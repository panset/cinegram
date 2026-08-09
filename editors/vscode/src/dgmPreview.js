// The standalone preview for a .dgm file.
//
// This is the other half of the extension, and it is deliberately not the page
// `cinegram preview` writes. That page inlines everything, including 2.7 MB of
// mermaid, which is right for a file you might email to someone and wrong for a
// panel that re-renders every time you save. Here the assets are webview URIs
// loaded once and cached by the webview, and only the timeline is replaced.
//
// It is also why pkg/emit/html never needed a nonce option: this panel writes
// its own shell, so the emitted page stays exactly as self-contained as it was.

'use strict';

const path = require('path');
const vscode = require('vscode');

const compile = require('./compile');

/** One panel per source file, so re-running the command reveals rather than duplicates. */
const panels = new Map();

function register(context) {
  context.subscriptions.push(
    vscode.commands.registerCommand('cinegram.showPreview', (uri) => {
      const target = uri || (vscode.window.activeTextEditor && vscode.window.activeTextEditor.document.uri);
      if (!target) {
        vscode.window.showInformationMessage('Open a .dgm file to preview it.');
        return;
      }
      show(context, target, vscode.ViewColumn.Beside);
    })
  );

  // Saving is the moment the file on disk becomes what the panel claims to be
  // showing. Rendering on every keystroke would mean the panel and the compiler
  // disagree about what "this file" means, since `view … from` reads from disk.
  context.subscriptions.push(
    vscode.workspace.onDidSaveTextDocument((doc) => {
      for (const [key, entry] of panels) {
        if (key === doc.uri.toString() || entry.dependsOn.has(doc.uri.fsPath)) {
          update(context, entry);
        }
      }
    })
  );
}

function show(context, uri, column) {
  const key = uri.toString();
  const existing = panels.get(key);
  if (existing) {
    existing.panel.reveal(column);
    return existing;
  }

  const panel = vscode.window.createWebviewPanel(
    'cinegram.preview',
    'Cinegram: ' + path.basename(uri.fsPath),
    column,
    {
      enableScripts: true,
      retainContextWhenHidden: true,
      localResourceRoots: [vscode.Uri.joinPath(context.extensionUri, 'media')]
    }
  );

  const entry = { panel, uri, context, dependsOn: new Set([uri.fsPath]) };
  panels.set(key, entry);
  panel.onDidDispose(() => panels.delete(key));

  update(context, entry);
  return entry;
}

function update(context, entry) {
  const result = compile.file(entry.uri.fsPath);

  // A diagram that pulls in other files should refresh when any of them is
  // saved, not only its own entry point.
  entry.dependsOn = new Set([entry.uri.fsPath]);
  for (const d of result.diagnostics || []) {
    if (d.file) entry.dependsOn.add(d.file);
  }

  entry.panel.webview.html = shell(context, entry.panel.webview, result);
}

/**
 * The page around the player: the three assets as webview URIs, the payload as
 * a JSON island, and a CSP that permits exactly that and nothing else.
 *
 * The payload is a <script type="application/json"> rather than a global, so
 * that nothing here has to be executable — the only scripts on the page are the
 * three files, each carrying the nonce.
 */
function shell(context, webview, result) {
  const nonce = String(Math.random()).slice(2) + String(Date.now());
  const asset = (name) =>
    webview.asWebviewUri(vscode.Uri.joinPath(context.extensionUri, 'media', name));

  // The payload sits inside a <script> element, so anything that could close
  // it early or break the parse is escaped. U+2028 and U+2029 are legal inside
  // a JSON string but are line terminators to a JavaScript parser, and are
  // written as escapes here because as literals they are invisible.
  const payload = JSON.stringify(result)
    .replace(/</g, '\\u003c')
    .replace(/\\u2028/g, '\\\\u2028')
    .replace(/\\u2029/g, '\\\\u2029');

  return `<!doctype html>
<html lang="en">
<head>
<meta charset="utf-8">
<meta http-equiv="Content-Security-Policy" content="default-src 'none'; img-src ${webview.cspSource} data:; style-src ${webview.cspSource} 'unsafe-inline'; font-src ${webview.cspSource} data:; script-src 'nonce-${nonce}';">
<meta name="viewport" content="width=device-width, initial-scale=1">
<title>Cinegram</title>
<link rel="stylesheet" href="${asset('runtime.css')}">
<link rel="stylesheet" href="${asset('preview.css')}">
<style>
  html, body { height: 100%; }
  body { margin: 0; }
</style>
</head>
<body class="dgm-standalone">
<div id="cinegram"></div>
<script type="application/json" id="cinegram-payload" nonce="${nonce}">${payload}</script>
<script nonce="${nonce}" src="${asset('mermaid.min.js')}"></script>
<script nonce="${nonce}" src="${asset('runtime.js')}"></script>
<script nonce="${nonce}" src="${asset('panel.js')}"></script>
</body>
</html>`;
}

// shell is exported so the page can be built and inspected without opening a
// panel — the CSP and the asset wiring are the parts most worth checking, and
// they are pure given a webview.
module.exports = { register, show, shell };
