# Diagramator

Animated architecture diagrams from a Mermaid-like DSL.

A static diagram shows you that an Ingress sits in front of a Service. It cannot
show you what happens to a single `GET /api/orders` as it travels
LB → Ingress → Service → Pod and back. Diagramator adds a small animation
language on top of Mermaid so that path becomes something you can watch.

```
bazel run //cmd/diagramator -- preview examples/k8s-request.dgm -o /tmp/k8s.html
open /tmp/k8s.html
```

## The language

A `.dgm` file is a Mermaid diagram followed by one or more `scenario` blocks.

```
flowchart LR
  client[External Client]
  lb[(Cloud Load Balancer)]

  subgraph cluster[Kubernetes Cluster]
    ing[Ingress Controller]
    subgraph ns[namespace: prod]
      svc[ClusterIP Service]
      pod1[Pod A]
      pod2[Pod B]
    end
  end

  client --> lb
  lb --> ing
  ing --> svc
  svc --> pod1
  svc --> pod2

scenario "GET /api/orders" { speed: 1.0, loop: true }

  step route "Ingress rule matches host and path" {
    note ing "host: api.example.com\npath: /api/*"
    flow ing -> svc { dur: 500ms }
  }

  step respond "Response travels back to the client" {
    flow pod1 -> svc -> ing -> lb -> client {
      label: "200 OK", dur: 1400ms, style: response
    }
  }
```

Two properties shape everything else:

**The diagram half is untouched Mermaid.** Delete the scenario blocks and you
have a file any Mermaid renderer will draw. `diagramator mermaid` does exactly
that, and it is lossless by construction — statements are reprinted from their
original source text, so `classDef`, `click`, and any Mermaid syntax added after
this parser was written all survive verbatim.

**Actions inside a step start together; steps run one after another.** That
single rule covers most real scenarios. Reach for `seq { … }` when you need
actions inside one step to chain instead.

### Actions

| Action | Form | Notes |
| --- | --- | --- |
| `flow` | `flow a -> b -> c` | A packet travelling the hops. Each hop becomes its own track. |
| `highlight` | `highlight a, b` | Emphasise nodes. |
| `note` | `note a "text"` | A callout anchored to a node. `\n` works. |
| `dim` | `dim a` | Fade a node back. |
| `pulse` | `pulse a` | Repeating pulse. |
| `show` / `hide` | `show a` | Reveal or conceal. |
| `wait` | `wait 500ms` | Consume time, draw nothing. |
| `seq` | `seq { … }` | Run the contained actions in sequence. |

### Attributes

Written as `{ key: value, … }` after an action, or as bare `key: value` lines
inside a step or scenario body. Blocks may span lines.

- `label` — text carried by a flow
- `dur` — `600ms`, `1.2s`, or a bare number of milliseconds
- `delay`, `at` — offset the start
- `style` — a name the renderer maps to CSS (`response` and `busy` ship styled)
- `color`, `ease`, `repeat`, `bidi`
- scenario only: `speed`, `loop`, `autoplay`

Durations behave predictably:

- A `flow` with no `dur` takes 600ms per hop; with one, the total is split
  evenly across hops and always sums exactly to what you asked for.
- A stateful action (`highlight`, `dim`, `note`, …) with no `dur` spans its
  whole step, which is what makes `highlight ing` beside a flow do the obvious
  thing.
- A step lasts as long as its longest action unless given an explicit `dur`.

A flow may travel **against** the direction an edge was drawn. Response paths
are the normal case, so `flow pod1 -> svc` reuses the `svc --> pod1` edge and
marks the track reversed rather than demanding you draw a second arrow.

## Commands

```
diagramator compile <file.dgm> [-o out.json]   # animation timeline JSON
diagramator mermaid <file.dgm> [-o out.mmd]    # the diagram as plain Mermaid
diagramator preview <file.dgm> [-o out.html]   # self-contained animated page
diagramator lint    <file.dgm>                 # diagnostics only
```

Relative paths are resolved against the directory you ran the command from,
including under `bazel run` — the binary executes from its runfiles tree, so
Diagramator honours Bazel's `BUILD_WORKING_DIRECTORY` to get this right.
`preview` with no `-o` writes beside its input.

Warnings never fail a build; errors do. Diagnostics carry a line, a column, and
usually a suggestion:

```
errors.dgm:11:15: error: "ingres" is not a node in this diagram
  hint: did you mean ing?
errors.dgm:15:20: error: no edge between "client" and "svc" to animate along
  hint: add `client --> svc` to the diagram, or route the flow through nodes that are connected
```

## How it fits together

```
source.dgm
   │
   ├─ parser ──────────────► ast.Document + symbol.Table
   │    ├── flowchart.go        the diagram half (pluggable per diagram type)
   │    └── scenario.go         the animation half (diagram-agnostic)
   │
   ├─ compile ────────────► ir.Timeline      absolute-millisecond tracks
   │
   └─ emit
        ├── mermaid ───────► plain Mermaid source
        └── html ─────────► self-contained animated page
```

Layout is delegated to mermaid.js. Diagramator emits clean Mermaid, mermaid.js
produces the SVG, and the runtime animates particles along the real edge
geometry using `getPointAtLength`. There is no layout engine here to maintain.

Two design decisions are load-bearing and worth preserving:

- **The timeline holds no geometry.** Tracks reference node and edge IDs with
  absolute start/end times, so a renderer is a clock and a scrubber. The same
  timeline could drive a Go-native SVG backend later without recompilation.

- **The animation layer never mentions flowcharts.** A diagram parser's only
  obligation is to produce a `symbol.Table`; scenario parsing, validation, and
  compilation work against that alone. Adding `sequenceDiagram` or
  `architecture-beta` costs one parser in `pkg/parser` registered via
  `pkg/registry`, and zero changes anywhere else.

The runtime binds to the rendered SVG defensively: nodes by mermaid's
`flowchart-<id>-<n>` id, and edges by matching path endpoints against node
centres rather than by parsing mermaid's edge-id format, which has changed
between releases. Anything that fails to bind is reported in a banner on the
page instead of silently not animating.

## Building

Bazel with bzlmod. Go is not required locally — `rules_go` fetches a hermetic
SDK, and Gazelle runs through Bazel.

```
bazel build //...
bazel test  //...
bazel run   //:gazelle          # after adding or moving packages
```

Golden fixtures are regenerated through `bazel run` (which sets
`BUILD_WORKSPACE_DIRECTORY`), not `bazel test`:

```
bazel run //pkg/parser:parser_test   -- -update
bazel run //pkg/compile:compile_test -- -update
```

There are no third-party Go dependencies, and the intent is to keep it that
way: a hand-rolled lexer and recursive-descent parser need nothing beyond the
standard library.

`pkg/emit/html/assets/` holds the vendored `mermaid.min.js` plus the runtime's
`runtime.js` and `runtime.css`. They live inside the package because `go:embed`
cannot reach outside it; a future VS Code plugin should consume them from there
rather than keeping a second copy. The runtime is a classic script, not an ES
module, because module scripts are blocked on `file://` and awkward in webviews.

## Status

Working today: flowcharts (`flowchart` / `graph`) with every Mermaid node shape
and link form, nested subgraphs, frontmatter, scenarios, the timeline compiler,
and the animated HTML preview.

Not built yet: sequence diagrams and `architecture-beta` (the registry seam
exists for them), the VS Code plugin, WASM builds, and animated SVG/GIF export.
