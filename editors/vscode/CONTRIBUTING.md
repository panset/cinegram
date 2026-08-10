# Working on the Cinegram extension

This is the maintainer's half of `editors/vscode/`. `README.md` is the
Marketplace listing and is written for someone who has just installed the
extension; everything about building, packaging and publishing it lives here,
so that none of it appears on the gallery page.

## Installing it from a working tree

The extension is plain JavaScript with no dependencies and no build step, so a
symlink is the whole install:

```sh
ln -s "$PWD/editors/vscode" ~/.vscode/extensions/tejaspanse.cinegram-0.1.0
```

Then **Developer: Reload Window** in VS Code. That folder name is not arbitrary:
VS Code reads it as `<publisher>.<name>-<version>`, so it has to match
`package.json` or the extension is ignored.

It needs the `cinegram` binary, which it looks for in this order:

1. the `cinegram.path` setting,
2. a copy bundled with the extension under `bin/<platform>-<arch>/`,
3. `bazel-bin/cmd/cinegram/cinegram_/cinegram` in an open workspace folder,
4. `cinegram` on your `PATH`.

Step 3 is the one that works straight after `bazel build //cmd/cinegram` with
this repository open, which is why nothing needs configuring to try it.

A binary older than the extension answers an unknown flag with Go's whole usage
text, which explains nothing. `describeSkew` in `src/record.js` recognises that
and says to rebuild — the skew is routine here, because a `git pull` updates
the extension without rebuilding the binary under `bazel-bin/`.

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

## The copied files

`media/runtime.js`, `media/runtime.css`, `media/mermaid.min.js` and
`LICENSE.txt` are copies of files this repository owns elsewhere. They have to
be copies: `go:embed` cannot reach outside its own package, VS Code's
`markdown.previewScripts` takes only extension-relative paths, and a `.vsix`
contains nothing from outside the extension folder.

`bazel test //editors/vscode:assets_test` fails when they drift, and
`bazel run //editors/vscode:sync_assets` fixes it.

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

All five at once, which is what a release needs:

```sh
for TARGET in darwin-arm64 darwin-x64 linux-x64 linux-arm64 win32-x64; do
  bazel build //cmd/cinegram:cinegram_$TARGET
  bazel run   //cmd/vsix -- \
    --target "$TARGET" \
    --binary "$PWD/$(bazel cquery --output=files //cmd/cinegram:cinegram_$TARGET)" \
    --out "dist/cinegram-$TARGET.vsix"
done
bazel build //cmd/cinegram   # restore the bazel-bin symlink, see below
```

Building a cross target clears Bazel's `bazel-bin` convenience symlink, because
those targets self-transition to another configuration. The final `bazel build
//cmd/cinegram` puts it back — the extension's development-mode binary lookup
goes through it.

`cmd/vsix` warns on stderr about anything the Marketplace listing will be
missing — no icon, no changelog, no repository link. None of those stop a
package installing, which is exactly why they are worth saying out loud: a
`.vsix` with no icon and no README installs perfectly and lists as a blank grey
square.

## Releasing

1. **Bump `version` in `package.json`** and add a section to `CHANGELOG.md`. A
   published version number can never be reused, so this comes first.
2. **`bazel test //...`** — `//editors/vscode:assets_test` is the one that
   catches a stale copied file, and `//cmd/vsix:vsix_test` the one that catches
   a listing that will render wrong.
3. **Build all five packages**, as above.
4. **Install one locally and open a diagram**, which is the only check that the
   bundled binary in a package actually runs:
   ```sh
   code --install-extension dist/cinegram-darwin-arm64.vsix
   ```
5. **Upload.** Publishing wants the publisher's Personal Access Token from
   Azure DevOps, and platform-specific packages have to be uploaded together:

   ```sh
   npx @vscode/vsce publish --packagePath dist/*.vsix
   ```

   That `npx` is the one place Node is needed, and it is deliberately not a
   build dependency — the packages are already built, and `vsce` only uploads
   them. The same files can be dragged into the
   [publisher management page](https://marketplace.visualstudio.com/manage)
   instead, which needs no toolchain at all.
6. **Tag the commit** `v<version>` so the `CHANGELOG.md` link resolves.

### Before the first release

- The **publisher ID `tejaspanse`** has to exist and be owned by the Azure
  DevOps account doing the upload. Create it once at
  <https://marketplace.visualstudio.com/manage/createpublisher>; it is the
  `publisher` field in `package.json`, and together with `name` it fixes the
  extension's identity as **`tejaspanse.cinegram`** — the string people install
  by, which cannot be changed afterwards without republishing under a new one.
  The listing then lives at
  <https://marketplace.visualstudio.com/items?itemName=tejaspanse.cinegram>.
- **A Personal Access Token** for that account, scoped to *Marketplace →
  Manage*, over the **all accessible organizations** scope. `vsce login
  tejaspanse` stores it; `vsce publish` will otherwise ask on every run.
- The **repository must be public**, or the README's images 404 on the gallery
  page: the Marketplace renders the README from inside the package but fetches
  images over the network, which is why every URL in it is absolute.
