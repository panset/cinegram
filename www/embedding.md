# Embedding cinegrams in your own site

`cinegram preview` gives you a page that *is* a diagram. `cinegram site` gives
you a site of them. Neither helps when the diagram belongs in the middle of
something you already wrote — an architecture doc, an RFC, an onboarding page —
where the diagram is a paragraph, not the point.

The **embed kit** is for that. It is five files and a `<div>`:

```html
<div class="cinegram" data-cinegram="storage-failover" data-height="900"></div>
```

Everything on this site works this way, including the diagram on the
[home page](index.md). If embedding breaks, these pages break first.

## What you install

```sh
cinegram assets -o docs/assets/cinegram
```

That writes five files into one folder:

| File | What it is |
| --- | --- |
| `cinegram-embed.js` | the loader — finds the divs, fetches what they need, mounts a player |
| `cinegram-embed.css` | the box a player mounts into, and its presenter/fullscreen layout |
| `runtime.js`, `runtime.css` | the player itself |
| `mermaid.min.js` | Mermaid, vendored — 2.6 MB, and the reason for everything below |

They belong in **one folder**, under the names given: the loader finds its
siblings and the `timelines/` folder relative to its own URL, not relative to
the site root. That is also what makes the kit survive a path prefix — a
GitHub Pages project site, or an enterprise install serving from
`/pages/org/repo/`.

Re-run the command after upgrading cinegram. It rewrites only what changed, so
it is safe in a build script.

## What you compile

A page plays a **timeline**, not a `.dgm`. Compile each source into
`timelines/` beside the loader:

```sh
cinegram compile diagrams/storage-failover.dgm \
  -o docs/assets/cinegram/timelines/storage-failover.json
```

`data-cinegram` is that path under `timelines/`, without the `.json`.
Subfolders work: `data-cinegram="architecture/storage-failover"` reads
`timelines/architecture/storage-failover.json`.

Lint before you compile — `cinegram lint` fails on a broken diagram at build
time, which is better than a page that renders an error at read time.

## Wiring it up

On a Zensical or Material for MkDocs site, two lines:

```toml
[project]
extra_css = ["assets/cinegram/cinegram-embed.css"]
extra_javascript = ["assets/cinegram/cinegram-embed.js"]
```

Site-wide, and deliberately cheap. **The 2.6 MB of Mermaid is not in that
list.** The loader is inert on a page with no `.cinegram` element — no fetch,
no parse, nothing — and the CSS is the box and nothing else. A page that shows
no diagram pays for neither. That is the whole reason the loader exists rather
than listing `mermaid.min.js` and `runtime.js` directly.

On any other site, a `<script src=…>` before `</body>` and a `<link rel=…>` in
the head do the same job. Nothing in the kit assumes a particular generator;
the palette hooks are Material's, and they fall back rather than break.

## The div

```html
<div class="cinegram" data-cinegram="storage-failover" data-height="900"></div>
```

`data-height` is the space the box **reserves before the player mounts**, in
pixels, and only until then — the player measures itself once it is up and the
reservation is dropped. Its whole job is to stop the article shifting as each
diagram arrives, which on a page of five is the whole page moving under the
reader.

Load the page once, measure the box, put that number in. There is no way to
know it ahead of time: the diagram's own height depends on how Mermaid lays it
out at your column width, and that is not knowable until it has been laid out.
Guessing high is the cheaper mistake.

## What you get

A player mounted this way is a **guest on your page**, not a page of its own:

- **Keys are scoped to the diagram.** Space, the arrows and `/` reach the
  player only when it has focus, so the theme's own search and navigation keep
  working everywhere else.
- **The address bar is left alone.** The theme writes heading anchors there;
  a player treating one as a view id would jump somewhere nobody asked to go.
  Drill-down and Back stay inside the player.
- **Nothing autoplays.** Five diagrams starting to move as a page renders is
  noise.
- **Diagrams mount lazily**, a little before they scroll into view, so a long
  page does not run Mermaid over five layouts to show the first one.
- **The palette follows the site.** Flip the theme's light/dark toggle and every
  player on the page redraws — Mermaid picks its colours per render, so this
  has to be a redraw rather than a restyle.

## Giving it your palette

The player's chrome — bar, captions, step list, panels — draws from a dozen CSS
custom properties, `--dgm-bg` through `--dgm-fail`, and by default they are a
neutral light/dark pair that sits quietly on any theme. Set them on `:root` in
your own stylesheet and the chrome follows, no different from retuning
Material's own variables.

Cinegram ships one made-up palette of its own for the players it owns — the
greenbar-and-phosphor look of this site, defined in `runtime.css` as a **skin**.
Any site may wear it, with one declaration:

```css
:root { --cinegram-skin: mainframe; }
```

The loader reads that on a page carrying a diagram and stamps
`data-dgm-skin="mainframe"` on `<html>`, which is what the skin's tokens hang
off. It follows the light/dark toggle like everything else: greenbar under
`default`, phosphor under `slate`. This site declares it in
`assets/stylesheets/mainframe.css`, beside the `--cg-*` colours the same theme
gives the prose — which is the argument for a stylesheet declaration over an
attribute on every `<div>`: a skin is a fact about the site, and the next page
you write cannot forget it.

Nothing is skinned by default. A site that declares no `--cinegram-skin` gets
the neutral palette, which is the right answer when the diagram should look
like it belongs to *your* theme rather than to cinegram's.

The diagram itself is not in scope either way: nodes and edges are Mermaid's own
light and dark render. The skin is the chrome around the picture.

## Presenter mode

Click **Present** and the diagram asks the browser for the screen: chrome off,
one beat per press of Space, narration at a size that survives a projector.
Escape or **Exit** comes back. Where the browser refuses element fullscreen —
iOS Safari, some webviews — it pins itself to the window instead, which is the
same box minus the browser's own chrome going away.

The kit does nothing here beyond making the button visible; all of the above is
the player's own, and it is the reason the kit is as small as it is.

A page carrying exactly **one** diagram can also open straight into it with
`?present` on the URL — the way `cinegram preview --serve` does. On a page of
several it is ignored, because there is no sensible answer to which one.

## When to use `cinegram site` instead

If what you have is a folder of `.dgm` files and no prose around them, you do
not need any of this:

```sh
cinegram site diagrams/ -o public/
```

That builds the browsable site — nav from the folder tree, an index per folder,
one shared copy of the runtime — with no static-site generator involved. The
embed kit is for when the diagrams live inside writing that already exists.
