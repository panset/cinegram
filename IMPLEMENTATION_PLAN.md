# Cinegram — Implementation Plan

Ten sequential phases, each sized for one engineer (or one Opus subagent) working
alone to completion. Every phase ships user-visible value, ends with all tests
green, and adds at least one new example under `examples/` that showcases what
it built. Later phases depend on earlier ones only where noted, but the order
below is the intended execution order.

## Ground rules for every phase

Read `CLAUDE.md` first. The invariants that no phase may break:

- **`pkg/parser` does no I/O.** `Parse(filename, content)` stays string-in.
- **No geometry in `pkg/ir`.** Tracks and bindings reference IDs only; times are
  absolute integer milliseconds. Renderers own geometry.
- **No third-party Go dependencies.** Standard library only.
- **Diagram vocabulary stays out of** `scenario.go`, `interact.go`,
  `validate.go`, `pkg/compile`, and `pkg/ir`. The halves meet only through
  `symbol.Table`.
- **Mermaid emission is a reprint.** Unmodeled syntax round-trips through
  `ast.RawStmt` verbatim.
- **The preview page stays self-contained** (no external URLs;
  `html_test.go` enforces it) and `runtime.js` stays a classic script.
- **Runtime frame application must be deterministic under seek.** `apply(t)`
  is a pure function of `t` — never accumulate animation state across frames.
  Anything persistent (reveal, prefs) lives outside the clock and must be
  listed in `STICKY` if it is expressed as a `dgm-*` class.

Definition of done for a phase:

1. `bazel build //...` and `bazel test //...` pass.
2. Golden fixtures regenerated with `bazel run //pkg/parser:parser_test -- -update`
   and `bazel run //pkg/compile:compile_test -- -update` (never under `bazel test`).
3. `bazel run //:gazelle` if packages were added/moved; `gofmt` run via the
   hermetic SDK binary (see CLAUDE.md).
4. The phase's new example compiles (`bazel run //cmd/cinegram -- compile examples/<x>.dgm`),
   lints clean, and previews correctly. Verify runtime changes in a browser by
   serving the page and driving `window.CINEGRAM_PLAYER.seek(ms)`
   deterministically (see "Verifying animation changes" in CLAUDE.md).
5. `README.md` documents any new DSL syntax, CLI flag, or player control.
6. IR changes bump nothing unless the schema shape changes incompatibly; new
   *optional* fields do not require a `Version` bump, new required semantics do.

Phase index:

| # | Phase | Theme |
|---|-------|-------|
| 1 | Honor the compiled contract | `color`, `ease`, `speed`, `autoplay` actually work |
| 2 | Flow storytelling | Edge highlighting, trails, waypoint pulses, failure flows |
| 3 | Narration layer | Step `desc`, stage caption, scrubber step markers, aria-live |
| 4 | Persistent state | `set` badges and `gauge` values that survive across steps |
| 5 | Attention control | `focus` spotlight, reveal affordances, note placement |
| 6 | Links & deep linking | `click -> url`, shareable time links, embed mode |
| 7 | Player polish | Keyboard help, prefs, reduced motion, pan/zoom, a11y |
| 8 | Toolchain for agents | `narrate` command, JSON lint output, deeper lint |
| 9 | Authoring loop | `preview --serve --watch`, `frame` capture |
| 10 | sequenceDiagram | Second diagram type, proving the registry abstraction |

---

## Phase 1 — Honor the compiled contract

**Problem.** The compiler already emits `Color`, `Ease`, `Speed` and `Autoplay`
into the IR (`compile.go:171,315-316,345-346`), but `runtime.js` reads none of
them. Users can write these attributes today and silently get nothing.

**Scope.**

- **`color`** on `flow`, `highlight`, `pulse`: the runtime applies it as an
  inline CSS custom property (e.g. `--dgm-color`) on the particle group / node
  group, and `runtime.css` consumes it with fallbacks to the current defaults.
  Accept any CSS color string; do not validate in Go.
- **`ease`** on `flow`: values `linear` (default), `in`, `out`, `in-out`.
  Implement as a progress remap in `applyFlows` (`p' = ease(p)`) so seeking
  stays deterministic. Lint rejects unknown ease names with a hint.
- **Scenario `speed`**: `selectScenario`/`setView` initialize `player.speed`
  from `scenario.speed`; the speed button continues to cycle absolute values
  and its label always reflects the effective speed.
- **`autoplay`**: after the first successful render of a view, if the selected
  scenario has `autoplay: true` (compiler default) and
  `matchMedia('(prefers-reduced-motion: reduce)')` does not match, start
  playback. Navigating between views re-applies the rule.
- **Lint**: introduce a per-action known-attribute table in
  `pkg/parser/validate.go` (or a new `attrs.go` beside it) and warn on unknown
  attribute keys with a "did you mean" hint. This catches `colour:`/`durr:`
  typos, which today vanish silently. Keep the table diagram-agnostic.

**Files.** `pkg/emit/html/assets/runtime.js`, `runtime.css`,
`pkg/parser/validate.go`, tests in `pkg/parser` and `pkg/compile`
(attr passthrough goldens), `README.md`.

**Example — `examples/deploy-pipeline.dgm`.** A CI/CD pipeline
(`dev → git → CI → registry → staging → prod` with a rollback edge). Uses
`autoplay: true`, `speed: 1.5`, green `color` on the deploy flows, a red
rollback flow, and `ease: in-out` on the promotion hops. Copy into
`pkg/compile/testdata/` with a golden JSON so attr passthrough is pinned.

**Acceptance.** Serving the example and calling `CINEGRAM_PLAYER.seek()`
mid-flow shows a colored particle; loading the page starts playback unattended;
a `.dgm` with `colour: red` lints with a warning naming `color`.

---

## Phase 2 — Flow storytelling

**Scope.**

- **Active-edge highlighting.** While a flow track is open at time `t`, its
  edge path gets a `dgm-flow-active` class (plus `dgm-flow-<style>` when the
  track has a style, and the Phase-1 `--dgm-color` variable). CSS gives it a
  brighter stroke and slightly increased width with a short transition.
  Applied in `applyFlows` via the same diff-against-previous-frame pattern as
  `applyNodeStates` — and removed classes must respect `STICKY`.
- **Trail.** A comet effect on the active edge using `stroke-dasharray`
  computed from the particle's current arc length: a second overlaid path
  (cloned into the `dgm-layer`) whose dash window trails the particle. Purely
  derived from `t` — no per-frame state.
- **Waypoint pulses.** A multi-hop flow already compiles to one track per hop.
  During the final 15% of each hop's time range, the hop's destination node
  gets a `dgm-waypoint` pulse class. Derived from `t`, so scrubbing backwards
  works.
- **Failure flows.** New flow attribute `status: fail` (parser: allow on
  `flow`; compiler: copy to a new optional `Track.Status string` IR field).
  Runtime: the particle takes an error appearance, the edge gets
  `dgm-flow-fail`, and during the final 20% of the track an ✕ marker is drawn
  at the destination end of the path (in the `dgm-layer`, so it sits above
  nodes). `style: error` remains a plain CSS hook; `status` is semantic.
- **Lint**: `status` accepts only `ok` (default) and `fail`.

**Files.** `runtime.js`, `runtime.css`, `pkg/ir/ir.go` (optional `Status`),
`pkg/compile/compile.go`, `pkg/parser/validate.go` (attr table), goldens,
`README.md`.

**Example — `examples/payment-checkout.dgm`.** Checkout flow:
`browser → checkout → payment-gw` where the first attempt has
`status: fail` ("gateway timeout"), a retry flows to a fallback provider,
succeeds, and the response returns. Two scenarios: "happy path" and
"gateway outage" — showcasing that failure paths are first-class.

**Acceptance.** Seeking into a hop shows the lit edge and trail; seeking to a
fail flow's final quarter shows the ✕; seeking backwards clears everything.

---

## Phase 3 — Narration layer

**Scope.**

- **Step `desc`.** A `desc: "..."` attribute inside a `step` block (the
  attr grammar already supports it — no scanner changes). Compiler copies to a
  new `ir.Step.Desc string`. Multi-sentence strings with `\n` allowed.
- **Stage caption.** A `dgm-caption` element between the stage and the footer
  showing the active step's name (bold) and desc (regular). Empty between
  steps/before start. Updated from `syncChrome`.
- **aria-live.** The caption region carries `aria-live="polite"` so step
  transitions are announced; the announcement is the step name + desc.
- **Scrubber step markers.** Keep the `<input type=range>` but overlay a
  `dgm-scrub-marks` element with one tick per step boundary, positioned by
  `start / duration`. Ticks are clickable (seek to step start) and carry the
  step name as a tooltip. Recomputed in `buildSteps`.
- **Step list stays** as the table of contents; the caption is the "you are
  here" narration. Step list items also show the first line of `desc` as a
  `title` tooltip.

**Files.** `pkg/ir/ir.go`, `pkg/compile/compile.go`, `runtime.js`,
`runtime.css`, goldens, `README.md`.

**Example — `examples/oauth-login.dgm`.** OAuth 2.0 authorization-code flow
(`browser`, `app`, `auth server`, `resource API`) where every step has a real
two-sentence `desc` explaining *why* ("The app never sees the user's password;
it only ever holds the short-lived code…"). This is the flagship "explorable
explanation" example — invest in the prose.

**Acceptance.** Scrubbing moves the caption in step with the highlight in the
step list; ticks are visible on the scrubber and clicking one seeks; a screen
reader (or inspecting the live region) sees step announcements.

---

## Phase 4 — Persistent state: `set` and `gauge`

**Design.** Tracks live inside steps, and `apply(t)` skips steps whose window
excludes `t` — so persistent state cannot be a step track. Add
`ir.Scenario.Persistent []Track`: tracks whose `Start` is when the action
fires and whose `End` is the scenario duration *or* the moment a later action
overwrites/clears the same key. The **compiler** computes all of this (it owns
timing); the runtime just applies any persistent track whose window contains
`t`. Seek-determinism falls out for free.

**Scope.**

- **`set <node|group> { badge: "leader" }`** — attaches a small pill to the
  element from this moment on. `set x { badge: "" }` (or `unset x`) ends it:
  the compiler closes the previous track's `End` at that time. Multiple nodes:
  `set a, b { badge: "follower" }` (arityTarget). Also support
  `state: <name>` in the same block as a persistent CSS class
  (`dgm-state-<name>`) for recoloring without a label.
- **`gauge <node> { label: "term", value: 3 }`** — a labeled value pill.
  Subsequent `gauge` actions on the same `(target, label)` update the value:
  compiler closes the previous track and opens a new one. Values are strings
  in the IR (render-only; no arithmetic).
- **Parser**: two new entries in `actionKinds` (`set`, `gauge` — arityTarget),
  new `ast.ActionKind`s. Validate: `gauge` requires `label` and `value`;
  targets must resolve in the symbol table (existing mechanism).
- **Runtime**: render badges and gauges as HTML overlay pills (same technique
  as notes: positioned off `getBoundingClientRect`), anchored to the node's
  top-right corner, stacking if a node has several. New track kinds
  `set`/`gauge` in a new `applyPersistent(t)` called from `apply`.
- **IR**: `TrackSet`, `TrackGauge` kinds; `Track.Value string` field;
  `Scenario.Persistent`.

**Files.** `pkg/ast/ast.go`, `pkg/parser/scenario.go` (+2 map entries),
`pkg/parser/validate.go`, `pkg/ir/ir.go`, `pkg/compile/compile.go` (the
open/close bookkeeping — this phase's real work), `runtime.js`, `runtime.css`,
goldens, `README.md`.

**Example — `examples/raft-election.dgm`.** Five-node Raft cluster. Steps:
leader heartbeats (badge `leader` on n1), leader dies (`hide`/`dim` + badge
cleared), election timeout (badge `candidate` on n3, gauge `term` increments,
gauge `votes` counts up per step), new leader elected (badge `leader` on n3),
log replication resumes. Badges and gauges make the cluster's state legible at
any scrub position — the feature's whole point.

**Acceptance.** Seeking to any time shows exactly the badges/gauges implied by
the actions before that time; scrubbing backwards removes them; the compile
golden pins the open/close windows.

---

## Phase 5 — Attention control

**Scope.**

- **`focus <group|node>[, ...]`** — a stateful action (spans its step when it
  has no `dur`). While active, every node *not* in the focus set gets
  `dgm-unfocused` (heavy dim + desaturate). Group targets include their
  transitive children — the **runtime** expands groups via `view.groups`
  (`Children` is already in the IR), keeping the track ID-only. Edges between
  two unfocused nodes also dim.
- **Reveal affordances.** Any element that is the `source` of a `reveal`
  binding gets a small "+N" chip (N = count of currently-hidden targets),
  rendered in the overlay and updated when reveal state toggles; it flips to
  "–" when open. Any source of a `view` binding gets a corner ⤢ glyph chip.
  Chips are clickable (same activation as the element).
- **Reveal transition.** `dgm-collapsed` gains an opacity/scale transition in
  CSS so reveals unfold instead of popping. Keep `display` off the table —
  animate opacity + `pointer-events`.
- **Note placement.** `note` gains optional attrs `side: above|below|left|right`
  (default `above`) and `dur` (already supported). Compiler copies `side` to a
  new optional `Track.Side`. Runtime positions accordingly and **clamps** the
  note into the stage rect. When two notes would overlap, nudge the later one
  below the earlier (simple one-pass vertical shove, not a solver).

**Files.** `pkg/parser/scenario.go` (`focus` entry), `pkg/ast`, `pkg/ir`
(`TrackFocus`, `Track.Side`), `pkg/compile`, `runtime.js`, `runtime.css`,
goldens, `README.md`.

**Example — `examples/layered-arch.dgm`.** A layered service (edge layer,
application layer, domain layer, data layer as subgraphs). The scenario walks a
request down the layers with `focus` isolating one layer per step; an
`interact` block reveals a hidden "cross-cutting concerns" subgraph
(observability, authn) via a "+3" chip; notes use `side: right` alongside the
focused layer.

**Acceptance.** Seeking into a focus step dims everything else; the +N chip is
visible before clicking and flips after; notes never render outside the stage.

---

## Phase 6 — Links & deep linking

**Scope.**

- **`click X -> url "https://…"`** — fourth binding kind in
  `pkg/parser/interact.go` (`Binding.Kind = "url"`, reuse `Binding.View`? no —
  add `URL string` to `ir.Binding`). Runtime opens with
  `window.open(url, '_blank', 'noopener')`. Lint warns on non-http(s) schemes.
- **Deep links.** Today the hash is a bare view id. New format, backward
  compatible: `#v=<viewID>&s=<scenarioID>&t=<ms>` (a hash without `=` is
  still treated as a bare view id). `applyHash` parses all three and applies
  view → scenario → seek, paused. `navigate()` keeps writing the short form
  for plain drills.
- **Share button.** A `Copy link` button in the control bar writes the full
  deep link for the current view/scenario/time to the clipboard and flashes
  confirmation. This is how "look at *this* step" gets into PRs and Slack.
- **Embed mode.** `?embed` (query string, not hash) hides the header bar and
  the step list, keeping stage + caption + scrubber. Document an iframe
  snippet in the README. The keyboard handler still works when the iframe has
  focus.

**Files.** `pkg/parser/interact.go`, `pkg/parser/validate.go`, `pkg/ir/ir.go`
(`Binding.URL`), `pkg/compile`, `runtime.js`, `runtime.css`, parser goldens
(extend `interact-corners.dgm`), `README.md`.

**Example — `examples/incident-triage.dgm`.** A service map used during
incident response: every service node has `click svc -> url` to its (example.com)
dashboard/runbook, the scenario replays the outage timeline, and the README
shows a deep link into the moment the cascade starts plus an embed snippet.

**Acceptance.** Opening `page.html#v=root&s=outage&t=3200` lands paused at
3.2s in the right scenario; Copy link round-trips; `?embed` renders chromeless;
clicking a service opens its URL in a new tab.

---

## Phase 7 — Player polish (input, prefs, motion, scale, a11y)

**Scope.**

- **Keyboard help.** `?` toggles an overlay listing all shortcuts. New keys:
  `Home`/`End` (start/end), digits `1–9` (jump to step *n*), existing
  Space/←/→/Esc documented.
- **Preference persistence.** Theme and speed persist in `localStorage`
  (`dgm.theme`, `dgm.speed`). Theme additionally follows live
  `prefers-color-scheme` *changes* until the user explicitly overrides via the
  button (store `'light' | 'dark' | null`-for-auto).
- **Reduced motion.** With `prefers-reduced-motion: reduce`: no autoplay
  (Phase 1 already gates this — verify), CSS transitions minimized, and the
  help overlay advertises arrow-key step-through as the primary mode.
- **Pan/zoom.** Wheel-zoom (cursor-anchored) and drag-pan on the stage,
  implemented as a CSS transform on the SVG holder; double-click or a `⌂`
  button resets. Notes/badges/gauges position off `getBoundingClientRect`,
  which reflects transforms — verify they track while zoomed. Zoom state
  resets on view change.
- **Keyboard a11y.** Clickable SVG elements get `tabindex="0"`,
  `role="button"`, an `aria-label` from the binding label, and
  Enter/Space activation; step list items become real `<button>`s; visible
  focus outlines in CSS. Chips from Phase 5 inherit the same treatment.

**Files.** `runtime.js`, `runtime.css`, `README.md`. No Go changes expected.

**Example — `examples/data-platform.dgm`.** A deliberately large diagram
(~25 nodes: sources → Kafka → stream processing → lake → warehouse → BI, plus
an orchestration subgraph) whose scenario tours ingestion. Its size is the
point: it demos pan/zoom, digit-key step jumps, and why `Home`/`End` matter.

**Acceptance.** Zoom in, play a flow — particles, notes and badges stay glued
to their nodes; reload restores theme/speed; tab reaches every clickable node
and Enter activates it; `?` shows the overlay.

---

## Phase 8 — Toolchain for agents: `narrate` and machine-readable lint

**Scope.**

- **`cinegram narrate <file> [-o out.md] [--format=md|json]`.** A new
  `pkg/emit/narrate` package, pure over `ir.Timeline` (no parser/loader
  imports beyond what `main.go` already wires). Markdown output: per view, per
  scenario, an ordered walkthrough — step heading (name), the `desc` prose
  (Phase 3), then one bullet per track in plain English:
  `- **client → lb** carries "GET /api/orders" (0.0s–0.7s)`,
  `- **lb** is highlighted (active)`, `- badge on **n3**: "candidate" (from 4.2s)`.
  Include interact bindings as a final "Interactions" list. JSON format emits
  the same structure as data. This is the artifact an AI agent reads instead
  of watching the animation.
- **`lint --format=json`.** Diagnostics as
  `[{"file","line","col","severity","message","hint"}]` on stdout, exit code
  semantics unchanged (warnings 0, errors 1). Plumb through `diag.Bag`.
- **Deeper lint.** New warnings: a node/group never referenced by any
  scenario, interact, or edge label (hint: "is `etcd` supposed to be
  animated?"); an `interact` step target that exists only in a non-first
  scenario (the runtime binds against the selected scenario — say so); a
  `view` declared but never targeted; duplicate scenario names.

**Files.** `pkg/emit/narrate/` (new — run gazelle), `cmd/cinegram/main.go`,
`pkg/parser/validate.go`, `pkg/loader` (only if narrate needs bundle access —
it should consume the compiled `ir.Timeline`), narrate golden tests
(`narrate_test.go` with `.golden.md` fixtures), `README.md`.

**Example.** Run narrate over the Phase-3 example and commit the output as
`examples/oauth-login.narrate.md`, referenced from the README as "what an AI
agent sees". Also add a deliberately imperfect `pkg/parser/testdata` fixture
exercising each new lint warning, pinned in `errors.golden`.

**Acceptance.** `bazel run //cmd/cinegram -- narrate examples/oauth-login.dgm`
reproduces the committed file; `lint --format=json examples/…` emits valid JSON
(pinned by a golden); the new warnings fire on the fixture and nowhere in
`examples/`.

---

## Phase 9 — Authoring loop: serve/watch and frame capture

**Scope.**

- **`preview --serve [addr] [--watch]`.** Stdlib `net/http` server that
  compiles in-memory and serves the page at `/` (no temp files). With
  `--watch`, poll the source file set's mtimes (the `loader.Bundle` knows
  every file) every ~300ms and bump a generation counter; the served page —
  only in serve mode — includes a tiny injected script that polls
  `/generation` and reloads on change. Default addr `127.0.0.1:8731`. This
  also retires the "python3 -m http.server" workaround in CLAUDE.md (update
  it).
- **`cinegram frame <file> --at 2400ms -o out.png [--scenario id] [--view id]`.**
  Renders one deterministic moment: start the serve server on an ephemeral
  port, build the Phase-6 deep link (`#v=…&s=…&t=…`), and shell out to a
  headless Chrome/Chromium (`--headless=new --screenshot=… --window-size=…`),
  found via `$CINEGRAM_CHROME`, then a small candidate list
  (`google-chrome`, `chromium`, the macOS app path). Clear error if none
  found. `--frames N -o dir/` captures N evenly spaced moments; the README
  documents the ffmpeg/ImageMagick one-liners to make a GIF from them —
  do **not** implement GIF encoding in Go.
- Keep the third-party-free rule: shelling out to an existing browser is fine,
  vendoring a screenshot library is not.

**Files.** `cmd/cinegram/main.go` (+ a `cmd/cinegram/serve.go`,
`capture.go`), `pkg/emit/html` (only if the reload script injection needs a
hook — prefer injecting in the serve handler, keeping the emitted file
byte-identical to non-serve output), `CLAUDE.md` (verification workflow),
`README.md`.

**Example.** No new `.dgm` needed; the showcase is workflow. Add a README
"Authoring" section with the watch loop, and a `frame` invocation over
`examples/payment-checkout.dgm` capturing the failure moment; commit that one
PNG as `examples/payment-checkout.fail.png` so the repo shows the feature.
Add a Go test for the serve handler (compile + serve + `/generation` bump via
an in-memory read function); gate an end-to-end frame smoke test behind
`CINEGRAM_CHROME` being set, skipping otherwise.

**Acceptance.** Editing a watched `.dgm` reloads the browser within ~1s;
`frame --at` produces a PNG whose content matches the seeked state; both
commands print actionable errors (port busy, chrome missing).

---

## Phase 10 — sequenceDiagram support

The riskiest phase, deliberately last. The architecture promises this costs
one new parser; the runtime binding is where the real risk lives — **start
with a spike**: hand-write a small mermaid `sequenceDiagram`, render it,
and inspect the SVG structure of the target mermaid version before writing
any Go.

**Scope.**

- **Parser.** `pkg/parser/sequence.go` implementing `registry.DiagramParser`,
  registered in `init()` for header `sequenceDiagram`. Model:
  `participant`/`actor` lines → nodes (alias handling: `participant A as Api`);
  message lines (`A->>B: text`, `A-->>B: text`, `A-)B: text` …) → edges, one
  per occurrence, ids unique per occurrence and ordered; `Note`, `activate`,
  `loop/alt/opt/end`, `box` → `ast.RawStmt` (reprint keeps them). Hand the
  cursor back at every keyword in `isTopLevelKeyword` — that is the registry
  contract.
- **Symbol table.** Repeated `A→B` messages mean `FindEdge` can be ambiguous.
  Rule: a `flow` resolves to the **first not-yet-used** matching message
  within the scenario, in message order; an optional flow attr `msg: <n>`
  (1-based occurrence) overrides. Validation warns when ambiguity was resolved
  implicitly. This logic lives in `symbol.Table`/validate — not in
  `scenario.go`, which stays vocabulary-free.
- **Runtime.** Sequence SVGs have no `g.node`/`.edgePaths`. Extend `index()`
  to detect diagram type from `ir.Diagram.Type` and use a second indexing
  strategy: actor boxes by their text/order for "nodes"; message lines/paths
  matched by vertical order for edges (they render top-to-bottom in message
  order — order-based matching is more robust than geometry here). Flows
  travel along the message line; `highlight` lights the actor box (top and
  bottom); notes anchor to the actor's lifeline at the current message's
  height. Anything unbindable surfaces in the existing warning banner.
- **Everything else is free.** Scenarios, `desc`, `set`/`gauge`, `focus`,
  interact (click an actor → view/url), narrate, frame capture must all work
  unchanged — that's the point. The example must prove it by using several.

**Files.** `pkg/parser/sequence.go` (+ tests, goldens), `pkg/registry` (only
if registration needs anything new — it shouldn't), `runtime.js` (second
indexer), `README.md`.

**Example — `examples/websocket-handshake.dgm`.** A `sequenceDiagram` of a
WebSocket upgrade (`client`, `lb`, `server`): HTTP GET with Upgrade header,
101 Switching Protocols, then ping/pong frames — with step `desc` narration, a
`gauge` for open connections, and `click server -> view` drilling into a
flowchart view of the server internals, proving cross-type view bundles work.

**Acceptance.** The example previews with flows travelling along message
arrows; `mermaid` subcommand reprints the sequence body byte-faithfully
(golden); the k8s flowchart examples are untouched by the runtime changes;
`narrate` output for the example reads correctly.

---

## Dependency notes

- Phase 2 uses Phase 1's `--dgm-color` plumbing.
- Phase 8's narrate is worth little without Phase 3's `desc` — keep the order.
- Phase 9's `frame` builds its URL from Phase 6's deep links.
- Phases 4 and 5 are independent of each other; either could be swapped.
- Phase 10 touches everything read-only and must come after the features it
  needs to prove diagram-agnostic (3, 4, 5, 6).

## Suggested subagent briefing template

> Implement Phase N of IMPLEMENTATION_PLAN.md in this repo. Read CLAUDE.md and
> the "Ground rules" section of the plan first; they override any default
> approach. Do not start work belonging to other phases. Finish with the
> phase's Definition of done: tests green via bazel, goldens regenerated via
> `bazel run … -- -update`, gazelle + gofmt run, the new example added and
> verified in a served browser via `CINEGRAM_PLAYER.seek()`, README updated.
