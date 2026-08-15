# Prior art for the next five workstreams

*Research pass, 2026-08-15. Every claim below is cited to the primary source
that owns it — official docs, the project's own repo/source, or the published
npm package contents — not secondary write-ups. Version/date noted wherever
behavior may have changed. Repo anchors referenced: today's runtime autoplays
by default (`pendingAutoplay = this.opts.autoplay !== false`,
`pkg/emit/html/assets/runtime.js:387`); the playground split is a fixed
`--pg-left: 42%` CSS variable consumed by `flex: 0 0 var(--pg-left)`
(`web/playground/playground.css`); the repo is MIT-licensed.*

The five decided workstreams these findings map onto:

1. **`cinegram site <dir>`** — folder hierarchy of `.dgm` files → browsable
   static site; nav mirrors folders; one shared copy of runtime assets;
   ordering via optional numeric filename prefixes stripped for display.
2. **Paused-on-load-by-default player** across all surfaces; autoplay becomes
   explicit opt-in.
3. **Draggable, localStorage-persisted editor/preview split** in the
   playground; double-click to collapse.
4. **Shared lightbox layer** in the runtime for storyboard scene images:
   click to expand with viewport auto-fit, cursor-anchored wheel zoom,
   Esc/click-out to close.
5. **Future "rolodex" navigation** — click a diagram object, flip through the
   scenes/scenarios that involve it (inverted object→scenes index over the
   timeline IR).

---

## Cluster A — diagram-scenario tools

### D2 (d2lang.com)

Declarative diagram language with a first-class multi-board composition
model. Docs fetched 2026-08; the composition section is flagged incomplete by
the project itself.

**The composition model, precisely.** One root board, three board kinds
distinguished *only* by what they inherit
([Intro to Composition](https://d2lang.com/tour/composition/)):

- `layers` — "Boards which do not inherit. They are a new base."
- `scenarios` — inherit from the base layer: "Any new objects are added onto
  all objects in the base Layer, and you can reference any objects from the
  base Layer to update them" ([Scenarios](https://d2lang.com/tour/scenarios/)).
- `steps` — chain: "Each Step inherits from the Step before it. The first
  step inherits from its parent, whether that's a Scenario or Layer"
  ([Steps](https://d2lang.com/tour/steps/)), and inheritance is transitive
  across the chain.

So: **step N inherits from step N−1 (chain), scenario fans out from its base,
layers reset.** Deletion inside an inheriting board is not a
composition-specific mechanism — `null` is the universal override/delete verb
("You may override with the value `null` to delete the
shape/connection/attribute"), explicitly recommended for compositions that
inherit everything with exceptions
([Overrides → Null](https://d2lang.com/tour/overrides/); since
[D2 0.6.0](https://d2lang.com/releases/0.6.0/)).

Multi-board output ([Export formats](https://d2lang.com/tour/composition-formats/)):
default is one SVG file per board with "Internal board links … rewritten to
point to those file paths"; `--animate-interval=1200` collapses everything
into one animated SVG that "stays on each board for 1200ms" (since
[v0.3.0](https://github.com/terrastruct/d2/releases/tag/v0.3.0)) — no
scrubber, equal time per board, and D2's own docs concede it degrades with
board count. Board-to-board navigation reuses the external-URL keyword:
`a.link: layers."2012.06"`, `_` walks parent *boards*, and rendered output
gets a clickable breadcrumb bar
([Linking between boards](https://d2lang.com/tour/linking/)).

**Steal**
- The doc discipline: "who inherits from whom" is the entire taxonomy, one
  sentence per board kind. Cinegram's `variant: "base", until: <step>` splice
  is D2's `steps` chain *plus* a truncation point D2 doesn't have — D2
  scenarios inherit state, not a prefix of a timeline — so `until:` is the
  more expressive primitive; document it with D2's economy.
- `null` as the single deletion verb, should variants ever need "inherit
  minus one action" — precedent over inventing a bespoke `except` syntax.
- For the rolodex (ws5): author-facing links that are format-independent and
  rewritten at emit time (file paths in multi-SVG, anchors in PDF) — the same
  shape as cinegram's hash routing, validated.

**Avoid**
- Fixed-interval auto-flip (`--animate-interval`) as an animated output mode.
  `ir.Timeline` + scrubber is strictly stronger; do not add an interval mode.

#### D2 as a Go library (the one-big-win question)

Investigated separately because the IR's no-geometry rule exists so a
Go-native SVG backend can someday replace mermaid + headless Chrome. Findings
as of 2026-08-15:

- **The premise "D2's layout is dagre.js under goja" was true until this
  month and is now false.** D2 reorganized in Aug 2026: module moved from
  `oss.terrastruct.com/d2` (now
  [deprecated](https://pkg.go.dev/oss.terrastruct.com/d2), repo renamed
  `d2lang/d2-legacy`) to
  [`github.com/d2lang/d2`](https://pkg.go.dev/github.com/d2lang/d2) v0.8.x
  (Aug 7 2026), replacing every JS-under-goja subsystem with native Go ports.
  Legacy ≤ v0.7.2 is confirmed JS-in-goja
  (`//go:embed setup.js` + `runner.RunString(dagreJS)` in
  [legacy layout.go](https://github.com/d2lang/d2-legacy/blob/master/d2layouts/d2dagrelayout/layout.go);
  goja in the [legacy go.mod](https://github.com/d2lang/d2-legacy/blob/master/go.mod)).
  Current [go.mod](https://github.com/d2lang/d2/blob/master/go.mod) has no
  goja anywhere; layout delegates to:
  - [`github.com/d2lang/dagro`](https://github.com/d2lang/dagro) — native Go
    port of dagre (behavioral target dagre 3.1.1), **MIT**, and its
    [go.mod](https://github.com/d2lang/dagro/blob/master/go.mod) has **zero
    require lines**.
  - [`github.com/d2lang/elk-go`](https://github.com/d2lang/elk-go) — native
    ELK port, **EPL-2.0** (avoidable: Go links per-import).
- **Licenses.** D2 is
  [MPL-2.0](https://github.com/terrastruct/d2/blob/master/LICENSE.txt);
  per the [Mozilla MPL FAQ](https://www.mozilla.org/en-US/MPL/2.0/FAQ/)
  (Q11/Q12) it is file-level copyleft only and does not infect an MIT
  importer using it as an unmodified module dependency. TALA is confirmed
  proprietary — "TALA is closed-source", paid, watermarks unlicensed output
  ([terrastruct/tala](https://github.com/terrastruct/tala)) — a binary
  plugin, irrelevant as a library.
- **Dep tree / API.** The full module has 27 direct requires (playwright-go,
  fpdf, chroma, goldmark …), but the layout+SVG path is decoupled:
  [`d2svg.Render(diagram *d2target.Diagram, opts) ([]byte, error)`](https://github.com/d2lang/d2/blob/master/d2renderers/d2svg/d2svg.go)
  imports neither d2lib nor the compiler, and the
  [documented low-level pipeline](https://github.com/d2lang/d2/blob/master/docs/examples/lib/3-lowlevel/lowlevel.go)
  is `d2compiler.Compile → textmeasure.NewRuler → SetDimensions →
  d2dagrelayout.Layout → d2exporter.Export → d2svg.Render`. Text measurement
  is solved via
  [embedded TTF/WOFF fonts](https://github.com/d2lang/d2/tree/master/d2renderers/d2fonts)
  and a custom ruler
  ([textmeasure.go](https://github.com/d2lang/d2/blob/master/lib/textmeasure/textmeasure.go)) —
  no x/image/font. Caveat: the documented entry point is D2 *text/AST*, not a
  programmatic node/edge list.
- **Alternatives.**
  [goccy/go-graphviz](https://github.com/goccy/go-graphviz) is C graphviz
  compiled to WASM and run under
  [wazero](https://github.com/goccy/go-graphviz/blob/master/go.mod) with an
  EPL wasm blob — a foreign runtime relocated, not eliminated.
  [gonum graph/layout](https://pkg.go.dev/gonum.org/v1/gonum/graph/layout)
  has only force-directed/Isomap — no layered layout at all.

**Verdicts under the "one big win" gate** (stdlib-only stays the default;
full case comes back as its own decision):

| Candidate | Verdict |
|---|---|
| `github.com/d2lang/dagro` alone | **BIG-WIN-CANDIDATE (strongest).** MIT, zero deps, pure Go, the exact layered algorithm mermaid flowcharts use. Cinegram keeps its own SVG emission and text measurement and buys only coordinates. Risk: born Aug 2026, git-tag-only releases, API shaped by D2, unproven outside it. |
| `github.com/d2lang/d2` v0.8+ (layout+d2svg path) | **BIG-WIN-CANDIDATE (qualified).** License fine, layout now native Go, renderer decoupled with a documented pipeline. Qualifiers: large module graph even if the linked subset is small, D2-text input model, one avoidable EPL-2.0 dep, ports are one release old. |
| `oss.terrastruct.com/d2` ≤ v0.7.2 | LEARN-ONLY — deprecated, and its layouts are JS-in-goja. |
| TALA | Non-starter — closed-source paid binary. |
| goccy/go-graphviz | LEARN-ONLY — C-in-wazero + EPL blob. |
| gonum graph/layout | LEARN-ONLY — no layered layout exists there. |

**Decision (2026-08-15):** `dagro` is the designated layout engine for the
future Go-native SVG backend — noted here so the backend work starts from
this finding rather than re-surveying. The dependency is **not** taken now;
the stdlib-only constraint stays in force until that backend work actually
begins, at which point dagro's maturity (releases, API stability, use
outside D2) gets re-evaluated before the import lands.

### Mermaid Live Editor (mermaid.live)

SvelteKit web app for editing/previewing/sharing Mermaid diagrams
([README](https://github.com/mermaid-js/mermaid-live-editor)). Code claims
pinned to `develop` @ `10ef944db` (2026-08-14).

- **The split is draggable AND persisted, nearly for free.** The edit page
  builds a horizontal pane group from
  [PaneForge](https://paneforge.com/docs/components/pane-group) with
  `autoSaveId="liveEditor"` — the entire persistence story is that one prop
  ("If provided, the layout will be saved to local storage when it changes").
  Editor pane `defaultSize={30}`, both panes `minSize={15}`; below 640px the
  split is *replaced* by an Edit/View toggle sliding a 200%-wide container,
  not squeezed
  ([edit/+page.svelte](https://github.com/mermaid-js/mermaid-live-editor/blob/develop/src/routes/(app)/edit/%2Bpage.svelte)).
- **Self-describing share hash.** The hash is `"${serde}:${payload}"` —
  `pako:` + URL-safe base64 of `deflate(json, {level: 9})`, with a no-prefix
  legacy base64 fallback
  ([serde.ts](https://github.com/mermaid-js/mermaid-live-editor/blob/develop/src/lib/util/serde.ts)).
  Written via `history.replaceState` behind a 250ms debounce
  (`initURLSubscription`, `state.svelte.ts`) — never `pushState`, so Back
  means "leave the page", not "undo a keystroke".
- **Autosave**: current state in localStorage key `codeStore`; a history
  timeline snapshots every 60s, capped at 30 entries
  (`historyState.svelte.ts`).
- **Avoid:** pan/zoom coordinates live in the shared state, which forced
  `isDirty`/ResizeObserver reconciliation machinery
  ([panZoom.ts](https://github.com/mermaid-js/mermaid-live-editor/blob/develop/src/lib/util/panZoom.ts)).
  For cinegram the playhead is the state worth carrying (the VS Code preview
  already snapshots it by source hash), not the camera.

### Structurizr (structurizr.com)

C4-model tooling: one workspace = one shared model projected into many views
(system landscape → context → container → component), plus docs/ADRs
([workspaces](https://docs.structurizr.com/workspaces)). Because views are
projections of one model, the same element intrinsically appears in many
views — exactly the property the rolodex wants.

- **Navigation between views**
  ([navigation](https://docs.structurizr.com/ui/diagrams/navigation)):
  double-click = zoom into the next level of detail *if one exists*; "When an
  element has a mix of 'zoom-ins', documentation, decisions, and URLs,
  double-clicking the element will open a modal from which you can choose
  where to go next." Arrow keys step prev/next diagram; `Space` opens a
  type-ahead quick navigation across everything
  ([quick-navigation](https://docs.structurizr.com/ui/quick-navigation)).
- **Static export keeps the runtime**: the CLI "can export a static site …
  hosted from a simple web server" with the interactive viewer intact —
  double-click navigation, quick-nav, bookmarkable diagrams, animation on
  `,`/`.` ([static](https://docs.structurizr.com/static)). The community
  [structurizr-site-generatr](https://github.com/avisi-cloud/structurizr-site-generatr)
  adds a hierarchy-mirroring left menu with a filter worth noting: "external
  systems excluded from navigation menus" — the tree drives nav, but not
  every artifact deserves an entry.
- **Auto-generated diagram key** from the styles actually used
  ([notation](https://docs.structurizr.com/ui/diagrams/notation)) — cheap to
  mirror later: derive a legend from the actions a scenario actually uses.
- **Avoid: perspectives as a separate annotation layer**
  ([perspectives](https://docs.structurizr.com/ui/diagrams/perspectives)) —
  Structurizr needs that second vocabulary because its views are static;
  cinegram's scenarios *are* per-concern replays. The rolodex should
  enumerate existing scenarios/views touching an element, not add a parallel
  annotation system.

---

## Cluster B — player & site conventions

### asciinema player

Standalone terminal-session player (v3.x, actively maintained), mounted via
`AsciinemaPlayer.create(src, container, opts)`; self-hostable as one JS + one
CSS file
([quick-start](https://docs.asciinema.org/manual/player/quick-start/)).
The canonical paused-by-default player. Exact defaults
([options](https://docs.asciinema.org/manual/player/options/)):

| Option | Default |
|---|---|
| `autoPlay` | **`false`** — opt-in is one boolean |
| `preload` | `false` |
| `poster` | blank terminal, or the frame at `startAt`; spellings `'npt:1:23'` (a moment rendered as a still) or `'data:text/plain,…'` |
| `controls` | `"auto"` — three-state: `true` / `false` / show-on-hover |
| `loop` | `false` |
| `startAt` / `speed` | `0` / `1` |
| `markers` | none; `[time, label]` pairs |
| `pauseOnMarkers` | `false` — auto-pause at each marker turns a linear cast into a step-through |

**Steal:** the entire stance for ws2 — paused on a poster with a play button,
one-boolean opt-in. Cinegram's analog of `poster: 'npt:…'` is nearly free
(deterministic `seek(ms)` already exists): a `posterAt`-style resting frame
beats a blank t=0 diagram. `pauseOnMarkers` maps one-to-one onto cinegram
steps as a *player option*, no source-format change. `controls: "auto"`
matters once `cinegram site` puts many players on one page.

### mdBook

Rust's standard book generator — the explicit-manifest counter-model to
numeric filename prefixes.

- **SUMMARY.md** ([format](https://rust-lang.github.io/mdBook/format/summary.html))
  is a strictly parsed list: prefix chapters, `# Part Title` headers, nested
  numbered chapters, suffix chapters, draft chapters `[Title]()`, `---`
  separators. "Its formatting is very strict… may cause an error."
- **Two silent-drift failure modes, both demonstrated by mdBook itself:**
  (1) `.md` files not listed in SUMMARY.md are simply not rendered — they
  vanish from the site with no warning
  ([creating](https://rust-lang.github.io/mdBook/guide/creating.html));
  (2) `create-missing` defaults to **`true`** — a mistyped path in the
  manifest is silently *created as an empty chapter* at build time rather
  than erroring
  ([general config](https://rust-lang.github.io/mdBook/format/configuration/general.html)).
  Filesystem-derived nav has neither failure mode by construction.
- **Worth copying anyway:** prev/next chapter arrows built into the template
  ([index.hbs](https://rust-lang.github.io/mdBook/format/theme/index-hbs.html));
  one copy of theme/static assets per site, fingerprinted (`hash-files =
  true` default); sidebar folding *off* by default (`[output.html.fold]
  enable = false`) — everything expanded with the current chapter
  highlighted
  ([renderer config](https://rust-lang.github.io/mdBook/format/configuration/renderers.html)).
- **If cinegram ever adds a title override**, use per-file front-matter, not
  a central manifest; and any file reference that misses must be a hard
  error — `create-missing=true` is the anti-pattern.

### Marp CLI (primary) / Slidev (brief)

Marp CLI ([README](https://github.com/marp-team/marp-cli)) has the stronger
folder story:

- `--input-dir` bulk-converts a directory "keeping the origin directory
  structure" — **but generates no index page**; the output is a bag of HTML
  files with no nav. That gap is exactly what `cinegram site` adds.
- `--server` mode serves a directory with on-demand per-request conversion,
  lists files at the root, and an `index.md` (or `PITCHME.md`) in a directory
  hijacks `/` as that folder's default page; `--watch` auto-refreshes.
  **Steal:** the `index.md`-at-folder-root convention — a hand-authored page
  overrides the generated listing, no manifest needed; and the serve/build
  twin pairing (cinegram already has `preview --serve --watch` for one file —
  serving the *tree* is the natural extension, with `site` as the
  write-to-disk twin).

Slidev ([hosting](https://sli.dev/guide/hosting),
[CLI](https://sli.dev/builtin/cli)): `slidev build slides1.md slides2.md`
emits one *complete SPA copy per deck* in per-deck subfolders — no shared
assets, no index. The anti-pattern for ws1: with a shared
`runtime.js`/`mermaid.min.js`, one asset copy per site is a large size win
over per-deck duplication. Slidev's click animations advance only on user
interaction by default — consistent with the paused-by-default consensus.

---

## Cluster C — zoom & pane UX

### PhotoSwipe v5 (v5.4.4)

The reference JS image lightbox; v5 is dependency-free. Source read from
master, 2026-08-15.

- **Three-tier zoom levels** —
  ([zoom-level.js](https://github.com/dimsemenov/PhotoSwipe/blob/master/src/js/slide/zoom-level.js)):
  `initialZoomLevel` defaults to `'fit'` = `min(1, min(panAreaW/imgW,
  panAreaH/imgH))` — fit to viewport but **never upscale past 1:1**; pan area
  = viewport minus the `padding` option (default all-zero,
  [options](https://photoswipe.com/options/)). `secondaryZoomLevel` (the
  click/double-tap target) defaults to `min(1, fit * 3)` capped so scaled
  width ≤ `MAX_IMAGE_WIDTH = 4000`px; `maxZoomLevel` defaults to
  `max(1, fit * 4)`. **The [docs page](https://photoswipe.com/adjusting-zoom-level/)
  still says 2.5x/3000px — stale; the shipped source says 3x/4000px.**
- **Wheel: ctrl+wheel zooms, plain wheel pans; bare-wheel zoom is opt-in**
  via `wheelToZoom` (default false). The whole handler is
  [scroll-wheel.js](https://github.com/dimsemenov/PhotoSwipe/blob/master/src/js/scroll-wheel.js):
  `if (e.ctrlKey || options.wheelToZoom)` → zoom with the exponential,
  deltaMode-normalized factor `2 ** (-deltaY * 0.002)` (0.05 for line mode,
  1 for page mode); else pan. Equal wheel ticks = equal zoom *ratios* — steal
  the formula verbatim.
- **Cursor anchoring, one formula**
  ([slide.js `calculateZoomToPanOffset`](https://github.com/dimsemenov/PhotoSwipe/blob/master/src/js/slide/slide.js)):
  `newPan = clamp((pan - point) * (currZoom/prevZoom) + point)` per axis —
  keeps the pixel under the cursor stationary. `toggleZoom(centerPoint)`
  flips fit ↔ secondary anchored at the click point.
- **Click/close defaults**
  ([photoswipe.js defaultOptions](https://github.com/dimsemenov/PhotoSwipe/blob/master/src/js/photoswipe.js),
  ~lines 422–434): `imageClickAction: 'zoom-or-close'`,
  `bgClickAction: 'close'`, `escKey: true`, `bgOpacity: 0.8`; touch gets
  *different* defaults (`tapAction: 'toggle-controls'`,
  `doubleTapAction: 'zoom'`) — the mouse/touch split is worth copying
  ([click-and-tap docs](https://photoswipe.com/click-and-tap-actions/)).
- **Open/close**: `showHideAnimationType: 'zoom'` animates from the
  thumbnail's bounds, falling back to `'fade'` with no thumb
  ([transition docs](https://photoswipe.com/opening-or-closing-transition/)) —
  the storyboard panel `<img>` provides exactly the thumb bounds cinegram's
  zoom-open needs.

### Compiler Explorer (godbolt.org) — what a layout framework costs

CE hosts its panes in GoldenLayout — and **couldn't stay on upstream**:
`package.json` pins
[`"golden-layout": "github:compiler-explorer/golden-layout#v1.6.0"`](https://github.com/compiler-explorer/compiler-explorer/blob/main/package.json)
— a self-maintained fork of the v1 line, frozen through the v1→v2 API break,
with a `fixBugsInConfig()` pass repairing invalid saved configs before
feeding them back to the framework
([static/main.ts ~299](https://github.com/compiler-explorer/compiler-explorer/blob/main/static/main.ts)).
The tax of a docking framework: your persisted state is *its* recursive
config tree and you own its bugs forever.

**Steal the persistence shape, which is framework-independent** (main.ts):
restore order URL-hash → localStorage key `'gl'` → default; save is one
`JSON.stringify(layout.toConfig())` on **`beforeunload`**, not per drag;
localStorage access goes through a prefixed try/catch wrapper
([static/local.ts](https://github.com/compiler-explorer/compiler-explorer/blob/main/static/local.ts))
because Safari private mode throws; "reset UI" is `remove('gl')` + reload.

**Lesson for ws3:** CE needs GoldenLayout because panes are open-ended (N×N,
tabs, docking). For exactly two fixed panes the mechanism reduces to: one
flex-basis, a divider div with `pointerdown/move/up` + `setPointerCapture`,
and one localStorage number.

### Excalidraw — canvas zoom conventions

Infinite-canvas whiteboard; the de-facto zoom-UX reference alongside Figma.

- **Wheel convention**
  ([App.tsx `handleWheel`](https://github.com/excalidraw/excalidraw/blob/master/packages/excalidraw/components/App.tsx)):
  plain wheel scrolls, `ctrlKey || metaKey` + wheel zooms (exponential,
  anchored at cursor), shift+wheel scrolls horizontally. Same
  exponential-anchored formula as PhotoSwipe, **opposite bare-wheel
  default** — because an infinite canvas embedded in a page must leave plain
  wheel meaning "scroll" or it hijacks the page. **A modal lightbox has no
  page competing for the wheel, so bare-wheel zoom is defensible there;
  supporting ctrl+wheel too costs one `||`.**
- **Bounds and keys**:
  [`MIN_ZOOM = 0.1`, `MAX_ZOOM = 30`, `ZOOM_STEP = 0.1`](https://github.com/excalidraw/excalidraw/blob/master/packages/common/src/constants.ts);
  [actionCanvas.tsx](https://github.com/excalidraw/excalidraw/blob/master/packages/excalidraw/actions/actionCanvas.tsx):
  Ctrl/Cmd+0 = reset to 100%, Shift+1 = zoom-to-fit, Shift+2 = zoom to
  selection. Keyboard zoom anchors at viewport *center*, pointer zoom at the
  cursor — a sensible asymmetry to copy.

### Two-pane divider precedent (brief)

[Split.js](https://github.com/nathancahill/split/tree/master/packages/splitjs)
is the canonical tiny splitter: `sizes: [25, 75]`, `minSize` (default 100),
`gutterSize: 10`, `snapOffset: 30` (drag near min snaps closed-ish),
continuous `onDrag`. **No built-in double-click-to-collapse** — only a
programmatic `.collapse(index)` the README suggests wiring to your own
event. Even the tiniest library leaves dblclick-collapse as user code, which
is the last argument for hand-rolling ws3 entirely.

---

## Cluster D — animation frameworks (background depth)

*Sources fetched 2026-08-15: Motion Canvas main branch, Remotion docs, manim
stable docs. motioncanvas.io itself 403'd fetches, so Motion Canvas claims
cite the `.mdx` sources in its GitHub repo.*

### Motion Canvas

Generator-based procedural animation: a scene is a generator; `yield*`
composes via `all()`, `chain()`, `sequence()`, `waitFor()`
([flow.mdx](https://github.com/motion-canvas/motion-canvas/blob/main/packages/docs/docs/getting-started/flow.mdx)).

- **Seeking through procedural time is expensive** —
  [`PlaybackManager.seek()`](https://github.com/motion-canvas/motion-canvas/blob/main/packages/core/src/app/PlaybackManager.ts)
  cannot jump backwards: it finds the best cached scene state, calls
  `scene.reset()`, and replays forward frame-by-frame; `recalculate()`
  re-runs from frame 0. Cinegram's absolute-ms IR gets O(1) seek for free —
  protect that explicitly: never introduce state only forward playback can
  compute.
- **Steal: named time events.** `yield* waitUntil('event')` names a sync
  point instead of hard-coding a duration; the editor shows events as
  draggable pills, dragging cascades to downstream events (SHIFT disables);
  `useDuration('event')` reuses an event's length
  ([time-events.mdx](https://github.com/motion-canvas/motion-canvas/blob/main/packages/docs/docs/getting-started/time-events.mdx)).
  Maps to named anchors in `.dgm` replacing `at:` offsets. (Where event
  offsets persist is unconfirmed from the docs read.)

### Remotion

React framework: video as a pure function of frame number — composition
declares `durationInFrames`/`fps`, components read `useCurrentFrame()`,
"given the same frame number, a component should render identically every
time" ([fundamentals](https://www.remotion.dev/docs/the-fundamentals)).
Validates cinegram's deterministic `frame --at`.

- **Player defaults, everything off**
  ([player docs](https://www.remotion.dev/docs/player/player)):
  `autoPlay: false`, `controls: false`, `loop: false`, `initialFrame: 0`
  (fixed at mount), and **`clickToPlay` defaults `true` only if `controls`
  is true** — chromeless embeds don't intercept clicks. Directly relevant to
  ws2, and the clickToPlay↔controls coupling matters for cinegram because
  `interact` bindings live on the SVG. Event split worth copying: `seeked` /
  `timeupdate` / `frameupdate`. `initialFrame`-fixed-at-mount contrasts with
  cinegram's re-navigable hash deep links — cinegram's is the stronger
  design.

### manim

Python batch scene renderer; `self.play(Transform(…), run_time=…)`; no
interactive player by default.

- **Steal: sections + manifest.** `self.next_section(name,
  skip_animations=…)` cuts a scene into individually rendered spans;
  `--save_sections` writes `sections/SceneName_XXXX.mp4` (zero-padded,
  declaration order) plus a `SceneName.json` sidecar with per-section name,
  duration, frame count
  ([output_and_config](https://docs.manim.community/en/stable/tutorials/output_and_config.html)).
  For cinegram: steps already are the section unit — the exportable idea is a
  machine-readable manifest (step id → start ms, duration) beside `record`
  output, plus per-step partial export. `skip_animations` is their
  fast-iteration hack; cinegram doesn't need it (seek is a lookup) — which is
  itself the takeaway about procedural vs declarative time.

**Cross-cutting:** all three converge on deterministic time-indexed
rendering; procedural time (Motion Canvas) pays replay cost for seeks,
declarative time (Remotion frames, cinegram ms) doesn't.

---

## Adoption verdicts

Gates: the runtime must stay a classic script (no ES modules; `file://` and
webview CSP), assets vendored into `pkg/emit/html/assets/`, repo is MIT.
Go side: **stdlib-only stays the default**, negotiable only for one big win
(see the D2/dagro table in Cluster A). Sizes are minified from the published
npm packages via unpkg; dates from deps.dev; all verified 2026-08-15
(registry.npmjs.org was unreachable, so package contents were read from the
tarballs unpkg serves verbatim).

| Library | Min size | Deps | License | Classic script? | Last release | Verdict |
|---|---|---|---|---|---|---|
| [Split.js](https://github.com/nathancahill/split/blob/master/packages/splitjs/README.md) | 6.77 kB ([UMD](https://unpkg.com/split.js@1.6.5/dist/split.min.js), global `Split`) | 0 | MIT | Yes | 1.6.5, 2022-01 | **Skip** — dblclick-collapse isn't built in anyway (only `.collapse(index)`), and the CE lesson says two fixed panes need one pointer handler + one localStorage number. Hand-roll ws3. |
| [medium-zoom](https://github.com/francoischalifour/medium-zoom/blob/master/README.md) | 9.64 kB (UMD, global `mediumZoom`) | 0 | MIT | Yes | 1.1.0, 2023-11 | **Rejected for ws4** — click-to-zoom-fit only; **no wheel zoom at all** (scrolling while zoomed *dismisses*, `scrollOffset` default 40px). Fails the decided cursor-anchored-wheel requirement. |
| [svg-pan-zoom](https://github.com/bumbu/svg-pan-zoom/blob/master/README.md) | 29.8 kB (UMD, global `svgPanZoom`) | 0 | BSD-2 | Yes | 3.6.2, 2024-10 | **Not for ws4** (SVG elements only — cannot handle raster `<img>`, which storyboard frames are). Keep on file if diagram-SVG pan/zoom is ever wanted. |
| [PhotoSwipe v5](https://app.unpkg.com/photoswipe@5.4.4/files/dist/umd) | 54.5 + 14.6 kB UMD (+7.4 kB CSS) | 0 | MIT | Yes, with caveat — 5.4.4 ships `dist/umd/` (globals `PhotoSwipe`, `PhotoSwipeLightbox`) but its own README calls UMD second-class ("use only if unable to use ESM") | 5.4.4, 2024-05 | **Viable but not recommended** — ~70 kB + CSS + officially deprecated build for one image class. **Steal its mechanics instead** (formulas above are ~30 lines). |
| [GoldenLayout v2](https://github.com/golden-layout/golden-layout/blob/master/docs/index.md) | no single file; cjs/ 633 kB | 0 | MIT | **No** — CJS+ESM only, no UMD/IIFE | 2.6.0, 2022-09; own docs say npm modules stale, build from source | **BLOCKED** (classic-script gate) and dormant; CE itself froze on a self-maintained v1 fork. Nothing here for a two-pane case. |
| [asciinema-player](https://github.com/asciinema/asciinema-player/blob/develop/README.md) | 185 kB (self-contained IIFE, global `AsciinemaPlayer`) | 3, all bundled | **Apache-2.0** (player specifically — not GPL) | Yes | 3.17.0, 2026-06 | **Not a vendoring candidate** (terminal player) — it's the *design reference* for ws2 defaults. License verified only because its option model is being copied. |
| [panzoom](https://github.com/anvaka/panzoom/blob/master/README.md) (anvaka) | 33.8 kB (UMD, deps bundled) | 3 bundled | MIT | Yes | 9.4.4, 2026-03 | **Fallback for ws4** if hand-rolling stalls: cursor-anchored wheel zoom on DOM *and* SVG, maintained, single file. Default remains hand-roll with PhotoSwipe's formulas. |
| Motion Canvas / Remotion / manim | — | — | — | — | — | **LEARN-ONLY, all three**: Motion Canvas and Remotion are TypeScript/React frameworks (fail classic-script + vendorable gates); manim is a Python toolchain (fails no-new-toolchain gate). Ideas only, per Cluster D. |

Go-side big-win candidates (dagro; D2 v0.8 layout+d2svg) are in the Cluster A
table — each would come back as its own decision, not adopted from this
document.

---

## Apply this

### WS1 — `cinegram site <dir>`

- **The generated index is the product**: Marp mirrors folders but emits no
  index; Slidev emits per-deck islands; mdBook makes a real site but demands
  a manifest with two silent-drift failure modes (unlisted files vanish;
  `create-missing=true` fabricates empty chapters). The decided
  filesystem-derived nav + numeric prefixes has none of those failure modes
  by construction — **nothing found contradicts the numeric-prefix
  decision**; the survey strengthens it.
- Steal: Marp's `index.md`-at-folder-root override for a folder's landing
  page; mdBook's prev/next arrows, everything-expanded sidebar with current
  page highlighted, and one fingerprinted copy of shared assets at site
  root; Structurizr's static-export principle of shipping the *interactive*
  runtime (bookmarkable per-diagram hash URLs; later, a `Space` type-ahead
  over all diagrams) and site-generatr's "the tree drives nav, but not every
  artifact deserves an entry" (mirrors the existing publish-site rule that
  drill-down-only `.dgm` files get no page).
- If title overrides are ever needed: per-file front-matter (or the existing
  leading `%%` block), never a central manifest; missing references must
  hard-error.

### WS2 — paused by default

- Unanimous across the survey: asciinema `autoPlay: false`, Remotion
  `autoPlay: false`, Slidev manual clicks. **No contradiction with the
  decision anywhere.** Spell opt-in as one boolean, per asciinema.
- Steal beyond the flip: a `poster`-style resting frame (`npt:`-like "render
  the frame at time t as the paused state" — cheap given deterministic
  `seek`); asciinema's `pauseOnMarkers` as a step-through *player option*;
  three-state `controls: "auto"` for many-players-per-page; Remotion's
  clickToPlay-follows-controls coupling so chromeless embeds never intercept
  clicks (`interact` bindings live on the SVG); the
  `seeked`/`timeupdate`/`frameupdate` event split if the runtime grows a
  player-events API.

### WS3 — draggable persisted split

- **Hand-roll it; no library.** CE's GoldenLayout history is the cautionary
  tale (frozen v1 fork + config-repair pass); GoldenLayout v2 is BLOCKED on
  the classic-script gate anyway; even Split.js leaves dblclick-collapse to
  user code. The mechanism: divider div, `pointerdown/move/up` with
  `setPointerCapture`, write `--pg-left` (the variable already exists in
  `playground.css`), clamp to min sizes.
- Copy the *shapes*: Mermaid Live's percentage-based, id-keyed persistence
  (`defaultSize 30`, `minSize 15`) and its <640px replace-with-toggle
  instead of squeezing; CE's save-on-`beforeunload` (or drag-end), try/catch
  prefixed localStorage wrapper, and reset = remove key + reload; dblclick
  toggles collapse by swapping the stored value with a sentinel.

### WS4 — storyboard lightbox

- Hand-roll using PhotoSwipe's shipped-source mechanics (trust source over
  its stale docs): open at `fit = min(1, viewport/image)` with padding
  subtracted; click toggles fit ↔ `min(1, fit*3)` anchored at the click
  point; wheel zoom factor `2 ** (-deltaY * 0.002)` (deltaMode-normalized);
  cursor anchor `newPan = clamp((pan - point) * ratio + point)`; max
  `fit*4`; Esc + backdrop-click close; zoom-open animated from the storyboard
  panel `<img>` bounds, fade fallback.
- **One nuance vs the decided "mouse-wheel zoom"**: both zoom references
  reserve bare wheel for pan/scroll and zoom on ctrl+wheel (PhotoSwipe
  default `wheelToZoom: false`; Excalidraw ctrl/cmd+wheel). The environment
  distinction resolves it: a modal overlay has no competing scroll, so
  bare-wheel zoom is defensible *in the lightbox* — support both
  (`e.ctrlKey || true`-style, one `||`), and never bare-wheel-zoom outside a
  modal context. Not a contradiction, a scoping rule.
- medium-zoom rejected (no wheel zoom); svg-pan-zoom rejected (no raster);
  anvaka/panzoom is the vetted fallback if hand-rolling stalls.

### WS5 — rolodex

- **The interaction is already proven**: Structurizr's double-click →
  next-level-if-unambiguous, else a modal listing every destination
  (views/docs/links) is exactly click-object → flip-through-scenes. Reserve
  double-click (single click stays free for existing `interact` bindings);
  route through `location.hash` as today.
- The inverted object→scenes index over `ir.Timeline` mirrors Structurizr's
  one-model-many-views property; D2 validates format-independent author-facing
  links rewritten at emit time, and its breadcrumb backlinks suit drill-down
  paths.
- Avoid Structurizr's perspectives-style second annotation layer — enumerate
  existing scenarios/views touching the element; do not invent a parallel
  vocabulary. D2's fixed-interval auto-flip is the anti-pattern for
  presenting the flip-through: give it the scrubber/stepper the runtime
  already has.
- Adjacent, from Motion Canvas: named time events (anchors) in `.dgm` would
  give rolodex entries stable landing points (`scenario s2 @ anchor
  'commit'`) instead of raw milliseconds.
