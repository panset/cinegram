package sitegen

import (
	"regexp"
	"sort"
	"strings"
	"testing"

	"github.com/tejaspanse/cinegram/pkg/emit/html"
)

// The bounded shell is stated in two sheets. runtime.css hands the player root
// the whole viewport so the narration and the transport stay on screen while a
// tall diagram scales into what is left; site.css divides that same viewport
// between the site chrome — sidebar, breadcrumbs, prev/next — and the player,
// because a sited page has a header and a footer the standalone one does not.
//
// Both give up at the same points. Past a certain narrowness or shortness there
// is no height left to divide, and each sheet relaxes its half back into an
// ordinary scrolling page. Relax one and not the other and the band between
// them is the worst of both: a player laid out to grow, inside a column still
// clipped to the window, with the bottom of the walkthrough cut off.
//
// Only a comment in each sheet ties those numbers together today. This is the
// tripwire that makes moving one without the other fail out loud.
func TestShellHatchesAgree(t *testing.T) {
	runtimeCSS := string(html.Assets()["runtime.css"])
	if runtimeCSS == "" {
		t.Fatal("pkg/emit/html no longer ships runtime.css; half of this check has gone missing")
	}

	// The two hatches, as the sheets state them: the narrow one and the short
	// one. Written out rather than derived, so adding a third is a deliberate
	// edit here as well as there.
	want := []string{"(max-height: 520px)", "(max-width: 900px)"}

	cases := []struct {
		name  string
		sheet string
		// relax matches the rule that hands the shell's own element back its
		// content height — the thing each hatch does, in that sheet's terms.
		relax *regexp.Regexp
	}{
		{
			name:  "runtime.css",
			sheet: runtimeCSS,
			relax: regexp.MustCompile(`(?s)\.dgm-standalone \.dgm:not\(\.dgm-reel\):not\(\.dgm-present\)\s*\{[^}]*height:\s*auto`),
		},
		{
			name:  "site.css",
			sheet: siteCSS,
			relax: regexp.MustCompile(`(?s)body\.dgm-sited\s*\{[^}]*height:\s*auto`),
		},
	}

	for _, c := range cases {
		got := hatchesIn(c.sheet, c.relax)
		if len(got) == 0 {
			t.Errorf("%s: found no rule relaxing the bounded shell; either the hatches went away or this test no longer recognises them", c.name)
			continue
		}
		if strings.Join(got, " ") != strings.Join(want, " ") {
			t.Errorf("%s relaxes the shell under %v, want %v — the two sheets have to give up at the same width and the same height", c.name, got, want)
		}
	}
}

// hatchesIn reports the @media conditions the given rule sits under, sorted and
// deduplicated. An occurrence outside every media block reports the empty
// string, which no expectation matches — an unconditional relax is a broken
// shell, not a hatch.
func hatchesIn(sheet string, relax *regexp.Regexp) []string {
	sheet = cssComments.ReplaceAllString(sheet, "")
	seen := map[string]bool{}
	for _, m := range relax.FindAllStringIndex(sheet, -1) {
		seen[mediaAt(sheet, m[0])] = true
	}
	out := make([]string, 0, len(seen))
	for cond := range seen {
		out = append(out, cond)
	}
	sort.Strings(out)
	return out
}

var cssComments = regexp.MustCompile(`(?s)/\*.*?\*/`)

// mediaAt returns the innermost @media condition open at pos, or "" if none is.
// It tracks brace depth rather than taking the nearest preceding @media, so a
// rule written after a media block closes is correctly read as unconditional.
func mediaAt(sheet string, pos int) string {
	var open []string
	for i := 0; i < pos; {
		switch {
		case strings.HasPrefix(sheet[i:], "@media"):
			brace := strings.IndexByte(sheet[i:], '{')
			if brace < 0 {
				return ""
			}
			open = append(open, strings.TrimSpace(sheet[i+len("@media"):i+brace]))
			i += brace + 1
		case sheet[i] == '{':
			open = append(open, "")
			i++
		case sheet[i] == '}':
			if len(open) > 0 {
				open = open[:len(open)-1]
			}
			i++
		default:
			i++
		}
	}
	for i := len(open) - 1; i >= 0; i-- {
		if open[i] != "" {
			return open[i]
		}
	}
	return ""
}
