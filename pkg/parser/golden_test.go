package parser

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/tejaspanse/diagramator/pkg/ast"
	"github.com/tejaspanse/diagramator/pkg/diag"
)

// update rewrites the .golden fixtures instead of comparing against them.
// Run it through `bazel run`, which sets BUILD_WORKSPACE_DIRECTORY:
//
//	bazel run //pkg/parser:parser_test -- -update
var update = flag.Bool("update", false, "rewrite golden files")

func TestGolden(t *testing.T) {
	sources, err := filepath.Glob("testdata/*.dgm")
	if err != nil || len(sources) == 0 {
		t.Fatalf("no testdata found: %v", err)
	}

	for _, src := range sources {
		name := strings.TrimSuffix(filepath.Base(src), ".dgm")
		t.Run(name, func(t *testing.T) {
			content, err := os.ReadFile(src)
			if err != nil {
				t.Fatal(err)
			}

			res, bag := Parse(filepath.Base(src), string(content))
			got := renderParse(res, bag)

			goldenPath := strings.TrimSuffix(src, ".dgm") + ".golden"
			if *update {
				writeGolden(t, goldenPath, got)
				return
			}

			want, err := os.ReadFile(goldenPath)
			if err != nil {
				t.Fatalf("missing golden file (run with -update): %v", err)
			}
			if got != string(want) {
				t.Errorf("parse output differs from %s\n--- got ---\n%s\n--- want ---\n%s",
					goldenPath, got, want)
			}
		})
	}
}

func writeGolden(t *testing.T, path, content string) {
	t.Helper()
	root := os.Getenv("BUILD_WORKSPACE_DIRECTORY")
	if root == "" {
		t.Fatalf("-update requires `bazel run`, which sets BUILD_WORKSPACE_DIRECTORY")
	}
	full := filepath.Join(root, "pkg", "parser", path)
	if err := os.WriteFile(full, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	t.Logf("wrote %s", full)
}

// renderParse renders a parse result as stable, reviewable text. It covers the
// symbol table (what scenarios are allowed to reference), the scenario tree,
// and every diagnostic, so one fixture exercises both halves of the language.
func renderParse(res *Result, bag *diag.Bag) string {
	var b strings.Builder

	if res.Document.Frontmatter != "" {
		fmt.Fprintf(&b, "frontmatter:\n%s\n\n", indentBlock(res.Document.Frontmatter, "  "))
	}

	if d := res.Document.Diagram; d != nil {
		fmt.Fprintf(&b, "diagram: %s (%s)\n", d.Kind(), d.Header())
		if fc, ok := d.(*ast.Flowchart); ok {
			fmt.Fprintf(&b, "direction: %s\n", fc.Direction)
		}
	} else {
		b.WriteString("diagram: <none>\n")
	}

	b.WriteString("\nnodes:\n")
	for _, n := range res.Symbols.Nodes() {
		fmt.Fprintf(&b, "  %-10s label=%-24q shape=%-10s", n.ID, n.Label, orDash(n.Shape))
		if n.Group != "" {
			fmt.Fprintf(&b, " group=%s", n.Group)
		}
		if n.Class != "" {
			fmt.Fprintf(&b, " class=%s", n.Class)
		}
		b.WriteByte('\n')
	}

	if groups := res.Symbols.Groups(); len(groups) > 0 {
		b.WriteString("\ngroups:\n")
		for _, g := range groups {
			fmt.Fprintf(&b, "  %-10s label=%-24q parent=%-10s children=[%s]\n",
				g.ID, g.Label, orDash(g.Parent), strings.Join(g.Children, " "))
		}
	}

	b.WriteString("\nedges:\n")
	for _, e := range res.Symbols.Edges() {
		fmt.Fprintf(&b, "  %-4s %s -> %s style=%s head=%s", e.ID, e.From, e.To, e.Style, e.Head)
		if e.Label != "" {
			fmt.Fprintf(&b, " label=%q", e.Label)
		}
		if e.Bidir {
			b.WriteString(" bidir")
		}
		b.WriteByte('\n')
	}

	if views := res.Document.Views; len(views) > 0 {
		b.WriteString("\nviews:\n")
		for _, v := range views {
			fmt.Fprintf(&b, "  %-10s title=%-20q from=%q\n", v.ID, v.Title, v.Path)
		}
	}

	if bindings := res.Document.Interactions; len(bindings) > 0 {
		b.WriteString("\ninteract:\n")
		for _, bd := range bindings {
			names := make([]string, 0, len(bd.Targets))
			for _, t := range bd.Targets {
				names = append(names, t.Name)
			}
			fmt.Fprintf(&b, "  click %-10s -> %s %s%s\n",
				bd.Source.Name, bd.Kind, strings.Join(names, ", "), renderAttrs(bd.Attrs))
		}
	}

	for _, sc := range res.Document.Scenarios {
		fmt.Fprintf(&b, "\nscenario %q%s\n", sc.Name, renderAttrs(sc.Attrs))
		for _, st := range sc.Steps {
			fmt.Fprintf(&b, "  step %s %q%s\n", orDash(st.ID), st.Name, renderAttrs(st.Attrs))
			renderActions(&b, st.Actions, "    ")
		}
	}

	if bag.Len() > 0 {
		b.WriteString("\ndiagnostics:\n")
		for _, d := range bag.All() {
			fmt.Fprintf(&b, "  %s: %s: %s\n", d.Pos, d.Severity, d.Msg)
			if d.Hint != "" {
				fmt.Fprintf(&b, "    hint: %s\n", d.Hint)
			}
		}
	}

	return b.String()
}

func renderActions(b *strings.Builder, actions []ast.Action, indent string) {
	for _, a := range actions {
		fmt.Fprintf(b, "%s%s", indent, a.Kind)
		names := make([]string, 0, len(a.Targets))
		for _, t := range a.Targets {
			names = append(names, t.Name)
		}
		if len(names) > 0 {
			sep := " "
			if a.Kind == ast.ActionFlow {
				sep = " -> "
			} else {
				sep = ", "
			}
			fmt.Fprintf(b, " %s", strings.Join(names, sep))
		}
		if a.Text != "" {
			fmt.Fprintf(b, " %q", a.Text)
		}
		b.WriteString(renderAttrs(a.Attrs))
		b.WriteByte('\n')
		if len(a.Body) > 0 {
			renderActions(b, a.Body, indent+"  ")
		}
	}
}

func renderAttrs(a ast.Attrs) string {
	keys := a.Keys()
	if len(keys) == 0 {
		return ""
	}
	parts := make([]string, 0, len(keys))
	for _, k := range keys {
		v, _ := a.Get(k)
		parts = append(parts, fmt.Sprintf("%s=%q", k, v.Raw))
	}
	return " {" + strings.Join(parts, " ") + "}"
}

func indentBlock(s, prefix string) string {
	lines := strings.Split(s, "\n")
	for i, l := range lines {
		lines[i] = prefix + l
	}
	return strings.Join(lines, "\n")
}

func orDash(s string) string {
	if s == "" {
		return "-"
	}
	return s
}
