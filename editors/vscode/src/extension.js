// Cinegram in VS Code.
//
// The split is the one the Markdown preview forces. Compiling has to happen in
// the extension host, because markdown-it's renderer is synchronous and the
// compiler is a separate process; drawing has to happen in the webview, because
// that is where the DOM is. So this file turns a ```dgm block into an inert
// placeholder carrying a compiled timeline, and media/preview.js turns that
// placeholder into a player.
//
// Two properties of the preview shape the code more than anything else:
//
//   - Its CSP is `script-src 'nonce-…'`, and the nonce is generated per render
//     and never handed to a plugin. Any <script> emitted here would be blocked
//     *and* would trip the "some content has been disabled" banner. So the
//     placeholder is data only, and every line of code the page runs arrives
//     through markdown.previewScripts, which VS Code nonces for us.
//
//   - Rendering is synchronous all the way down. There is no awaiting a
//     compile, which is why pkg/compile is reached through execFileSync and a
//     content-addressed cache.

'use strict';

const vscode = require('vscode');

const binary = require('./binary');
const compile = require('./compile');
const dgmPreview = require('./dgmPreview');

function activate(context) {
  dgmPreview.register(context);

  context.subscriptions.push(
    vscode.workspace.onDidChangeConfiguration((e) => {
      if (!e.affectsConfiguration('cinegram')) return;
      // The binary or the recognised languages may have moved; everything
      // derived from them is now suspect.
      binary.reset();
      compile.clearCache();
      vscode.commands.executeCommand('markdown.preview.refresh');
    })
  );

  return { extendMarkdownIt };
}

function deactivate() {}

// --- the markdown-it plugin -------------------------------------------------

/**
 * VS Code calls this once per engine, before installing its own renderers.
 *
 * Both hooks below are the ones VS Code's own bundled Mermaid support uses,
 * which is the strongest available evidence that they keep working: the fenced
 * *rule* is wrapped by VS Code after we run and has changed name between
 * releases, where `options.highlight` is plain markdown-it and has not.
 */
function extendMarkdownIt(md) {
  // Which document is being rendered is not passed to the highlight hook, but
  // it is passed to render(). Stashing it for the duration of one synchronous
  // render is enough, and it is what lets `view … from "pod-a.dgm"` inside a
  // fenced block resolve relative to the .md file the block lives in — the same
  // way VS Code resolves a relative image path.
  const render = md.renderer.render.bind(md.renderer);
  md.renderer.render = function (tokens, options, env) {
    const previousDoc = currentDoc;
    const previousIds = usedIds;
    currentDoc = (env && env.currentDocument) || undefined;
    // Ids are per document render, so two identical blocks in one file get
    // distinct ones and cannot share a playhead.
    usedIds = new Set();
    try {
      return render(tokens, options, env);
    } finally {
      currentDoc = previousDoc;
      usedIds = previousIds;
    }
  };

  const highlight = md.options.highlight;
  md.options.highlight = (code, lang, attrs) => {
    if (isBlockLanguage(lang)) return renderBlock(code, currentDoc);
    return highlight ? highlight(code, lang, attrs) : code;
  };

  return md;
}

let currentDoc;
let usedIds = new Set();

function isBlockLanguage(lang) {
  if (!lang) return false;
  // Only the first word: ```dgm and ```dgm autoplay both name the language,
  // where ```dgmx does not.
  const first = String(lang).trim().split(/\s+/)[0].toLowerCase();
  return languages().indexOf(first) >= 0;
}

function languages() {
  const configured = vscode.workspace.getConfiguration('cinegram').get('blockLanguages');
  return Array.isArray(configured) && configured.length
    ? configured.map((s) => String(s).toLowerCase())
    : ['dgm', 'cinegram'];
}

// --- turning a block into a placeholder --------------------------------------

/**
 * renderBlock returns the HTML for one fenced block.
 *
 * The wrapper is a <pre> because markdown-it only skips its own `<pre><code>`
 * wrapping when the highlight hook returns something already starting with
 * `<pre`. `all: unset` then cancels the code-block styling VS Code puts on it —
 * but only on the wrapper: the player mounts into a child, so that the reset
 * never lands on the element runtime.css has to style.
 */
function renderBlock(source, docUri) {
  const docPath = docUri && docUri.fsPath ? docUri.fsPath : '';
  const { result, key } = compile.block(source, docPath);
  const payload = Buffer.from(JSON.stringify(result), 'utf8').toString('base64');

  return (
    '<pre class="cinegram-block" style="all: unset; display: block"' +
    ' data-cinegram-id="' + attr(uniqueId(key)) + '"' +
    ' data-cinegram="' + attr(payload) + '"' +
    '>' + escapeHtml(source) + '</pre>\n'
  );
}

/**
 * uniqueId keeps the content hash as the identity — so a block that did not
 * change keeps its playhead across an edit elsewhere in the document — while
 * still separating two blocks that happen to hold the same source.
 */
function uniqueId(key) {
  let id = key;
  let n = 0;
  while (usedIds.has(id)) id = key + '-' + ++n;
  usedIds.add(id);
  return id;
}

function escapeHtml(s) {
  return String(s).replace(/[&<>]/g, (c) => ({ '&': '&amp;', '<': '&lt;', '>': '&gt;' }[c]));
}

function attr(s) {
  return String(s).replace(/[&<>"]/g, (c) => ({ '&': '&amp;', '<': '&lt;', '>': '&gt;', '"': '&quot;' }[c]));
}

module.exports = { activate, deactivate };
