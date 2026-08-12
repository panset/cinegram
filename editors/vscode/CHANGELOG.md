# Changelog

All notable changes to the Cinegram extension are recorded here. The format
follows [Keep a Changelog](https://keepachangelog.com/en/1.1.0/), and the
project uses [semantic versioning](https://semver.org/spec/v2.0.0.html).

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

[0.1.0]: https://github.com/panset/cinegram/releases/tag/v0.1.0
