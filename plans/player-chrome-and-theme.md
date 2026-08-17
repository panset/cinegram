# Player chrome and theme — the mainframe look everywhere, the clutter out of the bar

*2026-08-17. Prompted by the Zensical site landing (PR #27/#28): the landing
page's embedded players show two controls and look right; the day-to-day
players show ten and look like a different product.*

## What was observed

Two asks, one theme between them — the site now has an identity
(`www/assets/stylesheets/mainframe.css`: greenbar paper light, 3279 phosphor
dark, one monospace family) and the player everywhere else still wears the
neutral blue-accent look:

1. **Consistency.** Can the mainframe look extend to every surface — the
   standalone preview, `cinegram site` output, the playground — so the product
   reads as one thing?
2. **Clutter.** The full bar carries ten controls (scenario picker, Play,
   Restart, Speed, Copy link, Present, Cine, zoom-home, theme, help). The
   landing page's `inline` players carry three (Play, picker, Present) and are
   better for it. Day-to-day surfaces need the other controls but not in the
   reader's face: keep Play and Present in the bar, move the rest to a
   translucent icon rail on the right of the stage.

One naming correction that shapes everything below: the look is **not
Zensical's** — Zensical ships a stock Material palette. The look is
*cinegram's own theme*, authored in `mainframe.css`. So the job is not "adopt
Zensical everywhere"; it is "make the mainframe theme the player's native
skin", which every surface then inherits.

## Where the hooks already are

The groundwork exists; both asks land on seams the codebase already cut:

- `runtime.css` opens with a `--dgm-*` **token layer** (bg, panel, border, fg,
  muted, accent, …) with light/dark resolved via `data-theme` on `<html>` plus
  `prefers-color-scheme`. The playground's own workbench CSS and `pkg/sitegen`'s
  `site.css` both key off the same tokens, by design — "the site chrome
  repaints with the player's own theme toggle". A reskin is therefore a token
  override, not a selector hunt. The one gap: the font is hardcoded in two
  rules (`body.dgm-standalone`, `.dgm`) rather than a token.
- The bar already classifies its controls. `dgm-play` names the one control no
  mode may drop; `dgm-authoring` names the set presenter mode strips;
  `.dgm-inline` strips the bar to Play + picker **by name, not position** —
  the comment at `runtime.js` `build()` says adding a button cannot quietly
  change what a document shows. Moving buttons to a new container preserves
  all of that as long as the classes move with them.

Who mounts what today:

| Surface | Mount | Bar today | After this plan |
| --- | --- | --- | --- |
| www landing / example pages (`pkg/embedkit`) | `inline: true` | Play, picker (+Present re-shown) | unchanged — this is the look being copied |
| `cinegram preview` standalone page | full | all ten | Play/Present bar + right rail, mainframe skin |
| `cinegram site` (`pkg/sitegen`) | full | all ten | same |
| Playground | full | all ten | same |
| VS Code Markdown preview | `inline: true` | Play, picker | unchanged (guest in the editor's document) |
| VS Code animation editor (`media/panel.js`) | full | all ten | rail yes; skin **no** (see decision) |

## Part 1 — the right rail (declutter)

### Shape

- **Bar keeps:** Back, title, crumb; scenario picker (still hidden when there
  is only one scenario); **Play**; **Present**. That is the landing page's
  silhouette plus Present, which the user asked to keep prominent.
- **Rail gets:** Restart, Speed, Cine, Copy link, zoom-home (⌂), theme, help.
  A new `.dgm-rail` element, built in `build()` beside the stage, positioned
  absolute against the stage's right edge and **vertically centered** —
  centered, not top-anchored, because the minimap parks top-right (its
  overlap with the top-right corner is already an accepted collision in
  runtime.css; the rail should not join that pile-up). In presenter mode the
  storyboard thumbnail parks top-*left*, so no conflict there.
- **Look:** translucent — panel token at low alpha plus `backdrop-filter:
  blur()`, going opaque on hover/focus-within; a solid-panel fallback where
  `backdrop-filter` is unsupported. Icon buttons: inline SVG drawn in
  `currentColor` (the page must stay self-contained — no icon font, no
  external URL; `html_test.go` enforces this). Every button keeps
  `title` + `aria-label`; the rail is `role="toolbar"`. The Speed button's
  "1×/1.5×/2×" text *is* its icon and stays text.
- **Narrow screens** (the existing 520px breakpoint): the rail collapses to a
  single "⋯" toggle that expands the same rail, so a phone diagram is not
  overlaid by seven targets.

### Contracts that must not move

- Each relocated button keeps its classes (`dgm-authoring` on Speed and Copy
  link; none on Restart/Cine/theme/help/zoom). The presenter rule
  `.dgm-present .dgm-authoring { display: none }` then keeps stripping
  exactly what it strips today — the rail thins in present mode instead of
  vanishing, preserving "Restart and Cine survive present" semantics.
- `.dgm-inline` hides the rail wholesale (`.dgm-inline .dgm-rail { display:
  none }`). Embeds, the landing page, and the VS Code Markdown preview are
  untouched; `enablePresenter` in the embed kit keeps working because
  `presentBtn` stays in the bar.
- Keyboard shortcuts and the help overlay are unchanged; the rail is a second
  way to reach what keys already reach.
- Copy-link feedback ("Copied" / "Press ⌘C") becomes a transient label
  anchored to the rail button rather than a textContent swap that would
  stretch an icon button.
- z-order: rail above `.dgm-overlay` and the stage, below lightbox and help.
- `runtime.js` stays a classic ES5 script; `apply(t)` remains a pure function
  of t — the rail is chrome, and touches neither.

### Who gets it

Everything mounted without `inline: true`: the standalone page, `cinegram
site` pages, the playground, and the VS Code **animation editor**
(`media/panel.js` mounts full-chrome — the rail reaches it via
`sync_assets`, no extension change).

## Part 2 — the mainframe skin (consistency)

### Mechanism: an opt-in skin on the token layer

Add to `runtime.css`:

- a `--dgm-font` token, consumed by the two rules that hardcode the stack
  today (default value: the current sans stack — no visual change unskinned);
- a skin block, `[data-dgm-skin="mainframe"]`, overriding the `--dgm-*`
  tokens with the mainframe palette in both schemes (greenbar values under
  the light resolution, phosphor under dark — same
  `data-theme`/`prefers-color-scheme` machinery as the base tokens) and
  setting `--dgm-font` to the mainframe monospace stack.

**Opt-in, not default**, set on `<html>` by each surface we own:

- `pkg/emit/html` emits `data-dgm-skin="mainframe"` on the standalone page →
  `cinegram preview` and every `cinegram site` page (sitegen reuses the
  emitter and its index pages set the attribute in `chrome.go`).
- `web/playground/index.html` sets it; the workbench repaints for free since
  `playground.css` already keys off the tokens.
- The www site: `cinegram-embed.js` sets the attribute (or `Cinegram.mount`
  grows a `skin` option the loader passes) so the players *inside* the
  mainframe pages stop wearing the neutral look — that mismatch is the
  original observation. The two-button chrome there is unaffected; this is
  colors and type only.
- **VS Code sets nothing and keeps the neutral look.** The preview is a guest
  in the editor's document (the embed-kit header calls this the design), and
  a branded monospace player inside someone's Markdown reads as a broken
  font. The attribute makes flipping this a one-line decision later.

Why opt-in wins over making mainframe the default token values: the assets
are byte-synced into `editors/vscode/media/` (`assets_test`), so a default
would restyle VS Code with no way out short of an un-skin attribute, which is
the same mechanism pointed the ugly way.

### The duplication and its guard

The palette then exists twice: `--cg-*` literals in `www/assets/stylesheets/
mainframe.css` (feeding Material variables) and the skin block in
`runtime.css` (feeding `--dgm-*`). `go:embed` cannot reach across packages
and `www/` is not a Go package, so a shared source needs generation — not
worth it for ~14 hex values. Instead, the repo's own pattern for honest
copies: a test (beside `site/config_test.go`) that parses both files and
asserts the named hexes agree, failing with the pair that drifted.

### Deliberately out of scope, recorded so it is not half-done by accident

The skin recolors the **chrome** — bar, rail, captions, step list, panels.
The **diagram itself** stays mermaid's light/dark render. Restyling nodes and
edges to phosphor/greenbar means a `themeVariables` set per skin in the
mermaid init, and the 3279's semantic colors (green = actionable, turquoise =
read, yellow = literal) do not map onto mermaid's class model without design
work; contrast on greenbar paper is its own problem. Stretch phase, only if
the chrome-only result looks half-committed in practice.

## Phases

Each independently committable; every phase ends with the standard gate:

```sh
bazel run //editors/vscode:sync_assets
bazel run //site:sync
bazel test //...
```

plus gofmt via the hermetic SDK when Go changed, and browser verification via
`bazel run //cmd/cinegram -- preview … --serve` driving `CINEGRAM_PLAYER`
(seek, class-list assertions; mind the ~250ms transition trap).

1. **The rail.** `runtime.js` `build()` + `runtime.css`. Verify per surface:
   standalone full chrome; present mode strips `dgm-authoring` from the rail
   and keeps Restart/Cine; `inline` players byte-identical in behavior (bar
   rules unchanged); playground and animation editor pick it up via sync;
   narrow-viewport collapse; minimap overlap at high zoom.
2. **Skin tokens.** `--dgm-font` + skin block in `runtime.css`; attribute
   emitted by `pkg/emit/html`, `pkg/sitegen`, playground. Palette-agreement
   test. Verify light/dark toggle inside the skin, and that an unskinned
   mount (VS Code copies) is pixel-unchanged.
3. **The www site's players.** Embed-kit opt-in; `bazel run //site:sync`;
   check the example pages under the Zensical build (pages.yml already builds
   the site on deploy). Confirm the host-box CSS (`cinegram-embed.css`) still
   harmonizes — it colors from Material variables, which mainframe.css
   already retunes.
4. **Stretch, separately decided:** mermaid `themeVariables` per skin;
   auto-hiding the rail after inactivity in presenter mode;
   `prefers-reduced-transparency` handling beyond the solid fallback.

Phases 1 and 2 are independent and could land in either order; 3 needs 2.

## Decisions taken here (flip by saying so, not by relitigating silently)

- "Zensical theme" = the mainframe theme; the skin is cinegram's, not the
  site generator's.
- Skin is opt-in per surface; VS Code (both webviews) stays neutral.
- The landing page and all `inline` embeds keep the two-button chrome
  exactly as shipped.
- Present stays in the top bar next to Play (the user's stated pair), not in
  the rail.
- Rail is always visible but translucent — not hover-to-reveal — because an
  invisible control is a control the reader does not know exists; auto-hide
  is considered only for presenter mode, as stretch.
