// Opening a .dgm *as* an animation.
//
// A CustomTextEditorProvider registered at `priority: "option"`: the text
// editor stays the default and this appears in *Open With…*, which is the right
// trade for a source format. A file you can read and edit should not open as a
// picture with its own text behind a submenu — but a reader who only wants to
// watch the diagram should not have to know a command exists either.
//
// It is about fifty lines because it owns nothing new. The page is
// dgmPreview.shell — the one place that knows the CSP and the asset wiring —
// and the payload is a compile envelope, so a file with errors shows its
// diagnostics through media/panel.js rather than going blank.

'use strict';

const vscode = require('vscode');

const compile = require('./compile');
const dgmPreview = require('./dgmPreview');

const VIEW_TYPE = 'cinegram.animation';

function register(context) {
  context.subscriptions.push(
    vscode.window.registerCustomEditorProvider(
      VIEW_TYPE,
      { resolveCustomTextEditor: (document, panel) => resolve(context, document, panel) },
      {
        // The webview holds a running player and 2.7 MB of parsed mermaid.
        // Rebuilding that every time the tab loses focus would throw the
        // playhead away for nothing.
        webviewOptions: { retainContextWhenHidden: true },
        supportsMultipleEditorsPerDocument: true
      }
    )
  );
}

function resolve(context, document, panel) {
  panel.webview.options = {
    enableScripts: true,
    localResourceRoots: [vscode.Uri.joinPath(context.extensionUri, 'media')]
  };

  const render = () => {
    panel.webview.html = dgmPreview.shell(context, panel.webview, compile.file(document.uri.fsPath));
  };
  render();

  // Refresh on save, not on keystroke — the same rule the side panel follows,
  // and for two reasons. Assigning `webview.html` is a whole-page reload, so
  // per-keystroke rendering would reset the playhead and re-parse mermaid on
  // every character; and `view … from` reads from disk, so before a save the
  // compiler and the editor disagree about what this file even is.
  const saved = vscode.workspace.onDidSaveTextDocument((doc) => {
    if (doc.uri.toString() === document.uri.toString()) render();
  });
  panel.onDidDispose(() => saved.dispose());
}

module.exports = { register, VIEW_TYPE };
