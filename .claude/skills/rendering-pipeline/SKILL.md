---
name: rendering-pipeline
description: >
  The .dgm → timeline → animated SVG pipeline: pkg/loader, pkg/parser,
  pkg/compile, pkg/ir, pkg/emit and the browser runtime. Use when adding a
  Mermaid diagram type, a scenario action or a timing rule, when changing
  parsing, compilation, mermaid emission or runtime.js, and when an animation
  renders wrongly and has to be verified in a browser.
---

# The rendering pipeline

A `.dgm` source is a Mermaid diagram body followed by `scenario`, `view` and
`interact` blocks.

```
source.dgm ─► pkg/loader ─► the file and every `view` it references
                  │
                  └─► pkg/parser ─► ast.Document + symbol.Table   (per file)
                                      │
                                      ├─► pkg/compile ─► ir.Timeline
                                      │
                                      └─► pkg/emit/{mermaid,html}
```

## Do not undo

1. **`pkg/parser` does no I/O.** `Parse(filename, content)` takes a string, so
   the parser works from a webview or a WASM build with no filesystem.
   `pkg/loader` resolves `view … from "path"` declarations and takes its read
   function as an argument, so it is testable against an in-memory map.
2. **The two halves meet only at `symbol.Table`.** `pkg/parser/flowchart.go`
   parses the diagram; `scenario.go` and `interact.go` parse the animation and
   the click bindings and contain no diagram vocabulary at all — they emit
   nothing but named references. `validate.go` is the one place the halves
   meet, and it works purely against `symbol.Table`. Keep flowchart specifics
   out of `scenario.go`, `interact.go`, `validate.go`, `pkg/compile` and
   `pkg/ir`.
3. **Mermaid emission is a reprint, not a regeneration.** Every
   `ast.Statement` carries its original source text via `Raw()`, and
   `pkg/emit/mermaid` only re-indents and drops scenario blocks. Anything the
   parser does not model semantically falls through to `ast.RawStmt` and
   round-trips verbatim — that is what keeps `classDef`, `click` and future
   Mermaid syntax working. Parse for the symbol table; never rewrite the text.
4. **`pkg/ir` holds no geometry.** Absolute integer milliseconds only; tracks
   and bindings reference node, edge, group and storyboard-frame IDs. Storyboard
   images arrive as `data:` URIs the loader built, never as paths — because of
   1, and because the page has to work from the filesystem. Two consequences
   worth protecting: a renderer is just a clock plus a scrubber, and a
   Go-native SVG backend stays possible without touching the compiler. Do not
   add coordinates.

A `Timeline` holds one `View` per diagram in the bundle, plus a `Root` naming
the one to open on. `Compile` lowers a single document (wrapping it in one
view); `CompileBundle` lowers a whole `loader.Bundle`.

## Adding a diagram type

Because of 2, this costs one parser and one runtime indexer — scenario
parsing, validation and the timeline compiler need zero changes.
`pkg/parser/sequence.go` is the worked proof: `sequenceDiagram` cost exactly
that and nothing in between.

1. Write a parser implementing `registry.DiagramParser`, registering itself in
   `init()`.
2. Hand the cursor back at **every** keyword in `isTopLevelKeyword`
   (`interact.go`), not just at `scenario` — that list is the contract in
   `registry.DiagramParser.Parse`.
3. Add a runtime indexer. Read
   [references/runtime-binding.md](references/runtime-binding.md) first: there
   are two strategies, not one per diagram type, and reusing one is usually
   right.
4. Verify in a browser (below).

Done when an example of the new type animates under `--serve` with no warning
banner, and `bazel test //...` is green.

## Adding an action or a timing rule

Timing lives entirely in `pkg/compile`. Read
[references/timing-rules.md](references/timing-rules.md) before changing any of
it; per 4, express the result in milliseconds and IDs.

## Narration

`desc` prose can be spoken. `cinegram voice` records a clip per line into a
`<file>.voice/` sidecar; `pkg/loader` inlines them and `pkg/ir` carries them on
`Step.Audio`. Four rules hold it together:

1. **Loading narration is opt-in — `loader.Load(path, read, loader.WithVoice())`
   — and the default is the load-bearing half.** With it off, compilation stays a
   pure function of the checked-in sources, so `//site:site_test` and
   `//pkg/langref:langref_test` regenerate to the same bytes whether or not the
   person running them has recorded narration locally. It also keeps the clips —
   which outweigh the diagram, the timeline and the runtime together — out of
   every page that did not ask. `--with-voice` on `preview`, `compile` and
   `record` is that switch.
2. **A clip is keyed by the words it says**, `voice.Key(prose)`, never by the
   step. So a retelling reusing a sentence reuses its recording, reordering or
   renaming re-records nothing, and only a genuine rewrite costs a new clip.
3. **Clips reach `ir` as `data:` URIs**, exactly as storyboard images do and for
   the same reasons as rule 4 above. `voice.WAVMillis` measures a clip's length
   from a RIFF header at record time and the manifest carries the number, so
   nothing downstream ever decodes audio.
4. **Only `pace: voice` lets a recording move the clock.** A step is as long as
   its animation needs and a line as long as it takes to say, and the second is
   routinely an order of magnitude longer — so without it the voice is simply cut
   off. The stretch is applied *after* the `minStepMillis` clamp (that warning is
   about the span the author wrote) and *before* layout, so flows keep their own
   durations and only the step, and whatever spans one, grows.

In `runtime.js`, `window.speechSynthesis` is a **page-global queue**, and several
players per page is ordinary here — the playground, a folder listing, a Markdown
preview. So every call into it is guarded by that player's own `voiceOn`, and
`dispose()` calls `hush()`: pausing is not stopping, and a paused utterance
belonging to a dead player is still there for the next player's `resume()`.

## Parsing strategy

The diagram body is **line-oriented** (like Mermaid itself) and read through
`source.Cursor`. The scenario block is **brace-structured** and gets a real
tokenizer in `pkg/parser/scanner.go`.

`pkg/parser/link.go` implements Mermaid's link disambiguation, which is subtle
and easy to break: an operator of exactly two dashes or equals signs with no
arrowhead is the *opening half* of a labelled link (`A-- text -->B`); three or
more, or any arrowhead, closes it immediately. `link_test.go` pins this — run
it after touching that file.

## Diagnostics

Diagnostics carry a position and usually a `Hint`; `diag.Bag` de-duplicates
identical entries because several passes read the same attribute. Warnings
never fail a build; errors exit 1.

## Verify a runtime change

Bazel tests cover the parser, compiler and emitters, never the browser runtime.
Serve the page — Chrome blocks `file://` — and drive the player, exposed as
`window.CINEGRAM_PLAYER`:

```sh
bazel run //cmd/cinegram -- preview examples/01-basics/01-k8s-request.dgm --serve --watch
# http://127.0.0.1:8731/ — edit the .dgm and the page reloads itself
```

1. `CINEGRAM_PLAYER.seek(ms)` jumps to a moment deterministically, which is far
   more reliable than watching playback.
2. Wait ~250ms before asserting on a colour or a width. A CSS `transition`
   means `getComputedStyle` right after a class change returns the *starting*
   value.
3. Animate the shape inside a node with `transform-box: fill-box`, or animate
   something else. A CSS `transform` on a `g.node` replaces the `transform`
   **attribute** mermaid positions it with rather than composing, and the node
   jumps to the origin.

For a still, `frame` captures one exact millisecond with no race against the
animation, because the deep link it opens lands paused:

```sh
bazel run //cmd/cinegram -- frame examples/02-storytelling/01-payment-checkout.dgm \
  --at 1620ms --scenario s1 -o /tmp/fail.png
```

It shells out to a headless Chrome found on `PATH` or named by
`$CINEGRAM_CHROME`. Its opt-in end-to-end test must run **outside** the Bazel
sandbox, which denies the browser what it needs to start:

```sh
CINEGRAM_CHROME="/Applications/Google Chrome.app/Contents/MacOS/Google Chrome" \
  bazel-bin/cmd/cinegram/cinegram_test_/cinegram_test -test.run TestFrameEndToEnd
```

Done when the moment you seek to shows what you intended and the page raises no
warning banner.
