package main

import (
	"fmt"
	"io"
	"os"
	"sort"
	"strings"

	"github.com/tejaspanse/cinegram/pkg/compile"
	"github.com/tejaspanse/cinegram/pkg/envelope"
	"github.com/tejaspanse/cinegram/pkg/loader"
)

// `lint --fix` applies the edits the diagnostics already carry.
//
// Nothing here decides what a fix is: pkg/parser attaches one only where its
// own "did you mean" bound was met, so the CLI's job is purely mechanical —
// verify the text is still what the compiler saw, splice, and say what it did.
// That split is what makes the feature safe to run unattended: a fix the parser
// would not have suggested cannot be invented at this layer.

// maxFixRounds bounds the load → compile → apply loop.
//
// A round has to re-parse, because applying an edit moves every position after
// it on the line and can uncover a diagnostic that was previously masked — a
// step id corrected in one round is a step id a `click … -> step` can resolve
// against in the next. The bound exists so that a pathological document (one
// where a fix re-creates the problem it solved) stops rather than spins; five is
// far more than the cascades observed, and hitting it simply leaves the
// remaining diagnostics to be reported as usual.
const maxFixRounds = 5

// runFixes applies the edits a bundle's diagnostics carry and hands back the
// bundle the caller should report from.
//
// Reporting comes from a bundle re-parsed after the last edit, so `--fix`
// composes with everything already there: the exit status is exactly the one a
// plain lint of the repaired file would earn, and `--strict` and
// `--format=json` need to know nothing about fixing at all.
func runFixes(input string, bundle *loader.Bundle, stderr io.Writer) (*loader.Bundle, error) {
	total := 0

	for round := 0; round < maxFixRounds; round++ {
		diags, _ := envelope.Collect(bundle.Bags())
		n, err := fixRound(diags, stderr)
		if err != nil {
			return nil, err
		}
		if n == 0 {
			break
		}
		total += n

		// Re-parse rather than track offsets: an edit moves every position
		// after it on its line, and a repaired reference can uncover the
		// problem that was standing behind it.
		if bundle, err = loader.Load(input, os.ReadFile); err != nil {
			return nil, err
		}
		compile.CompileBundle(bundle)
	}

	if total > 0 {
		word := "fixes"
		if total == 1 {
			word = "fix"
		}
		fmt.Fprintf(stderr, "applied %d %s\n", total, word)
	}
	return bundle, nil
}

// applyFixes rewrites path with every fix that still matches the file.
//
// Edits are grouped by line and applied right to left within each line, so an
// earlier column is untouched by a later splice and no offset bookkeeping is
// needed. Every edit is verified against the bytes on disk first: a fix whose
// Old is no longer there describes a file that has since changed, and is
// skipped and reported rather than guessed at. The same check quietly covers
// the escape cases — a quoted value the parser unescaped may not reconstruct
// byte-for-byte, and mismatching is exactly the right outcome there.
func applyFixes(path string, fixes []envelope.Fix) (applied int, skipped []envelope.Fix, err error) {
	if len(fixes) == 0 {
		return 0, nil, nil
	}

	info, err := os.Stat(path)
	if err != nil {
		return 0, nil, err
	}
	raw, err := os.ReadFile(path)
	if err != nil {
		return 0, nil, err
	}

	lines := strings.Split(string(raw), "\n")

	byLine := map[int][]envelope.Fix{}
	var order []int
	for _, f := range fixes {
		if byLine[f.Line] == nil {
			order = append(order, f.Line)
		}
		byLine[f.Line] = append(byLine[f.Line], f)
	}
	sort.Ints(order) // so what is skipped comes back in a stable order

	for _, line := range order {
		group := byLine[line]
		// Right to left, so that applying one edit cannot move the next.
		sort.SliceStable(group, func(i, j int) bool { return group[i].Col > group[j].Col })

		if line < 1 || line > len(lines) {
			skipped = append(skipped, group...)
			continue
		}
		text := lines[line-1]

		// A CRLF file arrives here with the carriage return still attached, and
		// the parser never saw it: positions are columns of the line proper. So
		// the terminator is set aside and put back, which is also what keeps a
		// mixed-ending file from being silently normalised.
		eol := ""
		if strings.HasSuffix(text, "\r") {
			text, eol = text[:len(text)-1], "\r"
		}

		for _, f := range group {
			start := f.Col - 1
			if start < 0 || start > len(text) || start+len(f.Old) > len(text) ||
				text[start:start+len(f.Old)] != f.Old {
				skipped = append(skipped, f)
				continue
			}
			text = text[:start] + f.New + text[start+len(f.Old):]
			applied++
		}
		lines[line-1] = text + eol
	}

	if applied == 0 {
		return 0, skipped, nil
	}
	if err := os.WriteFile(path, []byte(strings.Join(lines, "\n")), info.Mode().Perm()); err != nil {
		return 0, skipped, err
	}
	return applied, skipped, nil
}

// fixRound applies every fix in one collection of diagnostics, file by file,
// and reports each edit on stderr.
//
// Diagnostics are the payload of a lint run, so what a fix run says about
// itself belongs on stderr beside them — a `--format=json` fix run must still
// put nothing but the array on stdout.
func fixRound(diags []envelope.Diagnostic, stderr io.Writer) (applied int, err error) {
	byFile := map[string][]envelope.Fix{}
	var order []string
	seen := map[string]map[envelope.Fix]bool{}

	for _, d := range diags {
		if d.Fix == nil || d.File == "" {
			continue
		}
		// Two diagnostics may legitimately carry the same edit — several passes
		// read the same attribute. Applying it twice would leave the second
		// looking like a stale fix, so identical edits collapse here.
		if seen[d.File] == nil {
			seen[d.File] = map[envelope.Fix]bool{}
			order = append(order, d.File)
		}
		if seen[d.File][*d.Fix] {
			continue
		}
		seen[d.File][*d.Fix] = true
		byFile[d.File] = append(byFile[d.File], *d.Fix)
	}

	for _, file := range order {
		fixes := byFile[file]
		n, skipped, err := applyFixes(file, fixes)
		if err != nil {
			return applied, fmt.Errorf("fixing %s: %w", file, err)
		}
		applied += n

		stale := map[envelope.Fix]bool{}
		for _, f := range skipped {
			stale[f] = true
		}
		// Reported in reading order, not the order they were applied in.
		sort.SliceStable(fixes, func(i, j int) bool {
			if fixes[i].Line != fixes[j].Line {
				return fixes[i].Line < fixes[j].Line
			}
			return fixes[i].Col < fixes[j].Col
		})
		for _, f := range fixes {
			if stale[f] {
				fmt.Fprintf(stderr, "skipped %s:%d:%d: %q is no longer there\n", file, f.Line, f.Col, f.Old)
				continue
			}
			fmt.Fprintf(stderr, "fixed %s:%d:%d: %s -> %s\n", file, f.Line, f.Col, f.Old, f.New)
		}
	}
	return applied, nil
}
