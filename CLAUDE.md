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

**`docs/` is the committed GitHub Pages site — regenerate it, never edit it.**
Pages serves it straight from `main` with no CI build, so after changing
anything under `examples/` or in the render pipeline:

```sh
bazel run //site:sync
```

`//site:site_test` fails while `docs/` is stale. The sweep only touches
`docs/demos/`; a hand-placed top-level file there (a `CNAME`, say) survives.
The full publish workflow — what gets a page, how the index blurb is chosen —
is `skills/publish-site/SKILL.md`.

Formatting — `gofmt` exists only inside the hermetic SDK, and Bazel does not
check it, so run it manually before finishing:

```sh
"$(find "$(bazel info output_base)/external" -name gofmt -type f | head -1)" -w ./cmd ./pkg ./site
```

Exercise the CLI (relative paths work; the binary resolves them against
`BUILD_WORKING_DIRECTORY`):

```sh
bazel run //cmd/cinegram -- preview examples/k8s-request.dgm -o /tmp/k8s.html
bazel run //cmd/cinegram -- compile examples/k8s-request.dgm
bazel run //cmd/cinegram -- mermaid examples/k8s-request.dgm
bazel run //cmd/cinegram -- lint    examples/k8s-request.dgm
bazel run //cmd/cinegram -- record  examples/payment-checkout.dgm -o /tmp/out.gif --fps 10
```

`record` shells out to the same headless Chrome `frame` uses, once per frame in
a small worker pool, then encodes. GIF goes through `pkg/gifenc` — stdlib only,
so it works with nothing installed; `--format mp4|webm` needs ffmpeg on `PATH`
or `$CINEGRAM_FFMPEG`.

## Architecture

A `.dgm` source is a Mermaid diagram body followed by `scenario`, `view` and
`interact` blocks. The pipeline is `loader → parser → compile → emit`:

```
source.dgm ─► pkg/loader ─► the file and every `view` it references
                  │
                  └─► pkg/parser ─► ast.Document + symbol.Table   (per file)
                                      │
                                      ├─► pkg/compile ─► ir.Timeline
                                      │
                                      └─► pkg/emit/{mermaid,html}
```

**`pkg/parser` does no I/O and must stay that way.** `Parse(filename, content)`
takes a string, so the parser works from a webview or a WASM build with no
filesystem. `pkg/loader` resolves `view … from "path"` declarations and takes
its read function as an argument, so it is testable against an in-memory map.

### The two halves never meet except through symbol.Table

This is the load-bearing decision in the codebase. `pkg/parser/flowchart.go`
parses the diagram; `pkg/parser/scenario.go` and `pkg/parser/interact.go` parse
the animation and the click bindings and contain **no diagram vocabulary at
all** — they emit nothing but named references. `pkg/parser/validate.go` is the
only place the halves meet, and it works purely against `symbol.Table`.

Adding a Mermaid diagram type (`architecture-beta`, say) therefore means writing
one parser that implements `registry.DiagramParser` and registers itself in
`init()`. Scenario parsing, validation, and the timeline compiler need zero
changes. **Keep flowchart specifics out of `scenario.go`, `interact.go`,
`validate.go`, `pkg/compile`, and `pkg/ir`.**

`pkg/parser/sequence.go` is the worked proof of that claim: `sequenceDiagram`
cost one parser plus one runtime indexer, and nothing in between. The runtime
is the part that does need a second strategy per diagram type — see below.

A diagram parser must hand the cursor back at every keyword in
`isTopLevelKeyword` (`interact.go`), not just at `scenario` — that list is the
contract in `registry.DiagramParser.Parse`.

### Mermaid emission is a reprint, not a regeneration

Every `ast.Statement` carries its original source text via `Raw()`, and
`pkg/emit/mermaid` only re-indents and drops scenario blocks. Anything the
parser does not model semantically falls through to `ast.RawStmt` and
round-trips verbatim — that is what keeps `classDef`, `click`, and future
Mermaid syntax working. When adding diagram syntax support, preserve this: parse
for the symbol table, never rewrite the text.

### The timeline is the renderer contract

`pkg/ir` holds absolute integer milliseconds and **no geometry** — tracks and
bindings reference node, edge, group and storyboard-frame IDs only. Storyboard
images arrive as `data:` URIs the loader built, never as paths: `pkg/parser`
does no I/O, and the page has to work from the filesystem. Two consequences worth
protecting: a renderer is just a clock plus a scrubber, and a Go-native SVG
backend remains possible later without touching the compiler. Do not add
coordinates to `ir`.

A `Timeline` holds one `View` per diagram in the bundle, plus a `Root` naming
the one to open on. `Compile` still lowers a single document (wrapping it in one
view); `CompileBundle` lowers a whole `loader.Bundle`.

Timing rules live entirely in `pkg/compile`:

- Actions inside a step start together; steps run in sequence. `seq { }` chains.
- A `flow` splits its `dur` across hops using `total*i/hops` so hops tile the
  duration exactly regardless of remainder.
- A stateful action (`highlight`, `dim`, `note`, …) with no `dur` spans its
  whole step.
- A flow may run against an edge's drawn direction; `symbol.Table.FindEdge`
  matches reversed edges and the track records `Reverse`.
- `scene` is a stateful action like any other here, but its target is a
  **storyboard frame**, not a node: it is the one action `validate.go` resolves
  against `doc.Storyboards` instead of the symbol table, and the runtime shows
  the latest scene track with `Start <= t` rather than the ones open at t. That
  stickiness is what makes the panel hold a screen across the steps where
  nothing the user can see changes.
- A `scene` inside a `seq` costs **zero** of the chain (`seqSpan`), like a
  persistent action: it fires where the chain has reached and the panel then
  holds, so `seq { flow a -> b; scene x }` means "the screen changes when the
  arrow lands" without the author computing an `at:`.
- `scenario … { variant: "base", until: <step> }` is spliced in
  `resolveVariants` **before** any timing runs, so the merged scenario is an
  ordinary `ast.Scenario` and every rule above applies to it unchanged.
  Depth-1 only; `until` is inclusive. Keep the splice at AST level — lowering
  it would mean reimplementing hop-occurrence and persistent-window resets.

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
`<renderer>-<id>-<n>` id, but matches **edges geometrically** by comparing path
endpoints to node centres. This is intentional: mermaid's edge-id format has
changed between releases, and geometric matching additionally detects paths
mermaid drew from the far end (composed with `Reverse` as `!reverse !== !flip`).
A backwards reading pays `REVERSE_COST`, so that two arrows running between the
same pair in opposite directions each take the one drawn for it instead of
scoring identically and swapping. Unbound nodes, edges or click sources surface
in a warning banner on the page rather than silently failing.

There are **two strategies, not one per diagram type**, and `index()` picks
between them on `ir.Diagram.Type`. A state diagram reuses the flowchart's whole
apparatus — `indexNodesBy`, `indexClusters`, `indexEdges`, `makeLayer` — and
differs in exactly two details: mermaid's id prefix is `state-` rather than
`flowchart-` (and its composite clusters are `g.statediagram-cluster` with no
counter suffix), and the node lookup handed to `indexEdges` is merged with the
clusters, because a transition into a composite stops at the cluster's border
rather than at any node. That merge is passed **only** in the state branch, so
flowchart matching is byte-identical.

A sequence diagram is the one that genuinely needs the second strategy: it has
neither `g.node` nor `.edgePaths`. Actors are recovered by **column** — the parts
of one actor are loose rects and texts sharing a lifeline x — and wrapped in a
`g.dgm-actor` so that every existing `.dgm-highlight rect` rule applies
unchanged. Messages are matched to edges by **order**, because mermaid draws
them top to bottom in message order and that is far more robust than recovering
identity from geometry. Only the direction a line was drawn in is read from
geometry, and it composes with `Reverse` exactly as the flowchart `flip` does.

One `Player` hosts every view and swaps between them, rather than one Player per
view: `build()` installs a **document-level** keydown handler, which would stack
up and fight over Space and the arrow keys if the chrome were built more than
once. Click listeners attach in `index()`, the one place that runs per mermaid
render with the id→element maps in hand; they live on SVG elements that
`render()` replaces wholesale, so they never need removing.

The storyboard panel is overlay-style HTML built once in `build()` and shown or
hidden by `syncBoard()`, which re-runs per render *and* per scenario change
because scene usage is per scenario. It is outside the SVG, so `baseClass` and
`STICKY` do not apply to it; `applyBoard` diffs on the frame id instead, and has
to — rewriting an `<img src>` every frame would restart the crossfade
transition forever.

**`baseClass` strips `dgm-*` classes and `applyNodeStates` rewrites the whole
class attribute every frame.** Anything that must outlive a frame — the click
affordance, reveal state — has to be listed in `STICKY` or the clock will erase
it the moment a node animates.

Navigation goes through `location.hash`: `navigate()` only moves the hash and
`applyHash()` does the work, so a click and a browser history move follow the
same path and the Back button cannot disagree with the browser's own.

### The VS Code extension is a consumer, not a second implementation

`editors/vscode/` renders ```` ```dgm ```` blocks inside VS Code's **built-in**
Markdown preview. It holds no diagram, scenario or timing knowledge: it shells
out to `cinegram compile - --as <path> --envelope` and mounts whatever timeline
comes back, so a new diagram type, action or timing rule reaches the preview
with no change there. Keep it that way.

Two constraints of that preview decide its shape, and both are load-bearing:

- **Its CSP is `script-src 'nonce-…'`, and the nonce never reaches a markdown-it
  plugin.** So the placeholder the extension host emits is *data only* — a
  `<pre>` with a base64 payload — and every line of code the page runs is
  contributed through `markdown.previewScripts`, which VS Code nonces. Emitting
  a `<script>` would be blocked *and* would raise the "content has been
  disabled" banner. There is no WASM there either: no `wasm-unsafe-eval`.
- **Content updates are a morphdom diff.** An edit anywhere in the file reverts
  a rendered block to its placeholder, disposing nothing and firing no event
  beyond `vscode.markdown.updateContent`. `media/preview.js` therefore asks the
  DOM whether each block is still mounted rather than remembering, and carries
  each player's playhead across by keying on a hash of the block's own source.

`Cinegram.mount(root, timeline, opts)` takes the options that make several
players share one page: `inline`, `keys: 'scoped'`, `hash: false`, `autoplay`,
`theme`. Every default is the standalone page's existing behaviour, so the
emitted page is unaffected. `runtime.css` scopes its page-level rules behind
`.dgm-standalone` for the same reason — the sheet is loaded whole into documents
the extension does not own.

The three browser assets are duplicated into `editors/vscode/media/` because
`go:embed` cannot reach out of its package and `previewScripts` cannot reach out
of the extension; `LICENSE.txt` is a fourth copy, because a `.vsix` contains
nothing from outside the extension folder. `bazel test
//editors/vscode:assets_test` is what keeps the copies honest; `bazel run
//editors/vscode:sync_assets` updates them. Both lists live in `sync/sync.go`
and `assets_test.go` and are deliberately not shared — a test that imported its
expectations from the thing it checks would pass either way.

**`editors/vscode/README.md` is the Marketplace listing, not developer
documentation.** The gallery renders it as the page body, so it is written for
someone who has just installed the extension, and every URL in it is absolute —
the Marketplace serves the README from inside the package but fetches images
over the network, so a relative path 404s. Everything about building,
packaging, publishing and how the extension works internally lives in
`editors/vscode/CONTRIBUTING.md` instead. `cmd/vsix` warns on stderr about
anything the listing will be missing, because none of it stops a package
installing: a `.vsix` with no icon and no README installs perfectly and lists as
a blank grey square.

**Export is `spawn`, compile is `execFileSync`, and that asymmetry is the
design, not an inconsistency to clean up.** `src/compile.js` is synchronous
because markdown-it's renderer cannot await, and it gets away with it because a
compile is milliseconds. `src/record.js` cannot: `cinegram record` spawns one
headless Chrome *per frame*, four at a time, so a ten-second scenario at 12fps
is 121 browsers and minutes of wall clock. Blocking the extension host for that
would freeze the editor. So export has its own module with `spawn`, a
cancellable `withProgress` notification, and the `--progress` protocol
(`cinegram-progress capture <i> <n>`, then `cinegram-progress encode`) that
`cmd/cinegram/record.go` writes to stderr for it. Two rules there:
`--progress` is **purely additive** — the two human-readable lines are
untouched, so anything already parsing today's output still works — and Cancel
kills the **process group**, not the child, because killing only the recorder
orphans its browsers. (Go's default `SIGTERM` skips deferred functions, so a
cancelled record leaves one `cinegram-record-*` temp directory behind. That is
knowingly traded for not adding signal handling to the CLI.)

The extension rewrites the CLI's failure messages in exactly one case, and the
exception is worth keeping narrow. `findChrome` and `findFFmpeg` already name
`CINEGRAM_CHROME` and `CINEGRAM_FFMPEG` and suggest recording a GIF instead; a
message rewritten in JavaScript could only say less. But a binary **older than
the extension** answers an unknown flag with Go's flag package printing the
whole usage text, which says nothing about the real problem and does not fit in
a notification — so `describeSkew` in `src/record.js` recognises
`flag provided but not defined` and `unknown command`, names the binary via
`binary.resolve().source`, and says to rebuild. That skew is routine in dev
mode, where the binary comes from `<workspace>/bazel-bin/` and a `git pull`
updates the extension without rebuilding it.

**`src/animationEditor.js` re-renders on save, never on keystroke.** It is a
`CustomTextEditorProvider` at `priority: "option"` — the text editor stays the
default and the animation appears in *Open With…*, because a source format that
opens as a picture with its text behind a submenu is a bad trade. It reuses
`dgmPreview.shell`, so exactly one place knows the CSP and the asset wiring.
Refreshing per keystroke would mean assigning `webview.html` wholesale on every
edit, which throws the playhead away and re-parses 2.7 MB of mermaid; it also
would not be *true*, since `view … from` reads from disk and only a save makes
the file on disk what the panel claims to show. Live-on-type would need the
payload delivered by `postMessage` plus the snapshot/restore dance
`media/preview.js` does for the Markdown path.

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

## Releasing

Everything ships from one `v*` tag — binaries on GitHub Releases, the VS Code
extension on the Marketplace — and **`RELEASING.md` is the procedure and the
contracts.** The short version: the version lives in three places
(`cmd/cinegram/version.go`, the extension `package.json`, the extension
changelog) kept equal by `//editors/vscode:assets_test`; the release workflow
is qualify → one job per distribution channel → verify; the asset names
`cinegram-<os>-<arch>` and the `releases/latest/download/` URL are contracts
that `skills/cinegram/SKILL.md` and `cinegram upgrade` both download by.
Anything that adds a way users obtain cinegram — a playground, a package
manager — must follow "Adding a distribution channel" in `RELEASING.md`
rather than publishing on its own.

## Verifying animation changes

Bazel tests cover the parser, compiler and emitters, but not the browser
runtime. To check a runtime change actually renders, serve the page — the
Chrome extension blocks `file://` — and drive the player, which is exposed as
`window.CINEGRAM_PLAYER`:

```sh
bazel run //cmd/cinegram -- preview examples/k8s-request.dgm --serve --watch
# http://127.0.0.1:8731/ — edit the .dgm and the page reloads itself
```

`CINEGRAM_PLAYER.seek(ms)` jumps to a moment deterministically, which is far
more reliable for verification than watching playback. Two traps to know about:

- A CSS `transition` means `getComputedStyle` right after a class change
  returns the *starting* value. Wait ~250ms before asserting on a colour or a
  width, or you will measure the frame before the change.
- Never put a CSS `transform` on a `g.node`. Mermaid positions nodes with a
  `transform` **attribute**, and the CSS property replaces it rather than
  composing — the node jumps to the origin. Animate the shape inside the node
  with `transform-box: fill-box`, or animate something else entirely.

For a still, `frame` captures one exact millisecond with no race against the
animation, because the deep link it opens lands paused:

```sh
bazel run //cmd/cinegram -- frame examples/payment-checkout.dgm \
  --at 1620ms --scenario s1 -o /tmp/fail.png
```

It shells out to a headless Chrome found on `PATH` or named by
`$CINEGRAM_CHROME`. The opt-in end-to-end test for it must run **outside**
the Bazel sandbox, which denies the browser what it needs to start:

```sh
CINEGRAM_CHROME="/Applications/Google Chrome.app/Contents/MacOS/Google Chrome" \
  bazel-bin/cmd/cinegram/cinegram_test_/cinegram_test -test.run TestFrameEndToEnd
```
