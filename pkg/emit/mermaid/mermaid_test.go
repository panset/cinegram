package mermaid_test

import (
	"strings"
	"testing"

	"github.com/tejaspanse/diagramator/pkg/emit/mermaid"
	"github.com/tejaspanse/diagramator/pkg/parser"
)

// TestEmitDropsScenariosOnly is the language's central compatibility promise:
// take any source file, remove the scenario blocks, and what remains is exactly
// the Mermaid the author wrote.
func TestEmitDropsScenariosOnly(t *testing.T) {
	const src = `%% a leading comment
flowchart LR
  client[External Client]
  lb[(Cloud LB)]

  subgraph cluster[K8s Cluster]
    ing[Ingress]
    subgraph ns[namespace: prod]
      svc[Service]
    end
  end

  client --> lb
  lb -->|HTTP| ing
  ing -.-> svc

  classDef busy fill:#f96
  class svc busy
  click ing "https://example.com" "docs"

view detail "Detail" from "detail.dgm"

interact {
  click svc -> view detail { label: "zoom" }
}

scenario "req"
  step one "hop" {
    flow client -> lb { dur: 400ms }
    highlight ing
  }
`
	res, bag := parser.Parse("t.dgm", src)
	if bag.HasErrors() {
		t.Fatalf("unexpected diagnostics:\n%s", bag)
	}

	got := mermaid.Emit(res.Document)

	if strings.Contains(got, "scenario") || strings.Contains(got, "highlight") {
		t.Errorf("scenario content leaked into the Mermaid output:\n%s", got)
	}

	// View and interact blocks drop out for the same reason scenarios do —
	// but mermaid's own `click` directive in the diagram body must survive,
	// which is why this checks for the block rather than for the word.
	if strings.Contains(got, "interact") || strings.Contains(got, "view detail") {
		t.Errorf("interaction content leaked into the Mermaid output:\n%s", got)
	}

	// Statements the parser does not model semantically must survive verbatim.
	for _, want := range []string{
		"%% a leading comment",
		"client[External Client]",
		"lb[(Cloud LB)]",
		"subgraph cluster[K8s Cluster]",
		"subgraph ns[namespace: prod]",
		"lb -->|HTTP| ing",
		"ing -.-> svc",
		"classDef busy fill:#f96",
		"class svc busy",
		`click ing "https://example.com" "docs"`,
	} {
		if !strings.Contains(got, want) {
			t.Errorf("output is missing %q:\n%s", want, got)
		}
	}

	// Both subgraphs must be closed.
	if n := strings.Count(got, "\n  end\n") + strings.Count(got, "\n    end\n"); n != 2 {
		t.Errorf("expected two closing `end` lines, got %d:\n%s", n, got)
	}
}

// TestEmitReparses guards against the emitter producing something the parser
// can no longer read, which would break any tool chaining the two.
func TestEmitReparses(t *testing.T) {
	const src = `flowchart TD
  a[Start] --> b{Choice}
  b -->|yes| c[Do it]
  b -->|no| d[Skip]

scenario "walk"
  step s "go" {
    flow a -> b
  }
`
	first, bag := parser.Parse("t.dgm", src)
	if bag.HasErrors() {
		t.Fatalf("unexpected diagnostics:\n%s", bag)
	}
	emitted := mermaid.Emit(first.Document)

	second, bag2 := parser.Parse("emitted.mmd", emitted)
	if bag2.HasErrors() {
		t.Fatalf("emitted Mermaid failed to reparse:\n%s\n--- source ---\n%s", bag2, emitted)
	}

	if got, want := len(second.Symbols.Nodes()), len(first.Symbols.Nodes()); got != want {
		t.Errorf("reparse found %d nodes, want %d", got, want)
	}
	if got, want := len(second.Symbols.Edges()), len(first.Symbols.Edges()); got != want {
		t.Errorf("reparse found %d edges, want %d", got, want)
	}

	// Emission must be idempotent: emitting the reparsed document again
	// produces the same text.
	if again := mermaid.Emit(second.Document); again != emitted {
		t.Errorf("emit is not idempotent\n--- first ---\n%s\n--- second ---\n%s", emitted, again)
	}
}
