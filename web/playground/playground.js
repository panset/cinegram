// The Cinegram playground.
//
// A classic script, like runtime.js and for the same reasons: it is loaded from
// a plain <script> on a static page and there is nothing here that modules would
// buy. The style below matches runtime.js — var, function, no arrow syntax — so
// the two files read as one codebase.
//
// The shape of the thing:
//
//   1. The Go compiler runs in the tab as WebAssembly. `cinegramCompile` and
//      `cinegramRenderHTML` are synchronous calls into it (see
//      web/playground/wasm/main.go); they are milliseconds for real documents,
//      which is why they can sit on the main thread. A Worker is the fix if a
//      document ever appears that makes that false.
//   2. There is no disk. A Map from path to text-or-bytes stands in for one, so
//      `view … from "pod-a.dgm"` and `img: "frames/x.svg"` resolve against
//      whatever the reader has pasted or dropped in.
//   3. Nothing is uploaded, ever. Share encodes the whole file set into the URL
//      fragment, which browsers do not send to servers.

(function () {
  'use strict';

  /** Keystrokes settle before a compile runs; a compile is cheap but a render is not. */
  var DEBOUNCE_MS = 300;

  /** Past roughly this much encoded fragment, browsers and chat clients start truncating. */
  var SHARE_WARN_BYTES = 50 * 1024;

  var DEFAULT_EXAMPLE = 'k8s-request';

  var TEXT_EXT = ['.dgm'];
  var IMAGE_EXT = ['.svg', '.png', '.jpg', '.jpeg', '.gif', '.webp'];

  // --- state -----------------------------------------------------------------

  /** path -> {text: string} for sources, {bytes: Uint8Array} for images. */
  var vfs = new Map();

  var entryPath = '';
  var openPath = '';

  var player = null;
  var title = '';
  var compiled = false;
  var ready = false;
  var timer = null;
  var examples = [];

  var ui = {};

  // --- boot -------------------------------------------------------------------

  function boot() {
    ui.editor = document.getElementById('pg-editor');
    ui.tabs = document.getElementById('pg-tabs');
    ui.left = document.getElementById('pg-left');
    ui.host = document.getElementById('pg-player-host');
    ui.bootMsg = document.getElementById('pg-boot');
    ui.diagnostics = document.getElementById('pg-diagnostics');
    ui.examples = document.getElementById('pg-examples');
    ui.attachInput = document.getElementById('pg-attach-input');
    ui.attachAdd = document.getElementById('pg-attach-add');
    ui.attachList = document.getElementById('pg-attach-list');
    ui.share = document.getElementById('pg-share');
    ui.download = document.getElementById('pg-download');

    wireUI();

    startWasm()
      .then(function () {
        ready = true;
        // mermaid registers a `load` handler that renders every element it
        // recognises, and its default config turns that on. The player calls
        // mermaid.render itself, so auto-run can only race it. Same lesson as
        // silenceMermaidAutoRun in the VS Code preview, and it has to land
        // before the first mount.
        try { window.mermaid.initialize({ startOnLoad: false }); } catch (e) { /* fails visibly later */ }
        return loadExamplesManifest();
      })
      .then(function () {
        return loadFromHash();
      })
      .then(function (restored) {
        if (restored) return null;
        return loadExample(DEFAULT_EXAMPLE);
      })
      .catch(function (e) {
        fail('The playground did not start: ' + message(e));
      });
  }

  function startWasm() {
    return new Promise(function (resolve, reject) {
      if (typeof window.Go !== 'function') {
        reject(new Error('wasm_exec.js did not load'));
        return;
      }
      var go = new window.Go();

      // The exports are set by the module's main, not by instantiation, so the
      // page cannot tell from the fetch alone when they are callable. The Go
      // side invokes this once it has published them.
      window.onCinegramReady = function () { resolve(); };

      instantiate(go).then(function (instance) {
        // go.run never settles — the Go main blocks on `select {}` so the
        // module stays alive — so its promise is deliberately not chained.
        go.run(instance);
      }).catch(reject);
    });
  }

  function instantiate(go) {
    if (typeof WebAssembly.instantiateStreaming === 'function') {
      return WebAssembly.instantiateStreaming(fetch('cinegram.wasm'), go.importObject)
        .then(function (r) { return r.instance; })
        .catch(function () {
          // Streaming needs `Content-Type: application/wasm`, which not every
          // static host sends. Buffering costs a copy and always works.
          return fromArrayBuffer(go);
        });
    }
    return fromArrayBuffer(go);
  }

  function fromArrayBuffer(go) {
    return fetch('cinegram.wasm').then(function (r) {
      if (!r.ok) throw new Error('cinegram.wasm: HTTP ' + r.status);
      return r.arrayBuffer();
    }).then(function (bytes) {
      return WebAssembly.instantiate(bytes, go.importObject);
    }).then(function (r) { return r.instance; });
  }

  function fail(text) {
    ui.bootMsg.hidden = false;
    ui.bootMsg.textContent = text;
  }

  // --- the virtual filesystem ---------------------------------------------------

  function putText(path, text) { vfs.set(path, { text: text }); }
  function putBytes(path, bytes) { vfs.set(path, { bytes: bytes }); }

  /**
   * The file set as the WASM side wants it: a plain object whose values are JS
   * strings for text and Uint8Arrays for binary. No base64 anywhere — the loader
   * builds its data: URIs from raw bytes.
   */
  function filesForGo() {
    var out = {};
    vfs.forEach(function (file, path) {
      out[path] = file.bytes ? file.bytes : file.text;
    });
    return out;
  }

  function paths() {
    var out = [];
    vfs.forEach(function (_, path) { out.push(path); });
    out.sort();
    return out;
  }

  function isSource(path) { return endsWithAny(path, TEXT_EXT); }
  function isImage(path) { return endsWithAny(path, IMAGE_EXT); }

  function endsWithAny(name, exts) {
    var lower = name.toLowerCase();
    for (var i = 0; i < exts.length; i++) {
      if (lower.length > exts[i].length && lower.slice(-exts[i].length) === exts[i]) return true;
    }
    return false;
  }

  // --- the compile loop ----------------------------------------------------------

  function schedule() {
    if (timer) clearTimeout(timer);
    timer = setTimeout(function () {
      timer = null;
      compile();
    }, DEBOUNCE_MS);
  }

  function saveEditor() {
    if (openPath) putText(openPath, ui.editor.value);
  }

  function compile() {
    if (!ready || !entryPath) return;

    var result;
    try {
      result = JSON.parse(window.cinegramCompile(entryPath, filesForGo()));
    } catch (e) {
      // The Go side catches its own panics and answers with an error envelope,
      // so reaching here means the module is gone. Say so rather than dying
      // silently in a callback.
      showDiagnostics([{ severity: 'error', message: 'The compiler stopped responding: ' + message(e) }]);
      stale(true);
      return;
    }

    var found = result.diagnostics || [];
    var errors = found.filter(isError);
    showDiagnostics(found);
    ui.download.disabled = errors.length > 0;

    if (errors.length || !result.timeline) {
      // Deliberately unlike the VS Code preview, which replaces the diagram
      // with its errors: here the reader is mid-keystroke and the last drawing
      // that worked is the more useful thing to leave on screen. It is dimmed
      // so it cannot be mistaken for what was just typed.
      stale(true);
      return;
    }

    stale(false);
    title = result.title || '';
    mount(result.timeline);
  }

  function isError(d) { return d.severity === 'error'; }

  function stale(yes) {
    // Nothing has ever been drawn: an error on the very first compile has no
    // previous diagram to dim, so the boot panel stays up and says so.
    if (yes && !compiled) {
      ui.bootMsg.hidden = false;
      ui.bootMsg.textContent = 'Nothing to draw yet — see the errors on the left.';
      return;
    }
    ui.host.classList.toggle('is-stale', !!yes);
  }

  // --- mounting -------------------------------------------------------------------

  function mount(timeline) {
    var was = player ? snapshot(player) : null;
    stop(player);
    player = null;

    // A fresh child rather than a reused one: the player owns everything below
    // its root, and handing it a node another player has already decorated is
    // asking for two sets of chrome.
    ui.host.textContent = '';
    var inner = document.createElement('div');
    ui.host.appendChild(inner);

    try {
      player = window.Cinegram.mount(inner, timeline, {
        keys: 'scoped',
        hash: false,
        // A first mount plays; a remount mid-edit does not start the clock
        // under the reader — restore puts it back where they left it.
        autoplay: !was
      });
    } catch (e) {
      showDiagnostics([{ severity: 'error', message: 'This diagram could not be drawn: ' + message(e) }]);
      stale(true);
      return;
    }

    // The same debug handle the emitted page exposes, so anything written
    // against `cinegram preview` output works here too.
    window.CINEGRAM_PLAYER = player;
    restore(player, was);

    compiled = true;
    ui.bootMsg.hidden = true;
  }

  function stop(p) {
    if (!p) return;
    // dispose takes back the window- and document-level listeners that pause
    // alone leaves behind. Without it every keystroke would cost another set.
    try { p.dispose ? p.dispose() : p.pause(); } catch (e) { /* already gone */ }
  }

  // Snapshot and restore are the VS Code preview's, unchanged in substance: the
  // reader's position in the animation has to survive a recompile or editing a
  // label would throw them back to the start.
  function snapshot(p) {
    try {
      return {
        time: p.time,
        scenarioIndex: p.scenarioIndex,
        viewId: p.view() ? p.view().id : null,
        stack: p.stack.slice(),
        playing: p.playing
      };
    } catch (e) {
      return null;
    }
  }

  function restore(p, was) {
    if (!was) return;
    try {
      // setView rather than navigate: the stack is restored wholesale just
      // below, and navigating would push onto it a second time.
      if (was.viewId && was.viewId !== p.view().id) p.setView(was.viewId);
      p.stack = was.stack.slice();
      p.syncNav();

      if (was.scenarioIndex && was.scenarioIndex !== p.scenarioIndex) {
        p.selectScenario(was.scenarioIndex);
      }
      // Seek last: setView and selectScenario both reset the clock.
      p.seek(was.time);
      if (was.playing) p.play();
    } catch (e) {
      // A timeline that changed shape under the reader just starts over; an
      // edit is allowed to delete the view they were looking at.
    }
  }

  // --- diagnostics ------------------------------------------------------------------

  // The strip carries two different things and has to keep them apart. The
  // compiler's diagnostics are replaced wholesale by the next compile; a notice
  // — "that file type is not an attachment" — is about something the reader
  // just did, and the recompile their action triggers would wipe it a third of
  // a second later if it lived in the same list. So it is held separately and
  // survives until the next edit.
  var notice = '';
  var diagnostics = [];

  function showDiagnostics(list) {
    diagnostics = list || [];
    paintStrip();
  }

  function note(text) {
    notice = text;
    paintStrip();
  }

  function clearNotice() {
    if (!notice) return;
    notice = '';
    paintStrip();
  }

  function paintStrip() {
    ui.diagnostics.textContent = '';

    if (notice) ui.diagnostics.appendChild(strip(notice, 'pg-diag'));

    // Errors first: they are what stopped the diagram, and a long tail of
    // warnings must not push them out of view.
    var errors = diagnostics.filter(isError);
    var ordered = errors.concat(diagnostics.filter(function (d) { return !isError(d); }));

    for (var i = 0; i < ordered.length; i++) {
      ui.diagnostics.appendChild(strip(describe(ordered[i]), 'pg-diag'));
      if (ordered[i].hint) {
        ui.diagnostics.appendChild(strip(ordered[i].hint, 'pg-diag pg-diag-hint'));
      }
    }

    ui.diagnostics.classList.toggle('has-errors', errors.length > 0);
    ui.diagnostics.hidden = !notice && ordered.length === 0;
  }

  function strip(text, className) {
    var line = document.createElement('p');
    line.className = className;
    line.textContent = text;
    return line;
  }

  function describe(d) {
    var where = '';
    if (d.file) where += d.file + ': ';
    if (d.line) where += 'line ' + d.line + (d.col ? ':' + d.col : '') + ': ';
    return where + (d.severity === 'warning' ? 'warning: ' : '') + d.message;
  }

  // --- open files ----------------------------------------------------------------------

  function openFile(path) {
    saveEditor();
    openPath = path;
    syncEditor();
    renderTabs();
    renderAttachments();
  }

  function syncEditor() {
    var file = vfs.get(openPath);
    ui.editor.value = file && file.text !== undefined ? file.text : '';
    ui.editor.disabled = !file || file.text === undefined;
  }

  function renderTabs() {
    ui.tabs.textContent = '';
    var list = paths().filter(isSource);
    if (list.length < 2) return; // one file needs no tab strip

    for (var i = 0; i < list.length; i++) {
      ui.tabs.appendChild(tab(list[i]));
    }
  }

  function tab(path) {
    var button = document.createElement('button');
    button.type = 'button';
    button.className = 'pg-tab' + (path === openPath ? ' is-open' : '');
    button.textContent = path;
    if (path === entryPath) {
      var mark = document.createElement('span');
      mark.className = 'pg-tab-entry';
      mark.textContent = 'entry';
      button.appendChild(mark);
    }
    button.addEventListener('click', function () { openFile(path); });
    return button;
  }

  // --- attachments -----------------------------------------------------------------------

  function renderAttachments() {
    ui.attachList.textContent = '';
    var list = paths();
    for (var i = 0; i < list.length; i++) {
      ui.attachList.appendChild(attachmentRow(list[i]));
    }
  }

  function attachmentRow(path) {
    var file = vfs.get(path);
    var row = document.createElement('li');
    row.className = 'pg-attach-row' + (isSource(path) ? ' is-openable' : '');

    var name = document.createElement('button');
    name.type = 'button';
    name.className = 'pg-attach-name';
    name.textContent = path;
    if (isSource(path)) {
      name.title = 'Open in the editor';
      name.addEventListener('click', function () { openFile(path); });
    } else {
      name.disabled = true;
    }
    row.appendChild(name);

    var size = document.createElement('span');
    size.className = 'pg-attach-size';
    size.textContent = human(file.bytes ? file.bytes.length : byteLength(file.text));
    row.appendChild(size);

    var remove = document.createElement('button');
    remove.type = 'button';
    remove.className = 'pg-attach-remove';
    remove.title = 'Remove';
    remove.textContent = '×';
    remove.addEventListener('click', function () { removeFile(path); });
    row.appendChild(remove);

    return row;
  }

  function removeFile(path) {
    vfs.delete(path);
    // Removing the file being edited, or the one the compile starts from,
    // leaves the page with no valid target; fall back to any source left.
    if (openPath === path || entryPath === path) {
      var next = paths().filter(isSource)[0] || '';
      if (entryPath === path) entryPath = next;
      if (openPath === path) openPath = next;
    }
    syncEditor();
    renderTabs();
    renderAttachments();
    schedule();
  }

  function attach(fileList) {
    var files = Array.prototype.slice.call(fileList || []);
    var rejected = [];
    var jobs = [];

    files.forEach(function (file) {
      var path = file.webkitRelativePath || file.name;
      if (isSource(path)) {
        jobs.push(file.text().then(function (text) { putText(path, text); }));
      } else if (isImage(path)) {
        jobs.push(file.arrayBuffer().then(function (buf) { putBytes(path, new Uint8Array(buf)); }));
      } else {
        rejected.push(file.name);
      }
    });

    return Promise.all(jobs).then(function () {
      if (!entryPath) {
        entryPath = paths().filter(isSource)[0] || '';
        openPath = entryPath;
        syncEditor();
      }
      renderTabs();
      renderAttachments();
      if (rejected.length) {
        note('Not attached (a .dgm or an image is expected): ' + rejected.join(', '));
      }
      schedule();
    });
  }

  // --- examples -------------------------------------------------------------------------

  function loadExamplesManifest() {
    return fetch('examples.json').then(function (r) {
      if (!r.ok) throw new Error('examples.json: HTTP ' + r.status);
      return r.json();
    }).then(function (list) {
      examples = list;
      ui.examples.textContent = '';
      for (var i = 0; i < list.length; i++) {
        var option = document.createElement('option');
        option.value = list[i].id;
        option.textContent = list[i].title;
        ui.examples.appendChild(option);
      }
    });
  }

  function loadExample(id) {
    var example = null;
    for (var i = 0; i < examples.length; i++) {
      if (examples[i].id === id) example = examples[i];
    }
    if (!example) return Promise.resolve();

    var jobs = example.files.map(function (path) {
      return fetch('examples/' + path).then(function (r) {
        if (!r.ok) throw new Error(path + ': HTTP ' + r.status);
        return isImage(path)
          ? r.arrayBuffer().then(function (buf) { return { path: path, bytes: new Uint8Array(buf) }; })
          : r.text().then(function (text) { return { path: path, text: text }; });
      });
    });

    return Promise.all(jobs).then(function (loaded) {
      vfs.clear();
      loaded.forEach(function (f) {
        if (f.bytes) putBytes(f.path, f.bytes);
        else putText(f.path, f.text);
      });

      entryPath = example.entry;
      openPath = example.entry;
      ui.examples.value = id;

      // A new document is a new reading, so let it play from the top: dropping
      // the player is what makes `autoplay: !was` true again.
      stop(player);
      player = null;
      compiled = false;

      clearHash();
      clearNotice();
      syncEditor();
      renderTabs();
      renderAttachments();
      compile();
    }).catch(function (e) {
      fail('That example could not be loaded: ' + message(e));
    });
  }

  // --- sharing ---------------------------------------------------------------------------
  //
  // The document set goes into the fragment, deflated and base64url'd. The
  // fragment is the only part of a URL browsers never send to a server, so a
  // shared link is a copy of the document and not an upload of it.

  function shareSupported() {
    return typeof window.CompressionStream === 'function' && typeof window.DecompressionStream === 'function';
  }

  function share() {
    encodeDoc().then(function (encoded) {
      var url = location.origin + location.pathname + location.search + '#doc=' + encoded;
      history.replaceState(null, '', url);

      var oversize = encoded.length > SHARE_WARN_BYTES
        ? 'This link is ' + human(encoded.length) + ' long. Some clients truncate URLs that size — prefer Download HTML for a document this big.'
        : '';

      // Both halves of this can have something to say, and the strip holds one
      // notice at a time, so they are joined rather than raced: a link that is
      // too long AND could not be copied must report both.
      return copy(url).then(function (trouble) {
        var say = [oversize, trouble].filter(Boolean).join(' ');
        if (say) note(say);
        else flash(ui.share, 'Copied');
      });
    }).catch(function (e) {
      note('That could not be turned into a link: ' + message(e));
    });
  }

  function encodeDoc() {
    var doc = { v: 1, entry: entryPath, open: openPath, files: [] };
    vfs.forEach(function (file, path) {
      if (file.bytes) doc.files.push({ p: path, t: 'b64', d: base64(file.bytes) });
      else doc.files.push({ p: path, t: 'text', d: file.text });
    });

    var raw = new TextEncoder().encode(JSON.stringify(doc));
    return deflate(raw).then(base64url);
  }

  function loadFromHash() {
    var hash = location.hash || '';
    if (hash.slice(0, 5) !== '#doc=') return Promise.resolve(false);
    if (!shareSupported()) {
      note('This browser cannot read shared links: it has no DecompressionStream.');
      return Promise.resolve(false);
    }

    return inflate(unbase64url(hash.slice(5))).then(function (raw) {
      var doc = JSON.parse(new TextDecoder().decode(raw));
      if (!doc || !doc.files) throw new Error('not a playground document');

      vfs.clear();
      doc.files.forEach(function (f) {
        if (f.t === 'b64') putBytes(f.p, unbase64(f.d));
        else putText(f.p, f.d);
      });

      entryPath = doc.entry || (paths().filter(isSource)[0] || '');
      openPath = doc.open || entryPath;
      ui.examples.value = '';

      syncEditor();
      renderTabs();
      renderAttachments();
      compile();
      return true;
    }).catch(function (e) {
      note('That shared link could not be read (' + message(e) + '); loading an example instead.');
      return false;
    });
  }

  function clearHash() {
    if (location.hash) history.replaceState(null, '', location.pathname + location.search);
  }

  function deflate(bytes) {
    return through(bytes, new window.CompressionStream('deflate-raw'));
  }

  function inflate(bytes) {
    return through(bytes, new window.DecompressionStream('deflate-raw'));
  }

  function through(bytes, stream) {
    // Response().arrayBuffer() is the shortest way to drain a ReadableStream
    // without hand-rolling a reader loop.
    return new Response(new Blob([bytes]).stream().pipeThrough(stream))
      .arrayBuffer()
      .then(function (buf) { return new Uint8Array(buf); });
  }

  function base64(bytes) {
    // btoa takes a string of char codes; building it in chunks keeps a large
    // image from blowing the argument limit on String.fromCharCode.
    var chunks = [];
    for (var i = 0; i < bytes.length; i += 0x8000) {
      chunks.push(String.fromCharCode.apply(null, bytes.subarray(i, i + 0x8000)));
    }
    return btoa(chunks.join(''));
  }

  function unbase64(text) {
    var chars = atob(text);
    var bytes = new Uint8Array(chars.length);
    for (var i = 0; i < chars.length; i++) bytes[i] = chars.charCodeAt(i);
    return bytes;
  }

  function base64url(bytes) {
    return base64(bytes).replace(/\+/g, '-').replace(/\//g, '_').replace(/=+$/, '');
  }

  function unbase64url(text) {
    var padded = text.replace(/-/g, '+').replace(/_/g, '/');
    while (padded.length % 4) padded += '=';
    return unbase64(padded);
  }

  // Resolves to what went wrong, or to '' when the link is on the clipboard.
  // The link is in the address bar either way, so a refusal is a note, never a
  // failure: clipboard access needs a secure context and a recent gesture, and
  // neither is this page's to guarantee.
  function copy(text) {
    if (!navigator.clipboard || !navigator.clipboard.writeText) {
      return Promise.resolve('The link is in the address bar — copying it needs a secure context.');
    }
    return navigator.clipboard.writeText(text).then(function () { return ''; }, function () {
      return 'The link is in the address bar; the clipboard refused it.';
    });
  }

  function flash(button, text) {
    var was = button.textContent;
    button.textContent = text;
    setTimeout(function () { button.textContent = was; }, 1200);
  }

  // --- download ----------------------------------------------------------------------------

  function download() {
    var result;
    try {
      result = JSON.parse(window.cinegramRenderHTML(entryPath, filesForGo()));
    } catch (e) {
      note('The page could not be rendered: ' + message(e));
      return;
    }
    if (!result.html) {
      showDiagnostics(result.diagnostics || [{ severity: 'error', message: 'nothing to render' }]);
      return;
    }

    var url = URL.createObjectURL(new Blob([result.html], { type: 'text/html' }));
    var link = document.createElement('a');
    link.href = url;
    link.download = slug(title || entryPath) + '.html';
    document.body.appendChild(link);
    link.click();
    link.remove();
    setTimeout(function () { URL.revokeObjectURL(url); }, 10000);
  }

  function slug(text) {
    var out = String(text).toLowerCase().replace(/\.dgm$/, '').replace(/[^a-z0-9]+/g, '-').replace(/^-+|-+$/g, '');
    return out || 'cinegram';
  }

  // --- wiring ---------------------------------------------------------------------------------

  function wireUI() {
    ui.editor.addEventListener('input', function () {
      clearNotice();
      saveEditor();
      schedule();
    });

    ui.examples.addEventListener('change', function () {
      loadExample(ui.examples.value);
    });

    ui.attachAdd.addEventListener('click', function () { ui.attachInput.click(); });
    ui.attachInput.addEventListener('change', function () {
      attach(ui.attachInput.files);
      // Re-picking the same file must fire `change` again.
      ui.attachInput.value = '';
    });

    ['dragenter', 'dragover'].forEach(function (type) {
      ui.left.addEventListener(type, function (e) {
        e.preventDefault();
        ui.left.classList.add('is-dropping');
      });
    });
    ['dragleave', 'dragend'].forEach(function (type) {
      ui.left.addEventListener(type, function () { ui.left.classList.remove('is-dropping'); });
    });
    ui.left.addEventListener('drop', function (e) {
      e.preventDefault();
      ui.left.classList.remove('is-dropping');
      attach(e.dataTransfer && e.dataTransfer.files);
    });

    if (shareSupported()) {
      ui.share.addEventListener('click', share);
    } else {
      ui.share.disabled = true;
      ui.share.title = 'Sharing needs CompressionStream, which this browser does not have.';
    }

    ui.download.addEventListener('click', download);

    // A debug handle, in the spirit of window.CINEGRAM_PLAYER: enough to drive
    // the page from a console or a test without reaching into the DOM.
    window.CINEGRAM_PLAYGROUND = {
      compile: compile,
      files: filesForGo,
      entry: function () { return entryPath; },
      open: openFile,
      put: function (path, value) {
        if (typeof value === 'string') putText(path, value);
        else putBytes(path, value);
        syncEditor();
        renderTabs();
        renderAttachments();
      },
      example: loadExample,
      share: encodeDoc
    };
  }

  // --- odds and ends -----------------------------------------------------------------------------

  function message(e) {
    return e && e.message ? e.message : String(e);
  }

  function byteLength(text) {
    return new TextEncoder().encode(text || '').length;
  }

  function human(n) {
    if (n < 1024) return n + ' B';
    if (n < 1024 * 1024) return (n / 1024).toFixed(1) + ' KB';
    return (n / (1024 * 1024)).toFixed(1) + ' MB';
  }

  if (document.readyState === 'loading') {
    document.addEventListener('DOMContentLoaded', boot);
  } else {
    boot();
  }
})();
