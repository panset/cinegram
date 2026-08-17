<!-- Generated from README.md by `bazel run //site:sync`. Do not edit: //site:site_test fails while this file disagrees with its source, and the next sync overwrites it. -->

# The project

## How it fits together

```
source.dgm
   │
   ├─ loader ──────────────► the file and every `view` it references
   │
   ├─ parser ──────────────► ast.Document + symbol.Table   (per file, no I/O)
   │    ├── flowchart.go        the diagram half (pluggable per diagram type)
   │    ├── scenario.go         the animation half (diagram-agnostic)
   │    └── interact.go         the interaction half (diagram-agnostic)
   │
   ├─ compile ────────────► ir.Timeline      one View per diagram,
   │                                         absolute-millisecond tracks
   └─ emit
        ├── mermaid ───────► plain Mermaid source
        └── html ─────────► self-contained animated page
```

Layout is delegated to mermaid.js. Cinegram emits clean Mermaid, mermaid.js
produces the SVG, and the runtime animates particles along the real edge
geometry using `getPointAtLength`. There is no layout engine here to maintain.

Two properties follow from that shape. The timeline holds **no geometry** —
tracks are node and edge IDs with absolute start/end times — so a renderer is a
clock and a scrubber. And the animation layer **never mentions flowcharts**: a
diagram parser's only obligation is to produce a `symbol.Table`, so adding
`sequenceDiagram` or `architecture-beta` costs one parser and nothing else.

The runtime binds to the rendered SVG defensively, and anything that fails to
bind is reported in a banner on the page instead of silently not animating.

Contributing? `.claude/skills/rendering-pipeline/` is the working reference for
all of it.

## Building

Bazel with bzlmod. Go is not required locally — `rules_go` fetches a hermetic
SDK, and Gazelle runs through Bazel.

```
bazel build //...
bazel test  //...
bazel run   //:gazelle          # after adding or moving packages
```

There are no third-party Go dependencies, and the intent is to keep it that
way: a hand-rolled lexer and recursive-descent parser need nothing beyond the
standard library. The website at <https://panset.github.io/cinegram/> is the
one exception, and it is held at arm's length: Zensical renders `www/` in the
`pages` workflow at a pinned version, and Bazel's job is to guarantee the
input. `bazel run //site:sync` writes the generated half of `www/` — the
examples tour and this README, cut into guide pages — and `//site:site_test`
fails while the committed copy is stale, so the renderer is never handed
anything out of date.

`.claude/skills/build/` has the rest: golden fixtures, formatting, and the
invariants a change has to preserve.

## Status

Working today: flowcharts (`flowchart` / `graph`) with every Mermaid node shape
and link form, `sequenceDiagram`, `stateDiagram-v2` with composites and
pseudostates, nested subgraphs, frontmatter, scenarios with
narration and persistent state, focus, deep links and embedding, the timeline
compiler, clickable drill-down between diagrams of different types, the
animated HTML preview, a serve/watch authoring loop, `narrate`, PNG frame
capture, GIF/mp4/webm recording, and the VS Code extension — ```dgm blocks
animate inside the built-in Markdown preview, and `.dgm` files get syntax
highlighting, a preview panel, an *Open With… → Cinegram Animation* editor, and
`Cinegram: Export Animation…` to record one straight to a GIF (see
`editors/vscode/`).

Not built yet: `architecture-beta` and the other Mermaid diagram types (the
registry seam exists for them).

## The VS Code extension

`editors/vscode/README.md` is the Marketplace listing and is written for someone
installing the extension. Building, packaging and publishing it are in
[`editors/vscode/CONTRIBUTING.md`](https://github.com/panset/cinegram/blob/main/editors/vscode/CONTRIBUTING.md).

Each package carries the `cinegram` binary for one platform, because the
extension shells out to the compiler rather than reimplementing any of it, and
VS Code installs the package matching the machine.

## License

[MIT](https://github.com/panset/cinegram/blob/main/LICENSE).
