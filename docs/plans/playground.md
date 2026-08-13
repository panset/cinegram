# Zero-install playground for cinegram

## Context

Cinegram's adoption funnel currently starts at "install something" (VS Code
extension, CLI binary, or the AI skill). The playground removes that step: a
static web page at `https://panset.github.io/cinegram/` where a developer
pastes a Mermaid diagram or `.dgm` document and watches it animate immediately
— the actual Go compiler (parser → loader → compile → emit) running as WASM in
the tab, feeding the same `runtime.js` player the CLI emits. Every shared link
is a demo; nothing is ever uploaded (the document lives in the URL fragment,
which browsers do not send to servers).

The codebase was built for this: `parser.Parse` does no I/O, `loader.Load`
takes its read function as an argument (its doc comment names "a WASM build
that has no disk at all"), and `bazel` cross-compilation via
`--platforms=@rules_go//go/toolchain:js_wasm` is confirmed available in
rules_go 0.55.1 with `wasm_exec.js` shipped inside the hermetic Go 1.24.0 SDK.

**This plan is written for execution by an Opus subagent**, in phases with a
`bazel test //...` gate after each phase. Each phase is independently
committable and leaves the repo green.

## Scope decisions (made with the user — do not relitigate)

- **Editor**: plain `<textarea>`, no CodeMirror, no vendored editor deps.
  Diagnostics in a panel below the editor, not inline squiggles.
- **Attachments in v1**: a drop-zone/file-picker panel; storyboard images and
  extra `.dgm` files live in an in-memory virtual FS so `view … from` and
  `img:` resolve. Clicking an attached `.dgm` swaps it into the editor.
- **No markdown extraction in v1** (pasting a `.md` with ```dgm fences is a
  later feature).
- **Share** = URL fragment (deflate + base64url of the whole file set), warn
  above ~50 KB encoded.
- **Download HTML** = the same self-contained page `cinegram preview -o`
  emits, produced by the same `pkg/emit/html.Render` code compiled into the
  WASM binary (byte-identical output; costs ~2.7 MB of embedded mermaid in
  the .wasm, accepted deliberately).
- **Hosting**: GitHub Pages via a new Actions workflow deploying on push to
  main. One-time manual step for the user: enable Pages (source: GitHub
  Actions) in repo settings.

## Verified facts the plan builds on

(from parser/runtime/Bazel exploration; the executor can trust these)

- `@rules_go//go/toolchain:js_wasm` platform + registered toolchain exist.
  A js/wasm `go_binary` output has **no** `.wasm` extension unless
  `out = "cinegram.wasm"` is set.
- `wasm_exec.js` (Go 1.24: `lib/wasm/wasm_exec.js`, not `misc/wasm/`) is
  exported by the SDK repo via `exports_files`; naming the SDK
  (`go_sdk.download(name = "go_sdk", …)` + `use_repo`) makes it addressable
  as `@go_sdk//:lib/wasm/wasm_exec.js`. Taking it from the same SDK that
  compiles the .wasm keeps the two version-locked.
- All library packages (`pkg/parser,loader,compile,ir,diag,symbol,registry,
  units,ast,source,emit/mermaid,emit/html`) import only js/wasm-clean stdlib.
- The envelope wire format (`jsonEnvelope`, `jsonDiagnostic`,
  `collectDiagnostics`) lives in `cmd/cinegram/main.go` (package main) and
  must be lifted to a shared package for the WASM main to reuse.
- `Cinegram.mount(root, timeline, opts)` (runtime.js) accepts
  `{keys:'scoped', hash:false, autoplay, theme}`; **no dispose exists** — a
  playground that remounts per edit must add `Player.prototype.dispose()`
  (window pointermove/pointerup from `bindStageGestures`, document keydown
  when `keys!=='scoped'`, window hashchange when `hash!==false`, matchMedia
  listener when no explicit theme, plus `cancelAnimationFrame`).
- Playhead preservation across remounts: copy the snapshot/restore pattern
  from `editors/vscode/media/preview.js` (restore order is load-bearing:
  `setView` → assign `stack` → `syncNav` → `selectScenario` → `seek` LAST →
  `play`).
- Call `mermaid.initialize({startOnLoad:false})` once before any mount
  (prior art: `silenceMermaidAutoRun` in preview.js).
- `runtime.css` page-level rules are scoped behind `body.dgm-standalone`; the
  playground page must NOT use that class and owns its own two-panel layout.
- `runtime.js` must stay a classic script (never an ES module).
- `.gitignore` contains `*.html` — the playground's `index.html` needs an
  explicit exception.
- Duplicated-asset policy: committed copies + sync tool + drift test
  (editors/vscode pattern) — but prefer NOT committing a third copy of
  mermaid.min.js; assemble the site at build/deploy time instead.

## Execution notes for the implementing agent

- Work in a git worktree; commit at the end of each phase; every phase ends
  with `bazel test //...` AND `bazel build //...` green, and gofmt run via the
  hermetic SDK:
  `"$(find "$(bazel info output_base)/external" -name gofmt -type f | head -1)" -w ./cmd ./pkg ./web`
- Go is NOT installed locally — never run bare `go build`/`go test`; Bazel only.
- After adding Go packages run `bazel run //:gazelle`; protect hand-written
  BUILD content with `# gazelle:ignore` (whole file) or `# keep` (attribute).
- The repo remote is `panset/cinegram`; the Pages URL will be
  `https://panset.github.io/cinegram/`.
- Resolved build question: `go_binary` in rules_go 0.55.1 supports
  `goos = "js", goarch = "wasm"` attributes (verified in
  `go/private/rules/binary.bzl`: goos line ~401, goarch ~412, `out` ~310,
  `pure` ~358; applied via `go_transition` to the whole dep tree). This is the
  mechanism — no `--platforms` needed, `bazel build //...` stays green.

## Phase 1 — Extract the envelope wire format into `pkg/envelope`

Pure refactor so the WASM main can reuse the exact JSON shape the VS Code
preview consumes.

**New `pkg/envelope/envelope.go`** — moves from `cmd/cinegram/main.go`
(carry over the "deliberately a wire format" comments at main.go:338-344,
472-475):
- `Envelope` (was `jsonEnvelope`, main.go:345-348): `Timeline *ir.Timeline`
  + `Diagnostics []Diagnostic`, same JSON tags.
- `Diagnostic` (was `jsonDiagnostic`, main.go:476-483): file/line/col/
  severity/message/hint(omitempty), identical tags.
- `Collect(bags []*diag.Bag) ([]Diagnostic, int)` (was `collectDiagnostics`,
  main.go:489-509) — verbatim body; always returns a non-nil slice.

**Modify `cmd/cinegram/main.go`**: delete the moved types/func;
`writeEnvelope` STAYS (it does output I/O) but takes/builds
`envelope.Envelope`; update `cmdCompile` (lines ~279-290) and `lintJSON`
(~515) to use `envelope.Collect` / `envelope.Diagnostic`.

**Modify `cmd/cinegram/main_test.go`**: `TestEnvelopeAlwaysCarriesBothHalves`
(~197-236) keeps pinning `writeEnvelope` output; `mustCollect` (~238),
`TestLintJSONExitCodes` (~250-288) switch to the envelope types.
**New `pkg/envelope/envelope_test.go`**: Collect-specific assertions (hint
survives, filename propagation, error count, empty bag → `[]` not `null`).

**Gate**: `bazel run //:gazelle && bazel test //...`. Then byte-compare
envelope output against the pre-phase commit:
`cat examples/k8s-request.dgm | bazel run //cmd/cinegram -- compile - --as examples/k8s-request.dgm --envelope`
must be byte-identical before/after. Also `lint --format=json` unchanged.

## Phase 2 — `Player.prototype.dispose()` in runtime.js

The playground remounts per edit; without dispose each mount leaks listeners.

**Modify `pkg/emit/html/assets/runtime.js`** (match its var/function classic
style — no arrow functions, no modules):
- Constructor (~line 341): add `this._unbind = [];`
- Module-scope helper `own(player, target, type, fn, opts)` — addEventListener
  + push a remover onto `player._unbind`.
- Rewrite the five leaking sites to use it: matchMedia `change` (~563-576,
  keep the `addListener` legacy fallback, pushing a manual remover),
  `document` keydown (~589, only when `keys !== 'scoped'`), `window`
  hashchange (~600, only when `hash !== false`), `window` pointermove (~922)
  and pointerup (~936) in `bindStageGestures`.
- New `Player.prototype.dispose()` near `pause` (~1712): `this.pause()`, run
  every `_unbind` entry in try/catch, reset the array. Do NOT touch
  `data-theme` on documentElement. Element-level listeners need no handling —
  they die with the DOM when the host clears the root.

**Modify `editors/vscode/media/preview.js`** `stop()` (~211-214):
`player.dispose ? player.dispose() : player.pause()` — fixes a real existing
leak in the Markdown preview, feature-detected so a lagging copy stays safe.

**Sync**: `bazel run //editors/vscode:sync_assets` (no list changes — no new
files). **Gate**: `bazel test //...` (includes `assets_test` and `html_test`);
then `bazel run //cmd/cinegram -- preview examples/k8s-request.dgm -o /tmp/k8s.html`,
open it, verify in console: `typeof CINEGRAM_PLAYER.dispose === 'function'`;
after `dispose()`, Space no longer toggles and no errors are thrown.

## Phase 3 — WASM binary (`web/playground/wasm`) + MODULE.bazel

**MODULE.bazel diff**:
```python
go_sdk = use_extension("@rules_go//go:extensions.bzl", "go_sdk")
go_sdk.download(
    name = "go_sdk",
    version = "1.24.0",
)
# wasm_exec.js must match the SDK that compiled the .wasm; taking it from the
# same repo makes that a fact rather than a discipline.
use_repo(go_sdk, "go_sdk")
```
This renames the SDK repo (lockfile regenerates; nothing references the old
name). `bazel mod tidy` fixes any use_repo complaints. wasm_exec.js is then
`@go_sdk//:lib/wasm/wasm_exec.js` (Go 1.24 keeps it in `lib/wasm/`, exported
via the SDK BUILD's `exports_files(glob(["*/wasm/**"]))`).

**New `web/playground/wasm/main.go`** with `//go:build js && wasm`:
- `main()`: `js.Global().Set("cinegramCompile", js.FuncOf(compileFn))`, same
  for `cinegramRenderHTML`; invoke `onCinegramReady` if the page defined it;
  `select {}` to keep the runtime alive.
- `filesFromJS(v js.Value) map[string][]byte`: iterate `Object.keys`; JS
  string values → `[]byte(s)` (text .dgm), Uint8Array values →
  `js.CopyBytesToGo` (images). Keys stored under `filepath.Clean`.
  **Encoding decision: text crosses as JS strings, images as Uint8Array — no
  base64 round-trip**; `loader` inlines images to data: URIs itself from raw
  bytes (loader.go:186-196).
- `readerFor(files) loader.ReadFileFunc`: cleaned-path map lookup, else
  `fs.ErrNotExist` (loader degrades to its normal diagnostic + hint).
- `compileFn`: validate `len(args)==2`; `defer recover()` → error envelope,
  never panic across the JS boundary. `loader.Load(entry, readerFor(files))`;
  unreadable entry → envelope with single error diagnostic (mirror
  main.go:279-284); else `compile.CompileBundle` + `envelope.Collect`; return
  JSON of `struct { envelope.Envelope; Title string }` with
  `html.DefaultTitle(t)`.
- `renderFn`: same load/compile; `html.Render(t, html.Options{})` (empty
  Options = the CLI `preview` defaults, so **Download HTML is byte-identical
  to `cinegram preview` output**); returns `{"html": …, "diagnostics": […]}`.
- Comment the size tradeoff: importing `pkg/emit/html` embeds mermaid.min.js
  (2.7 MB) into the .wasm — the price of byte-identical output, deliberate.

**New `web/playground/wasm/BUILD.bazel`** (hand-written, first line
`# gazelle:ignore`):
```python
load("@rules_go//go:def.bzl", "go_binary")
go_binary(
    name = "cinegram_wasm",
    srcs = ["main.go"],
    out = "cinegram.wasm",       # js_wasm output has no extension otherwise
    goarch = "wasm",
    goos = "js",
    pure = "on",
    visibility = ["//web/playground:__subpackages__"],
    deps = ["//pkg/compile", "//pkg/emit/html", "//pkg/envelope", "//pkg/loader"],
)
```

**Gate**: `bazel build //... && bazel test //...` (host build must not break —
that's the invariant the goos/goarch attrs buy);
`file bazel-bin/web/playground/wasm/cinegram.wasm` → "WebAssembly (wasm)
binary module" (~15–20 MB expected); `bazel build @go_sdk//:lib/wasm/wasm_exec.js`
resolves.

## Phase 4 — The playground site + assembler/dev-server

**New `examples/BUILD.bazel`**: `filegroup(name = "playground_examples",
srcs = glob(["*.dgm", "frames/*.svg"]), visibility = ["//web/playground:__pkg__"])`.

**New `web/playground/index.html`** — classic scripts in order:
`mermaid.min.js`, `runtime.js`, `wasm_exec.js`, `playground.js`; stylesheets
`runtime.css` + `playground.css`. Body class `pg` (NOT `dgm-standalone` — the
page owns its two-panel layout). Left: examples `<select>`, open-file tabs,
`<textarea id="pg-editor">`, attachments panel (hidden `<input type="file"
multiple accept=".dgm,.svg,.png,.jpg,.jpeg,.gif,.webp">` + list), diagnostics
strip. Right: `#pg-player-host` + `#pg-boot` loading indicator. Topbar: Share,
Download HTML, GitHub link. (`.gitignore` gains `!web/**/*.html`.)

**New `web/playground/playground.css`** — page layout only; key the page's own
colors off `html[data-theme="dark"]` so it follows the player's theme toggle
(the player sets that attribute); never restyle `.dgm` internals.

**New `web/playground/playground.js`** — one classic-script IIFE:
1. *Boot*: `new Go()`; `WebAssembly.instantiateStreaming(fetch('cinegram.wasm'),
   go.importObject)` with `arrayBuffer()` fallback; await `onCinegramReady`;
   then `mermaid.initialize({startOnLoad:false})` (the preview.js lesson),
   wire UI, `loadFromHash() || loadExample(default)`.
2. *Virtual FS*: `Map` path → `{text}` (.dgm) or `{bytes: Uint8Array}`
   (images); `entry` and `open` path vars; `filesForGo()` returns a plain
   object of strings/Uint8Arrays (matches Phase 3 encoding).
3. *Compile loop*: `input` → save buffer to vfs → 300 ms debounce →
   `JSON.parse(cinegramCompile(entry, filesForGo()))` (synchronous) → paint
   diagnostics (`file: line N: message` + hint; errors before warnings; block
   mount only on `severity === 'error'` — the preview.js contract). Deliberate
   divergence from preview.js: on error keep the last good player mounted and
   dim it (`is-stale` class) instead of tearing down — mid-keystroke errors
   are the common case in an editor.
4. *Mount lifecycle*: snapshot → `player.dispose()` → fresh child div →
   `Cinegram.mount(inner, timeline, {keys:'scoped', hash:false,
   autoplay:!hadPrevious})` → restore. Snapshot/restore copied from
   preview.js:130-209; restore order is load-bearing: `setView` → assign
   `stack` → `syncNav` → `selectScenario` → `seek` LAST → `play`. Set
   `window.CINEGRAM_PLAYER = player` (same debug handle as the standalone
   page).
5. *Attachments*: file picker + dragover/drop on the left panel. `.dgm` →
   `file.text()`; images → `new Uint8Array(await file.arrayBuffer())`; others
   rejected with a strip message. Rows: name, size, remove; clicking a `.dgm`
   row saves the editor and swaps `open`. Every mutation triggers the
   debounced recompile.
6. *Share codec*: `{v:1, entry, open, files:[{p, t:'text'|'b64', d}]}` →
   `CompressionStream('deflate-raw')` → base64url → `#doc=…` via
   `history.replaceState`; copy URL to clipboard; warn above ~50 KB encoded
   ("prefer Download HTML"). Feature-detect CompressionStream (Chrome 103+/
   Safari 16.4+/Firefox 113+) → disable Share with tooltip, never throw. On
   load, `#doc=` → decode → hydrate vfs → recompile. The player runs
   `hash:false` so it never fights for the fragment.
7. *Examples picker*: `fetch('examples.json')` → populate select; on pick
   reset vfs, fetch each listed file (`.dgm` as text, images as arrayBuffer),
   recompile, clear the hash.
8. *Download HTML*: `cinegramRenderHTML(entry, filesForGo())` → Blob →
   `<a download="<slug(title)>.html">`. Disabled while errors exist.

**New `web/playground/examples.json`** — id/title/entry/files per example.
Verified multi-file sets: `k8s-request.dgm` and `websocket-handshake.dgm` each
also need `pod-a.dgm`; `oidc-login.dgm` needs the five `frames/*.svg`. All
other examples are single-file (`embedded.md`, `*.narrate.md`,
`payment-checkout.fail.png`, `frames/README.md` are not examples).

**New `web/playground/site.go`** — assembler + dev server (`bazel run
//web/playground:site -- --serve [--addr]` or `-o dir` for the workflow).
Resolves via `@rules_go//go/runfiles`: the four static sources +
`pkg/emit/html/assets/{runtime.js,runtime.css,mermaid.min.js}` (canonical — no
committed copies, nothing to drift) + `cinegram.wasm` +
`go_sdk/lib/wasm/wasm_exec.js` + the example files. Resolve each file named in
`examples.json` individually (that doubles as the manifest-drift check — fail
loudly on a miss; it runs in CI). Relative `-o` resolves against
`BUILD_WORKING_DIRECTORY` (same convention as `resolvePath`,
cmd/cinegram/main.go:204-212). Serve with `http.ServeContent` (Go's mime table
knows `.wasm` → `application/wasm`, required for `instantiateStreaming`).

**New `web/playground/BUILD.bazel`** — gazelle skeleton then hand-adjust with
`# keep`: `filegroup(name="static", srcs=[examples.json, index.html,
playground.css, playground.js])`; `go_binary(name="site",
embed=[":playground_lib"], data=[":static", "//examples:playground_examples",
"//pkg/emit/html:browser_assets", "//web/playground/wasm:cinegram_wasm",
"@go_sdk//:lib/wasm/wasm_exec.js"])`. Gazelle resolves the
`github.com/bazelbuild/rules_go/go/runfiles` import to `@rules_go//go/runfiles`
automatically; hand-pin with `# keep` if it guesses wrong.

**Gate**: `bazel run //:gazelle && bazel test //... && bazel build //...`;
`bazel run //web/playground:site -- -o /tmp/pg-site && ls -R /tmp/pg-site`;
then `--serve` and browser-verify (manually or via claude-in-chrome):
1. Boot completes, no console errors, default example autoplays.
2. `CINEGRAM_PLAYER.timeline.views.length >= 2` on k8s; drill into Pod A;
   `location.hash` unchanged (hash:false honored).
3. Seek + pause, edit a label → after ~300 ms diagram updates with playhead
   ≈ preserved, still paused. Introduce a syntax error → red strip with
   `line N:`, previous diagram dimmed; fix → clears.
4. Listener hygiene: 30 scripted recompiles, then DevTools
   `getEventListeners(window).pointermove.length === 1`.
5. oidc-login example → storyboard images render; websocket-handshake →
   drill-down works.
6. Attach a .png, reference from `img:` → renders; attach a .dgm, add a
   `view … from` → drill-down resolves; click attached .dgm → editor swaps.
7. Share → `#doc=` link reproduces the document set in a private window;
   >50 KB doc → warning.
8. Download HTML → byte-identical to
   `bazel run //cmd/cinegram -- preview <same>.dgm -o -` for a single-file doc.

## Phase 5 — GitHub Pages workflow + docs

**New `.github/workflows/pages.yml`** (modeled on release.yml —
setup-bazelisk@v3 reads `.bazelversion`):
`on: push: branches: [main]` + `workflow_dispatch`; permissions
`contents: read, pages: write, id-token: write`; concurrency group `pages`;
steps: checkout → setup-bazelisk → `bazel test //...` (qualify) →
`bazel run //web/playground:site -- -o "$PWD/_site"` →
`actions/configure-pages@v5` → `actions/upload-pages-artifact@v3` (path
`_site`) → `actions/deploy-pages@v4` with the `github-pages` environment.

**Docs**: README gains a "Try it in the browser" line pointing at
`https://panset.github.io/cinegram/` plus the dev-serve command; CLAUDE.md
gains a short subsection (playground assembled-not-committed, wasm BUILD is
hand-written/gazelle:ignore, envelope wire format now in `pkg/envelope`).

**Gate**: `bazel test //...`; YAML parses
(`python3 -c "import yaml,sys; yaml.safe_load(open('.github/workflows/pages.yml'))"`);
clean-tree assembly works (`bazel clean && bazel run //web/playground:site --
-o /tmp/pg-clean`, smoke-test in a browser).

**One-time manual step for the user (not a file — the executor cannot do
it)**: repo Settings → Pages → Source = "GitHub Actions". Note it in the PR
description. Post-merge: watch the `pages` run and verify the deployed URL
boots (network tab: if Pages' content-type breaks `instantiateStreaming`, the
`arrayBuffer()` fallback covers it).

## Risks (known and accepted)

1. `use_repo(go_sdk, "go_sdk")` naming regenerates the lockfile — validate
   first thing in Phase 3; `bazel mod tidy` fixes complaints.
2. WASM is ~15–20 MB raw (mermaid embed included); Pages gzip cuts it to
   roughly a third. Accepted for v1 — do NOT "optimize" by dropping
   `pkg/emit/html`; byte-identical Download HTML is the point.
3. Synchronous compile on the main thread — ms-scale for real docs; a Worker
   is the v2 fix (leave a comment, don't build it).
4. `restore` on a reshaped timeline just restarts via try/catch (preview.js
   precedent).
5. Textarea vs player keys: `keys:'scoped'` + runtime.js's input-tag guard
   (onKey ~655) means typing never fights the transport.

## Phase order and dependencies

1 (envelope) → 3 (wasm); 2 (dispose) → 4 (site); 3 → 4 → 5.
Execute 1, 2, 3, 4, 5. Each phase: gate green, gofmt run, commit.
