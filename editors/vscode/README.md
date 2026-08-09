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
their own (`Cinegram: Open Preview to the Side`).

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

## Known limitations

- **Exporting the Markdown to HTML or PDF will not include the diagrams.**
  Exporters run the markdown-it plugin but load none of the preview scripts, so
  they see an empty placeholder.
- **VS Code for Web is not supported**, because rendering a block spawns a
  native binary.
- **Markdown cells in notebooks** need a separate `notebookRenderer`
  contribution, which this does not have yet.
