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
    ui.main = document.getElementById('pg-main');
    ui.divider = document.getElementById('pg-divider');
    ui.host = document.getElementById('pg-player-host');
    ui.bootMsg = document.getElementById('pg-boot');
    ui.diagnostics = document.getElementById('pg-diagnostics');
    ui.examples = document.getElementById('pg-examples');
    ui.attachInput = document.getElementById('pg-attach-input');
    ui.attachAdd = document.getElementById('pg-attach-add');
    ui.attachFolder = document.getElementById('pg-attach-folder');
    ui.attachFolderInput = document.getElementById('pg-attach-folder-input');
    ui.attachList = document.getElementById('pg-attach-list');
    ui.files = document.getElementById('pg-files');
    ui.viewEditor = document.getElementById('pg-view-editor');
    ui.viewFiles = document.getElementById('pg-view-files');
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
        playing: p.playing,
        present: p.present,
        follow: p.follow,
        // The pending presenter stop: without it, a recompile in the middle of
        // a beat hands the reader a player that plays to the end of the
        // scenario the next time they press Space.
        stopAt: p.stopAt
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
      // The modes before the clock, because both of them move it: setPresenter
      // pauses unconditionally and drops any pending stop, so restoring it
      // after play() would stop the playback it was meant to preserve.
      //
      // A recompile replaces the player's DOM node, which drops fullscreen with
      // it; the new player asks for it back, and with no gesture behind the
      // request the browser refuses and the fill fallback takes over. So typing
      // mid-presentation degrades to a full-window overlay rather than snapping
      // back into the split view.
      if (was.present) p.setPresenter(true);
      if (was.follow) p.setFollow(true);

      // Seek last: setView, selectScenario and setPresenter all reset the
      // clock. The presenter stop goes back after it, since seek clears it —
      // and then play() resumes the beat exactly where it was interrupted.
      p.seek(was.time);
      p.stopAt = was.stopAt === undefined ? null : was.stopAt;
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
    renderFiles();
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
    renderFiles();
    schedule();
  }

  // attach takes either plain File objects (from an <input> or a flat drop) or
  // {file, path} pairs (from the folder walker), because only the walker knows
  // where in a dropped tree each file sat.
  function attach(fileList) {
    var files = Array.prototype.slice.call(fileList || []);
    var rejected = [];
    var jobs = [];
    var sawFolder = false;

    files.forEach(function (item) {
      var file = item.file || item;
      var path = item.path || file.webkitRelativePath || file.name;
      // A folder pick sweeps in .DS_Store and its ilk; junk nobody chose by
      // name is skipped without comment rather than reported as rejected.
      if (path.split('/').some(function (part) { return part.charAt(0) === '.'; })) return;
      if (path.indexOf('/') >= 0) sawFolder = true;
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
      renderFiles();
      if (rejected.length > 3) {
        note('Skipped ' + rejected.length + ' files that are neither .dgm nor images.');
      } else if (rejected.length) {
        note('Not attached (a .dgm or an image is expected): ' + rejected.join(', '));
      }
      // A whole tree just landed: the tree view is what makes it browsable.
      if (sawFolder) setLeftView('files');
      schedule();
    });
  }

  // --- folders --------------------------------------------------------------
  //
  // dataTransfer.files flattens a dropped directory to nothing useful; the
  // webkitGetAsEntry API is what yields the tree. It is prefixed but it is
  // also the only game in town, in every current browser, and the code falls
  // back to the flat file list where it is missing.

  function collectDropped(dt) {
    var items = dt && dt.items;
    if (!items || !items.length || typeof items[0].webkitGetAsEntry !== 'function') {
      return Promise.resolve(Array.prototype.slice.call((dt && dt.files) || []));
    }
    var jobs = [];
    for (var i = 0; i < items.length; i++) {
      var entry = items[i].webkitGetAsEntry();
      if (entry) {
        jobs.push(walkEntry(entry));
      } else if (typeof items[i].getAsFile === 'function') {
        // A null entry is an item with no filesystem behind it (a synthetic
        // DataTransfer, some text drags); the plain File is still there.
        var file = items[i].getAsFile();
        if (file) jobs.push(Promise.resolve([file]));
      }
    }
    return Promise.all(jobs).then(function (lists) {
      return Array.prototype.concat.apply([], lists);
    });
  }

  function walkEntry(entry) {
    if (entry.isFile) {
      return new Promise(function (resolve) {
        entry.file(function (file) {
          // fullPath is rooted at the drop ("/frames/x.svg"); the vfs wants
          // it relative, matching webkitRelativePath from the folder input.
          resolve([{ file: file, path: entry.fullPath.replace(/^\//, '') }]);
        }, function () { resolve([]); });
      });
    }
    if (!entry.isDirectory) return Promise.resolve([]);
    if (entry.name.charAt(0) === '.') return Promise.resolve([]); // a .git is nobody's diagram

    var reader = entry.createReader();
    return new Promise(function (resolve) {
      var children = [];
      (function drain() {
        // readEntries hands out at most ~100 entries per call and signals the
        // end with an empty batch; a single call silently truncates big dirs.
        reader.readEntries(function (batch) {
          if (batch.length) {
            children = children.concat(Array.prototype.slice.call(batch));
            drain();
            return;
          }
          Promise.all(children.map(walkEntry)).then(function (lists) {
            resolve(Array.prototype.concat.apply([], lists));
          });
        }, function () { resolve([]); });
      })();
    });
  }

  // --- the Files view --------------------------------------------------------
  //
  // The left pane shows one of two things: the editor (tabs + textarea) or the
  // file tree. The tree is the site experience for a local folder — entries in
  // numeric-prefix order, folders collapsible, a click compiles and plays —
  // without a server and without anything leaving the tab.

  var leftView = 'editor';

  function setLeftView(view) {
    leftView = view;
    ui.left.classList.toggle('is-files', view === 'files');
    ui.viewEditor.classList.toggle('is-on', view === 'editor');
    ui.viewFiles.classList.toggle('is-on', view === 'files');
    if (view === 'files') renderFiles();
  }

  // The same ordering the site generator uses: an optional numeric filename
  // prefix first (01-intro before 02-deploy before anything unprefixed), then
  // the name, folders and files interleaved.
  var ORDER_PREFIX = /^(\d+)[-_. ]\s*/;

  function orderKey(base) {
    var m = ORDER_PREFIX.exec(base);
    if (m) return { prefixed: true, num: parseInt(m[1], 10), name: base.slice(m[0].length) };
    return { prefixed: false, num: 0, name: base };
  }

  function keyLess(a, b) {
    if (a.prefixed !== b.prefixed) return a.prefixed ? -1 : 1;
    if (a.prefixed && a.num !== b.num) return a.num - b.num;
    return a.name < b.name ? -1 : a.name > b.name ? 1 : 0;
  }

  function buildFileTree() {
    var root = { dirs: {}, files: [] };
    paths().forEach(function (path) {
      var parts = path.split('/');
      var node = root;
      for (var i = 0; i < parts.length - 1; i++) {
        node = node.dirs[parts[i]] || (node.dirs[parts[i]] = { dirs: {}, files: [] });
      }
      node.files.push({ name: parts[parts.length - 1], path: path });
    });
    return root;
  }

  function renderFiles() {
    ui.files.textContent = '';
    if (vfs.size === 0) {
      var empty = document.createElement('p');
      empty.className = 'pg-files-hint';
      empty.textContent = 'Nothing here yet — Add folder… below, or drop one anywhere on this panel.';
      ui.files.appendChild(empty);
      return;
    }
    ui.files.appendChild(treeList(buildFileTree()));
  }

  function treeList(node) {
    var list = document.createElement('ul');

    var entries = [];
    Object.keys(node.dirs).forEach(function (name) {
      entries.push({ key: orderKey(name), name: name, dir: node.dirs[name] });
    });
    node.files.forEach(function (f) {
      entries.push({ key: orderKey(f.name), file: f });
    });
    entries.sort(function (a, b) { return keyLess(a.key, b.key); });

    entries.forEach(function (e) {
      var item = document.createElement('li');
      if (e.dir) {
        var details = document.createElement('details');
        details.open = true;
        var summary = document.createElement('summary');
        summary.textContent = e.name + '/';
        details.appendChild(summary);
        details.appendChild(treeList(e.dir));
        item.appendChild(details);
      } else {
        item.appendChild(fileRow(e.file));
      }
      list.appendChild(item);
    });
    return list;
  }

  function fileRow(f) {
    var file = vfs.get(f.path);
    var row = document.createElement('div');
    row.className = 'pg-file'
      + (isSource(f.path) ? ' is-dgm' : '')
      + (f.path === openPath ? ' is-open' : '');

    var name = document.createElement('button');
    name.type = 'button';
    name.className = 'pg-file-name';
    name.textContent = f.name;
    if (isSource(f.path)) {
      name.title = 'Play this document';
      name.addEventListener('click', function () { playFile(f.path); });
    } else {
      name.disabled = true;
    }
    row.appendChild(name);

    if (f.path === entryPath) {
      var mark = document.createElement('span');
      mark.className = 'pg-file-entry';
      mark.textContent = 'entry';
      row.appendChild(mark);
    }

    var size = document.createElement('span');
    size.className = 'pg-file-size';
    size.textContent = human(file.bytes ? file.bytes.length : byteLength(file.text));
    row.appendChild(size);

    return row;
  }

  // playFile makes the clicked document the one being played: entry and open
  // both move, and the player starts over — clicking a file in the tree is
  // opening a new document, exactly like picking an example, not an edit to
  // the current one. Toggling back to Editor then edits that same file.
  function playFile(path) {
    if (path === entryPath) {
      openFile(path);
      return;
    }
    saveEditor();
    entryPath = path;
    openPath = path;

    stop(player);
    player = null;
    compiled = false;

    clearNotice();
    syncEditor();
    renderTabs();
    renderAttachments();
    renderFiles();
    compile();
  }

  // --- examples -------------------------------------------------------------------------

  // The picker's first entry is not an example: it resets the working set to a
  // small skeleton. A truly empty editor's first feedback would be a parse
  // error, so "fresh" means a two-node diagram and a scenario to grow from.
  var NEW_DOC_ID = '__new__';
  var STARTER = [
    '%% A fresh cinegram document: a Mermaid diagram, then a scenario that',
    '%% animates it. Edit anything — the page recompiles as you type.',
    'flowchart LR',
    '  client[Client]',
    '  server[Server]',
    '  db[(Database)]',
    '',
    '  client --> server',
    '  server --> db',
    '',
    'scenario "a request" { speed: 1.0 }',
    '',
    '  step ask "The client asks" {',
    '    desc: "flow sends a pulse along an edge; highlight holds a node."',
    '    flow client -> server { label: "GET /thing", dur: 600ms }',
    '    highlight server',
    '  }',
    '',
    '  step fetch "The server reads" {',
    '    flow server -> db { dur: 500ms }',
    '    flow db -> server { dur: 500ms, delay: 600ms }',
    '  }',
    '',
    '  step answer "And answers" {',
    '    flow server -> client { label: "200 OK", dur: 600ms }',
    '  }',
    ''
  ].join('\n');

  function loadExamplesManifest() {
    return fetch('examples.json').then(function (r) {
      if (!r.ok) throw new Error('examples.json: HTTP ' + r.status);
      return r.json();
    }).then(function (list) {
      examples = list;
      ui.examples.textContent = '';
      var fresh = document.createElement('option');
      fresh.value = NEW_DOC_ID;
      fresh.textContent = 'New document';
      ui.examples.appendChild(fresh);
      for (var i = 0; i < list.length; i++) {
        var option = document.createElement('option');
        option.value = list[i].id;
        option.textContent = list[i].title;
        ui.examples.appendChild(option);
      }
    });
  }

  // adoptDocument replaces the whole working set: vfs, entry, player. A new
  // document is a new reading, so let it play from the top: dropping the
  // player is what makes `autoplay: !was` true again.
  function adoptDocument(files, entry) {
    vfs.clear();
    files.forEach(function (f) {
      if (f.bytes) putBytes(f.path, f.bytes);
      else putText(f.path, f.text);
    });

    entryPath = entry;
    openPath = entry;

    stop(player);
    player = null;
    compiled = false;

    clearHash();
    clearNotice();
    syncEditor();
    renderTabs();
    renderAttachments();
    renderFiles();
    compile();
  }

  function newDocument() {
    adoptDocument([{ path: 'untitled.dgm', text: STARTER }], 'untitled.dgm');
    ui.examples.value = NEW_DOC_ID;
  }

  function loadExample(id) {
    if (id === NEW_DOC_ID) {
      newDocument();
      return Promise.resolve();
    }

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
      adoptDocument(loaded, example.entry);
      ui.examples.value = id;
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
      renderFiles();
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

    // The folder input's files carry webkitRelativePath, so plain attach()
    // lands them at their tree positions.
    ui.attachFolder.addEventListener('click', function () { ui.attachFolderInput.click(); });
    ui.attachFolderInput.addEventListener('change', function () {
      attach(ui.attachFolderInput.files);
      ui.attachFolderInput.value = '';
    });

    ui.viewEditor.addEventListener('click', function () { setLeftView('editor'); });
    ui.viewFiles.addEventListener('click', function () { setLeftView('files'); });

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
      collectDropped(e.dataTransfer).then(attach);
    });

    if (shareSupported()) {
      ui.share.addEventListener('click', share);
    } else {
      ui.share.disabled = true;
      ui.share.title = 'Sharing needs CompressionStream, which this browser does not have.';
    }

    ui.download.addEventListener('click', download);

    wireDivider();

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
        renderFiles();
      },
      example: loadExample,
      share: encodeDoc,
      view: setLeftView,
      play: playFile
    };
  }

  // --- split divider ----------------------------------------------------------

  // The split is the --pg-left CSS variable; dragging rewrites it on .pg-main
  // as a percentage and localStorage carries it across visits. Double-click
  // collapses the editor entirely (the divider stays, so double-click also
  // brings it back). Below the stacked-layout breakpoint the divider is
  // display:none and none of this runs.
  var SPLIT_KEY = 'cinegram.playground.split';
  var SPLIT_MIN = 15;
  var SPLIT_MAX = 85;

  function wireDivider() {
    // The last non-collapsed width, so expanding restores where you were.
    var restore = 42;
    var dragging = false;

    function clampPct(pct) {
      return Math.min(SPLIT_MAX, Math.max(SPLIT_MIN, pct));
    }

    function apply(pct) {
      ui.main.style.setProperty('--pg-left', pct + '%');
    }

    function persist(value) {
      try { localStorage.setItem(SPLIT_KEY, value); } catch (e) { /* private browsing */ }
    }

    function collapsed() {
      return ui.main.classList.contains('is-collapsed');
    }

    function setCollapsed(want) {
      ui.main.classList.toggle('is-collapsed', want);
      // The collapsed value keeps the width, so expanding after a reload
      // still lands where the split was dragged to.
      if (want) persist('collapsed:' + restore);
      else { apply(restore); persist(String(restore)); }
    }

    var saved = null;
    try { saved = localStorage.getItem(SPLIT_KEY); } catch (e) { /* private browsing */ }
    if (saved !== null) {
      var wasCollapsed = saved.indexOf('collapsed') === 0;
      var pct = parseFloat(wasCollapsed ? saved.slice('collapsed:'.length) : saved);
      if (isFinite(pct)) {
        restore = clampPct(pct);
        if (!wasCollapsed) apply(restore);
      }
      if (wasCollapsed) ui.main.classList.add('is-collapsed');
    }

    ui.divider.addEventListener('pointerdown', function (e) {
      if (e.button !== 0) return;
      dragging = true;
      ui.divider.setPointerCapture(e.pointerId);
      ui.divider.classList.add('is-dragging');
    });

    ui.divider.addEventListener('pointermove', function (e) {
      if (!dragging) return;
      var rect = ui.main.getBoundingClientRect();
      if (rect.width === 0) return;
      // Dragging out of a collapse is expanding, not a special mode.
      if (collapsed()) ui.main.classList.remove('is-collapsed');
      restore = clampPct(((e.clientX - rect.left) / rect.width) * 100);
      apply(restore);
    });

    ['pointerup', 'pointercancel'].forEach(function (type) {
      ui.divider.addEventListener(type, function () {
        if (!dragging) return;
        dragging = false;
        ui.divider.classList.remove('is-dragging');
        persist(collapsed() ? 'collapsed:' + restore : String(restore));
      });
    });

    ui.divider.addEventListener('dblclick', function () {
      setCollapsed(!collapsed());
    });

    // The separator is focusable, so the split works without a pointer:
    // arrows nudge it, Enter/Space toggles the collapse.
    ui.divider.addEventListener('keydown', function (e) {
      if (e.key === 'ArrowLeft' || e.key === 'ArrowRight') {
        if (collapsed()) ui.main.classList.remove('is-collapsed');
        restore = clampPct(restore + (e.key === 'ArrowLeft' ? -2 : 2));
        apply(restore);
        persist(String(restore));
        e.preventDefault();
      } else if (e.key === 'Enter' || e.key === ' ') {
        setCollapsed(!collapsed());
        e.preventDefault();
      }
    });
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
