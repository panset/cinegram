# Confirmed workstreams (2026-08-15)

The decision record behind the next round of work, settled in review and
confirmed. Evidence and mechanics for each choice live in
[prior-art.md](prior-art.md); this file is the what-and-why, kept short.

## Scope

The site shell gets replaced; the compiler, the `.dgm` format, and the
runtime stay. No existing OSS framework is a drop-in for "browse a folder of
`.dgm` files on the web" — Markdown SSGs (Zensical, MkDocs, Hugo) would need
a plugin whose whole job is shelling out to the cinegram binary, plus a
toolchain in CI. Zensical is off the table unless a prose manual becomes a
goal; demos would embed into such a site, not be replaced by it.

## WS1 — `cinegram site <dir>`

A first-class CLI subcommand: walk a folder tree of `.dgm` files, emit a
browsable static site whose nav mirrors the folders, with **one shared copy**
of mermaid/runtime/CSS instead of a self-contained 2.8 MB per page (the
demo site drops from 33 MB to ~4 MB). `preview -o` stays self-contained; the
site still works over `file://` because the runtime is a classic script
loaded by relative path.

Output contract (settled 2026-08-15; no command previously existed — this
generalizes `site/site.go`'s Build, and the new primitive is an emit mode in
`pkg/emit/html` that links assets by relative path instead of inlining):

- **Mirrored tree**: one page per `.dgm` at the same relative path, an
  `index.html` per folder, one `assets/` at the output root, `.nojekyll`.
  Assets keep **plain names** — not fingerprinted; mdBook fingerprints for
  aggressive CDN caching, and GitHub Pages serves with max-age=600, so the
  rationale does not transfer.
- Ordering: folders give hierarchy; alphabetical within; an optional numeric
  prefix (`01-name.dgm`) forces order and is stripped from display names. No
  manifest file — mdBook's SUMMARY.md model silently drops unlisted files.
- **mdBook-style sidebar on every page**: the full site tree, collapsible
  folders, current page highlighted, embedded at build time. Breadcrumbs and
  global depth-first prev/next arrows stay alongside it.
- No prose override in v1 — folder indexes are listings, nothing more.
- **Playground presentation**: an "Edit in playground" button per demo page,
  its `#doc=` payload minted by the generator at build time (stdlib
  deflate + base64url of the playground's share JSON) so the reader lands in
  the playground with that exact document loaded — enabled by
  `--playground <url>`, omitted when unset; generic `--link NAME=URL` header
  links on every page; a hero card on the root index (copy via flag).
  Long `#doc=` URLs for image-heavy demos are accepted — ugly, never broken.
- Conventions carried over from the demos index: blurb from the leading `%%`
  block; `view`-referenced files get drill-down, not their own listing.
- **`-o DIR` XOR `--serve --watch`**, mirroring preview's contract.
- Repo's own site: `//site:sync` roots the generator at `docs/demos/` —
  live URLs preserved, the sweep contract untouched. The generated index
  becomes the landing page; top-level `docs/index.html` shrinks to a
  meta-refresh stub; `site/`'s hand-written landing template is deleted.
  `docs/` stays committed verbatim, `site_test` still gates staleness,
  `pages.yml` untouched.

## WS2 — paused on load, everywhere

All surfaces (standalone preview, site, VS Code, playground) load paused at
a resting frame; autoplay becomes explicit opt-in. `record`/`frame` are
unaffected (they drive the clock directly). Unanimous in prior art
(asciinema, Remotion both default autoplay off).

Options adopted into the spec from prior art:

- **Poster frame**: the page can rest at a chosen millisecond instead of
  0 ms — free, since seek is a lookup.
- **Step-through mode**: an asciinema `pauseOnMarkers`-style player option
  that pauses at each step boundary.
- Remotion's coupling: click-to-play only when player chrome is visible, so
  a chromeless embed never intercepts clicks meant for `interact` bindings.

## WS3 — draggable playground split

A real drag handle between editor and preview, hand-rolled onto the existing
`--pg-left` CSS variable (no layout framework — Compiler Explorer's frozen
GoldenLayout fork is the cautionary tale). Percent-based, persisted in
localStorage, double-click collapses the editor, the 720 px editor max-width
cap goes away, and below ~640 px the panes toggle rather than squeeze.

## WS4 — storyboard lightbox

One shared zoom layer in the runtime, used by every surface: click a
storyboard frame → expand to viewport fit; mouse-wheel zoom anchored at the
cursor; Esc/click-out closes. Hand-rolled using PhotoSwipe's shipped math
(fit = `min(1, viewport/image)`, wheel factor `2^(-deltaY·0.002)`,
cursor-anchored pan transform) — PhotoSwipe itself is ~70 kB for what is one
image class here.

- Wheel scoping rule: inside the modal both bare-wheel and ctrl+wheel zoom
  (no competing scroll exists there); bare-wheel zoom is never used outside
  a modal.
- Must reuse the already-mounted board `<img>` layer — `applyBoard` diffs on
  frame id so the `src` is never rewritten per frame; a fresh `<img>` would
  restart the crossfade.
- Sharpness caveat: zoom only reveals pixels the source has. SVG frames stay
  sharp at any size; the playground hints when an attached raster frame is
  notably small (~<1200 px wide) so softness is explained, not mysterious.

## WS5 — rolodex (object → scenes)

Click a diagram object, flip through the scenes/scenarios that involve it —
an inverted index over the timeline IR (which already records which node and
edge IDs each track targets; no format change needed). From Structurizr:
double-click drills straight through when unambiguous, otherwise a small
chooser lists destinations; routed through `location.hash` so Back works.
No fixed-interval auto-flip.

## WS6 — playground folder browsing (after WS1)

"Add folder…" (`webkitdirectory`) and folder drag-drop land a whole tree in
the playground's in-memory file set, relative paths preserved. The left pane
gains an **Editor | Files** toggle: Files shows the tree in numeric-prefix
order with collapsible folders; clicking a `.dgm` compiles and plays it,
toggling back to Editor edits it. Nothing uploads anywhere — it is the site
experience for a local folder, without a server.

## Go-native backend (deferred)

**Decision:** `github.com/d2lang/dagro` (MIT, zero deps, pure-Go dagre) is
the designated layout engine for the eventual mermaid-free SVG backend — see
the full case in [prior-art.md](prior-art.md). **The dependency is not taken
yet**; stdlib-only remains in force until backend work begins, and dagro's
maturity is re-evaluated then. Until that day, the IR's no-geometry rule is
what keeps this door open — protect it.

## Properties to protect while building any of this

- O(1) seek: absolute-millisecond IR makes every seek a lookup (Motion
  Canvas pays checkpoint-replay for the same operation). `frame`, `record`,
  the lightbox and the rolodex all lean on it.
- Deep links stay re-navigable through `location.hash` (stronger than
  Remotion's mount-time-fixed initial frame).
- A step manifest (id → start ms, duration) alongside `record` output is the
  cheap export that also feeds the rolodex — manim's section sidecar is the
  model.
