// Mounting Cinegram players inside VS Code's Markdown preview.
//
// This runs in the preview webview, loaded through markdown.previewScripts so
// that VS Code gives it the CSP nonce. It is a classic script, not a module,
// for the same reason runtime.js is: `document.currentScript` and plain globals
// are worth more here than import syntax.
//
// Three properties of the preview drive everything below.
//
//   1. There is no event for the first render. The rendered Markdown is not in
//      the initial DOM at all — it arrives in a <meta data-initial-md-content>
//      attribute and is injected at DOMContentLoaded, while this script is
//      loaded `async` and may well have run already. So mounting is driven by a
//      MutationObserver as well as by the update event.
//
//   2. Updates are a morphdom diff, not a replacement. Editing a paragraph two
//      pages away leaves our <pre> in place but reverts its contents to the
//      placeholder the extension host just produced, disposing nothing and
//      telling nobody. So "is this still mounted?" has to be asked of the DOM
//      each pass, not remembered.
//
//   3. Which means state has to be carried across, or a diagram would jump back
//      to the start on every keystroke anywhere in the file. The placeholder's
//      id is a hash of the block's own source, so an unchanged block keeps its
//      identity — and its playhead — while an edited one correctly starts over.

(function () {
  'use strict';

  var BLOCK = 'pre.cinegram-block';

  /** id -> the Player currently or most recently mounted for it. */
  var players = Object.create(null);

  var scheduled = false;
  var mounting = false;

  // --- mermaid ---------------------------------------------------------------

  // mermaid registers a `load` handler that renders every `.mermaid` element it
  // can find, and its default config turns that on. VS Code's own Markdown
  // preview emits exactly that class for its Mermaid blocks, so left alone our
  // copy would race VS Code's over diagrams that are not ours.
  //
  // Called eagerly and again from the readiness poll: `load` waits for every
  // async script to run, so either path lands before it fires.
  function silenceMermaidAutoRun() {
    if (silenceMermaidAutoRun.done || !window.mermaid) return;
    try {
      window.mermaid.initialize({ startOnLoad: false });
      silenceMermaidAutoRun.done = true;
    } catch (e) { /* a mermaid that cannot be configured will fail visibly later */ }
  }

  function ready() {
    return !!(window.Cinegram && window.mermaid);
  }

  // --- lifecycle -------------------------------------------------------------

  function schedule() {
    if (scheduled) return;
    scheduled = true;
    // One pass per turn of the event loop: a morphdom update arrives as a burst
    // of mutations, and mounting per mutation would rebuild every diagram on the
    // page several times over.
    setTimeout(function () {
      scheduled = false;
      mountAll();
    }, 0);
  }

  function mountAll() {
    silenceMermaidAutoRun();

    var blocks = document.querySelectorAll(BLOCK);
    if (!blocks.length) return;

    if (!ready()) {
      // Contributed scripts are `async` with no guaranteed order, so this one
      // may run first. Wait rather than fail: the assets are already on the way.
      waitForReady();
      return;
    }

    mounting = true;
    try {
      var live = Object.create(null);
      for (var i = 0; i < blocks.length; i++) {
        var el = blocks[i];
        var id = el.getAttribute('data-cinegram-id') || ('block-' + i);
        live[id] = true;
        if (!isMounted(el)) mountBlock(el, id);
      }

      // A block that has left the page keeps a live clock until its player is
      // stopped: the element is detached but the animation loop is not.
      for (var key in players) {
        if (live[key]) continue;
        stop(players[key]);
        delete players[key];
      }

      // Switching the editor's theme refreshes the preview, but the refreshed
      // HTML is identical, so the diff changes nothing and no block is
      // re-mounted. Nothing else would ever tell a player its colours are now
      // wrong, so tell it here — on every pass, since this is the one place
      // that runs after every update.
      var kind = themeKind();
      for (var id in players) {
        try { players[id].setTheme(kind); } catch (e) { /* older runtime */ }
      }
    } finally {
      mounting = false;
    }
  }

  /**
   * Whether this element still holds what we put in it.
   *
   * Asked of the DOM rather than of a flag we set, because morphdom reverts the
   * contents in place: the element, and any property hung off it, survive an
   * edit that has just thrown the player away.
   */
  function isMounted(el) {
    var first = el.firstElementChild;
    return !!first && (first.classList.contains('dgm') || first.classList.contains('cinegram-error'));
  }

  function mountBlock(el, id) {
    // Read the outgoing player before replacing it — it is detached by now, but
    // its clock is still the truth about where the reader had got to.
    var was = players[id] ? snapshot(players[id]) : null;
    stop(players[id]);
    delete players[id];

    var payload = decode(el.getAttribute('data-cinegram'));
    if (!payload) {
      showMessages(el, [{ message: 'This block did not compile, and no diagnostics came back.' }]);
      return;
    }

    var errors = (payload.diagnostics || []).filter(function (d) { return d.severity === 'error'; });
    if (errors.length || !payload.timeline) {
      showMessages(el, errors.length ? errors : [{ message: 'This block produced no diagram.' }]);
      return;
    }

    // The placeholder is a <pre> so that markdown-it leaves it alone, and the
    // player mounts into a child of it: the `all: unset` that cancels VS Code's
    // code-block styling would otherwise beat every rule in runtime.css.
    el.textContent = '';
    var host = document.createElement('div');
    el.appendChild(host);

    var player;
    try {
      player = window.Cinegram.mount(host, payload.timeline, {
        inline: true,
        keys: 'scoped',
        hash: false,
        autoplay: false,
        theme: themeKind()
      });
    } catch (e) {
      showMessages(el, [{ message: 'This diagram could not be drawn: ' + (e && e.message ? e.message : e) }]);
      return;
    }

    restore(player, was);
    players[id] = player;
  }

  // --- carrying state across a re-render --------------------------------------

  function snapshot(player) {
    try {
      return {
        time: player.time,
        scenarioIndex: player.scenarioIndex,
        viewId: player.view() ? player.view().id : null,
        stack: player.stack.slice(),
        playing: player.playing
      };
    } catch (e) {
      return null;
    }
  }

  function restore(player, was) {
    if (!was) return;
    try {
      // setView rather than navigate: the stack is restored wholesale just
      // below, and navigating would push onto it a second time.
      if (was.viewId && was.viewId !== player.view().id) player.setView(was.viewId);
      player.stack = was.stack.slice();
      player.syncNav();

      if (was.scenarioIndex && was.scenarioIndex !== player.scenarioIndex) {
        player.selectScenario(was.scenarioIndex);
      }
      // Seek last: setView and selectScenario both reset the clock.
      player.seek(was.time);
      if (was.playing) player.play();
    } catch (e) {
      // A timeline that changed shape under a stable id should not take the page
      // down; the diagram simply starts from the beginning.
    }
  }

  function stop(player) {
    if (!player) return;
    // dispose also takes the player's document- and window-level listeners off
    // again, which pause alone leaves behind — a real leak here, since every
    // edit to the file remounts every block. Feature-detected so a runtime.js
    // copy older than this file still stops the clock.
    try { player.dispose ? player.dispose() : player.pause(); } catch (e) { /* already gone */ }
  }

  // --- rendering a failure -----------------------------------------------------

  function showMessages(el, diagnostics) {
    el.textContent = '';
    var box = document.createElement('div');
    box.className = 'cinegram-error';

    for (var i = 0; i < diagnostics.length; i++) {
      var d = diagnostics[i];

      var line = document.createElement('div');
      line.className = 'cinegram-error-msg';
      line.textContent = (d.line ? 'line ' + d.line + ': ' : '') + d.message;
      box.appendChild(line);

      if (d.hint) {
        var hint = document.createElement('div');
        hint.className = 'cinegram-error-hint';
        hint.textContent = d.hint;
        box.appendChild(hint);
      }
    }
    el.appendChild(box);
  }

  // --- helpers ----------------------------------------------------------------

  function decode(b64) {
    if (!b64) return null;
    try {
      // atob yields one byte per character; the payload is UTF-8, and a label
      // with an accent in it would otherwise arrive mangled.
      var chars = atob(b64);
      var bytes = new Uint8Array(chars.length);
      for (var i = 0; i < chars.length; i++) bytes[i] = chars.charCodeAt(i);
      return JSON.parse(new TextDecoder('utf-8').decode(bytes));
    } catch (e) {
      return null;
    }
  }

  // The preview marks its theme on <body>; the runtime wants light or dark.
  // High contrast maps to whichever side it is a variant of.
  function themeKind() {
    var c = document.body.classList;
    if (c.contains('vscode-light') || c.contains('vscode-high-contrast-light')) return 'light';
    return 'dark';
  }

  function waitForReady() {
    if (waitForReady.timer) return;
    var tries = 0;
    waitForReady.timer = setInterval(function () {
      silenceMermaidAutoRun();
      if (ready()) {
        clearInterval(waitForReady.timer);
        waitForReady.timer = null;
        mountAll();
        return;
      }
      if (++tries < 200) return; // ~20 seconds

      clearInterval(waitForReady.timer);
      waitForReady.timer = null;
      var missing = window.mermaid ? 'runtime.js' : 'mermaid.min.js';
      var blocks = document.querySelectorAll(BLOCK);
      for (var i = 0; i < blocks.length; i++) {
        showMessages(blocks[i], [{ message: 'The Cinegram runtime did not load: ' + missing + ' is missing.' }]);
      }
    }, 100);
  }

  /**
   * Whether a mutation is a player animating rather than the document changing.
   *
   * The clock rewrites class attributes inside the SVG on every frame, so
   * reacting to those would run a full mounting pass sixty times a second. The
   * test is which comes first walking outwards: a mounted player's own root
   * means it is our own animation, the block wrapper means something changed the
   * block itself — which is exactly how a morphdom revert arrives, and must not
   * be filtered out.
   */
  function isPlayerChurn(node) {
    while (node && node !== document.body) {
      if (node.nodeType === 1 && node.classList) {
        if (node.classList.contains('dgm')) return true;
        if (node.classList.contains('cinegram-block')) return false;
      }
      node = node.parentNode;
    }
    return false;
  }

  // --- entry points ------------------------------------------------------------

  silenceMermaidAutoRun();

  // The documented signal, fired after every content update — but never for the
  // first one.
  window.addEventListener('vscode.markdown.updateContent', schedule);

  new MutationObserver(function (records) {
    if (mounting) return;
    for (var i = 0; i < records.length; i++) {
      if (!isPlayerChurn(records[i].target)) {
        schedule();
        return;
      }
    }
  }).observe(document.body, { childList: true, subtree: true });

  // The theme is written as a class on <body>, and switching it does not
  // necessarily change a single byte of the rendered document — so the diff
  // above may see nothing at all. A separate, narrow observer catches it.
  new MutationObserver(schedule).observe(document.body, {
    attributes: true,
    attributeFilter: ['class']
  });

  if (document.readyState === 'loading') {
    document.addEventListener('DOMContentLoaded', schedule);
  }
  schedule();
})();
