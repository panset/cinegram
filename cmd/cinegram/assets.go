package main

import (
	"bytes"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"

	"github.com/tejaspanse/cinegram/pkg/embedkit"
)

// cmdAssets writes the embed kit — the loader, its stylesheet and the three
// runtime files — into a directory a site serves.
//
// It exists because the alternative was a lie. A site that embeds cinegrams
// needs those five files beside each other and had no way to get them: they
// live inside this binary, and the only command that ever wrote them out was
// `site`, which writes a whole site around them. A downstream site was
// building into a temporary folder and copying three files out of the
// wreckage, which works right up until `site` changes what it emits.
//
// The output is generated, not editable: the command overwrites without
// asking, because the ordinary reason to run it twice is that cinegram was
// upgraded and the files should change.
func cmdAssets(args []string, stdout, stderr io.Writer) error {
	var output string
	fs := flag.NewFlagSet("assets", flag.ContinueOnError)
	fs.SetOutput(stderr)
	fs.StringVar(&output, "o", "", "directory to write the kit into")
	if err := fs.Parse(hoistFlags(args, valueFlagNames(fs))); err != nil {
		return err
	}
	if fs.NArg() > 0 {
		return fmt.Errorf("assets takes no input file, only -o DIR (got %q)", fs.Arg(0))
	}
	if output == "" {
		return fmt.Errorf("assets needs -o DIR")
	}

	kit := embedkit.Assets()
	names := make([]string, 0, len(kit))
	for name := range kit {
		names = append(names, name)
	}
	sort.Strings(names)

	if err := os.MkdirAll(output, 0o755); err != nil {
		return err
	}
	written := 0
	for _, name := range names {
		to := filepath.Join(output, name)
		// Skip a byte-identical file rather than rewriting it. mermaid.min.js
		// is 2.7 MB and changes about once a year; touching its mtime on every
		// build is enough to make a Make rule or a watcher think it has work.
		if old, err := os.ReadFile(to); err == nil && bytes.Equal(old, kit[name]) {
			continue
		}
		if err := os.WriteFile(to, kit[name], 0o644); err != nil {
			return err
		}
		written++
	}
	fmt.Fprintf(stdout, "%s: %d of %d files written, %d already current\n",
		output, written, len(names), len(names)-written)
	return nil
}
