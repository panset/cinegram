# Theme out of the rail, and the rail down to three

## Context

Dark/light is a property of the *page*, and the runtime already treats it as
one — `buildRail`'s theme button writes
`document.documentElement.setAttribute('data-theme', …)` and persists to a
global `dgm.theme` key. What sits in the wrong place is the **control**: a
page-scoped switch living inside one diagram's tool rail. That misplacement
produces three real defects, not just an aesthetic one:

1. **The VS Code panel preview overrides the editor.** `panel.js` passes
   `theme: themeKind()`, making the editor the authority — but the rail still
   builds the button, and pressing it writes `dgm.theme` and wins from then on.
   A light diagram in a dark editor, with no way back. (The Markdown-preview
   path escapes only by accident: `.dgm-inline .dgm-rail` is `display: none`.)
2. **There is no way back to "follow the system."** `themePref` is only ever
   set, never cleared, and `follow()` returns early once it is non-null. One
   press and that browser stops tracking the OS for every cinegram page, for
   good.
3. **Two players on one page fight over one attribute.**

Fixing the placement is also the moment to answer the second question the
review raised: the rail carries seven tools, and three of them are either a
page property, a duplicate of a documented gesture, or a preference that does
not deserve prime real estate. This plan is three phases, each independently
committable, each sized for one Opus subagent, each ending green.

## Ground rules for every phase

Read `CLAUDE.md` and the `build` skill first. Then:

- **`runtime.js` stays a classic ES5-style script**: `var`/`function`, no
  arrows, no `const`/`let`, no optional chaining, prose comments that say why.
  It is loaded as a classic script from `file://` and under a webview CSP.
- **The emitted page carries no external URL.** `pkg/emit/html/html_test.go`
  enforces it; a theme icon is inline SVG or nothing.
- **`apply(t)` stays a pure function of t.** Nothing here may accumulate state
  across frames.
- **`pkg/emit/html/assets/` is the only canonical copy.** `editors/vscode/media/*`
  is a byte-synced copy — never hand-edit it; run the sync.
- **Every phase gate**:

  ```sh
  bazel run //editors/vscode:sync_assets
  bazel run //site:sync
  bazel test //...
  ```

  plus gofmt from the hermetic SDK if any Go changed, and `bazel run //:gazelle`
  if a file was added to a Go package.

- **Verify in a browser, not by reading.** Per the `rendering-pipeline` skill:

  ```sh
  bazel run //cmd/cinegram -- preview examples/01-basics/01-k8s-request.dgm --serve
  ```

  Drive `window.CINEGRAM_PLAYER`; `seek(ms)` is deterministic where playback is
  not. Wait ~250ms before reading a computed style — a CSS transition returns
  the starting value otherwise.

## Settled decisions — do not relitigate

- **The theme control is three-state: System / Light / Dark**, not two. The
  current model already *has* a following-the-system state that a two-state
  button cannot express or return to (defect 2). `system` is the default.

  **Revised 2026-08-18 — the control is two-state, Light and Dark.** The user
  chose this after seeing phase 1 built. A page still *opens* following the
  system, because that is still the absence of `data-theme` and still costs no
  script; what goes is the ability to get back to following it once a side has
  been picked. `dgm.theme` therefore only ever holds `light` or `dark`, a press
  flips the *effective* theme (the stored side, or `systemDark()` while there is
  none), and the glyph gets a `prefers-color-scheme` listener of its own so it
  stays truthful during the following state, which fires no attribute mutation
  for the player's observer to see. Everything else below stands as written.
- **"System" is the attribute being absent, not a value.** runtime.css already
  resolves an unstamped page through `prefers-color-scheme` — that is what
  `:root:not([data-theme='light'])` and the `@media (prefers-color-scheme: light)`
  block are for. So `system` **removes** `data-theme` and lets the sheet decide.
  No `matchMedia` listener, no flash, less JS than today.
- **`<html data-theme>` is the single source of truth for every host.** The
  player reads it at mount and observes it; page chrome only ever writes it.
  This is what lets the same runtime serve a standalone page, a webview whose
  editor owns the theme, and a MkDocs page whose Material palette toggle owns
  it, with one code path.
- **The storage key stays `dgm.theme`**, so a reader who already chose dark
  keeps dark. It gains a third legal value, `system`.
- **VS Code gets no control at all.** The editor is the authority. This needs
  no suppression logic: neither webview emits page chrome, so with the rail
  button gone there is simply nothing to press. Defect 1 dies by construction.
- **Losing the word costs something, and it is paid for.** The current button
  is deliberately labelled "Light"/"Dark" because the word *is* the state
  (comment above `this.themeBtn`). A 24px glyph cannot carry that, so the
  replacement draws the **current** state and names the **action** in
  `aria-label`/`title` ("Theme: system — click for light"). Three states make
  a glyph honest in a way two never did.

## Phase index

| # | Phase | Result |
|---|-------|--------|
| 1 | Theme becomes page chrome | Rail 7 → 6; VS Code defect fixed; FOUC fixed; System state restored |
| 2 | Fit folds into the minimap; Restart earns its place | Rail 6 → 4 at rest |
| 3 | Speed moves to a settings sheet; the ⋯ collapse deletes | Rail 4 → 3; a state machine removed |

Strictly sequential — all three edit `buildRail` and the rail's CSS block, and
phase 3's deletion of the collapse is only safe once phases 1 and 2 have got
the button count down. Do not run them in parallel.

---

## Phase 1 — Theme becomes page chrome

**Problem.** Above. A page-scoped control in a diagram-scoped widget.

**The model to build.**

`<html data-theme>` is the truth. Three states, one storage key:

| `dgm.theme` | `data-theme` | who decides |
| --- | --- | --- |
| absent or `system` | *attribute removed* | runtime.css's `prefers-color-scheme` |
| `light` | `light` | the reader |
| `dark` | `dark` | the reader |

**Change — `pkg/emit/html/assets/runtime.js`.**

- Delete `this.themeBtn` and its `items.appendChild` from `buildRail`, and the
  comment block above it.
- Rework the theme fields in the `Player` constructor. A new
  `this.pageTheme = document.documentElement.getAttribute('data-theme')`
  distinguishes the two worlds: **non-null means the page owns its theme** (a
  cinegram-emitted page, whose chrome stamped it pre-paint), **null means the
  player is a guest** (a webview, a MkDocs page, someone else's document) and
  `opts.theme` or the system decides as it does today.
- The player stops reading and stops writing `dgm.theme`. That key now belongs
  to the page chrome. `prefGet`/`prefSet` stay — phase 3's speed setting still
  uses them.
- Resolve the initial theme in this order: `this.pageTheme` → `this.hostTheme`
  (`opts.theme`) → `systemDark() ? 'dark' : 'light'`. Mermaid needs a concrete
  `light`/`dark`, so an absent attribute still resolves through `systemDark()`.
- The player keeps mirroring its resolved theme onto `<html data-theme>` **only
  when `this.pageTheme` was null at mount.** That is what `pkg/embedkit` relies
  on today (see the comment at `cinegram-embed.js` about the runtime stamping
  the attribute) and it must not regress. On a page that already stamped, the
  player never writes.
- **Observe the attribute.** Add a `MutationObserver` on `documentElement`
  filtered to `data-theme`, calling `this.setTheme(resolved)` — where a removed
  attribute resolves through `systemDark()`. Register its `disconnect()` on the
  `_unbind` list `own()` maintains, so `dispose()` cleans it up; the playground
  disposes and remounts a player on every keystroke and must not leak
  observers. `setTheme`'s existing `if (this.theme === kind) return;` guard is
  what keeps the player's own mirror-write from re-entering.
- Keep the existing `prefers-color-scheme` follow block **only for the guest
  case** (`!this.pageTheme && !this.hostTheme`) — a bare embed still tracks the
  OS. On a page-owned page the stylesheet does it with no JS. Drop the
  `self.themeBtn.textContent` line from `follow()` and from `setTheme`; the
  button is gone.
- `setTheme` keeps its contract for hosts (`embedkit`, `preview.js`, `panel.js`
  all call it) and must keep re-rendering — mermaid picks its theme per render.

**Change — one implementation of the control, three placements.**

Put the control itself in `runtime.js` so it is written once and carried to
every surface by the existing sync/asset machinery, and expose it on the
`Cinegram` namespace beside `mount`:

- `Cinegram.themeToggle()` → returns a ready `<button class="dgm-page-theme">`
  with the three-state cycle wired (system → light → dark → system), writing
  `dgm.theme` and setting/removing `data-theme`. Every player on the page picks
  the change up through its observer; the function needs no player handle,
  which is exactly why the playground can remount players under it freely.
- Three inline SVG glyphs on the same 24-unit grid and the same
  `stroke="currentColor"` convention as `ICONS`: sun, moon, and a
  half-filled circle for system. Reuse `icon()`/`iconButton()`.
- `aria-label` and `title` both read "Theme: <state> — click for <next>", and
  the button carries `aria-live="polite"` so the state change is announced.

**Change — the pre-paint boot script.**

`data-theme` has to land before first paint or a standalone page opened from
disk flashes. `runtime.js` loads at the end of `<body>`, so this cannot come
from it — it is three lines of inline `<head>` script. Emit it from **one** Go
helper so the three surfaces cannot drift:

- New exported `html.ThemeBootScript() string` in `pkg/emit/html/html.go`,
  returning the `<script>` element. It reads `localStorage['dgm.theme']` inside
  a `try` (Safari private mode throws) and sets or removes the attribute.
- New exported `html.ThemeToggleHTML() string`, returning the button's markup
  placeholder — a `<button class="dgm-page-theme" data-dgm-theme-toggle>` that
  `runtime.js` upgrades on load if it finds one, so the control exists in the
  HTML (and is styled) before any JS runs.

**Change — the three placements.**

| Surface | Where |
| --- | --- |
| `pkg/emit/html/html.go` (standalone) | boot script into `<head>` after the stylesheet; toggle emitted right after `<body>` **only when `opts.Nav` is empty** — that is already the "no site chrome" branch. Fixed top-right via runtime.css. |
| `pkg/sitegen/chrome.go` | boot script into `head`; toggle appended inside `<div class="dgm-site-actions">` in `renderPage`, **and** in `renderIndex`'s own `dgm-site-top` — folder listings need it too. `.dgm-site-top` is already `flex` with `justify-content: space-between`. |
| `web/playground/index.html` | boot script inline in `<head>`; toggle inside `<div class="pg-actions">`, after the GitHub link. Hand-written, like the `data-dgm-skin` attribute beside it — hence the test below. |

VS Code and MkDocs get nothing, deliberately.

**Change — `pkg/emit/html/assets/runtime.css`.** Style `.dgm-page-theme` from
the `--dgm-*` tokens like every other button. The fixed top-right placement for
the standalone case is scoped `.dgm-standalone > .dgm-page-theme` — do **not**
add a bare `body` rule; `TestPageClaimsTheDocument` fails on one, and the whole
point of that scoping is that a guest page keeps its own layout. `z-index` must
clear the rail and the minimap but sit below the lightbox and help overlay.

**Tests to add.**

In `pkg/emit/html/html_test.go`:
- `TestThePageCarriesItsOwnThemeControl` — a plain `Render` emits exactly one
  `data-dgm-theme-toggle`, and it is outside the `#cinegram` element.
- `TestTheThemeBootScriptBeatsTheFirstPaint` — the boot script appears in the
  page **before** `#cinegram` and before any `runtime.js` reference. Pin the
  ordering by index comparison, not by presence: the whole value is the order.
- `TestASitedPageDoesNotDoubleTheControl` — `Render` with `Nav` set emits zero
  toggles of its own (the site header owns it), so a site page never shows two.

In `pkg/sitegen/discover_test.go` (or a sibling): every generated page and every
folder index carries exactly one toggle and one boot script.

In `site/palette_test.go`'s idiom — it already reads
`web/playground/index.html` from disk to pin a hand-written attribute against a
Go constant, so follow it exactly:
- `TestThePlaygroundCarriesTheThemeControl` — the page contains
  `html.ThemeToggleHTML()`'s marker and the same `dgm.theme` key that
  `html.ThemeBootScript()` uses. A hand-written page that drifts from the
  emitted one is a workbench whose toggle silently stops working.

A contract test that the rail no longer owns the theme — the mechanical proof
of this phase, in the style of the existing "read the asset and assert" tests:
- `TestTheRailDoesNotOwnTheTheme` — `runtime.js` contains no `themeBtn`, and
  contains no `prefSet('dgm.theme'` / `prefGet('dgm.theme'`.

**Verify in a browser** (all four, they exercise different code paths):
1. **Standalone**, served. Cycle the toggle through all three states: `data-theme`
   goes `light` → `dark` → *absent*, `localStorage['dgm.theme']` tracks it, the
   diagram's mermaid colours change (read a node fill 250ms after each click),
   and reload preserves the state. With `system` selected, flip the OS/emulated
   `prefers-color-scheme` and confirm the page follows with no JS involvement.
2. **No flash.** Set `dgm.theme=dark`, hard-reload, and confirm the first
   painted frame is dark — screenshot at the earliest paint, or assert
   `getComputedStyle(document.documentElement).getPropertyValue('--dgm-bg')`
   from a `<script>` placed immediately after the boot script.
3. **Playground.** Toggle, then type in the editor to force a player remount:
   the new player must come up in the chosen theme, and the workbench chrome
   must repaint with it (`playground.css` draws from `--dgm-*`). Then confirm
   no observer leak — remount 20 times and check the player count and that
   toggling still costs one re-render, not twenty.
4. **VS Code, both webviews** (`code --extensionDevelopmentPath=editors/vscode`).
   The panel preview and a Markdown block must show **no** theme control, and
   both must follow `Preferences: Toggle between Light/Dark Themes` live. This
   is the defect the phase exists to fix — demonstrate it explicitly.
5. **MkDocs / embedkit unchanged.** Build the site and confirm a diagram still
   follows Material's palette toggle. `pkg/embedkit` should need no edit; if it
   does, that is a signal the mirror-write rule above was got wrong.

---

## Phase 2 — Fit folds into the minimap; Restart earns its place

**Problem.** Two of the six remaining rail buttons are dead most of the time.

`Fit` (`this.zoomBtn` → `resetZoom`) does nothing until something has zoomed,
duplicates the stage's double-click, and sits directly beside a minimap that
already knows the exact answer: `syncMap` — *"At fit the whole diagram is on
the stage, so there is nothing to orient and the map is off."* Fit and the
minimap are the same signal, drawn twice.

`Restart` duplicates `Home`, duplicates dragging the scrub to zero, and
duplicates clicking step 1. Its justification in the source is presenter mode —
and that is correct, but *only* there: presenter and reel both hide
`.dgm-foot`, so those are the only modes with no scrub to drag.

**Change.**

- Delete `this.zoomBtn` from `buildRail`. Add a small fit control **inside the
  minimap**, appearing and disappearing with it: the map is already built in
  `build()` as `.dgm-map is-off` with `.dgm-map-body` and `.dgm-map-rect`, so
  add a `.dgm-map-fit` button beside them. It inherits the map's visibility for
  free, which is the whole point — the control exists exactly when it can act.
  The map already `stopPropagation`s its own `click`/`dblclick`, so the button
  will not also advance a presenter step; confirm that, do not assume it.
- The map carries `aria-hidden="true"` today. A focusable button inside an
  `aria-hidden` subtree is an accessibility error. Move `aria-hidden` off
  `.dgm-map` and onto `.dgm-map-body` and `.dgm-map-rect` — the decorative
  parts — leaving the button reachable, and give the map an `aria-label`.
- Also bind **double-click on the map** to `resetZoom`, matching the stage's
  established idiom. Add a line to `SHORTCUTS` so the help sheet says so;
  `SHORTCUTS` is deliberately both the handler's documentation and the
  overlay's content, so it must not fall behind.
- Restart stays in the rail but gains a class — `dgm-nofoot`, say — that is
  `display: none` by default and shown only under `.dgm-present` and
  `.dgm-reel`. Watch the specificity: `.dgm-present .dgm-authoring { display: none }`
  is (0,2,0) and the existing
  `.dgm-inline.dgm-present .dgm-controls > .dgm-authoring` comment in
  runtime.css explains the arithmetic the file already depends on. Restart is
  *not* `dgm-authoring`, so it should not collide — verify rather than reason.

**Tests to add.**
- `TestTheMapCarriesItsOwnFitControl` and `TestRestartIsForTheModesWithNoScrub`
  — assert against the runtime assets in the established read-the-asset style:
  `dgm-map-fit` exists in both `runtime.js` and `runtime.css`, and `dgm-nofoot`
  is styled under both `.dgm-present` and `.dgm-reel`.
- `TestTheRailHasNoFitButton` — `zoomBtn` no longer appears in `runtime.js`.

**Verify in a browser.**
- At rest: no minimap, no fit control, no Restart. Zoom via wheel: the minimap
  appears *and* the fit control with it. Click it → `zoom === 1`, pan zeroed,
  minimap gone. Double-click the map → same, and the click must not have been
  read as a pan-jump.
- Presenter mode: Restart visible, and pressing it does not exit presenter.
  Reel: same. Default view and inline: Restart absent.
- Tab order: the map's fit button is reachable, and no screen reader hits an
  `aria-hidden` focusable — check with Chrome's accessibility tree, not by eye.
- The map's press-is-a-jump and drag-to-pan gestures are unchanged; releasing a
  map drag over the stage's padding still must not advance a beat (that
  ordering is load-bearing and commented in `build()`).

---

## Phase 3 — Speed moves to a settings sheet; the ⋯ collapse deletes

**Problem.** `Speed` is a persisted global preference (`dgm.speed`) that is
also `dgm-authoring`, so presenters lose it anyway. On a player where the
author timed every beat in the `.dgm`, a reader overriding pacing across every
cinegram they ever open is not worth prime real estate. It is also the most
expensive button in the file: runtime.css sizes the **entire 56px column**
around fitting the string `"0.25x"`, with a paragraph of comment explaining the
arithmetic. And cycling one-way through five presets means four clicks to get
from 2x back to 0.25x.

Meanwhile the help overlay is a read-only sheet, and there is nowhere for a
preference to live.

**Change.**

- Promote the help overlay to a settings-and-shortcuts sheet. Keep `?`, keep
  the rail's Help button, keep the `role="dialog"` and the Esc/backdrop close.
  Add a "Playback" section above the existing shortcut list holding speed as a
  `<select>` (0.25 / 0.5 / 1 / 1.5 / 2), labelled, writing `dgm.speed` exactly
  as `cycleSpeed` does today.
- Delete `this.speedBtn` and `cycleSpeed` from `buildRail`; `syncSpeed` now
  writes the select's value and its `aria-label` moves with it. Anything else
  calling `cycleSpeed` must be found and moved, not left dangling.
- **`onKey` already refuses to steal keys from a `select`** (`/input|select|textarea/i`
  on `ev.target`), so the new control is safe inside the sheet — but the sheet
  is open over a player whose Space/arrows are live, so re-check that focus
  actually lands in the select and that Esc still closes the sheet from it.
- The rail is now Cine, Share, Help, plus Restart in presenter/reel — three
  buttons at rest. Three stack fine under 520px, so **delete the collapse
  entirely**: `this.railMore`, `setRailOpen`, `watchRailCollapse`, the
  `items` click-walker that closes it, the `is-open` class, the
  `.dgm-rail-more` CSS and the 520px rail block. That removes a state machine,
  a `matchMedia` watcher and its legacy `addListener` fallback.
- Shrink the rail from 56px to 42px and **rewrite the comment above
  `.dgm-rail`**: its entire argument is the width needed to fit `"0.25x"`, and
  that reason is gone. A comment that survives the thing it explains is worse
  than no comment.
- Give Cine `aria-pressed` while you are in `buildRail` — it toggles an
  `is-on` class and nothing tells a screen reader the camera is following.

**Tests to add.**
- `TestTheRailCollapseIsGone` — `runtime.js` and `runtime.css` contain no
  `rail-more`, no `setRailOpen`, no `watchRailCollapse`. This is the phase's
  mechanical proof.
- `TestSpeedLivesInTheSettingsSheet` — `speedBtn` is absent from `runtime.js`
  and the sheet markup carries a speed control bound to `dgm.speed`.
- `TestCineAnnouncesItsState` — `aria-pressed` is set on the cine button.

**Verify in a browser.**
- Open the sheet with `?` and with the Help button. Change speed, close, play:
  the rate actually changed (time two `seek`-free playback windows, or read
  `CINEGRAM_PLAYER.speed`). Reload: the choice persisted.
- With the sheet open, Space and the arrow keys must not move the player while
  focus is in the select; Esc closes the sheet and the player's keys come back.
- At 400px wide: three rail buttons, no ⋯, nothing clipped, the rail does not
  cover the diagram. At 1400px: unchanged from phase 2.
- Presenter mode at both widths: Restart, Cine, Help — no speed, no ⋯.
- A scenario declaring a non-preset speed (0.8) still shows something sensible
  in the select rather than snapping — the reason `cycleSpeed` picked by value
  rather than by index. Preserve that property or state in the comment why it
  no longer applies.

---

## Where this lands

| | Rail today | After phase 3 |
| --- | --- | --- |
| Default | Restart, Speed, Cine, Share, Fit, Theme, Help (7) | **Cine, Share, Help (3)** |
| Presenting | Restart, Cine, Fit, Theme, Help (5) | Restart, Cine, Help (3) |
| Page chrome | — | Theme (System / Light / Dark), top right |
| Minimap | orientation only | orientation + fit, appearing together |
| Help overlay | read-only shortcut list | shortcuts + playback speed |
