// Package registry maps a diagram's opening keyword to the parser that handles
// it.
//
// Adding a Mermaid diagram type (sequenceDiagram, architecture-beta, ...) means
// implementing DiagramParser and registering it. Nothing in the scenario parser
// or the timeline compiler needs to change, because both work purely against
// the symbol.Table a DiagramParser returns.
package registry

import (
	"fmt"
	"sort"
	"sync"

	"github.com/tejaspanse/cinegram/pkg/ast"
	"github.com/tejaspanse/cinegram/pkg/diag"
	"github.com/tejaspanse/cinegram/pkg/source"
	"github.com/tejaspanse/cinegram/pkg/symbol"
)

// DiagramParser parses one Mermaid diagram type.
type DiagramParser interface {
	// Keywords lists the opening keywords this parser claims, e.g.
	// "flowchart" and "graph".
	Keywords() []string

	// Parse consumes the diagram header and body from c, stopping at end of
	// input or at the first top-level block that belongs to the scenario
	// half — `scenario`, `view` or `interact`. Problems are reported into b;
	// Parse should recover and keep going where it can so that one bad line
	// does not mask the rest of the file.
	Parse(c *source.Cursor, b *diag.Bag) (ast.Diagram, *symbol.Table)
}

var (
	mu        sync.RWMutex
	byKeyword = make(map[string]DiagramParser)
)

// Register makes p available for its keywords. It panics on a duplicate
// registration, which can only be a programming error.
func Register(p DiagramParser) {
	mu.Lock()
	defer mu.Unlock()
	for _, kw := range p.Keywords() {
		if _, dup := byKeyword[kw]; dup {
			panic(fmt.Sprintf("registry: duplicate diagram parser for %q", kw))
		}
		byKeyword[kw] = p
	}
}

// Lookup returns the parser registered for keyword.
func Lookup(keyword string) (DiagramParser, bool) {
	mu.RLock()
	defer mu.RUnlock()
	p, ok := byKeyword[keyword]
	return p, ok
}

// Keywords lists every registered keyword, sorted.
func Keywords() []string {
	mu.RLock()
	defer mu.RUnlock()
	out := make([]string, 0, len(byKeyword))
	for kw := range byKeyword {
		out = append(out, kw)
	}
	sort.Strings(out)
	return out
}
