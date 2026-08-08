package parser

import (
	"testing"

	"github.com/tejaspanse/diagramator/pkg/ast"
)

func TestSplitLinks(t *testing.T) {
	tests := []struct {
		name  string
		in    string
		segs  []string
		links []ast.Link
	}{
		{
			name: "no link",
			in:   "client[External Client]",
			segs: []string{"client[External Client]"},
		},
		{
			name:  "simple arrow",
			in:    "a --> b",
			segs:  []string{"a ", " b"},
			links: []ast.Link{{Style: ast.LinkNormal, Head: ast.ArrowNormal, Length: 2, Operator: "-->"}},
		},
		{
			name: "chain",
			in:   "a --> b --> c",
			segs: []string{"a ", " b ", " c"},
			links: []ast.Link{
				{Style: ast.LinkNormal, Head: ast.ArrowNormal, Length: 2, Operator: "-->"},
				{Style: ast.LinkNormal, Head: ast.ArrowNormal, Length: 2, Operator: "-->"},
			},
		},
		{
			name:  "open link needs three dashes",
			in:    "a --- b",
			segs:  []string{"a ", " b"},
			links: []ast.Link{{Style: ast.LinkNormal, Head: ast.ArrowNone, Length: 3, Operator: "---"}},
		},
		{
			// The disambiguation rule: exactly two dashes with no arrowhead
			// opens a labelled link rather than closing a plain one.
			name:  "mid-text label",
			in:    "a -- carries --> b",
			segs:  []string{"a ", " b"},
			links: []ast.Link{{Style: ast.LinkNormal, Head: ast.ArrowNormal, Label: "carries", Length: 2, Operator: "-- carries -->"}},
		},
		{
			name:  "mid-text label without arrowhead",
			in:    "a -- carries --- b",
			segs:  []string{"a ", " b"},
			links: []ast.Link{{Style: ast.LinkNormal, Head: ast.ArrowNone, Label: "carries", Length: 2, Operator: "-- carries ---"}},
		},
		{
			name:  "pipe label",
			in:    "a -->|GET /api| b",
			segs:  []string{"a ", " b"},
			links: []ast.Link{{Style: ast.LinkNormal, Head: ast.ArrowNormal, Label: "GET /api", Length: 2, Operator: "-->"}},
		},
		{
			name:  "dotted",
			in:    "a -.-> b",
			segs:  []string{"a ", " b"},
			links: []ast.Link{{Style: ast.LinkDotted, Head: ast.ArrowNormal, Length: 2, Operator: "-.->"}},
		},
		{
			name:  "dotted with label",
			in:    "a -. retry .-> b",
			segs:  []string{"a ", " b"},
			links: []ast.Link{{Style: ast.LinkDotted, Head: ast.ArrowNormal, Label: "retry", Length: 2, Operator: "-. retry .->"}},
		},
		{
			name:  "thick",
			in:    "a ==> b",
			segs:  []string{"a ", " b"},
			links: []ast.Link{{Style: ast.LinkThick, Head: ast.ArrowNormal, Length: 2, Operator: "==>"}},
		},
		{
			name:  "thick with label",
			in:    "a == bulk ==> b",
			segs:  []string{"a ", " b"},
			links: []ast.Link{{Style: ast.LinkThick, Head: ast.ArrowNormal, Label: "bulk", Length: 2, Operator: "== bulk ==>"}},
		},
		{
			name:  "circle head",
			in:    "a --o b",
			segs:  []string{"a ", " b"},
			links: []ast.Link{{Style: ast.LinkNormal, Head: ast.ArrowCircle, Length: 2, Operator: "--o"}},
		},
		{
			name:  "cross head",
			in:    "a --x b",
			segs:  []string{"a ", " b"},
			links: []ast.Link{{Style: ast.LinkNormal, Head: ast.ArrowCross, Length: 2, Operator: "--x"}},
		},
		{
			name:  "bidirectional",
			in:    "a <--> b",
			segs:  []string{"a ", " b"},
			links: []ast.Link{{Style: ast.LinkNormal, Head: ast.ArrowNormal, Bidir: true, Length: 2, Operator: "<-->"}},
		},
		{
			name:  "long link",
			in:    "a ----> b",
			segs:  []string{"a ", " b"},
			links: []ast.Link{{Style: ast.LinkNormal, Head: ast.ArrowNormal, Length: 4, Operator: "---->"}},
		},
		{
			// Arrows inside a quoted label must not be seen as links.
			name: "arrow inside label",
			in:   `a["x --> y"]`,
			segs: []string{`a["x --> y"]`},
		},
		{
			name:  "inline declarations on both ends",
			in:    "client[Client] --> ing[Ingress]",
			segs:  []string{"client[Client] ", " ing[Ingress]"},
			links: []ast.Link{{Style: ast.LinkNormal, Head: ast.ArrowNormal, Length: 2, Operator: "-->"}},
		},
		{
			// A hyphen inside a bracketed label is not a link operator.
			name: "hyphens inside label",
			in:   "svc[kube-proxy --- internal]",
			segs: []string{"svc[kube-proxy --- internal]"},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			segs, offs, links := splitLinks(tc.in)

			if len(segs) != len(tc.segs) {
				t.Fatalf("got %d segments %q, want %d %q", len(segs), segs, len(tc.segs), tc.segs)
			}
			for i := range segs {
				if segs[i] != tc.segs[i] {
					t.Errorf("segment %d = %q, want %q", i, segs[i], tc.segs[i])
				}
				if got := tc.in[offs[i] : offs[i]+len(segs[i])]; got != segs[i] {
					t.Errorf("offset %d does not point at segment %q, got %q", i, segs[i], got)
				}
			}
			if len(segs) != len(links)+1 {
				t.Fatalf("invariant broken: %d segments for %d links", len(segs), len(links))
			}
			if len(links) != len(tc.links) {
				t.Fatalf("got %d links, want %d", len(links), len(tc.links))
			}
			for i, want := range tc.links {
				if links[i].link != want {
					t.Errorf("link %d =\n  %+v\nwant\n  %+v", i, links[i].link, want)
				}
			}
		})
	}
}

func TestSplitLinksChainIsContiguous(t *testing.T) {
	// Segments and operators together must reconstruct the input exactly;
	// otherwise a hop would silently lose text.
	in := "a -- x --> b -.-> c ==> d"
	segs, _, links := splitLinks(in)

	var rebuilt string
	for i, s := range segs {
		rebuilt += s
		if i < len(links) {
			rebuilt += links[i].link.Operator
		}
	}
	if rebuilt != in {
		t.Fatalf("reconstruction = %q, want %q", rebuilt, in)
	}
}

func TestStripComment(t *testing.T) {
	tests := []struct{ in, code, comment string }{
		{"a --> b", "a --> b", ""},
		{"a --> b %% trailing", "a --> b", "%% trailing"},
		{`a["100%% done"]`, `a["100%% done"]`, ""},
		{"%% whole line", "", "%% whole line"},
	}
	for _, tc := range tests {
		code, comment := stripComment(tc.in)
		if code != tc.code || comment != tc.comment {
			t.Errorf("stripComment(%q) = (%q, %q), want (%q, %q)", tc.in, code, comment, tc.code, tc.comment)
		}
	}
}
