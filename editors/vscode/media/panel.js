// Mounting the player in the standalone .dgm preview panel.
//
// Simpler than the Markdown preview's job in every respect: one player, a page
// of its own, and no diffing host to fight. The panel is rebuilt wholesale on
// save, so there is nothing to carry across and nothing to observe.
//
// It reads its payload from a <script type="application/json"> island rather
// than a global, so that the only executable scripts on the page are the three
// files the shell nonces.

(function () {
  'use strict';

  function payload() {
    var el = document.getElementById('cinegram-payload');
    if (!el) return null;
    try {
      return JSON.parse(el.textContent);
    } catch (e) {
      return null;
    }
  }

  function showMessages(root, diagnostics) {
    root.textContent = '';
    var box = document.createElement('div');
    box.className = 'cinegram-error';

    for (var i = 0; i < diagnostics.length; i++) {
      var d = diagnostics[i];

      var line = document.createElement('div');
      line.className = 'cinegram-error-msg';
      line.textContent = (d.file ? d.file + ' ' : '') + (d.line ? 'line ' + d.line + ': ' : '') + d.message;
      box.appendChild(line);

      if (d.hint) {
        var hint = document.createElement('div');
        hint.className = 'cinegram-error-hint';
        hint.textContent = d.hint;
        box.appendChild(hint);
      }
    }
    root.appendChild(box);
  }

  function themeKind() {
    var c = document.body.classList;
    if (c.contains('vscode-light') || c.contains('vscode-high-contrast-light')) return 'light';
    return 'dark';
  }

  function start() {
    var root = document.getElementById('cinegram');
    if (!root) return;

    var data = payload();
    if (!data) {
      showMessages(root, [{ message: 'The compiled diagram did not reach the panel.' }]);
      return;
    }

    var errors = (data.diagnostics || []).filter(function (d) { return d.severity === 'error'; });
    if (errors.length || !data.timeline) {
      showMessages(root, errors.length ? errors : [{ message: 'This file produced no diagram.' }]);
      return;
    }

    // mermaid's default config renders every `.mermaid` element on load. There
    // are none here, but initialising it before that fires is free insurance
    // and matches what the runtime does on its own page.
    try { window.mermaid.initialize({ startOnLoad: false }); } catch (e) { /* reported below if fatal */ }

    // Full chrome, unlike a block in a document: this panel *is* the page, so
    // the scenario picker, the step list and presenter mode all belong. Only the
    // address is withheld — a webview's URL is not the reader's to share.
    window.CINEGRAM_PLAYER = window.Cinegram.mount(root, data.timeline, {
      hash: false,
      theme: themeKind()
    });
  }

  function whenReady() {
    if (window.Cinegram && window.mermaid) {
      start();
      return;
    }
    var tries = 0;
    var timer = setInterval(function () {
      if (window.Cinegram && window.mermaid) {
        clearInterval(timer);
        start();
      } else if (++tries > 200) {
        clearInterval(timer);
        var root = document.getElementById('cinegram');
        if (root) showMessages(root, [{ message: 'The Cinegram runtime did not load.' }]);
      }
    }, 50);
  }

  if (document.readyState === 'loading') {
    document.addEventListener('DOMContentLoaded', whenReady);
  } else {
    whenReady();
  }
})();
