# Changelog

All notable changes to the Cinegram extension are recorded here. The format
follows [Keep a Changelog](https://keepachangelog.com/en/1.1.0/), and the
project uses [semantic versioning](https://semver.org/spec/v2.0.0.html).

## [0.3.0] — 2026-08-17

### Added

- **`stateDiagram-v2` as a third diagram type.** States, transitions, `[*]`
  markers, composite states and `<<choice>>`/`<<fork>>`/`<<join>>`
  pseudostates are all animation targets, alongside `flowchart`/`graph` and
  `sequenceDiagram`. Notes, `direction`, `classDef` and the `--` divider
  round-trip verbatim, and the `.dgm` grammar highlights the new keywords.
- **Reel mode.** `?reel` is a vertical, phone-shaped story layout: a tap plays
  exactly one step, the chrome goes entirely, the caption narrates at phone
  size, and a segmented step bar tracks progress. An auto-follow camera rides
  the action behind a Cine toggle. `record --reel` and `frame --reel` shoot
  that page at 1080×1920 instead of the landscape embed.
- **`poster:` and `stepwise:` scenario attributes.** A resting moment to show
  while paused, and a Play button that advances one step per press.
- **A lightbox for storyboard scenes.** Click a frame for a viewport-fit view
  with cursor-anchored wheel zoom; Esc closes the innermost thing open.
- **`cinegram site <dir>` turns a folder of `.dgm` files into a browsable
  static site.** A player page per document at its own relative path, an index
  per folder, prev/next threaded depth-first like a book, and one shared copy
  of the runtime assets rather than mermaid inlined into every page. Folders
  order by an optional numeric filename prefix that readers never see.
  `-o` writes it; `--serve --watch` develops against it.
- **`cinegram assets -o DIR` writes the player out of the binary** — the three
  runtime files plus the embed kit, `cinegram-embed.js` and
  `cinegram-embed.css`, which mounts a player into any
  `<div class="cinegram" data-cinegram="…">` on an ordinary page.
- **An opt-in mainframe skin.** A page that declares
  `:root { --cinegram-skin: mainframe }` gets cinegram's own palette — greenbar
  paper in light, 3279 phosphor in dark. Unskinned pages, the Markdown preview
  among them, are unchanged to the pixel.

### Changed

- **The player's tools moved off the top bar and onto a rail beside the
  stage.** The bar keeps Play and Present; Restart, speed, Cine, Copy link,
  reset zoom, theme and help are now one translucent icon column over the
  stage's right edge. A diagram inline in a document — the Markdown preview
  included — hides the rail and shows exactly what it showed before.
- **The player fits the screen wherever a host gives it the page.** The stage
  absorbs the leftover height and the diagram scales into it, so the narration
  and the transport stay on screen instead of a tall diagram pushing them
  below the fold. Every one of those rules is a no-op in an auto-height host
  like the Markdown preview.
- **Present asks the browser for the screen** through the Fullscreen API, with
  a fixed full-window fallback where the request is refused (iOS Safari,
  webviews, no gesture). Esc leaves fullscreen and presenter mode together.
- **Scenarios open at rest.** Autoplay now defaults to false — a page that is
  already moving when you arrive is stressful.

## [0.2.0] — 2026-08-12

### Added

- **The bundled `cinegram` binary can update itself when used outside the
  extension.** `cinegram upgrade` replaces a release-installed binary with
  the latest checksum-verified GitHub release; `cinegram upgrade --check`
  reports whether one exists (exit 1 when it does).

### Changed

- The stale-binary message now names the fix for the binary actually in use:
  rebuild for a workspace Bazel build, an extension update for the bundled
  copy, `cinegram upgrade` for a release install.

## [0.1.0] — 2026-08-10

First public release.

### Added

- **Animated ```` ```dgm ```` blocks in the built-in Markdown preview.** A
  fenced block compiles through the bundled `cinegram` binary and plays where
  the code block was, with a scrubber, a scenario picker and narration.
- **A preview panel for `.dgm` files** — `Cinegram: Open Preview to the Side`,
  on the editor title bar and the command palette.
- **An *Open With… → Cinegram Animation* editor**, which opens a `.dgm` as the
  animation itself. It re-renders on save; the text editor stays the default.
- **`Cinegram: Export Animation…`**, which records one scenario to a GIF, mp4 or
  webm beside the diagram, with a cancellable progress notification and a
  **Copy Markdown** action that yields a relative `![…](…)` link.
- **Syntax highlighting** for `.dgm` files and for ```` ```dgm ```` blocks
  inside Markdown.
- **A bundled compiler.** Each platform package carries the `cinegram` binary
  for that platform, so nothing needs installing to render a diagram. GIF
  export is encoded in the binary too; only mp4 and webm need `ffmpeg`.

### Known limitations

- Exporting Markdown to HTML or PDF does not include the diagrams: exporters
  run the markdown-it plugin but load none of the preview scripts.
- VS Code for Web is unsupported — rendering spawns a native binary.
- Markdown cells in notebooks need a `notebookRenderer` contribution, which
  this does not have yet.
- Recording needs a Chrome or Chromium on `PATH` (or `cinegram.chromePath`);
  every frame is a separate headless screenshot.

[0.2.0]: https://github.com/panset/cinegram/releases/tag/v0.2.0
[0.1.0]: https://github.com/panset/cinegram/releases/tag/v0.1.0
