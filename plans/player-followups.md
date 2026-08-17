# Player follow-ups — the debt the presentation round deferred

## Context

PR #25 ("Fit the player to the screen…") shipped nine player behaviors and
fixed every *correctness* finding from its 8-angle review. What it deliberately
deferred is the review's architectural residue: duplicated protocols, state
with two spellings, and one real multi-player hazard. None of it changes what
a reader sees today; all of it changes how safely the next feature lands.
This plan turns that residue into six phases, each independently committable,
each sized for one Opus subagent, each ending green.

Everything here lives in `pkg/emit/html/assets/runtime.{js,css}` unless a
phase says otherwise. The four copies of those files
(`editors/vscode/media/*`, `docs/demos/assets/*`) are generated — never edit
them; the sync commands in the gate below are what keep `assets_test` and
`site_test` green.

## Ground rules for every phase

Read `CLAUDE.md` first. Additional invariants for this plan:

- **`runtime.js` stays a classic ES5-style script**: `var`/`function`, no
  arrows, no `const`/`let`, no optional chaining, prose comments that say why.
- **`apply(t)` stays a pure function of t.** Nothing in these refactors may
  accumulate animation state across frames; a `record` frame at any
  millisecond must be pixel-identical to live playback paused there.
- **Refactors prove equivalence by measurement, not by reading.** The harness
  technique from PR #25's verification: `bazel run //cmd/cinegram -- preview
  examples/oidc-login.dgm -o /tmp/x/app.html`, an iframe harness html beside
  it, `python3 -m http.server`, drive headless Chrome with
  `--virtual-time-budget` and read state via `CINEGRAM_PLAYER` /
  `document.title` markers. rAF playback does not advance under virtual time —
  verify via `seek()` and direct state reads. Where a phase claims "behavior
  unchanged", capture the observable (client rects, class lists, zoom/pan
  values, listener effects) before and after and diff it.
- **Every phase gate**:

  ```sh
  bazel run //editors/vscode:sync_assets
  bazel run //site:sync
  bazel test //...        # 13/13, incl. assets_test, site_test, sitegen_test
  ```

  plus gofmt via the hermetic SDK if any Go changed, and `bazel run
  //:gazelle` if files were added to a Go package.

## Settled decisions — do not relitigate

- **Minimap stays a DOM clone, not an `<img>`.** Serializing the SVG to a
  data: URI would delete the whole id machinery, but mermaid draws flowchart
  labels in `foreignObject` (`htmlLabels: true`), and SVG loaded as an image
  does not render `foreignObject` — the thumbnail would lose every label.
  Rejected once in the minimap's original design review; recorded here so it
  is not retried.
- **The 900px/520px breakpoints stay as duplicated media queries.** The
  hazard was silent drift, and `pkg/sitegen`'s `TestShellHatchesAgree` now
  fails loudly on it. A JS-toggled `dgm-narrow` class (one `matchMedia`, both
  sheets keying off it) is the deeper fix, but it rewires selectors across
  two stylesheets for a risk the test already retired. Revisit only if a
  third consumer of the breakpoint appears.
- **A `scene` firing exactly at a step seam appears on the next beat's
  press**, not during the pause — a consequence of the presenter stop landing
  1ms before the seam, reviewed with the user and accepted.

## Phase index

| # | Phase | Theme |
|---|-------|-------|
| 1 | One window manager | Fullscreen ownership in one place; 21 CSS declarations become 7 |
| 2 | One drag protocol | Five hand-rolled gesture copies become one helper |
| 3 | One coordinate story | Camera and minimap share their geometry math |
| 4 | A self-contained thumbnail | The minimap clone stops borrowing another SVG's defs |
| 5 | One spelling per fact | `follow`/`following()` merge; board dismissal gets one owner |
| 6 | Transport edge | Steps of ≤1ms cannot stall the presenter |

Order matters for 1→2 (phase 2 touches listeners phase 1 deletes) and 3→4
(phase 4 builds on the helpers phase 3 extracts). 5 and 6 are independent.

---

## Phase 1 — One window manager

**Problem.** "Which screen does the player occupy" is currently answered in
five places: `setPresenter` (requests fullscreen, adds the fill class via an
async refusal callback), `requestFull` (a 150ms probe for legacy webkit),
`onFull` (drops the mode, with an overlay-open exception), `dispose`, and
three CSS rules whose bodies are identical
(`.dgm:fullscreen`, `.dgm:-webkit-full-screen`, `.dgm.dgm-present-fill`) —
kept as separate rules because an unknown pseudo-class invalidates a whole
selector list. Review finding: 21 declarations where 7 would do, and an
enter/exit/enter inside one refusal round-trip can leave the fill class on a
player that *did* get real fullscreen.

**Change.** Make `dgm-present-fill` unconditional: `setPresenter(true)` always
adds it, `setPresenter(false)` always removes it. On the element that *is*
the fullscreen element, `position: fixed; inset: 0` resolves against the
viewport and `z-index` is inert in the top layer, so the box is identical
whether the request was granted or refused. Then:

- The two pseudo-class rules delete; one plain class rule
  (`.dgm.dgm-present-fill`) carries the seven declarations. A plain class
  list cannot be invalidated by an unsupported pseudo-class.
- `requestFull` loses its `onFail` parameter and the 150ms probe — it becomes
  fire-and-forget, three lines. The async refusal guard in `setPresenter`
  goes with it.
- `_hadFull` moves from a Player field to a local above `onFull` in `build()`
  — it is a two-event latch read in exactly one closure, and as instance
  state it invites snapshot/restore code to wonder about it.
- Keep `onFull`'s two behaviors verbatim: our-fullscreen-lost → leave
  presenter mode, except when the lightbox/help overlay was open (Esc was for
  the overlay) → survive on the fill.
- Add the deferred one-line comment in `editors/vscode/media/preview.js`'s
  snapshot: present/follow are intentionally absent — the inline Markdown
  preview hides both buttons, so the states are unreachable there.

**Verify** (headless harness + one real browser):
- Enter present with fullscreen refused (headless default): fill class on,
  `position: fixed`, covers the iframe viewport. Exit: class off.
- Enter/exit/exit-again races: toggle present twice within 200ms; the fill
  class must match `this.present` at rest in every interleaving.
- In a *real* Chrome (manual, one minute): Present → real fullscreen; confirm
  the fill class being present changes nothing visually (screenshot diff of
  the bar/stage geometry vs. a build with the class suppressed); Esc with the
  lightbox open survives on fill; Esc again leaves.
- Playground recompile mid-present still restores (`present=true` after a
  keystroke), and no `webkitfullscreen*` listeners remain anywhere in the
  file (`grep -c` = 0 outside `fullscreenElement`/`exitFull`).

---

## Phase 2 — One drag protocol

**Problem.** The pointerdown/move/up/cancel + `setPointerCapture` +
capture-phase click-swallow dance is hand-rolled five times —
`bindStageGestures`, `bindMapGestures`, `bindBoardGestures`, and twice in
`buildLightbox` — with per-copy drift: only the two newest guard
`setPointerCapture` with try/catch, the flag sets disagree
(`moved` / `down,swallow` / `down,dragging,swallow`), and two independent
capture-phase click-swallowers sit on the same stage element, where
same-node registration order (now load-bearing, commented in `build()`)
decides which runs first.

**Change.** Extract one helper near the other free functions:

```js
// drag(el, opts) owns the bookkeeping every gesture repeats: primary-button
// guard, pointer capture (with the try/catch some browsers need), the
// down/move lifecycle, and cancel. opts: {start, move, end, cancel} — each
// optional, each called with (ev, state) where state carries dx/dy/moved.
```

- All five sites move onto it. Behavior-preserving: the stage's 4px slop, the
  board's 8px horizontal-intent test and the map's press-is-a-jump live in
  their `opts`, not in the helper.
- The click-swallowing collapses to **one** capture-phase listener on the
  stage, installed once in `build()`, reading a single `swallowNext` flag that
  stage, map and board gestures all arm. The registration-order comment
  deletes because the ordering stops existing.
- The board fling moves to the stylesheet idiom: classes `is-flinging` /
  `is-springing` with transitions declared in runtime.css, `.dgm-board` added
  to the `prefers-reduced-motion` block's selector list, and the JS
  `transitionend` + `setTimeout(300)` + `settled` dance replaced by listening
  for `transitionend` once with the CSS owning the duration. The
  reduced-motion JS branch deletes — the sheet's 0.01ms rule handles it —
  but the `setTimeout(settle, 0)` ordering fix (click must land on a visible
  board) must survive the rewrite; re-verify it explicitly.

**Verify.** Re-run PR #25's gesture measurements and diff against recorded
values: stage pan swallow, map drag released over stage padding (time and
stopAt unchanged, next genuine click advances), board swipe past/short of
half-width (flings / springs back), reduced-motion swipe (release lands on
the board, `stopAt` unchanged, hidden one tick later), tap opens lightbox,
lightbox pan unchanged. Then the count that proves the phase:
`grep -c setPointerCapture runtime.js` = 1.

---

## Phase 3 — One coordinate story

**Problem.** The stage↔holder mapping is written three times: `cameraKeys`
(`o`, `localRect`), `mapGeom` (byte-identical `o`, inlined `localRect` for
the svg), and `centreOn` in the map gestures (the pan-that-centres formula,
also `cameraApply`'s last two lines). A comment claims the math is "reused
rather than derived a second time" — it is derived a second time, and a
change to the framing convention (a stage border, a different
transform-origin) would desynchronize the camera from the map that claims to
describe it.

**Change.** Three small prototype methods, used by both consumers:

- `holderOrigin()` — the untransformed origin of the holder inside the stage
  (`holderR - pan - stageR`), used by `cameraKeys` and `mapGeom`.
- `localRect(el)` — an element's holder-local untransformed box, used by
  `cameraKeys.localRect` and `mapGeom`'s `d`.
- `centreOnLocal(cx, cy, z)` — writes the pan that centres a holder-local
  point; `cameraApply` and the map's `centreOn` both call it (`cameraApply`
  keeps calling `setTransform` itself — it must not re-enter `apply()`).

Fix the "reused rather than derived" comment to say what is now true.
Optional, only if free: write `mapRect` as one `transform:
translate()/scale()` instead of four box properties, so the compositor moves
it. Skip if it complicates the clamp math.

**Verify.** The map-click centring measurement from PR #25 must reproduce to
the pixel (pan `-300,-60 → 0,-481` on the recorded setup), camera poses across
all steps byte-identical before/after (serialize `{zoom,panX,panY}` per step
seek into the title and diff), minimap rectangle positions identical at three
zoom/pan combinations.

---

## Phase 4 — A self-contained thumbnail

**Problem.** The minimap clone strips its ids so `url(#…)` references resolve
document-wide to the live SVG's defs. With **one** player that is the trick
that makes it work; with **two** (a VS Code Markdown preview with two ```dgm
blocks — a supported, routine configuration), mermaid's *unprefixed* sequence
marker ids (`arrowhead`, `crosshead`, `sequencenumber`) resolve to whichever
SVG comes first in document order. The second player's thumbnail borrows the
first player's markers — wrong theme, or none at all once the preview's
morphdom diff reverts the first block to its placeholder.

**Change.** Rename instead of strip. In `ensureMapClone`:

- Give every id inside the clone a unique prefix (`dgm-map-<rand>-<oldid>`)
  instead of removing it.
- Rewrite every reference: `url(#…)` in `marker-start/mid/end`, `clip-path`,
  `mask`, `fill`, `stroke`, `filter`, and style attributes; `href`/
  `xlink:href` beginning with `#`; and the copied `<style>` text (the
  root-id retargeting already there generalizes to this).
- Keep the existing script/`on*` stripping — it is a security measure, not id
  hygiene — and the root-id/style retarget merges into the same pass.

**Verify.** A two-player harness page (`Cinegram.mount` twice with different
diagrams, one flowchart + one sequence): zoom both maps in, assert each
clone's `url(#…)` references resolve (`document.querySelectorAll` per target
id → exactly 1, inside that clone), then **remove player A's SVG from the
DOM** and assert player B's thumbnail still renders its arrowheads
(screenshot or marker-element presence). Single-player regression: the PR #25
minimap measurements reproduce (labels present, live diagram still animating,
zero ids *shared* with the live SVG).

---

## Phase 5 — One spelling per fact

**Problem A.** `this.follow` is false in a reel while the camera follows, so
the field does not mean what its name says; `following()` (= `reel || follow`)
papers over it at 11 call sites, and `setFollow` carries a dead
`else if (!this.reel)` branch (the Cine button does not exist in a reel's
hidden bar).

**Change A.** Set `this.follow = this.reel` where `reel` is first known in
`build()`; clamp in `setFollow` (`this.follow = this.reel || !!on`); delete
`following()` and use `this.follow` at all sites; the dead branch goes. The
playground's `snapshot()` keeps recording `p.follow` — in a reel that is now
`true`, which restores correctly because of the clamp.

**Problem B.** "The reader dismissed the thumbnail" is a DOM class
(`is-flung`) cleared from three unrelated places (`syncBoard`, `setPresenter`,
a media-query scope), while `board.style.display` decides visibility two
lines above one of the clears — and a rotate-to-landscape-and-back resurrects
then re-hides the panel with no gesture, because `resize` clears nothing.

**Change B.** A `this.boardDismissed` field, folded into `syncBoard`'s
existing `boardOn` computation so `syncBoard` is the *only* writer of the
panel's visibility. The swipe sets the field and calls `syncBoard()`; the
sites that today remove the class instead reset the field and call
`syncBoard()`; the resize handler resets it too (a layout change is a new
context); the CSS `is-flung` rule and its media-query scoping delete.

**Also.** Harden the swipe-arming probe: require `this.present` *and*
computed `position: absolute`, so a host stylesheet that floats the board for
its own reasons cannot arm flick-to-dismiss on a full-size panel.

**Verify.** Reel: camera still follows (zoom 2.5 at a step), `follow === true`,
Cine button absent. Normal/present: Cine toggles exactly as PR #25 measured.
Board: swipe-dismiss on phone-present; rotate wide (>900px) → panel returns
*and stays*; rotate back → thumbnail returns (dismissal did not survive the
layout change); scenario switch restores; desktop panel cannot be swiped.

---

## Phase 6 — Transport edge: ≤1ms steps

**Problem.** `advanceStep` stops at `end - (span > 2 ? 1 : 0)` and its scan
skips a step only when `start < time - 1`. For a step of span ≤1ms no
interior stop point exists: the press re-selects the same step forever. No
committed example emits one, but nothing in the compiler forbids it.

**Change.** Decide in `pkg/compile`, not the runtime: this is a timing rule,
and timing rules live in the compiler (CLAUDE.md). Smallest correct fix:
during lowering, clamp any step's duration to a floor of **4ms** (comfortably
above the transport's ±1ms tolerances; invisible at any playback speed), and
add a `lint` note when the clamp fires so an author who wrote `dur: 1ms`
learns why it played as 4. Golden fixtures: regenerate with
`bazel run //pkg/compile:compile_test -- -update` (never under `bazel test`)
and eyeball the diff — only synthetic zero/1ms fixtures should move; if any
real fixture changes, stop and reassess.

**Verify.** A synthetic scenario with a 1ms step: `advanceStep` walks through
it (time progresses past it on the second press at worst); compile test
covers the clamp; `lint` surfaces the note; every existing example's timeline
is byte-identical (`cinegram compile` output diff, empty).

---

## Definition of done for the plan

All six phases merged; `grep -c setPointerCapture runtime.js` = 1;
`grep -c "following()" runtime.js` = 0; a two-player page's thumbnails are
independent; the gate green after every phase; and no measurement recorded in
PR #25's verification changed except where a phase's "Verify" section says it
should.
