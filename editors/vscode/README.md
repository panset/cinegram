# Cinegram for VS Code

Animated architecture diagrams inside the built-in Markdown preview. Write a
fenced block, press `Cmd+Shift+V`, and the diagram plays where the code block
was.

````markdown
```dgm
flowchart LR
  client[Client]
  api[API]
  client --> api

scenario "A request"

  step call "The client calls the API" {
    flow client -> api { label: "GET /orders", dur: 700ms }
    highlight api { style: active }
  }
```
````

It also gives `.dgm` files syntax highlighting and a full-chrome preview panel of
their own (`Cinegram: Open Preview to the Side`). A `.dgm` can be opened *as* the
animation instead of as text — right-click it and choose **Open With… →
Cinegram Animation** — and either way it can be exported to a GIF from inside
the editor.

## Exporting a GIF

`Cinegram: Export Animation…` — on the editor title bar's `…` menu, the
explorer right-click menu, or the command palette — records one scenario and
writes it beside the diagram:

1. It compiles first, so a typo fails in milliseconds rather than after a
   minute of recording.
2. It asks which view and which scenario, but only when there is more than one
   of each.
3. The save dialog's extension **is** the format: name the file `demo.gif`,
   `demo.mp4` or `demo.webm` and there is no second prompt.
4. A progress notification counts the frames and can be cancelled. Cancelling
   kills the recorder's headless browsers along with it.
5. When it lands, **Copy Markdown** puts `![scenario](demo.gif)` on the
   clipboard, relative to the diagram's own directory — which is the pull
   request this was built for.

**GIF needs nothing installed**; it is encoded inside the `cinegram` binary.
mp4 and webm shell out to `ffmpeg`, found on your `PATH` or named by
`cinegram.ffmpegPath`. Recording needs a Chrome or Chromium, found the same way
or named by `cinegram.chromePath` — every frame is a separate headless
screenshot, which is what makes the result independent of how fast the machine
is, and also what makes `cinegram.record.fps` the setting that decides how long
an export takes.

Fenced ```` ```dgm ```` blocks inside Markdown cannot be exported: `record`
takes a path on disk, and a block has none.

## Installing for development

The extension is plain JavaScript with no dependencies and no build step, so a
symlink is the whole install:

```sh
ln -s "$PWD/editors/vscode" ~/.vscode/extensions/cinegram.cinegram-0.1.0
```

Then **Developer: Reload Window** in VS Code.

It needs the `cinegram` binary, which it looks for in this order:

1. the `cinegram.path` setting,
2. a copy bundled with the extension under `bin/<platform>-<arch>/`,
3. `bazel-bin/cmd/cinegram/cinegram_/cinegram` in an open workspace folder,
4. `cinegram` on your `PATH`.

Step 3 is the one that works straight after `bazel build //cmd/cinegram` with
this repository open, which is why nothing needs configuring to try it.

## Packaging

A `.vsix` is a ZIP with an XML manifest, so `cmd/vsix` builds one with
`archive/zip` and nothing else. There is no `vsce`, no npm and no lockfile —
which matters, because this repository has no JavaScript toolchain and does not
want one.

The extension spawns the compiler, so each package carries the binary for one
platform and VS Code installs the matching package:

```sh
TARGET=darwin-arm64          # or darwin-x64, linux-x64, linux-arm64, win32-x64
bazel build //cmd/cinegram:cinegram_$TARGET
bazel run   //cmd/vsix -- \
  --target "$TARGET" \
  --binary "$PWD/$(bazel cquery --output=files //cmd/cinegram:cinegram_$TARGET)" \
  --out "dist/cinegram-$TARGET.vsix"
```

Building a cross target clears Bazel's `bazel-bin` convenience symlink, because
those targets self-transition to another configuration. Run `bazel build
//cmd/cinegram` afterwards to put it back — the extension's development-mode
binary lookup goes through it.

## How it fits together

The extension holds no knowledge of the diagram language. It shells out to
`cinegram compile` and renders whatever comes back, so a new diagram type,
action or timing rule reaches the preview with no change here.

```
.md file
  └─ extension host: extendMarkdownIt hooks ```dgm fences
        └─ cinegram compile - --as <mdpath> --envelope      (cached by content hash)
              └─ <pre class="cinegram-block" data-cinegram="<base64 envelope>">
                    └─ webview: media/preview.js → Cinegram.mount(el, timeline, {inline: true, …})
```

The preview's CSP is `script-src 'nonce-…'` and the nonce never reaches a
markdown-it plugin, so the placeholder the host emits is **data only** — every
line of code the page runs is contributed through `markdown.previewScripts`,
which VS Code nonces for us.

Export is the one path that does not go through `src/compile.js`. Compiling is
`execFileSync` because markdown-it cannot await and a compile is milliseconds;
recording is one headless browser *per frame* and runs for minutes, so
`src/record.js` uses `spawn`, a cancellable progress notification, and the
`--progress` lines `cinegram record` writes to stderr for it.

`src/animationEditor.js` is the *Open With…* entry. It is a
`CustomTextEditorProvider` that reuses `dgmPreview.shell`, so there stays one
place that knows the CSP and the asset wiring, and it re-renders on save rather
than on keystroke — assigning `webview.html` is a whole-page reload, and doing
that per character would reset the playhead and re-parse 2.7 MB of mermaid.

`media/runtime.js`, `media/runtime.css` and `media/mermaid.min.js` are copies of
the canonical files in `pkg/emit/html/assets/`. They have to be copies: `go:embed`
cannot reach outside its own package, and `markdown.previewScripts` takes only
extension-relative paths. `//editors/vscode:assets_test` fails when they drift,
and `bazel run //editors/vscode:sync_assets` fixes it.

## Settings

| Setting | Default | Meaning |
|---|---|---|
| `cinegram.path` | `""` | Where the binary is. Empty means search, as above. |
| `cinegram.blockLanguages` | `["dgm", "cinegram"]` | Fence languages rendered as diagrams. |
| `cinegram.compileTimeout` | `5000` | Milliseconds a block may take to compile. |
| `cinegram.record.fps` | `12` | Frames per second to export at, and so what an export costs. |
| `cinegram.record.width` | `1280` | Viewport width to record at, rounded up to even. |
| `cinegram.record.height` | `720` | Viewport height to record at, rounded up to even. |
| `cinegram.chromePath` | `""` | The browser that captures frames, as `CINEGRAM_CHROME`. |
| `cinegram.ffmpegPath` | `""` | `ffmpeg`, as `CINEGRAM_FFMPEG`. Only mp4 and webm need it. |

## Known limitations

- **Exporting the Markdown to HTML or PDF will not include the diagrams.**
  Exporters run the markdown-it plugin but load none of the preview scripts, so
  they see an empty placeholder.
- **VS Code for Web is not supported**, because rendering a block spawns a
  native binary.
- **Markdown cells in notebooks** need a separate `notebookRenderer`
  contribution, which this does not have yet.
