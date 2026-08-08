# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Commands

Bazel with bzlmod is the only build system. **Go is not installed locally** —
`rules_go` fetches a hermetic SDK and Gazelle runs through Bazel, so never
reach for a bare `go build` or `go test`.

```sh
bazel build //...
bazel test  //...
bazel run   //:gazelle                  # after adding, moving or renaming packages
```

Run a single test:

```sh
bazel test //pkg/parser:parser_test --test_filter=TestSplitLinks
bazel test //pkg/compile:compile_test --test_output=all
```

**Regenerate golden fixtures with `bazel run`, not `bazel test`.** The `-update`
flag writes through `BUILD_WORKSPACE_DIRECTORY`, which only `bazel run` sets;
under `bazel test` the sandbox makes the write fail:

```sh
bazel run //pkg/parser:parser_test   -- -update
bazel run //pkg/compile:compile_test -- -update
```

Formatting — `gofmt` exists only inside the hermetic SDK, and Bazel does not
check it, so run it manually before finishing:

```sh
"$(find "$(bazel info output_base)/external" -name gofmt -type f | head -1)" -w ./cmd ./pkg
```

Exercise the CLI (relative paths work; the binary resolves them against
`BUILD_WORKING_DIRECTORY`):

```sh
bazel run //cmd/diagramator -- preview examples/k8s-request.dgm -o /tmp/k8s.html
bazel run //cmd/diagramator -- compile examples/k8s-request.dgm
bazel run //cmd/diagramator -- mermaid examples/k8s-request.dgm
bazel run //cmd/diagramator -- lint    examples/k8s-request.dgm
```

## Architecture

A `.dgm` source is a Mermaid diagram body followed by `scenario` blocks. The
pipeline is `parser → compile → emit`:

```
source.dgm ─► pkg/parser ─► ast.Document + symbol.Table
                              │
                              ├─► pkg/compile ─► ir.Timeline
                              │
                              └─► pkg/emit/{mermaid,html}
```

### The two halves never meet except through symbol.Table

This is the load-bearing decision in the codebase. `pkg/parser/flowchart.go`
parses the diagram; `pkg/parser/scenario.go` parses the animation and contains
**no diagram vocabulary at all** — it emits nothing but named references.
`pkg/parser/validate.go` is the only place the halves meet, and it works purely
against `symbol.Table`.

Adding a Mermaid diagram type (`sequenceDiagram`, `architecture-beta`) therefore
means writing one parser that implements `registry.DiagramParser` and registers
itself in `init()`. Scenario parsing, validation, and the timeline compiler need
zero changes. **Keep flowchart specifics out of `scenario.go`, `validate.go`,
`pkg/compile`, and `pkg/ir`.**

### Mermaid emission is a reprint, not a regeneration

Every `ast.Statement` carries its original source text via `Raw()`, and
`pkg/emit/mermaid` only re-indents and drops scenario blocks. Anything the
parser does not model semantically falls through to `ast.RawStmt` and
round-trips verbatim — that is what keeps `classDef`, `click`, and future
Mermaid syntax working. When adding diagram syntax support, preserve this: parse
for the symbol table, never rewrite the text.

### The timeline is the renderer contract

`pkg/ir` holds absolute integer milliseconds and **no geometry** — tracks
reference node and edge IDs only. Two consequences worth protecting: a renderer
is just a clock plus a scrubber, and a Go-native SVG backend remains possible
later without touching the compiler. Do not add coordinates to `ir`.

Timing rules live entirely in `pkg/compile`:

- Actions inside a step start together; steps run in sequence. `seq { }` chains.
- A `flow` splits its `dur` across hops using `total*i/hops` so hops tile the
  duration exactly regardless of remainder.
- A stateful action (`highlight`, `dim`, `note`, …) with no `dur` spans its
  whole step.
- A flow may run against an edge's drawn direction; `symbol.Table.FindEdge`
  matches reversed edges and the track records `Reverse`.

### Parsing strategy differs by half, deliberately

The diagram body is **line-oriented** (like Mermaid itself) and read through
`source.Cursor`. The scenario block is **brace-structured** and gets a real
tokenizer in `pkg/parser/scanner.go`.

`pkg/parser/link.go` implements Mermaid's link disambiguation, which is subtle
and easy to break: an operator of exactly two dashes or equals signs with no
arrowhead is the *opening half* of a labelled link (`A-- text -->B`); three or
more, or any arrowhead, closes it immediately. `link_test.go` pins this — run it
after touching that file.

### Runtime binds to the SVG defensively

`pkg/emit/html/assets/runtime.js` finds nodes by mermaid's
`flowchart-<id>-<n>` id, but matches **edges geometrically** by comparing path
endpoints to node centres. This is intentional: mermaid's edge-id format has
changed between releases, and geometric matching additionally detects paths
mermaid drew from the far end (composed with `Reverse` as `!reverse !== !flip`).
Unbound nodes or edges surface in a warning banner on the page rather than
silently failing to animate.

## Constraints

- **No third-party Go dependencies.** Standard library only. A hand-rolled lexer
  and recursive-descent parser need nothing more, and it keeps the build
  hermetic with no `go_deps` lockfile churn.
- **Runtime assets must live in `pkg/emit/html/assets/`.** `go:embed` cannot
  reach outside its own package directory. That directory is the single
  canonical copy of `runtime.js`, `runtime.css` and the vendored
  `mermaid.min.js` (2.7 MB, committed because the build needs it). Gazelle
  generates `embedsrcs` from the `//go:embed` directives automatically.
- **`runtime.js` is a classic script, never an ES module.** Module scripts are
  blocked on `file://` by CORS and are awkward in VS Code webviews.
- **The preview page must stay self-contained** — no external URLs at all, since
  it has to work from the filesystem and under a webview CSP.
  `pkg/emit/html/html_test.go` enforces this.
- Diagnostics carry a position and usually a `Hint`; `diag.Bag` de-duplicates
  identical entries because several passes read the same attribute.
- Warnings never fail a build; errors exit 1.

## Verifying animation changes

Bazel tests cover the parser, compiler and emitters, but not the browser
runtime. To check a runtime change actually renders, serve the page (the Chrome
extension blocks `file://`) and drive the player, which is exposed as
`window.DIAGRAMATOR_PLAYER`:

```sh
bazel run //cmd/diagramator -- preview examples/k8s-request.dgm -o /tmp/dgm/k8s.html
(cd /tmp/dgm && python3 -m http.server 8731)   # http://127.0.0.1:8731/k8s.html
```

`DIAGRAMATOR_PLAYER.seek(ms)` jumps to a moment deterministically, which is far
more reliable for verification than watching playback.
