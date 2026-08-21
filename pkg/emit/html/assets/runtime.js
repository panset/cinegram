/*
 * Cinegram animation runtime.
 *
 * Deliberately a classic script rather than an ES module: pages are opened
 * straight off the filesystem, where `<script type="module">` is blocked by
 * CORS, and VS Code webviews are happier with classic scripts too.
 *
 * The runtime owns no timing logic. Every track in the timeline carries
 * absolute start/end milliseconds, so playback is just a clock plus a lookup of
 * which tracks are open at time t. That keeps the Go compiler as the single
 * source of truth for semantics.
 */
(function () {
  'use strict';

  var SVG_NS = 'http://www.w3.org/2000/svg';

  var EMPTY_VIEW = { id: '', nodes: [], groups: [], edges: [], scenarios: [], bindings: [], hidden: [] };

  // ---------------------------------------------------------------------
  // SVG indexing
  //
  // Nodes are found by id: mermaid builds them as `<prefix>-<id>-<counter>`,
  // where the prefix names the renderer — `flowchart-` for a flowchart,
  // `state-` for a state diagram. Edges are matched geometrically instead, by
  // comparing each path's endpoints against node centres. That avoids
  // depending on mermaid's edge-id format, which has changed between releases,
  // and as a bonus tells us whether a path happens to be drawn against the
  // direction we declared.
  // ---------------------------------------------------------------------

  var FLOWCHART_NODE_ID = /^flowchart-(.+)-\d+$/;
  var STATE_NODE_ID = /^state-(.+)-\d+$/;

  // indexNodesBy maps declared ids onto their g.node, for whichever id format
  // the renderer that drew this diagram uses.
  //
  // The capture is greedy on purpose: a state called `retry-3` renders as
  // `state-retry-3-7`, and only the last `-<digits>` is mermaid's counter.
  function indexNodesBy(svg, re) {
    var map = {};
    var groups = svg.querySelectorAll('g.node');
    for (var i = 0; i < groups.length; i++) {
      var g = groups[i];
      var m = re.exec(g.id || '');
      if (m) {
        map[m[1]] = g;
      }
    }
    return map;
  }

  function indexNodes(svg) { return indexNodesBy(svg, FLOWCHART_NODE_ID); }

  function indexStateNodes(svg) { return indexNodesBy(svg, STATE_NODE_ID); }

  // withClusters is a node lookup that falls back to composites.
  //
  // A transition drawn into a composite state stops at the cluster's border,
  // not at any node, so the geometric matcher has to be able to score a path
  // against the composite itself. Nodes win on a name collision: an edge to a
  // real node is always about that node.
  function withClusters(nodes, clusters) {
    var map = {};
    var id;
    for (id in clusters) map[id] = clusters[id];
    for (id in nodes) map[id] = nodes[id];
    return map;
  }

  function indexClusters(svg, view) {
    var map = {};
    // A flowchart subgraph is `g.cluster`; a state composite is
    // `g.statediagram-cluster` and carries no counter suffix on its id. The
    // two never appear in the same SVG, so one selector covers both.
    var els = svg.querySelectorAll('g.cluster, g.statediagram-cluster');
    for (var i = 0; i < els.length; i++) {
      var el = els[i];
      var id = (el.id || '').replace(/-\d+$/, '');
      map[id] = el;
    }
    // Clusters are keyed by the subgraph id in every version we support, but
    // fall back to declaration order if that ever stops being true.
    var missing = view.groups.filter(function (g) { return !map[g.id]; });
    if (missing.length === els.length && els.length === view.groups.length) {
      for (var j = 0; j < view.groups.length; j++) {
        map[view.groups[j].id] = els[j];
      }
    }
    return map;
  }

  function clientPoint(path, pt) {
    var m = path.getScreenCTM();
    if (!m) return { x: pt.x, y: pt.y };
    return { x: pt.x * m.a + pt.y * m.c + m.e, y: pt.x * m.b + pt.y * m.d + m.f };
  }

  function centreOf(el) {
    var r = el.getBoundingClientRect();
    return { x: r.left + r.width / 2, y: r.top + r.height / 2 };
  }

  function dist(a, b) {
    var dx = a.x - b.x, dy = a.y - b.y;
    return Math.sqrt(dx * dx + dy * dy);
  }

  // relativeMatrix maps points from `from`'s user space into `into`'s.
  function relativeMatrix(into, from) {
    try {
      return into.getScreenCTM().inverse().multiply(from.getScreenCTM());
    } catch (e) {
      return null;
    }
  }

  // REVERSE_COST is how much worse a path read backwards has to look before it
  // is preferred to one read forwards.
  //
  // Reading a path backwards has to stay possible — mermaid does sometimes draw
  // an edge from the far end, which is the whole reason the match is geometric.
  // But when two arrows run between the same pair in *opposite* directions, both
  // fit both edges almost equally well, and scoring them symmetrically lets each
  // declared edge take the other one's arrow. Making the backwards reading pay
  // for itself settles that in favour of the arrow actually drawn for the edge,
  // while leaving the genuine far-end case — where nothing else comes close —
  // matching as before.
  var REVERSE_COST = 1.5;

  function indexEdges(svg, view, nodes) {
    var paths = Array.prototype.slice.call(
      svg.querySelectorAll('.edgePaths path, path.flowchart-link')
    );
    // De-duplicate: a path can match both selectors.
    paths = paths.filter(function (p, i) { return paths.indexOf(p) === i; });

    var ends = [];
    for (var i = 0; i < paths.length; i++) {
      var p = paths[i];
      var len = 0;
      try { len = p.getTotalLength(); } catch (e) { len = 0; }
      if (!len) { ends.push(null); continue; }
      ends.push({
        start: clientPoint(p, p.getPointAtLength(0)),
        end: clientPoint(p, p.getPointAtLength(len))
      });
    }

    var map = {};
    var used = {};
    for (var e = 0; e < view.edges.length; e++) {
      var edge = view.edges[e];
      var from = nodes[edge.from], to = nodes[edge.to];
      if (!from || !to) continue;
      var a = centreOf(from), b = centreOf(to);

      var best = -1, bestScore = Infinity, bestFlip = false;
      for (var k = 0; k < paths.length; k++) {
        if (used[k] || !ends[k]) continue;
        var fwd = dist(ends[k].start, a) + dist(ends[k].end, b);
        var rev = REVERSE_COST * (dist(ends[k].start, b) + dist(ends[k].end, a));
        var score = Math.min(fwd, rev);
        if (score < bestScore) {
          bestScore = score;
          best = k;
          bestFlip = rev < fwd;
        }
      }
      if (best >= 0) {
        used[best] = true;
        map[edge.id] = { path: paths[best], flip: bestFlip };
      }
    }
    return map;
  }

  // ---------------------------------------------------------------------
  // Sequence diagrams
  //
  // Nothing here looks like a flowchart. There is no g.node and no .edgePaths:
  // an actor is a pair of loose rects and a pair of loose texts that happen to
  // share a column, and a message is a loose <line> (or a <path>, when an actor
  // messages itself) sitting at the top level of the SVG.
  //
  // Actors are found by column, because the column is the one thing that
  // actually identifies an actor in the output. Messages are matched by order,
  // because mermaid emits them top to bottom in message order and that is far
  // more robust than trying to recover identity from geometry.
  // ---------------------------------------------------------------------

  // COLUMN_SLOP is how far from a lifeline a rect or label may sit and still
  // belong to it. Half an actor box is the natural bound.
  var COLUMN_SLOP = 90;

  function indexActors(svg, view, anchors) {
    var columns = actorColumns(svg);
    if (!columns.length) return {};

    // The SVG knows actors by their display label; the timeline knows them by
    // id. `participant C as Client` is exactly the case that makes these
    // different, so translate rather than assuming they match.
    var byLabel = {};
    view.nodes.forEach(function (n) {
      if (!byLabel[n.label]) byLabel[n.label] = n.id;
      if (!byLabel[n.id]) byLabel[n.id] = n.id;
    });

    var map = {};
    var claimed = {};
    for (var i = 0; i < columns.length; i++) {
      var col = columns[i];
      var id = byLabel[col.label];
      // Order is the fallback when a label was not matched, or when the label
      // points at an id another column already took — `participant A as B`
      // followed by `participant B` puts the display text "B" on two columns,
      // and without the claim check the second would steal the first's id.
      if ((!id || claimed[id]) && view.nodes[i]) id = view.nodes[i].id;
      if (!id || claimed[id]) continue;
      claimed[id] = true;

      map[id] = wrapActor(svg, col);
      // Notes and pills anchor to the top box rather than the wrapper, whose
      // rect spans the full height of the lifeline.
      anchors[id] = col.top || map[id];
    }
    return map;
  }

  // actorColumns groups the loose parts of the diagram by the lifeline they
  // belong to.
  function actorColumns(svg) {
    var columns = [];
    var lifelines = svg.querySelectorAll('line.actor-line');
    for (var i = 0; i < lifelines.length; i++) {
      columns.push({
        x: parseFloat(lifelines[i].getAttribute('x1')),
        parts: [lifelines[i]],
        top: null,
        label: ''
      });
    }
    if (!columns.length) return columns;
    columns.sort(function (a, b) { return a.x - b.x; });

    function nearest(x) {
      var best = null, bestGap = COLUMN_SLOP;
      for (var k = 0; k < columns.length; k++) {
        var gap = Math.abs(columns[k].x - x);
        if (gap <= bestGap) { bestGap = gap; best = columns[k]; }
      }
      return best;
    }

    var rects = svg.querySelectorAll('rect.actor, rect.actor-top, rect.actor-bottom');
    for (var r = 0; r < rects.length; r++) {
      var rect = rects[r];
      var cx = parseFloat(rect.getAttribute('x')) + parseFloat(rect.getAttribute('width')) / 2;
      var col = nearest(cx);
      if (!col) continue;
      col.parts.push(rect);
      // The top box is the one nearer the start of the diagram.
      var y = parseFloat(rect.getAttribute('y'));
      if (!col.top || y < parseFloat(col.top.getAttribute('y'))) col.top = rect;
    }

    var texts = svg.querySelectorAll('text.actor-box, text.actor');
    for (var t = 0; t < texts.length; t++) {
      var text = texts[t];
      var tcol = nearest(parseFloat(text.getAttribute('x')));
      if (!tcol) continue;
      tcol.parts.push(text);
      if (!tcol.label) tcol.label = (text.textContent || '').trim();
    }

    return columns;
  }

  // wrapActor gathers a column's parts into one group.
  //
  // This is what makes an actor animate like a node without a second set of
  // CSS: every existing rule is written as `.dgm-highlight rect`, so giving the
  // parts a common parent to put the class on is the whole adaptation.
  function wrapActor(svg, col) {
    var g = document.createElementNS(SVG_NS, 'g');
    g.setAttribute('class', 'dgm-actor');

    var first = col.parts[0];
    if (first && first.parentNode) {
      first.parentNode.insertBefore(g, first);
    } else {
      svg.appendChild(g);
    }
    for (var i = 0; i < col.parts.length; i++) {
      g.appendChild(col.parts[i]);
    }
    return g;
  }

  // indexMessages pairs each compiled edge with the line mermaid drew for it.
  //
  // Strictly by order: the compiler emits one edge per message occurrence in
  // source order, and mermaid draws them top to bottom in the same order.
  // Geometry is used only to decide which end the line starts at.
  function indexMessages(svg, view, anchors) {
    var lines = svg.querySelectorAll(
      'line.messageLine0, line.messageLine1, path.messageLine0, path.messageLine1');

    var map = {};
    for (var i = 0; i < view.edges.length && i < lines.length; i++) {
      var edge = view.edges[i];
      var el = lines[i];

      var ends = null;
      try {
        var len = el.getTotalLength();
        ends = { start: el.getPointAtLength(0), end: el.getPointAtLength(len) };
      } catch (e) { ends = null; }

      map[edge.id] = {
        path: el,
        flip: ends ? startsAtTheFarEnd(ends, edge, anchors) : false,
        matrix: null
      };
    }
    return map;
  }

  // startsAtTheFarEnd reports whether mermaid drew this line from the message's
  // destination rather than its source, which composes with a flow's own
  // `reverse` exactly as it does for a flowchart path.
  function startsAtTheFarEnd(ends, edge, anchors) {
    var from = centreX(anchors[edge.from]);
    var to = centreX(anchors[edge.to]);
    if (from === null || to === null || from === to) return false;
    return Math.abs(ends.start.x - to) < Math.abs(ends.start.x - from);
  }

  function centreX(el) {
    if (!el) return null;
    var x = parseFloat(el.getAttribute('x'));
    var w = parseFloat(el.getAttribute('width'));
    if (isNaN(x)) return null;
    return isNaN(w) ? x : x + w / 2;
  }

  // ---------------------------------------------------------------------
  // Player
  // ---------------------------------------------------------------------

  // opts says how the player is hosted, for the cases a page cannot express in
  // its URL. The standalone page passes nothing and every default below is what
  // it has always done; a host that puts several players in one document — the
  // VS Code Markdown preview puts one per ```dgm block — turns off the things
  // that are singular per page:
  //
  //   inline    strip the page chrome down to the stage, transport and caption
  //   keys      'document' (default) or 'scoped', which listens on this.root
  //             only and acts only while focus is inside it
  //   hash      false to leave location.hash alone entirely, navigating and
  //             going back within the player instead
  //   theme     'light' | 'dark' to follow the host instead of the system
  //   autoplay  false to open at rest
  //   reel      true for the vertical story mode the standalone page enters
  //             with `?reel`: tap plays one step, a segmented bar replaces
  //             the scrubber, and the page opens at rest
  //
  // Anything a host does not set keeps the page behaviour, so this stays one
  // renderer rather than two that have to be kept in step.
  function Player(root, timeline, opts) {
    this.root = root;
    this.timeline = timeline;
    this.opts = opts || {};
    this.viewIndex = 0;
    this.scenarioIndex = 0;
    this.time = 0;
    this.playing = false;
    // Overwritten from the selected scenario's compiled speed before the first
    // frame; the button then cycles absolute values from wherever that landed.
    this.speed = 1;
    // Set when a view is entered and cleared by the render that consumes it, so
    // autoplay fires once per view rather than on every re-render (a theme
    // toggle re-renders too, and must not restart the animation).
    //
    // A diagram embedded in a document opens at rest: several of them starting
    // to move the moment the page renders is noise, not information.
    this.pendingAutoplay = this.opts.autoplay !== false;
    this.raf = null;
    this.lastFrame = 0;
    this.nodeState = {};
    this.edgeState = {};
    this.particles = {};
    this.notes = {};
    this.pills = {};
    // The trail of view ids drilled through, so Back knows where to return.
    this.stack = [];
    // Elements a reveal binding has opened in the current view. Unlike a
    // track this is not owned by the clock: it persists across seeks and is
    // cleared only when the view changes.
    this.revealed = {};

    // `<html data-theme>` is the page's answer, not this player's: page chrome
    // writes it, an editor writes it, a site's palette toggle writes it, and
    // every player on the document reads the same one. pageTheme is what it
    // said at mount, or null for a page that has not answered.
    //
    // A host that has its own theme states it instead, and then it is not a
    // system setting to follow: an editor's light/dark is already the reader's
    // answered question.
    this.pageTheme = document.documentElement.getAttribute('data-theme');
    if (this.pageTheme !== 'dark' && this.pageTheme !== 'light') this.pageTheme = null;
    this.hostTheme = this.opts.theme === 'dark' || this.opts.theme === 'light'
      ? this.opts.theme
      : null;
    // Whether the page carries a theme control of its own — the marker the
    // chrome emits, upgraded at the bottom of this file. The attribute alone
    // cannot answer that question, because "follow the system" *is* the
    // attribute's absence: a cinegram page whose reader chose to follow the
    // system looks exactly like a document that has never heard of cinegram.
    // So it is the control that says who writes, and a player that finds one
    // never writes the attribute itself.
    this.pageChrome = !!document.querySelector('[data-dgm-theme-toggle]');
    // mermaid needs a concrete palette per render, so an unanswered page still
    // resolves to one — through the same media query the stylesheet uses.
    this.theme = this.pageTheme || this.hostTheme || (systemDark() ? 'dark' : 'light');

    // Stage transform. Reset per view: a zoom that made sense for one diagram
    // is meaningless over the next.
    this.zoom = 1;
    this.panX = 0;
    this.panY = 0;

    // Presenter mode plays exactly one step at a time. stopAt is the moment the
    // clock should stop at, in timeline milliseconds — the same domain as
    // this.time, so the speed multiplier composes with it for free.
    this.present = false;
    this.stopAt = null;

    // Armed by a drag's release, read once by the click-swallower in build().
    this.swallowNext = false;

    // Auto-follow camera state, live wherever `follow` is. camOverride is
    // user-state like `revealed`, not clock-state: a manual gesture takes the
    // wheel, a double-click hands it back, and a seek changes neither. camKeys
    // caches the per-step pose keyframes; null means "rebuild from the DOM".
    // `follow` means the camera follows, full stop: true from the start in a
    // reel (set in build(), where reel is first known, and clamped true by
    // setFollow), toggled by the Cine button everywhere else.
    this.follow = false;
    this.camOverride = false;
    this.camKeys = null;
    this._camMoved = false;

    // The minimap's own cache, invalidated in exactly the places camKeys is:
    // it holds the layout half of the stage↔diagram mapping, which a resize, a
    // render or a mode change moves and a pan or a zoom does not. See mapGeom.
    this.mapKeys = null;

    // Removers for every listener this player puts somewhere that outlives its
    // own DOM — window, document, a media query. See `own` and dispose: a host
    // that mounts a fresh player per edit would otherwise gain one more set of
    // them with every mount, each holding the player it belonged to alive.
    this._unbind = [];

    this.build();
  }

  // own binds a listener the player has to take back off itself, and records
  // how. Listeners on elements inside the root need none of this: they die with
  // the nodes when the host clears or replaces the root.
  function own(player, target, type, fn, opts) {
    target.addEventListener(type, fn, opts);
    player._unbind.push(function () { target.removeEventListener(type, fn, opts); });
  }

  // drag(el, opts) owns the bookkeeping every pointer gesture repeats: the
  // primary-button guard, pointer capture (with the try/catch some browsers
  // need), the down/move/up lifecycle, and cancel.
  //
  // opts: {start, move, end, cancel} — each optional, each called with
  // (ev, state). state is fresh per gesture and carries startX/startY and
  // dx/dy, the total travel since the press; anything else a gesture needs to
  // remember between callbacks it writes onto state itself (the stage keeps a
  // moved flag there, the board its horizontal-intent verdict). start may
  // return false to decline the gesture, so a declined press stays an
  // ordinary click for whoever else wants it.
  //
  // state.claim(ev) takes the pointer for the gesture: after it, moves and the
  // release arrive here wherever the pointer wanders. Sites call it the moment
  // a press stops being a possible click — at start for surfaces whose every
  // press is a gesture (the map jumps, the lightbox pans), at the intent
  // threshold for surfaces that are also click targets (the stage's nodes, the
  // board's tap-for-lightbox). The timing matters: capture retargets the
  // pointerup and with it the click the release manufactures, so claiming a
  // press that stays a tap would send its click to this element instead of the
  // node under the finger.
  function drag(el, opts) {
    var state = null;
    function claim(ev) {
      try { if (el.setPointerCapture) el.setPointerCapture(ev.pointerId); } catch (e) { /* ignore */ }
    }
    el.addEventListener('pointerdown', function (ev) {
      if (ev.button !== 0) return;
      var st = { startX: ev.clientX, startY: ev.clientY, dx: 0, dy: 0, claim: claim };
      if (opts.start && opts.start(ev, st) === false) return;
      state = st;
    });
    el.addEventListener('pointermove', function (ev) {
      if (!state) return;
      // An unclaimed press whose release happened off the element never
      // delivers its pointerup here. The button being up says it ended.
      if (!ev.buttons) {
        var lost = state;
        state = null;
        if (opts.cancel) opts.cancel(ev, lost);
        return;
      }
      state.dx = ev.clientX - state.startX;
      state.dy = ev.clientY - state.startY;
      if (opts.move) opts.move(ev, state);
    });
    el.addEventListener('pointerup', function (ev) {
      if (!state) return;
      var st = state;
      state = null;
      st.dx = ev.clientX - st.startX;
      st.dy = ev.clientY - st.startY;
      if (opts.end) opts.end(ev, st);
    });
    el.addEventListener('pointercancel', function (ev) {
      if (!state) return;
      var st = state;
      state = null;
      if (opts.cancel) opts.cancel(ev, st);
    });
  }

  // --- preferences ------------------------------------------------------
  //
  // localStorage throws outright in a sandboxed iframe and in some private
  // modes, so every access is guarded: a page that cannot remember a
  // preference should still play.

  function prefGet(key) {
    try { return localStorage.getItem(key); } catch (e) { return null; }
  }

  function prefSet(key, value) {
    try {
      if (value === null) localStorage.removeItem(key);
      else localStorage.setItem(key, value);
    } catch (e) { /* not remembering is not a failure */ }
  }

  function systemDark() {
    try { return matchMedia('(prefers-color-scheme: dark)').matches; } catch (e) { return false; }
  }

  Player.prototype.view = function () {
    return this.timeline.views[this.viewIndex] || EMPTY_VIEW;
  };

  Player.prototype.scenario = function () {
    return this.view().scenarios[this.scenarioIndex] || { steps: [], duration: 0 };
  };

  Player.prototype.viewIndexOf = function (id) {
    for (var i = 0; i < this.timeline.views.length; i++) {
      if (this.timeline.views[i].id === id) return i;
    }
    return -1;
  };

  Player.prototype.build = function () {
    var self = this;
    this.root.innerHTML = '';
    this.root.className = 'dgm';

    var bar = el('div', 'dgm-bar');

    var heading = el('div', 'dgm-heading');
    // Back sits before the title so drilling in does not shift the title
    // sideways as the trail grows.
    this.backBtn = button('← Back', 'dgm-btn dgm-back', function () { self.back(); });
    this.backBtn.style.display = 'none';
    heading.appendChild(this.backBtn);
    this.title = el('div', 'dgm-title');
    heading.appendChild(this.title);
    this.crumb = el('div', 'dgm-crumb');
    heading.appendChild(this.crumb);
    bar.appendChild(heading);

    var controls = el('div', 'dgm-controls');

    // The scenario picker is rebuilt per view, since each view brings its
    // own scenarios. It stays in the DOM and hides when there is nothing to
    // choose between.
    this.picker = el('select', 'dgm-select');
    this.picker.addEventListener('change', function () {
      self.selectScenario(parseInt(self.picker.value, 10));
      // Only here: a pick is the one scenario change the address does not
      // already know about. See syncHash.
      self.syncHash();
    });
    controls.appendChild(this.picker);

    // `dgm-authoring` marks the controls that belong to building a diagram
    // rather than showing one; presenter mode hides exactly that set.
    // dgm-play names the one control no mode can do without: inline strips the
    // bar down to it by name rather than by position, so adding a button here
    // later cannot quietly change what a document shows.
    this.playBtn = button('Play', 'dgm-btn dgm-btn-primary dgm-authoring dgm-play', function () { self.toggle(); });
    controls.appendChild(this.playBtn);

    // Presenter mode is a toggle rather than a link, so leaving it does not
    // reload and lose the moment the presenter had reached. It doubles as the
    // way out: in presenter mode almost everything else is hidden.
    //
    // Play and Present are the whole bar now — the pair a reader is offered,
    // which is also the silhouette an inline player has always shown. Every
    // other control moved to the rail; see buildRail.
    this.presentBtn = button('Present', 'dgm-btn', function () {
      self.setPresenter(!self.present);
    });
    controls.appendChild(this.presentBtn);

    bar.appendChild(controls);
    this.root.appendChild(bar);

    // The reel's segmented progress bar: one segment per step, story-style.
    // Built for every host and shown only under .dgm-reel, so the shared
    // stylesheet costs other hosts nothing. aria-hidden because the caption
    // is the live region and digits 1–9 remain the accessible step jump.
    this.reelbar = el('div', 'dgm-reelbar');
    this.reelbar.setAttribute('aria-hidden', 'true');
    this.root.appendChild(this.reelbar);

    this.warning = el('div', 'dgm-warning');
    this.warning.style.display = 'none';
    this.root.appendChild(this.warning);

    var body = el('div', 'dgm-body');
    this.stage = el('div', 'dgm-stage');
    this.overlay = el('div', 'dgm-overlay');
    this.stage.appendChild(this.overlay);

    // The minimap, built once beside the overlay and re-appended by render()
    // for the same reason: the stage is emptied there. It starts off screen
    // and stays there until something zooms past fit — see syncMap.
    this.map = el('div', 'dgm-map is-off');
    this.mapBody = el('div', 'dgm-map-body');
    this.mapRect = el('div', 'dgm-map-rect');
    // The whole box used to be aria-hidden: a thumbnail of the diagram and a
    // rectangle drawn on it are decoration, and a screen reader has no use for
    // a picture of a picture. It holds a real control now, and a focusable
    // element inside an aria-hidden subtree is the one arrangement the
    // standard calls an error — the reader can tab to it and hear nothing
    // there. So the attribute moves down onto the two decorative children and
    // the box itself becomes a named group with one button in it.
    this.mapBody.setAttribute('aria-hidden', 'true');
    this.mapRect.setAttribute('aria-hidden', 'true');
    this.map.setAttribute('role', 'group');
    this.map.setAttribute('aria-label', 'Diagram minimap');
    this.map.appendChild(this.mapBody);
    this.map.appendChild(this.mapRect);

    // Fit belongs here rather than in the rail because the map already knows
    // the answer the button gives: syncMap shows this box exactly while the
    // stage is showing less than the whole diagram, which is exactly when
    // there is anything to fit. In the rail the same button sat lit or dead
    // beside a map saying the same thing twice; here it inherits the map's
    // appearing and disappearing for free and exists only when it can act.
    this.mapFit = iconButton('fit', 'Fit the whole diagram', 'dgm-map-fit', function () {
      self.resetZoom();
    });
    this.map.appendChild(this.mapFit);
    this.stage.appendChild(this.map);

    // The tool rail, overlaying the stage's right edge. Built once here and
    // re-appended by render() for the same reason the minimap is: the stage is
    // emptied there.
    this.rail = this.buildRail();
    this.stage.appendChild(this.rail);

    body.appendChild(this.stage);
    // Every drag ends in a manufactured click, and that click belongs to the
    // drag: panning across a node must not drill into it, releasing a map
    // scrub over the stage must not advance a beat, a swiped-away thumbnail
    // must not open the lightbox. One capture-phase listener on the root
    // swallows exactly the next click after a gesture arms it — capture phase
    // so it runs before any element's own handler, the root so every gesture
    // surface (stage, map, board) is under it. armSwallow disarms on a zero
    // timer for the release that produces no click at all: a real drag-end
    // click is dispatched synchronously out of its pointerup, so it always
    // beats the timer.
    this.root.addEventListener('click', function (ev) {
      if (!self.swallowNext) return;
      self.swallowNext = false;
      ev.stopImmediatePropagation();
      ev.preventDefault();
    }, true);
    this.bindMapGestures();
    this.bindStageGestures();
    this.bindRailGestures();

    // Camera keyframes bake the stage size in, so a resize invalidates them.
    own(self, window, 'resize', function () {
      self.camKeys = null;
      self.mapKeys = null;
      // A layout change is a new context, so a dismissed thumbnail returns:
      // otherwise a rotate to landscape and back would resurrect and then
      // re-hide the panel with no gesture ever made in the new layout.
      if (self.boardDismissed) {
        self.boardDismissed = false;
        self.syncBoard();
      }
      if (self.follow && !self.camOverride && self.svg) {
        self.apply(self.time);
        self.syncChips();
      }
      // The map's box is a share of the stage and its rectangle is measured
      // against it, so a resize moves both even when nothing else changed.
      self.syncMap();
    });

    // The browser owns the way out of fullscreen — Esc, F11, the system
    // control — so the mode flag has to follow it rather than the other way
    // round, or the button would read "Exit" over a windowed page. It is also
    // why the Escape branch in onKey needs no change: in fullscreen the
    // browser eats Esc, and this puts the player back.
    //
    // Whichever way it went, the stage is a different size than the camera
    // measured against, and the change is not always accompanied by a resize.
    //
    // Only ever about *our* fullscreen, which is what hadFull records — a
    // two-event latch read in exactly this closure, which is why it is a local
    // and not a Player field. The event is document-level, so a page holding
    // two players hears every change either of them makes — and the playground
    // disposes one player and mounts the next on every keystroke, so a mount
    // can hear the exit of the player it replaced. Reading only
    // `fullscreenElement !== root` would take that as "the browser dropped me"
    // and drop presenter mode on the reader mid-sentence.
    var hadFull = false;
    var onFull = function () {
      var mine = fullscreenElement() === self.root;
      if (self.present && hadFull && !mine) {
        // The browser exits fullscreen on Esc even when onKey meant that Esc
        // for the innermost overlay — the lightbox or the help sheet. Ending
        // the whole presentation because someone closed a storyboard frame is
        // wrong, so with an overlay open the mode survives on the fill (the
        // class is already on — setPresenter never took it off); Esc pressed
        // again then leaves presenter mode properly.
        if (!(self.lightboxIsOpen() || self.helpOpen())) {
          self.setPresenter(false);
        }
      }
      hadFull = mine;
      self.camKeys = null;
      self.mapKeys = null;
      if (self.svg) {
        self.apply(self.time);
        self.syncChips();
        self.syncMap();
      }
    };
    own(self, document, 'fullscreenchange', onFull);

    // The storyboard sits between the stage and the step list: what the human
    // sees, beside what the system does. It is built once and hidden when the
    // showing scenario has no scenes, rather than being created per render —
    // the crossfade needs two layers that outlive a frame.
    body.appendChild(this.buildBoard());
    this.bindBoardGestures();

    this.steps = el('ol', 'dgm-steps');
    body.appendChild(this.steps);
    this.root.appendChild(body);

    // The caption is the "you are here" narration; the step list beside the
    // stage stays the table of contents. It is a live region, so a screen
    // reader hears the walkthrough rather than only seeing it.
    this.caption = el('div', 'dgm-caption');
    this.caption.setAttribute('role', 'status');
    this.caption.setAttribute('aria-live', 'polite');
    this.captionKey = null;
    this.root.appendChild(this.caption);

    var foot = el('div', 'dgm-foot');

    var track = el('div', 'dgm-scrub-wrap');
    this.scrub = document.createElement('input');
    this.scrub.type = 'range';
    this.scrub.min = '0';
    this.scrub.step = '10';
    this.scrub.className = 'dgm-scrub';
    this.scrub.addEventListener('input', function () {
      self.pause();
      self.seek(parseInt(self.scrub.value, 10));
    });
    track.appendChild(this.scrub);
    this.marks = el('div', 'dgm-scrub-marks');
    track.appendChild(this.marks);
    foot.appendChild(track);
    this.clock = el('div', 'dgm-clock');
    foot.appendChild(this.clock);
    this.root.appendChild(foot);

    this.help = this.buildHelp();
    this.root.appendChild(this.help);

    this.lightbox = this.buildLightbox();
    this.root.appendChild(this.lightbox);

    // The palette a document is wearing has to be legible from the document
    // itself: pkg/embedkit reads the stamp back, and runtime.css keys its
    // tokens off it. So a player that is the only thing here which knows the
    // answer says it — and a player on a page with chrome of its own says
    // nothing, because the chrome already did and the reader may have asked to
    // follow the system, which is this attribute being absent.
    if (!this.pageChrome) document.documentElement.setAttribute('data-theme', this.theme);

    // Watch the attribute rather than any particular control. Page chrome, an
    // editor's theme switch and a Material palette toggle all write the same
    // one thing, so one observer serves every host — and two players on a page
    // can no longer disagree about the palette, because neither of them owns
    // it. A removed attribute is "follow the system" and resolves like a page
    // that never had one.
    if (window.MutationObserver) {
      var watcher = new MutationObserver(function () {
        var at = document.documentElement.getAttribute('data-theme');
        self.setTheme(at === 'dark' || at === 'light' ? at : (systemDark() ? 'dark' : 'light'));
      });
      watcher.observe(document.documentElement, { attributes: true, attributeFilter: ['data-theme'] });
      // Straight onto the same list `own` keeps, so dispose() reaches it: the
      // playground disposes a player and mounts the next on every keystroke,
      // and an observer nobody disconnects holds the player it was built for
      // for as long as the document lives.
      this._unbind.push(function () { watcher.disconnect(); });
    }

    // Follow the system theme while nothing else has answered. Without this a
    // page left open across a scheduled light/dark switch keeps the old one —
    // and mermaid picks its palette per render, so following it means redrawing
    // rather than merely restyling, which is what setTheme does.
    //
    // A host that states a theme is already tracking its own, and following the
    // OS underneath it would make an editor in forced-light show a dark diagram.
    if (!this.hostTheme) {
      try {
        var mq = matchMedia('(prefers-color-scheme: dark)');
        var follow = function (ev) {
          // On a page with chrome, the attribute is the reader's explicit
          // choice and the OS does not overrule it; its absence is the reader
          // asking for exactly this. In a document with no chrome the only
          // data-theme on the page is the one this player stamped, so there is
          // nothing to ask and the system decides as it always has.
          if (self.pageChrome && document.documentElement.getAttribute('data-theme')) return;
          self.setTheme(ev.matches ? 'dark' : 'light');
        };
        if (mq.addEventListener) {
          own(self, mq, 'change', follow);
        } else if (mq.addListener) {
          // The legacy pair predates EventTarget on MediaQueryList, so `own`
          // cannot express it; the remover goes on by hand.
          mq.addListener(follow);
          self._unbind.push(function () { mq.removeListener(follow); });
        }
      } catch (e) { /* no matchMedia: the initial theme stands */ }
    }

    // One document-level handler for the whole page: the player swaps views
    // rather than being replaced, so these never stack up.
    //
    // They would stack up across *players*, though, and a document holding
    // three diagrams holds three of them — each one calling preventDefault on
    // Space, so the page could no longer scroll. A scoped player listens on its
    // own element and answers only when the reader is looking at it.
    if (this.opts.keys === 'scoped') {
      if (!this.root.hasAttribute('tabindex')) this.root.setAttribute('tabindex', '0');
      this.root.addEventListener('keydown', function (ev) { self.onKey(ev); });
    } else {
      own(self, document, 'keydown', function (ev) { self.onKey(ev); });
    }

    // The hash is the single source of truth for which view is showing, and
    // every navigation goes through it. That is what keeps the Back button
    // and the browser's own history from ever disagreeing.
    //
    // There is only one address, so only one player can own it. Embedded in a
    // document the player owns none of it: navigation happens in the player and
    // the URL is left to whatever else is on the page.
    if (this.usesHash()) {
      own(self, window, 'hashchange', function () { self.applyHash(); });
    }

    // Embed mode drops the chrome around the diagram but not the diagram's own
    // controls: an iframe in a doc page wants the stage, the narration and the
    // scrubber, and has its own heading and navigation already.
    if (isEmbedded(this.opts)) this.root.classList.add('dgm-embed');
    // Inline goes one step further back: embed hides the whole bar, and a
    // diagram in a document still needs somewhere to press Play. The class
    // brings the bar back holding only what a reader — rather than an author —
    // has any use for.
    if (this.opts.inline) this.root.classList.add('dgm-inline');
    // Reel is a page state like embed, not a toggle like presenter: it is
    // entered by URL and left by leaving the page, so it is a class and a
    // flag rather than anything setPresenter-shaped.
    this.reel = isReel(this.opts);
    if (this.reel) this.root.classList.add('dgm-reel');
    // A reel cannot work without the camera — a 9:16 frame shows nothing at
    // fit — so in a reel `follow` is simply true from the start, and
    // setFollow's clamp keeps it there. One spelling for one fact.
    this.follow = this.reel;
    this.setPresenter(isPresenter(this.opts));

    this.viewIndex = Math.max(0, this.viewIndexOf(this.hashView()));
    this.buildPicker();
    this.adoptScenarioSpeed();

    // Read before the first render so a deep link's scenario is the one that
    // gets built, rather than being swapped a frame later.
    var deep = this.readHash();
    if (deep.s) {
      var i = this.scenarioIndexOf(deep.s);
      if (i >= 0) {
        this.scenarioIndex = i;
        this.picker.value = String(i);
        this.adoptScenarioSpeed();
      }
    }
    if (deep.t !== undefined) {
      var ms = parseInt(deep.t, 10);
      if (!isNaN(ms)) {
        this.time = Math.max(0, ms);
        this.pendingAutoplay = false;
      }
    } else {
      // No deep link: rest at the scenario's poster moment, not a blank 0ms.
      this.time = this.restingTime();
    }

    this.render();
  };

  // --- the tool rail ------------------------------------------------------
  //
  // Everything that is not Play or Present, in one translucent column over the
  // stage's right edge. The bar carried ten controls and read as a control
  // panel with a diagram attached; the reader needs two of them, and the rest
  // are a keystroke or a hover away here instead.
  //
  // Three of them at rest — Cine, Copy link, help — which is few enough to
  // stand at any width. That is why there is no ⋯ collapse behind them any
  // more: a toggle, an is-open state and a media query that had to reset it
  // were a state machine standing in for a column that no longer needs one.
  //
  // What is *not* here is as deliberate:
  //
  //   - Dark and light is a property of the page, so its control is page
  //     chrome and lives at the bottom of this file, not in one diagram's
  //     column of tools.
  //   - Fit did nothing until something had zoomed, and it stood beside a
  //     minimap that appears at exactly that moment — so it moved into the
  //     map, in build(), where its own visibility is the answer.
  //   - Speed was the most expensive control here: its label was its own
  //     value, so the whole column had to be wide enough for "0.25x". It is
  //     also a preference, remembered for every cinegram this browser opens,
  //     which is not what prime real estate is for. It moved into the settings
  //     sheet, in buildHelp — where a presenter can reach it too, which
  //     dgm-authoring never let them.
  //
  // Vertically centred, because the corners are taken: the minimap parks
  // top-right and the presenter storyboard top-left.
  //
  // Nothing here is a mode. Each button keeps the classes it carried in the
  // bar, so `.dgm-present .dgm-authoring` strips exactly the set it always
  // stripped — the rail thins to Restart, Cine and help while presenting
  // rather than disappearing — and `.dgm-inline .dgm-rail` hides the whole
  // column, which is what keeps a diagram in a document showing what it showed
  // before the rail existed.
  Player.prototype.buildRail = function () {
    var self = this;
    var rail = el('div', 'dgm-rail');
    // A group, not a toolbar. `role="toolbar"` is a promise of single-tab-stop
    // arrow-key navigation, and these are four plain tab stops, three of them
    // at rest — Restart only stands in the modes with no scrub — but the
    // roving tabindex that would make the promise true has to own the arrow
    // keys, and the transport already does: onKey reads ArrowLeft and
    // ArrowRight as previous and next step, which is the more valuable binding
    // on a player. A group says what the buttons are without contracting for
    // keys something else owns.
    rail.setAttribute('role', 'group');
    rail.setAttribute('aria-label', 'Diagram tools');

    // The buttons are children of the column itself. They used to sit in a
    // `.dgm-rail-items` wrapper, so that the narrow-screen collapse could hide
    // the whole set by name without a `display` rule on the buttons — one on
    // them would have had to outrank the presenter rule, and would then have
    // put the authoring set back mid-talk. With the collapse gone the wrapper
    // wraps nothing, and a box whose only job was to be hidden is one fewer
    // element between a press and the stylesheet.

    // Restart is not authoring: taking a demo from the top is one of the most
    // ordinary things a presenter does, so it stays when presenter mode strips
    // the building tools.
    //
    // It is not much anywhere else, though — Home does it, clicking step 1 does
    // it, and dragging the scrub to zero does it while being the gesture a
    // reader reaches for first. So it carries dgm-nofoot, which shows it in
    // exactly the modes that hide `.dgm-foot` and so have no scrub to drag:
    // presenter and reel. The class says that reason rather than naming the
    // modes, because the reason is what would have to change first.
    rail.appendChild(iconButton('restart', 'Restart', 'dgm-btn dgm-nofoot', function () { self.seek(0); }));

    // Cine turns the reel's auto-follow camera on anywhere: each step framed
    // and zoomed, reel-style. Off is the default everywhere but a reel — a
    // diagram that fits the screen is best introduced whole — and the toggle
    // survives presenter mode, so it sits outside dgm-authoring.
    this.cineBtn = iconButton('cine', 'Camera follows each step', 'dgm-btn', function () {
      self.setFollow(!self.follow);
    });
    // A toggle has to say which way it is set. The is-on class draws that for
    // an eye and said nothing at all to a screen reader, which heard the same
    // "Camera follows each step" whether the camera was following or not — a
    // name that describes the button rather than its state. setFollow is the
    // one writer of the class, so it writes this beside it; here is only the
    // resting value.
    this.cineBtn.setAttribute('aria-pressed', this.follow ? 'true' : 'false');
    rail.appendChild(this.cineBtn);

    // "Look at *this* step" is most of why anyone sends a diagram to a
    // colleague, and reproducing a moment by describing it never works.
    this.shareBtn = iconButton('copy', SHARE_LABEL, 'dgm-btn dgm-authoring', function () { self.copyLink(); });
    // The confirmation lives inside the button so that hiding the button hides
    // it, and absolutely positioned so that saying "Copied" cannot resize the
    // rail under the cursor that just clicked. See flashShare.
    this.shareNote = el('span', 'dgm-rail-note');
    this.shareNote.setAttribute('aria-hidden', 'true');
    this.shareBtn.appendChild(this.shareNote);
    rail.appendChild(this.shareBtn);

    this.helpBtn = iconButton('help', 'Settings and shortcuts', 'dgm-btn', function () { self.toggleHelp(); });
    rail.appendChild(this.helpBtn);

    return rail;
  };

  // The rail is a child of the stage, which reads a press as the start of a pan
  // and — presenting, or in a reel — a click as "advance a step". Pressing a
  // tool is neither, so the rail keeps those two events to itself, and the
  // double-click that would otherwise reset the zoom behind it. Same reasoning
  // as bindMapGestures, and cheaper: nothing in the rail is a drag.
  //
  // The wheel is deliberately left alone. Scrolling over the rail zooms the
  // diagram exactly as scrolling over the stage does, which is what a control
  // floating on the stage should feel like.
  Player.prototype.bindRailGestures = function () {
    var rail = this.rail;
    ['pointerdown', 'click', 'dblclick'].forEach(function (type) {
      rail.addEventListener(type, function (ev) { ev.stopPropagation(); });
    });
  };

  // ICONS are the runtime's glyphs, as path data on a 24-unit grid, stroked in
  // currentColor so a button inherits the theme's foreground. Drawn here rather
  // than fetched or set in a font because the emitted page carries no external
  // URL at all — html_test.go enforces it — and because an icon font would be
  // a second copy of the alphabet for a handful of pictures. Every glyph in
  // the table is stroke-only, which is what lets icon() below be a loop with
  // no special cases in it.
  var ICONS = {
    // A reload arc, three quarters of a circle with its head at the top.
    restart: ['M20 12a8 8 0 1 1-8-8', 'M9.5 1.5 12 4 9.5 6.5'],
    // A video camera: body and lens.
    cine: ['M3.5 7.5h10v9h-10z', 'M13.5 12l7-3.5v7z'],
    // Two sheets, the front one over the back: copy.
    copy: ['M9.5 9.5h10v10h-10z', 'M5.5 14.5H4.5a1 1 0 0 1-1-1v-9a1 1 0 0 1 1-1h9a1 1 0 0 1 1 1v1'],
    // Four corner brackets: the whole diagram back inside its frame.
    fit: [
      'M4 9.5V6a2 2 0 0 1 2-2h3.5',
      'M14.5 4H18a2 2 0 0 1 2 2v3.5',
      'M20 14.5V18a2 2 0 0 1-2 2h-3.5',
      'M9.5 20H6a2 2 0 0 1-2-2v-3.5'
    ],
    help: ['M21 12a9 9 0 1 1-18 0 9 9 0 0 1 18 0z', 'M9.3 9.4a2.8 2.8 0 0 1 5.5.9c0 1.9-2.7 2.3-2.7 4.1', 'M12 17.6v.01'],
    // The two theme states, named for the state each *is* rather than the one
    // a press goes to: a sun for light, a moon for dark.
    'theme-light': [
      'M16.2 12a4.2 4.2 0 1 1-8.4 0 4.2 4.2 0 0 1 8.4 0',
      'M12 3v2', 'M12 19v2', 'M3 12h2', 'M19 12h2',
      'M5.6 5.6l1.5 1.5', 'M16.9 16.9l1.5 1.5',
      'M18.4 5.6l-1.5 1.5', 'M7.1 16.9l-1.5 1.5'
    ],
    'theme-dark': ['M20.5 14.6A8.6 8.6 0 0 1 9.4 3.5a8.6 8.6 0 1 0 11.1 11.1z']
  };

  function icon(name) {
    var svg = document.createElementNS(SVG_NS, 'svg');
    svg.setAttribute('viewBox', '0 0 24 24');
    svg.setAttribute('width', '16');
    svg.setAttribute('height', '16');
    // The glyph is decoration: the button's own aria-label names the action.
    // focusable=false is for the browsers that made an inline svg a tab stop.
    svg.setAttribute('aria-hidden', 'true');
    svg.setAttribute('focusable', 'false');
    svg.setAttribute('fill', 'none');
    svg.setAttribute('stroke', 'currentColor');
    svg.setAttribute('stroke-width', '1.7');
    svg.setAttribute('stroke-linecap', 'round');
    svg.setAttribute('stroke-linejoin', 'round');
    var d = ICONS[name] || [];
    for (var i = 0; i < d.length; i++) {
      var p = document.createElementNS(SVG_NS, 'path');
      p.setAttribute('d', d[i]);
      svg.appendChild(p);
    }
    return svg;
  }

  // iconButton is a button whose label is a picture. The name a bar button
  // carried as its own text has to be said twice instead: `title` for the
  // pointer, `aria-label` for everything else.
  function iconButton(glyph, label, cls, fn) {
    var b = button('', cls, fn);
    b.appendChild(icon(glyph));
    b.title = label;
    b.setAttribute('aria-label', label);
    return b;
  }

  // --- the page's theme control -------------------------------------------
  //
  // Dark and light describe the *page*, so the control belongs to the page's
  // chrome and every player on it merely listens. It is written here all the
  // same, because runtime.js is the one file all three of cinegram's own
  // surfaces already carry — the emitted page, the listings `cinegram site`
  // builds, the playground — and a control implemented three times is a
  // control that works on two of them.
  //
  // Two states: a press flips light to dark and back, and that is the whole
  // vocabulary. A page nobody has pressed carries no data-theme at all, so
  // runtime.css's prefers-color-scheme rules answer for it — a fresh reader
  // opens in whatever the system is showing, and an OS switch moves the page
  // with no script in the way. That is the state every page starts in.
  //
  // The first press ends the following. It stores a side, and from then on
  // this browser shows that side whatever the system does. The trade is
  // deliberate rather than overlooked: a reader who asked for dark meant dark,
  // including at sunrise. It does mean the glyph can outlive the system it was
  // drawn from, which is why drawing goes through the *effective* theme below
  // rather than the stored one, and why the control watches the media query
  // itself for as long as nothing is stored.

  // The one storage key, unchanged from the rail button so a reader who chose
  // dark keeps dark, and the same key the boot script in pkg/emit/html reads
  // before the first paint. Only 'light' and 'dark' are ever written to it.
  var THEME_KEY = 'dgm.theme';

  // themeChoice is the side the reader picked, or null while they have picked
  // none. Anything else in the key reads as null — a value some other page
  // wrote, or a stale one from a build that had a third state — which is
  // exactly how the boot script treats it, so the attribute and the glyph
  // agree from the very first frame.
  function themeChoice() {
    var v = prefGet(THEME_KEY);
    return v === 'light' || v === 'dark' ? v : null;
  }

  // effectiveTheme is what the reader is actually looking at: the stored side
  // if there is one, and the system's answer if there is not. It is what the
  // glyph draws and what a press flips, because "the other one" only means
  // anything against the palette on screen.
  function effectiveTheme() {
    return themeChoice() || (systemDark() ? 'dark' : 'light');
  }

  // chooseTheme is the only writer of the page's theme, and it writes both
  // halves of it: the key the next page load reads, and the attribute every
  // player on this one is watching. Nothing here needs a player handle — which
  // is why the playground can dispose and remount players underneath it all
  // day without the control noticing.
  function chooseTheme(state) {
    prefSet(THEME_KEY, state);
    document.documentElement.setAttribute('data-theme', state);
  }

  // drawThemeToggle draws the state the page is *in* and names the action the
  // press will take. The rail's button carried the word "Light" or "Dark"
  // because the word was the state, and a 24px glyph cannot say that on its
  // own; the accessible name says it instead, and aria-live on the button
  // means the new state is announced rather than only shown.
  function drawThemeToggle(btn) {
    var state = effectiveTheme();
    var label = 'Theme: ' + state + ' — click for ' + (state === 'dark' ? 'light' : 'dark');
    btn.title = label;
    btn.setAttribute('aria-label', label);
    btn.innerHTML = '';
    btn.appendChild(icon('theme-' + state));
  }

  // While nothing is stored, the effective theme can change without this page
  // doing anything at all: the stylesheet re-resolves the media query and no
  // data-theme mutation fires, so a button that only redrew on its own press
  // would sit there showing a sun on a dark page. This is the listener that
  // keeps it honest, and it goes quiet the moment a choice exists, because
  // from then on the system is no longer what the page is showing.
  //
  // Nothing takes it off again: unlike a player, which the playground disposes
  // and remounts on every keystroke, the control is built once and lives as
  // long as the document does.
  function watchSystemTheme(btn) {
    try {
      var mq = matchMedia('(prefers-color-scheme: dark)');
      var redraw = function () {
        if (themeChoice()) return;
        drawThemeToggle(btn);
      };
      if (mq.addEventListener) {
        mq.addEventListener('change', redraw);
      } else if (mq.addListener) {
        // The legacy pair predates EventTarget on MediaQueryList.
        mq.addListener(redraw);
      }
    } catch (e) { /* no matchMedia: the glyph drawn at load stands */ }
  }

  // wireThemeToggle turns a bare button into the control: one this file built,
  // or the placeholder a page emitted in its own HTML so that the control is
  // there, and styled, before a single script has run.
  function wireThemeToggle(btn) {
    // A page is free to emit the placeholder *and* call themeToggle(); the
    // flag is what keeps the sweep below from binding a second click.
    if (btn.dgmThemeWired) return btn;
    btn.dgmThemeWired = true;
    btn.addEventListener('click', function () {
      chooseTheme(effectiveTheme() === 'dark' ? 'light' : 'dark');
      drawThemeToggle(btn);
    });
    drawThemeToggle(btn);
    watchSystemTheme(btn);
    return btn;
  }

  function upgradeThemeToggles() {
    var all = document.querySelectorAll('[data-dgm-theme-toggle]');
    for (var i = 0; i < all.length; i++) {
      // `?embed` and `?reel` say the page is a picture rather than a page: an
      // embed's furniture belongs to the document around it, a reel is watched
      // rather than operated, and `cinegram record` photographs `?embed` frame
      // by frame — so a control here would be in every GIF the recorder makes.
      // The rail hides itself in those modes for the same three reasons; this
      // is the page's half of it, and it is a removal rather than a `display`
      // rule because the mode is in the URL, which no stylesheet can read.
      if (isEmbedded(null) || isReel(null)) {
        if (all[i].parentNode) all[i].parentNode.removeChild(all[i]);
        continue;
      }
      wireThemeToggle(all[i]);
    }
  }

  // SHARE_LABEL is the copy button's resting name, restored after the
  // confirmation flashShare puts in its place.
  var SHARE_LABEL = 'Copy link';

  // SHORTCUTS is both the key handler's documentation and the help overlay's
  // content, so the two cannot drift apart.
  var SHORTCUTS = [
    ['Space', 'Play or pause — in presenter mode, play exactly the next step'],
    ['← / →', 'Previous or next step'],
    ['Home / End', 'Jump to the start or the end'],
    ['1 – 9', 'Jump to step n'],
    ['Esc', 'Leave presenter mode, or back out of a drilled-in view'],
    ['Click stage', 'In presenter mode, advance one step'],
    ['Click scene', 'Open the storyboard frame full size — scroll zooms, Esc closes'],
    ['?', 'Show or hide this sheet — settings and shortcuts'],
    ['Scroll', 'Zoom the diagram; drag to pan, double-click to reset'],
    ['Minimap', 'Shown while zoomed: click or drag it to move the view, double-click it to fit']
  ];

  Player.prototype.onKey = function (ev) {
    // Never steal a key from a control that is taking text or arrow input.
    if (ev.target && /input|select|textarea/i.test(ev.target.tagName)) return;
    if (ev.metaKey || ev.ctrlKey || ev.altKey) return;

    var sc = this.scenario();

    if (ev.key === '?') { ev.preventDefault(); this.toggleHelp(); return; }
    if (ev.key === 'Escape') {
      if (this.lightboxIsOpen()) { ev.preventDefault(); this.closeLightbox(); return; }
      if (this.helpOpen()) { ev.preventDefault(); this.toggleHelp(); return; }
      // Escape means "get me out of the mode I am in", innermost first.
      if (this.present) { ev.preventDefault(); this.setPresenter(false); return; }
      if (this.stack.length) { ev.preventDefault(); this.back(); }
      return;
    }

    // In presenter mode — and in a reel, which shares its one-beat transport —
    // Space plays the next beat and stops at its end, rather than starting a
    // run the presenter then has to catch.
    if (this.stepwise() && (ev.key === ' ' || ev.key === 'ArrowRight')) {
      ev.preventDefault();
      this.advanceStep();
      return;
    }
    if (this.stepwise() && ev.key === 'ArrowLeft') {
      ev.preventDefault();
      this.prevStep();
      return;
    }

    if (ev.key === ' ') { ev.preventDefault(); this.toggle(); return; }
    if (ev.key === 'ArrowRight') { ev.preventDefault(); this.nextStep(1); return; }
    if (ev.key === 'ArrowLeft') { ev.preventDefault(); this.nextStep(-1); return; }
    if (ev.key === 'Home') { ev.preventDefault(); this.pause(); this.seek(0); return; }
    if (ev.key === 'End') { ev.preventDefault(); this.pause(); this.seek(sc.duration); return; }

    if (ev.key >= '1' && ev.key <= '9') {
      var i = parseInt(ev.key, 10) - 1;
      if (i < sc.steps.length) {
        ev.preventDefault();
        this.pause();
        this.seek(sc.steps[i].start);
      }
    }
  };

  // SPEED_PRESETS is the sheet's speed menu. Five rates, coarse on purpose:
  // this is a reader asking for slower or faster, not an author timing a beat
  // — `speed:` in the scenario is for that, and outranks this.
  var SPEED_PRESETS = [0.25, 0.5, 1, 1.5, 2];

  // The sheet is settings *and* shortcuts. It was a read-only list, and speed
  // was a button in the rail: a preference, persisted across every diagram on
  // the origin, sitting in the narrowest and most contested column on the page,
  // hidden from presenters by dgm-authoring, and reachable only by cycling
  // forwards through five rates — four clicks to get from 2x back to 0.25x.
  // Somewhere a reader opens deliberately is where a preference belongs, and
  // once there is one such place there is somewhere for the next one to go.
  Player.prototype.buildHelp = function () {
    var self = this;
    var box = el('div', 'dgm-help');
    box.setAttribute('role', 'dialog');
    box.setAttribute('aria-label', 'Settings and shortcuts');
    box.style.display = 'none';

    var panel = el('div', 'dgm-help-panel');
    panel.appendChild(elText('div', 'dgm-help-title', 'Settings and shortcuts'));

    // With reduced motion there is no autoplay, so stepping is not a fallback
    // — it is how the diagram is meant to be read. Say so where it is useful.
    if (prefersReducedMotion()) {
      panel.appendChild(elText('div', 'dgm-help-note',
        'Your system asks for reduced motion, so playback does not start on its own. ' +
        'Step through with the arrow keys.'));
    }

    panel.appendChild(elText('div', 'dgm-help-section', 'Playback'));

    // A <label> wrapping the control rather than a `for`/id pair: one document
    // can hold several players, ids have to be unique across it, and the pair
    // would need a counter nothing else in this file needs. Wrapping asks for
    // no id at all.
    var speedRow = el('label', 'dgm-help-row');
    speedRow.appendChild(elText('span', '', 'Speed'));
    this.speedSel = el('select', 'dgm-select dgm-help-speed');
    for (var i = 0; i < SPEED_PRESETS.length; i++) {
      var opt = document.createElement('option');
      opt.value = String(SPEED_PRESETS[i]);
      opt.textContent = speedLabel(SPEED_PRESETS[i]);
      this.speedSel.appendChild(opt);
    }
    this.speedSel.addEventListener('change', function () {
      self.setSpeed(parseFloat(self.speedSel.value));
    });
    speedRow.appendChild(this.speedSel);
    panel.appendChild(speedRow);
    // The menu exists before any scenario has been adopted, so it is written
    // once from whatever this.speed already is; adoptScenarioSpeed writes it
    // again the moment a scenario has an opinion.
    this.syncSpeed();

    panel.appendChild(elText('div', 'dgm-help-section', 'Shortcuts'));

    var list = el('dl', 'dgm-help-list');
    SHORTCUTS.forEach(function (row) {
      list.appendChild(elText('dt', '', row[0]));
      list.appendChild(elText('dd', '', row[1]));
    });
    panel.appendChild(list);

    var close = button('Close', 'dgm-btn', function () { self.toggleHelp(); });
    panel.appendChild(close);

    box.appendChild(panel);
    box.addEventListener('click', function (ev) {
      if (ev.target === box) self.toggleHelp();
    });
    // Escape is caught here rather than in onKey, and that is the price of
    // putting a control in the sheet: onKey's first line refuses to steal a key
    // from an input, a select or a textarea — which is what makes the speed
    // menu safe to open over a live player — so with focus in the menu, which
    // is exactly where opening the sheet puts it, onKey never sees the Escape
    // at all. A listener on the sheet does, ahead of the one on the root or the
    // document, and stops it there so nothing else reads the same press as
    // "leave presenter mode".
    box.addEventListener('keydown', function (ev) {
      if (ev.key !== 'Escape' || !self.helpOpen()) return;
      ev.preventDefault();
      ev.stopPropagation();
      self.toggleHelp();
    });
    return box;
  };

  Player.prototype.helpOpen = function () {
    return this.help && this.help.style.display !== 'none';
  };

  // Opening moves focus into the sheet, and that is not a nicety: the player
  // behind it is still holding Space and the arrow keys, so a dialog that
  // leaves focus outside itself is a dialog whose reader is driving the thing
  // underneath it. Focus lands on the speed menu, and onKey's
  // input/select/textarea guard then keeps all of those keys inside it.
  //
  // Closing puts focus back where it came from rather than on the Help button:
  // `?` opens the sheet from wherever the reader already was, and leaving them
  // on a button they never pressed would mean the next Space re-opened the
  // sheet instead of playing the diagram.
  Player.prototype.toggleHelp = function () {
    var open = !this.helpOpen();
    if (open) this.helpReturn = document.activeElement;
    this.help.style.display = open ? '' : 'none';
    var to = open ? this.speedSel : this.helpReturn;
    if (!open) this.helpReturn = null;
    // The element focus came from can be gone by now — the playground disposes
    // a player and mounts the next on every keystroke — so it is checked
    // rather than assumed. Focus is a courtesy; the sheet opens either way.
    try {
      if (to && to.focus && document.contains(to)) to.focus();
    } catch (e) { /* nothing to hand the focus to */ }
  };

  // ---------------------------------------------------------------------
  // Storyboard
  //
  // The side-stage: the screens a person would be looking at while the diagram
  // animates. Cinegram supplies the synchronisation; the pictures are the
  // author's, which is what keeps the feature bounded — an email screenshot is
  // as valid a frame as a login form.
  //
  // A scene is *sticky*: at time t the panel shows the latest scene track whose
  // start is at or before t, across the whole scenario rather than the current
  // step. That is a pure function of t, so scrubbing backwards lands exactly
  // where playing forwards would, and the panel does not blank between scenes —
  // what someone is looking at does not vanish because a step ended.
  //
  // The panel is overlay-style HTML, not part of the SVG, so baseClass and
  // STICKY have no bearing on it. Its state is keyed off the diffed frame id.
  // ---------------------------------------------------------------------

  Player.prototype.buildBoard = function () {
    var self = this;
    this.board = el('div', 'dgm-board');
    this.board.style.display = 'none';

    this.boardTitle = el('div', 'dgm-board-title');
    this.board.appendChild(this.boardTitle);

    var stack = el('div', 'dgm-board-stack');
    // Two layers, absolutely stacked, taking it in turns to be the front one.
    // A single element swapping its src would flash the page background at
    // every scene change.
    this.boardLayers = [el('div', 'dgm-board-layer'), el('div', 'dgm-board-layer')];
    for (var i = 0; i < this.boardLayers.length; i++) {
      var img = document.createElement('img');
      img.className = 'dgm-board-img';
      img.alt = '';
      this.boardLayers[i].appendChild(img);
      stack.appendChild(this.boardLayers[i]);
    }
    this.boardFront = 0;
    this.boardLayers[0].classList.add('is-front');
    // The panel is a preview; the lightbox is the full-size look at it.
    stack.addEventListener('click', function () {
      var frame = self.boardKey ? self.frames[self.boardKey] : null;
      if (frame && frame.image) self.openLightbox(frame);
    });
    this.board.appendChild(stack);

    this.boardCaption = el('div', 'dgm-board-caption');
    this.board.appendChild(this.boardCaption);

    this.boardOn = false;
    this.boardKey = null;
    // "The reader dismissed the thumbnail", as a fact rather than a DOM class:
    // the swipe sets it, syncBoard folds it into boardOn, and the sites that
    // change the context a dismissal was made in — a render, a scenario
    // switch, a mode change, a resize — reset it and ask syncBoard again.
    this.boardDismissed = false;
    this.frames = {};
    return this.board;
  };

  // syncBoard decides whether the panel belongs on the page at all.
  //
  // Scene usage is per scenario, not per view: a document may storyboard its
  // happy path and say nothing about its failure path, and the failure path
  // should get the full width rather than an empty panel. So this re-runs on
  // every scenario change as well as every render.
  Player.prototype.syncBoard = function () {
    var sb = this.view().storyboard;
    var frames = (sb && sb.frames) || [];

    this.frames = {};
    for (var i = 0; i < frames.length; i++) {
      this.frames[frames[i].id] = frames[i];
    }

    // The one writer of the panel's visibility. boardDismissed is part of the
    // computation rather than a class competing with it, so "should the panel
    // show" has exactly one answer in exactly one place; whoever changes the
    // context a dismissal was made in resets the field and calls back here.
    this.boardOn = frames.length > 0 && hasScenes(this.scenario()) &&
      !this.boardDismissed;
    this.root.classList.toggle('dgm-has-board', this.boardOn);
    this.board.style.display = this.boardOn ? '' : 'none';

    var title = (sb && sb.title) || '';
    this.boardTitle.textContent = title;
    this.boardTitle.style.display = title ? '' : 'none';

    // Force the next applyBoard to redraw: the same frame id can mean a
    // different picture once the view has changed.
    this.boardKey = null;
    this.applyBoard(this.activeScene(this.time, this.scenario()));
  };

  function hasScenes(sc) {
    var steps = sc.steps || [];
    for (var s = 0; s < steps.length; s++) {
      var tracks = steps[s].tracks || [];
      for (var k = 0; k < tracks.length; k++) {
        if (tracks[k].kind === 'scene') return true;
      }
    }
    return false;
  }

  // activeScene is the frame showing at time t: the last one to have started.
  //
  // It scans every step rather than the ones whose window contains t, which is
  // exactly what makes a scene outlast its step — the panel holds the last
  // screen until something replaces it.
  Player.prototype.activeScene = function (t, sc) {
    var best = null;
    var steps = sc.steps || [];
    for (var s = 0; s < steps.length; s++) {
      var tracks = steps[s].tracks || [];
      for (var k = 0; k < tracks.length; k++) {
        var tr = tracks[k];
        if (tr.kind !== 'scene' || tr.start > t) continue;
        if (!best || tr.start >= best.start) best = tr;
      }
    }
    return best;
  };

  // applyBoard swaps the panel to a frame, crossfading.
  //
  // It diffs on the frame id first and touches nothing when the answer has not
  // changed. That is the same discipline syncCaption follows and for a stronger
  // reason: rewriting an <img> src sixty times a second would restart the
  // opacity transition on every frame and the crossfade would never finish.
  Player.prototype.applyBoard = function (tr) {
    if (!this.boardOn) return;

    var frame = tr ? this.frames[tr.target] : null;
    var key = frame ? frame.id : '';
    if (key === this.boardKey) return;
    this.boardKey = key;

    var back = this.boardLayers[this.boardFront ^ 1];
    var img = back.querySelector('.dgm-board-img');
    if (frame && frame.image) {
      img.src = frame.image;
      img.style.display = '';
    } else {
      img.removeAttribute('src');
      img.style.display = 'none';
    }

    back.classList.add('is-front');
    this.boardLayers[this.boardFront].classList.remove('is-front');
    this.boardFront ^= 1;

    this.boardCaption.textContent = (frame && frame.caption) || '';
    // A caption-only frame is text, not a picture with a label under it, so it
    // gets the room the image box would have taken.
    this.board.classList.toggle('is-wordy', !!(frame && !frame.image));
    this.board.classList.toggle('is-pictorial', !!(frame && frame.image));

    // An open lightbox shows *the current scene*, not a snapshot of one: when
    // the sticky frame changes underneath it, it follows (and closes if the
    // scenario moves onto a frame with no picture).
    if (this.lightboxIsOpen()) {
      if (frame && frame.image) this.openLightbox(frame);
      else this.closeLightbox();
    }
  };

  // --- storyboard lightbox ----------------------------------------------
  //
  // The board panel is a thumbnail-sized preview; this is the full-size look.
  // It is the same overlay pattern as the help dialog — fixed, backdrop,
  // click-out or Esc to leave — plus the stage's own zoom conventions. One
  // deliberate difference from the stage: inside a modal there is no page
  // behind the pointer to scroll, so a bare wheel zooms with no modifier.
  //
  // Fit is PhotoSwipe's rule: a raster frame never opens larger than its own
  // pixels (min(1, viewport/image)) because upscaling raster is only blur; an
  // SVG frame fills the viewport, because it can.

  Player.prototype.buildLightbox = function () {
    var self = this;
    var box = el('div', 'dgm-lightbox');
    box.setAttribute('role', 'dialog');
    box.setAttribute('aria-label', 'Storyboard frame');
    box.setAttribute('tabindex', '-1');
    box.style.display = 'none';

    this.lbImg = document.createElement('img');
    this.lbImg.className = 'dgm-lightbox-img';
    this.lbImg.alt = '';
    this.lbImg.draggable = false;
    box.appendChild(this.lbImg);

    this.lbCaption = el('div', 'dgm-lightbox-caption');
    box.appendChild(this.lbCaption);

    // Zoom state: lbBase is the fitted layout size (crisp: it is real layout,
    // not a transform, so an SVG frame re-rasterises at it), lbK the zoom on
    // top of it, lbX/lbY the pan in screen pixels.
    this.lbFrame = null;
    this.lbBase = { w: 0, h: 0 };
    this.lbK = 1;
    this.lbKMax = 1;
    this.lbX = 0;
    this.lbY = 0;

    box.addEventListener('wheel', function (ev) {
      ev.preventDefault();
      self.lightboxZoomAt(ev.clientX, ev.clientY, Math.exp(-ev.deltaY * 0.0015));
    }, { passive: false });

    var lbDone = function () { self.lbImg.classList.remove('is-panning'); };
    drag(this.lbImg, {
      start: function (ev, st) {
        st.px = ev.clientX;
        st.py = ev.clientY;
        self.lbImg.classList.add('is-panning');
        ev.preventDefault();
        // Every press on the image is a pan; nothing here is a click target.
        st.claim(ev);
      },
      move: function (ev, st) {
        self.lbX += ev.clientX - st.px;
        self.lbY += ev.clientY - st.py;
        st.px = ev.clientX;
        st.py = ev.clientY;
        self.clampLightboxPan();
        self.applyLightbox();
      },
      end: lbDone,
      cancel: lbDone
    });

    // Double-click toggles between fit and the secondary zoom, anchored where
    // the pointer is, exactly like the wheel.
    this.lbImg.addEventListener('dblclick', function (ev) {
      var target = self.lbK > 1.001 ? 1 : Math.min(self.lbKMax, 3);
      self.lightboxZoomAt(ev.clientX, ev.clientY, target / self.lbK);
    });

    box.addEventListener('click', function (ev) {
      if (ev.target === box) self.closeLightbox();
    });

    // The fit is a function of the viewport, so a resize while open re-fits.
    own(self, window, 'resize', function () {
      if (self.lightboxIsOpen()) self.layoutLightbox();
    });

    return box;
  };

  Player.prototype.lightboxIsOpen = function () {
    return this.lightbox && this.lightbox.style.display !== 'none';
  };

  Player.prototype.openLightbox = function (frame) {
    var self = this;
    // Frame ids are per view, so the image is part of the identity: the same
    // id in another view is a different picture.
    var fresh = !this.lbFrame || this.lbFrame.id !== frame.id ||
      this.lbFrame.image !== frame.image;
    this.lbFrame = frame;
    if (fresh) {
      this.lbK = 1;
      this.lbX = 0;
      this.lbY = 0;
      if (this.lbImg.src !== frame.image) this.lbImg.src = frame.image;
    }
    this.lbCaption.textContent = frame.caption || '';
    this.lightbox.style.display = '';

    if (this.lbImg.complete && this.lbImg.naturalWidth) {
      this.layoutLightbox();
    } else {
      this.lbImg.onload = function () {
        self.lbImg.onload = null;
        self.layoutLightbox();
      };
    }
    this.lightbox.focus();
  };

  Player.prototype.closeLightbox = function () {
    if (!this.lightboxIsOpen()) return;
    this.lightbox.style.display = 'none';
    this.lbFrame = null;
    // Hand the keys back to the player the reader was in.
    if (this.opts.keys === 'scoped') this.root.focus();
  };

  // layoutLightbox computes the fit and writes it as real width/height. The
  // zoom on top stays a transform: layout at fit keeps the resting view crisp
  // (an SVG re-rasterises), and transform while zooming keeps the gesture
  // cheap.
  Player.prototype.layoutLightbox = function () {
    var w = this.lbImg.naturalWidth, h = this.lbImg.naturalHeight;
    if (!w || !h || !this.lbFrame) return;
    var fit = Math.min(
      (window.innerWidth * 0.94) / w,
      (window.innerHeight * 0.88) / h
    );
    var vector = /^data:image\/svg/.test(this.lbFrame.image);
    if (!vector) fit = Math.min(1, fit);
    this.lbBase.w = w * fit;
    this.lbBase.h = h * fit;
    // Raster earns zoom up to its own pixels (1:1), vector as far as useful.
    this.lbKMax = vector ? 8 : Math.max(2, 1 / fit);
    this.lbImg.style.width = this.lbBase.w + 'px';
    this.lbImg.style.height = this.lbBase.h + 'px';
    this.lbK = Math.min(this.lbK, this.lbKMax);
    this.clampLightboxPan();
    this.applyLightbox();
  };

  // Cursor-anchored zoom: the image point under the pointer stays put. The
  // image is flex-centred, so its displayed centre is centre-plus-pan; scaling
  // about that centre moves the anchor by (anchor − centre)·(factor − 1), and
  // the pan absorbs exactly that.
  Player.prototype.lightboxZoomAt = function (cx, cy, factor) {
    var k = Math.min(this.lbKMax, Math.max(1, this.lbK * factor));
    factor = k / this.lbK;
    if (factor === 1) return;
    var rect = this.lbImg.getBoundingClientRect();
    var ccx = rect.left + rect.width / 2;
    var ccy = rect.top + rect.height / 2;
    this.lbK = k;
    this.lbX -= (cx - ccx) * (factor - 1);
    this.lbY -= (cy - ccy) * (factor - 1);
    if (k === 1) { this.lbX = 0; this.lbY = 0; }
    this.clampLightboxPan();
    this.applyLightbox();
  };

  // Panning may push the image around, never lose it: at least an edge stays
  // within reach of the viewport.
  Player.prototype.clampLightboxPan = function () {
    var mx = Math.max(0, (this.lbBase.w * this.lbK - window.innerWidth) / 2) + 40;
    var my = Math.max(0, (this.lbBase.h * this.lbK - window.innerHeight) / 2) + 40;
    this.lbX = Math.min(mx, Math.max(-mx, this.lbX));
    this.lbY = Math.min(my, Math.max(-my, this.lbY));
  };

  Player.prototype.applyLightbox = function () {
    this.lbImg.style.transform =
      'translate(' + this.lbX + 'px,' + this.lbY + 'px) scale(' + this.lbK + ')';
  };

  // --- pan and zoom -----------------------------------------------------
  //
  // The transform goes on the SVG holder rather than the SVG's viewBox, which
  // keeps it out of mermaid's way entirely. It also means getBoundingClientRect
  // on a node already reflects the zoom, so notes, badges and gauges follow
  // their elements with no extra arithmetic.

  var ZOOM_MIN = 0.4;
  var ZOOM_MAX = 4;

  // armSwallow marks the next click as the tail of a drag; the capture-phase
  // listener in build() takes it. The zero timer disarms when the release
  // produced no click at all — outside the window, or on nothing clickable.
  Player.prototype.armSwallow = function () {
    var self = this;
    this.swallowNext = true;
    setTimeout(function () { self.swallowNext = false; }, 0);
  };

  Player.prototype.bindStageGestures = function () {
    var self = this;

    this.stage.addEventListener('wheel', function (ev) {
      // On a page of its own the stage is the thing being scrolled over, so a
      // wheel means zoom. Inside a document it is not: the reader is scrolling
      // past, and swallowing that would strand them on the diagram. Ctrl or ⌘
      // is the same modifier a map asks for, and the same one the browser's own
      // pinch-zoom arrives with.
      if (self.opts.inline && !ev.ctrlKey && !ev.metaKey) return;
      ev.preventDefault();
      // A manual zoom takes the wheel from the auto-follow camera.
      if (self.follow) self.camOverride = true;
      var factor = Math.exp(-ev.deltaY * 0.0015);
      self.zoomAt(ev.clientX, ev.clientY, factor);
    }, { passive: false });

    this.stage.addEventListener('dblclick', function () {
      // A double-click hands the framing back to the camera; the resetZoom
      // below re-applies the frame, and the camera reposes in the same apply
      // pass now that the override is gone.
      if (self.follow) self.camOverride = false;
      self.resetZoom();
    });

    // The pan. A few pixels of slop, so a click on a node is not read as a pan
    // and the node still activates; the moment the slop is crossed the whole
    // accumulated travel is applied at once, so the diagram does not start a
    // few pixels behind the hand.
    drag(this.stage, {
      move: function (ev, st) {
        if (!st.moved) {
          if (Math.abs(st.dx) + Math.abs(st.dy) < 4) return;
          st.moved = true;
          st.ax = 0;
          st.ay = 0;
          // Now it is a pan, not a click on its way to a node.
          st.claim(ev);
        }
        if (self.follow) self.camOverride = true;
        self.panX += st.dx - st.ax;
        self.panY += st.dy - st.ay;
        st.ax = st.dx;
        st.ay = st.dy;
        self.applyTransform();
      },
      end: function (ev, st) {
        // The click this release manufactures belonged to the pan: swallowed,
        // so panning across a node does not also drill into it.
        if (st.moved) self.armSwallow();
      }
    });

    // Presenting from a lectern means a clicker, and a clicker sends a click;
    // a reel on a phone means a tap, and a tap sends one too. A click on a
    // bound element stops propagating before it gets here, so drilling into a
    // view still wins over advancing.
    this.stage.addEventListener('click', function () {
      if (self.stepwise()) self.advanceStep();
    });
  };

  // Presenter mode, reel mode and a `stepwise` scenario share the one-beat
  // transport: Space, →, a stage click and the Play button all mean "play
  // exactly the next step, then stop".
  Player.prototype.stepwise = function () {
    return this.present || this.reel || !!this.scenario().stepwise;
  };

  // setFollow flips the Cine toggle. Turning it on hands the framing to the
  // camera afresh; turning it off gives the whole diagram back, because being
  // left stranded at the last step's zoom with the camera gone is exactly the
  // exit experience setPresenter already refuses to give.
  // setFollow flips what `follow` says, except in a reel, where the clamp
  // keeps it true: the format cannot work without the camera, and the Cine
  // button does not exist in a reel's hidden bar anyway. `follow` is
  // deliberately not stepwise(): the transport playing one beat at a time and
  // the camera zooming to each beat are different promises — presenter mode
  // and `stepwise:` scenarios start by seeing the whole diagram.
  Player.prototype.setFollow = function (on) {
    this.follow = this.reel || !!on;
    this.cineBtn.classList.toggle('is-on', this.follow);
    // Beside the class, not anywhere else: one writer means the tint an eye
    // reads and the state a screen reader hears cannot come apart.
    this.cineBtn.setAttribute('aria-pressed', this.follow ? 'true' : 'false');
    this.camKeys = null;
    this.mapKeys = null;
    this.camOverride = false;
    if (this.follow) {
      this.apply(this.time);
      this.syncChips();
    } else {
      // resetZoom relays out the overlays itself, via applyTransform.
      this.resetZoom();
    }
  };

  // restingTime is where an idle page sits: the author's poster moment, or the
  // start. The deep link wins over both — see build().
  Player.prototype.restingTime = function () {
    var sc = this.scenario();
    return (sc && sc.poster) || 0;
  };

  Player.prototype.zoomAt = function (clientX, clientY, factor) {
    var next = clamp(this.zoom * factor, ZOOM_MIN, ZOOM_MAX);
    if (next === this.zoom) return;

    // Keep the point under the cursor where it is: solve for the pan that
    // leaves this stage-local point fixed across the scale change.
    var r = this.stage.getBoundingClientRect();
    var x = clientX - r.left, y = clientY - r.top;
    this.panX = x - (x - this.panX) * (next / this.zoom);
    this.panY = y - (y - this.panY) * (next / this.zoom);
    this.zoom = next;
    this.applyTransform();
  };

  Player.prototype.resetZoom = function () {
    this.zoom = 1;
    this.panX = 0;
    this.panY = 0;
    this.applyTransform();
  };

  // setTransform is the style write alone. The camera calls it from inside
  // apply(), where re-entering apply() would recurse; every other caller wants
  // applyTransform below, which also relays out the overlays.
  Player.prototype.setTransform = function () {
    if (this.holder) {
      this.holder.style.transformOrigin = '0 0';
      this.holder.style.transform =
        'translate(' + this.panX + 'px,' + this.panY + 'px) scale(' + this.zoom + ')';
    }
    // Nothing here has to light a "zoomed" state any more: the minimap *is*
    // that state, drawn as the thing it means, and syncMap below is what shows
    // and hides it.
    //
    // This is the one place zoom/panX/panY are ever written, which is what
    // lets the wheel, the drag, resetZoom and the camera all move the map's
    // rectangle without any of them knowing the map exists.
    this.syncMap();
  };

  Player.prototype.applyTransform = function () {
    this.setTransform();
    // Overlay content is positioned from client rects, which the transform has
    // just changed, so it has to be laid out again.
    this.apply(this.time);
    this.syncChips();
  };

  // --- minimap -----------------------------------------------------------
  //
  // Zoomed in — by wheel, by drag, or by the auto-follow camera — nothing on
  // screen says which part of the diagram is showing. The map answers it
  // spatially: the whole diagram in the stage's top-right corner with a
  // rectangle around the visible region. It is on screen only while the stage
  // shows less than the whole diagram, so a page at rest looks exactly as it
  // did — and so does every recording, since `record` photographs `?embed` at
  // rest, where the map is `is-off`.
  //
  // The thumbnail is a clone of the rendered SVG taken once per render, never
  // per frame, and it is deliberately static: applyNodeStates rewrites the
  // class attribute of live elements every frame and baseClass strips `dgm-*`,
  // but the clone is outside that machinery entirely. The map says where you
  // are, not what is happening.
  //
  // Every id *inside* the clone is renamed under a per-clone prefix, and every
  // reference to one is rewritten to follow it, so the thumbnail is
  // self-contained. The earlier trick — stripping the ids so `url(#…)`
  // resolved document-wide to the live SVG's defs — worked with one player and
  // broke with two: mermaid's sequence marker ids are unprefixed (`arrowhead`,
  // `crosshead`, `sequencenumber`), so both players' clones borrowed whichever
  // SVG came first in document order, and a VS Code Markdown preview with two
  // ```dgm blocks — a supported, routine configuration — gave the second
  // thumbnail the first diagram's markers, or none at all once the preview's
  // diff reverted that block to its placeholder. Renaming keeps each clone on
  // its own defs whatever else the document holds. Nothing the runtime binds
  // to can be shadowed in return, because every lookup in index() is scoped to
  // the live svg (`svg.querySelectorAll`) and this file contains no
  // getElementById at all.
  //
  // The root id gets the same treatment for its own reason: mermaid prefixes
  // every rule of the <style> it embeds with the SVG's own id, so a clone with
  // no fresh id would draw unstyled — black boxes, no theme. Its copy of that
  // stylesheet is retargeted in the same pass that follows the renamed ids.

  var MAP_MAX_W = 220;    // px, and the cap the CSS states as a share
  var MAP_MAX_H = 240;
  var MAP_SHARE_W = 0.28; // of the stage, whichever comes out smaller
  var MAP_SHARE_H = 0.4;

  // buildMap only resets: the clone itself is built lazily by ensureMapClone,
  // on the first frame that would actually show the map. A page at rest — and
  // every `?embed` capture record photographs — then never pays for cloning
  // and walking the whole diagram for a thumbnail that stays hidden.
  Player.prototype.buildMap = function () {
    if (!this.map) return;
    this.mapBody.innerHTML = '';
    this.mapClone = null;
    this.mapKeys = null;
    this.syncMap();
  };

  Player.prototype.ensureMapClone = function () {
    if (this.mapClone || !this.svg) return;
    // A reel is a 9:16 frame with no room to spare, and its camera is the
    // point of the format rather than something to orient against.
    if (this.reel) return;
    // Between the moment render() puts the new holder on the stage and the
    // moment it finds the svg inside it, `this.svg` is still the previous
    // diagram — detached, and about to be replaced. setTransform runs in that
    // window, so without this a re-render at any zoom would clone the old
    // diagram in full just to have buildMap throw it away a few lines later.
    if (!this.holder || !this.holder.contains(this.svg)) return;
    var clone = this.svg.cloneNode(true);
    var old = clone.getAttribute('id') || '';
    // One namespace per clone: ids keep their names but move under a prefix,
    // so two players' thumbnails can share a document without sharing defs.
    var prefix = 'dgm-map-' + Math.floor(Math.random() * 1e9) + '-';
    var fresh = prefix + (old || 'svg');

    var ids = {};
    var kids = clone.querySelectorAll('[id]');
    for (var i = 0; i < kids.length; i++) {
      ids[kids[i].getAttribute('id')] = true;
      kids[i].setAttribute('id', prefix + kids[i].getAttribute('id'));
    }
    clone.setAttribute('id', fresh);

    // followUrls rewrites url(#…) references — markers, clip paths, masks,
    // paint servers, filters — onto the renamed ids. Only ids the clone
    // actually holds are touched, so a reference out of the document (or a
    // colour that merely looks like one, `#fff`) passes through untouched.
    function followUrls(value) {
      return value.replace(/url\(\s*(["']?)#([^"')\s]+)\1\s*\)/g, function (whole, q, id) {
        return ids[id] ? 'url(' + q + '#' + prefix + id + q + ')' : whole;
      });
    }
    var REF_ATTRS = ['marker-start', 'marker-mid', 'marker-end', 'clip-path',
                     'mask', 'fill', 'stroke', 'filter', 'style'];

    var sheets = clone.querySelectorAll('style');
    for (var j = 0; j < sheets.length; j++) {
      var css = sheets[j].textContent;
      if (old) css = css.split('#' + old).join('#' + fresh);
      css = css.replace(/#([A-Za-z_][-\w]*)/g, function (whole, id) {
        return ids[id] ? '#' + prefix + id : whole;
      });
      sheets[j].textContent = css;
    }

    // Cloning re-arms what innerHTML left inert: an SVG <script> parsed via
    // innerHTML never runs, but its clone does the moment it is inserted. The
    // live SVG's safety rests on mermaid's sanitizers; the thumbnail should
    // not depend on them, so anything executable is dropped from the copy.
    var scripts = clone.querySelectorAll('script');
    for (var k = 0; k < scripts.length; k++) scripts[k].parentNode.removeChild(scripts[k]);
    var all = clone.querySelectorAll('*');
    for (var m = 0; m < all.length; m++) {
      var attrs = all[m].attributes;
      for (var a = attrs.length - 1; a >= 0; a--) {
        if (/^on/i.test(attrs[a].name)) all[m].removeAttribute(attrs[a].name);
      }
      for (var ra = 0; ra < REF_ATTRS.length; ra++) {
        var v = all[m].getAttribute(REF_ATTRS[ra]);
        if (v && v.indexOf('url(') !== -1) all[m].setAttribute(REF_ATTRS[ra], followUrls(v));
      }
      var href = all[m].getAttribute('href');
      if (href && href.charAt(0) === '#' && ids[href.slice(1)]) {
        all[m].setAttribute('href', '#' + prefix + href.slice(1));
      }
      var xhref = all[m].getAttribute('xlink:href');
      if (xhref && xhref.charAt(0) === '#' && ids[xhref.slice(1)]) {
        all[m].setAttribute('xlink:href', '#' + prefix + xhref.slice(1));
      }
      // And the keyboard affordances a clickable node carries. bindClicks gives
      // every one of them tabindex, role and a name; the *listeners* are not
      // cloned, so the copies are tab stops that do nothing — a second set of
      // every control on the diagram, sitting in a subtree marked aria-hidden.
      // A focusable element inside one of those is the same error the map's
      // own attribute was moved down to avoid, and it is why the map can hold
      // a real button at all.
      all[m].removeAttribute('tabindex');
      all[m].removeAttribute('role');
      all[m].removeAttribute('aria-label');
    }
    clone.setAttribute('aria-hidden', 'true');
    clone.removeAttribute('role');
    // The box is already the diagram's own aspect ratio, so the viewBox does
    // the scaling and the picture fills it exactly.
    clone.style.width = '100%';
    clone.style.height = '100%';
    clone.style.maxWidth = 'none';
    clone.style.maxHeight = 'none';
    clone.style.pointerEvents = 'none';
    this.mapBody.appendChild(clone);
    this.mapClone = clone;
  };

  // --- the stage↔holder mapping, written once --------------------------
  //
  // The camera and the minimap describe the same world, so the arithmetic
  // that relates the stage to the holder lives here and only here: cameraKeys
  // and mapGeom both build their caches on these, and a change to the framing
  // convention — a stage border, a different transform-origin — moves both
  // consumers together instead of desynchronizing them.

  // holderOrigin is the holder's untransformed origin inside the stage: a
  // holder-local point q draws at origin + pan + q*zoom. The caller passes the
  // client rects it already measured, so building a cache costs no extra
  // layout.
  Player.prototype.holderOrigin = function (stageR, holderR) {
    return { x: holderR.left - this.panX - stageR.left,
             y: holderR.top - this.panY - stageR.top };
  };

  // localRect is an element's holder-local untransformed box. Client rects are
  // used deliberately: they already compose mermaid's inner transforms and the
  // SVG's CSS scaling, so dividing our own holder transform back out is the
  // whole coordinate story.
  Player.prototype.localRect = function (el, holderR) {
    var r = el.getBoundingClientRect();
    if (!r.width && !r.height) return null;
    var z = this.zoom || 1;
    return { x: (r.left - holderR.left) / z, y: (r.top - holderR.top) / z,
             w: r.width / z, h: r.height / z };
  };

  // centreOnLocal writes the pan that puts a holder-local point at the centre
  // of the stage, at zoom z. K supplies the frame — holderOrigin `o` and the
  // stage box W×H — from whichever cache the caller maintains (camKeys,
  // mapKeys): both record the same layout and are invalidated together. It
  // writes this.panX/panY and nothing else; the caller owns the style write,
  // because cameraApply must not re-enter apply().
  Player.prototype.centreOnLocal = function (cx, cy, z, K) {
    this.panX = K.W / 2 - K.o.x - z * cx;
    this.panY = K.H / 2 - K.o.y - z * cy;
  };

  // mapGeom measures the stage, the holder and the diagram once, and returns
  // the mapping between them. `o` and `d` are untransformed — divided back out
  // of the client rects by the current zoom, exactly as cameraKeys does — so
  // they describe the layout rather than the pose, and only a resize or a
  // render can change them. Null means "there is nothing to draw a map of".
  //
  // Which is exactly why the answer is cached in mapKeys, the same way
  // cameraKeys caches its poses: syncMap runs per frame through a camera glide
  // and per pointermove through a pan, and three forced layouts on each of
  // those is a cost with nothing to show for it. Everything that does move the
  // mapping — a resize, a render, a mode change — nulls the cache, beside the
  // camera's own. The box's size is written here for the same reason: it is a
  // function of the keys, so it changes exactly when they are rebuilt.
  Player.prototype.mapGeom = function () {
    if (!this.map || !this.mapClone || !this.holder || !this.svg) return null;
    if (this.mapKeys) return this.mapKeys;
    var stageR = this.stage.getBoundingClientRect();
    var holderR = this.holder.getBoundingClientRect();
    var svgR = this.svg.getBoundingClientRect();
    if (!stageR.width || !stageR.height || !svgR.width || !svgR.height) return null;

    // holderOrigin and localRect are the same functions cameraKeys builds on,
    // so the map that claims to describe the camera's world is derived from
    // the camera's own arithmetic rather than derived a second time beside it.
    var o = this.holderOrigin(stageR, holderR);

    // The diagram's own untransformed box, holder-local. The map is a picture
    // of the diagram, not of the empty room around it: a diagram narrower than
    // its stage would otherwise draw as a sliver in a mostly blank box, and the
    // rectangle would not line up with the picture it is drawn over.
    var d = this.localRect(this.svg, holderR);
    if (!d) return null;

    // One scale factor for the whole map: fit the diagram's box into the cap,
    // which keeps a tall diagram from claiming the height of the stage.
    var s = Math.min(Math.min(MAP_MAX_W, stageR.width * MAP_SHARE_W) / d.w,
                     Math.min(MAP_MAX_H, stageR.height * MAP_SHARE_H) / d.h);
    this.mapKeys = { o: o, d: d, s: s, W: stageR.width, H: stageR.height,
                     bw: d.w * s, bh: d.h * s };
    this.map.style.width = this.mapKeys.bw + 'px';
    this.map.style.height = this.mapKeys.bh + 'px';
    return this.mapKeys;
  };

  // syncMap places the rectangle, and is the one place the map is ever built.
  // It runs per frame during a camera glide and per pointermove during a pan,
  // so it stays cheap: at fit it measures nothing at all, and off fit it reads
  // a cache and writes four styles.
  Player.prototype.syncMap = function () {
    if (!this.map) return;

    // At fit the whole diagram is on the stage, so there is nothing to orient
    // and the map is off. Answering that from the transform alone — before any
    // measuring, and before the clone exists — is what keeps a page at rest,
    // an `?embed` capture and a resize storm from paying for a map nobody is
    // going to see. It is also why the clone is lazy: this is the only path to
    // it, so a page that never zooms never builds one.
    if (this.zoom === 1 && this.panX === 0 && this.panY === 0) {
      this.map.classList.add('is-off');
      return;
    }

    this.ensureMapClone();
    var g = this.mapGeom();
    if (!g) { this.map.classList.add('is-off'); return; }

    // Invert the mapping: the stage's own [0,W]x[0,H] is this holder-local
    // rect, which the diagram's corner and one scale factor turn into map px.
    var z = this.zoom || 1;
    var l = ((0 - g.o.x - this.panX) / z - g.d.x) * g.s;
    var t = ((0 - g.o.y - this.panY) / z - g.d.y) * g.s;
    var r = l + (g.W / z) * g.s;
    var b = t + (g.H / z) * g.s;
    var cl = clamp(l, 0, g.bw), ct = clamp(t, 0, g.bh);
    var cr = clamp(r, 0, g.bw), cb = clamp(b, 0, g.bh);

    // A rectangle covering the whole thumbnail means the stage is already
    // showing the whole diagram: there is nothing to orient, so the map goes
    // away. The early-out above is the cheap half of this test — it catches
    // the untouched transform; this catches zooming back out past fit, and a
    // diagram small enough that a pan never takes it off the stage.
    var off = cl <= 0.5 && ct <= 0.5 && cr >= g.bw - 0.5 && cb >= g.bh - 0.5;
    this.map.classList.toggle('is-off', off);
    if (off) return;
    this.mapRect.style.left = cl + 'px';
    this.mapRect.style.top = ct + 'px';
    this.mapRect.style.width = Math.max(0, cr - cl) + 'px';
    this.mapRect.style.height = Math.max(0, cb - ct) + 'px';
  };

  Player.prototype.bindMapGestures = function () {
    var self = this;

    // Centre the stage on the point pressed: centreOnLocal is the same pan
    // the camera writes, so a view moved by hand and one moved by the camera
    // stay in one coordinate system.
    function centreOn(ev) {
      var g = self.mapGeom();
      if (!g) return;
      var r = self.map.getBoundingClientRect();
      var qx = g.d.x + (ev.clientX - r.left) / g.s;
      var qy = g.d.y + (ev.clientY - r.top) / g.s;
      self.centreOnLocal(qx, qy, self.zoom, g);
      self.applyTransform();
    }

    // A press is already a jump — centreOn from the first pointerdown — and a
    // drag scrubs; both live here, not in the helper.
    drag(this.map, {
      start: function (ev, st) {
        // The stage underneath reads a press as the start of a pan and, when
        // the transport is stepwise, a click as "advance a step". Neither is
        // what a press on the map means — and every press here is already a
        // jump, so the pointer is claimed at once: a drag that wanders off the
        // little box keeps scrubbing anyway.
        ev.preventDefault();
        ev.stopPropagation();
        st.claim(ev);
        // Moving the view by hand takes it from the camera, exactly as a stage
        // drag does — otherwise the next frame would put it straight back.
        if (self.follow) self.camOverride = true;
        centreOn(ev);
      },
      move: function (ev) {
        ev.preventDefault();
        centreOn(ev);
      },
      end: function () {
        // The click that follows the captured drag lands on whatever is under
        // the release — usually the stage, where a stepwise transport reads it
        // as "advance a step". Take that one click back.
        self.armSwallow();
      }
    });

    this.map.addEventListener('click', function (ev) { ev.stopPropagation(); });

    // Double-click means "put the whole diagram back" on the stage, so it means
    // it on the picture of the stage too — the gesture a reader has already
    // learned, in the one place they are looking while zoomed in. The
    // stopPropagation is the same one the plain click needs: the stage below
    // reads a double-click as a fit as well, and would run a second one.
    //
    // Unlike the stage's, this one does not also clear camOverride. The stage's
    // double-click is documented as handing the framing back to the camera;
    // here the camera is often the reason the map is on screen at all, so
    // handing it back would re-zoom to the current step and the fit would look
    // like it had failed. Both ways to fit from this box therefore agree, and
    // agree with the button the rail used to carry.
    this.map.addEventListener('dblclick', function (ev) {
      ev.stopPropagation();
      self.resetZoom();
    });

    // The fit control sits inside the box whose every press is "centre the view
    // here", so its own press has to stop before the listeners above see it —
    // otherwise fitting the diagram would first jump the pan somewhere. One
    // stop covers the stage too: a stopped event reaches no ancestor at all, so
    // the stage neither starts a pan nor, in a stepwise transport, advances a
    // beat. Same three events and same reasoning as bindRailGestures.
    ['pointerdown', 'click', 'dblclick'].forEach(function (type) {
      self.mapFit.addEventListener(type, function (ev) { ev.stopPropagation(); });
    });
  };

  // The picture-in-picture storyboard can be swiped away. Only while it is
  // floating — the phone-present layout, where the stylesheet makes it
  // position:absolute — which is read from the computed style at press time
  // rather than from a duplicated breakpoint: as a panel in the grid it is
  // furniture, and furniture does not fly. A dismissal lasts exactly as long
  // as the context it was made in: it is the boardDismissed field, and every
  // context change — a render, a scenario switch, presenter entered or left,
  // a resize — resets it and asks syncBoard again.
  Player.prototype.bindBoardGestures = function () {
    var self = this;

    function clearInline() {
      self.board.style.transform = '';
      self.board.style.opacity = '';
    }

    // settleOn waits for the board's own transform transition to finish, then
    // settles. Filtered to the board and to transform because the storyboard's
    // crossfade layers bubble their own transitionends through here, and one
    // of those must not spend the listener.
    function settleOn(cls, settle) {
      var onEnd = function (tev) {
        if (tev.target !== self.board || tev.propertyName !== 'transform') return;
        self.board.removeEventListener('transitionend', onEnd);
        self.board.classList.remove(cls);
        settle();
      };
      self.board.classList.add(cls);
      self.board.addEventListener('transitionend', onEnd);
    }

    drag(this.board, {
      start: function () {
        // Only while presenting *and* while the stylesheet floats it — the
        // phone-present layout, where it is position:absolute — read from the
        // computed style at press time rather than from a duplicated
        // breakpoint: as a panel in the grid it is furniture, and furniture
        // does not fly. The mode check is the belt to that brace: a host
        // stylesheet that floats the board for reasons of its own must not
        // arm flick-to-dismiss on a full-size panel.
        if (!self.present) return false;
        if (getComputedStyle(self.board).position !== 'absolute') return false;
      },
      move: function (ev, st) {
        if (!st.dragging) {
          // Horizontal intent only: a small wobble is a tap on its way to the
          // lightbox, and a vertical pull means nothing here. Claiming waits
          // for the verdict — a claimed tap would retarget its click to the
          // board and the stack's tap-for-lightbox would never hear it.
          if (Math.abs(st.dx) < 8 || Math.abs(st.dx) < Math.abs(st.dy)) return;
          st.dragging = true;
          st.claim(ev);
        }
        ev.preventDefault();
        self.board.style.transform = 'translateX(' + st.dx + 'px)';
        // Fade with distance, so the gesture announces what letting go will do.
        self.board.style.opacity = String(
          Math.max(0.25, 1 - Math.abs(st.dx) / ((self.board.offsetWidth || 120) * 1.2)));
      },
      end: function (ev, st) {
        if (!st.dragging) return;
        // The click the release manufactures belonged to the drag, not to the
        // stack's tap-for-lightbox.
        self.armSwallow();
        var w = self.board.offsetWidth || 120;
        if (Math.abs(st.dx) > w * 0.5) {
          // Past the point of no return: fly off in the direction of travel,
          // then hide. The stylesheet owns the flight — is-flinging carries
          // the transition, so duration, easing and the reduced-motion
          // shortening live where every other motion does. Hiding waits for
          // the transitionend, which is necessarily after the release's click:
          // even a 0.01ms transition settles on a later frame, so the click
          // finds the board still there and the swallower takes it.
          settleOn('is-flinging', function () {
            self.boardDismissed = true;
            self.syncBoard();
            clearInline();
          });
          self.board.style.transform =
            'translateX(' + ((st.dx < 0 ? -1 : 1) * (w * 2 + 40)) + 'px)';
          self.board.style.opacity = '0';
        } else {
          // Not far enough: spring back.
          settleOn('is-springing', function () { /* home again */ });
          clearInline();
        }
      },
      cancel: function () {
        clearInline();
      }
    });
  };

  // --- auto-follow camera ------------------------------------------------
  //
  // It runs wherever `follow` is — a reel, or anywhere the reader pressed
  // Cine — and nowhere else: by default a page shows the whole diagram and
  // keeps the reader's own hands on the zoom, and only a reel, which is
  // nothing but framing, follows without being asked.
  //
  // The camera frames each step's action: one pose per step, precomputed from
  // the timeline, with a short glide at each step boundary. The pose is a
  // remap of the clock, evaluated at t, never integrated between frames — so
  // scrubbing to a moment shows exactly what playing to it would, and a frame
  // `record` photographs mid-glide is pixel-identical to live playback. That
  // is the same doctrine as flow easing, and it is why there is no CSS
  // transition on the holder and never should be.
  //
  // Persistent set/gauge state does not steer the camera: it is scenario-
  // scoped background fact, not the step's action.

  var CAM_MARGIN = 0.85; // headroom around the framed rect
  var CAM_MAX = 2.5;     // an automatic move should never reach ZOOM_MAX
  var CAM_SNAP = 0.02;   // a computed zoom this close to fit holds fit
  var CAM_TRANS = 500;   // ms; capped at half the step so short beats glide too

  // cameraKeys measures one target rect per step in holder-local CSS px and
  // turns each into a pose. Client rects are used deliberately: they already
  // compose mermaid's inner transforms and the SVG's CSS scaling, so dividing
  // out our own holder transform is the whole coordinate story.
  Player.prototype.cameraKeys = function () {
    if (!this.holder || !this.svg) return null;
    var sc = this.scenario();
    var stageR = this.stage.getBoundingClientRect();
    var holderR = this.holder.getBoundingClientRect();
    if (!stageR.width || !stageR.height) return null;

    var o = this.holderOrigin(stageR, holderR);
    var W = stageR.width, H = stageR.height;
    var fit = { cx: W / 2 - o.x, cy: H / 2 - o.y, z: 1 };

    var self = this;
    function localRect(el) {
      return self.localRect(el, holderR);
    }
    function anchor(id) {
      return (self.anchors && self.anchors[id]) || self.elementFor(id);
    }
    function union(a, b) {
      if (!b) return a;
      if (!a) return b;
      var x = Math.min(a.x, b.x), y = Math.min(a.y, b.y);
      return { x: x, y: y,
               w: Math.max(a.x + a.w, b.x + b.w) - x,
               h: Math.max(a.y + a.h, b.y + b.h) - y };
    }

    // A step's `focus` tracks alone define its rect when present — that is
    // the authored "look here". Otherwise every spatial target contributes.
    // In sequence mode a flow uses the message path alone: it already spans
    // the two lifelines, and unioning the top-anchored actor boxes would pin
    // the camera to the top of the page.
    //
    // That pin is also why a participant is never framed by its element in
    // sequence mode while a message is in flight: the element is the actor
    // box at the top of the page, but what the step is about is that actor's
    // column at the height of this step's messages — the same place
    // anchorRect puts a note. So a participant's rect is its column cut to
    // the flows' vertical span, and a focused participant still shows the
    // messages it is busy with. Without a flow in the step there is nothing
    // to cut to, and the box itself is the only honest answer.
    function rectOf(step) {
      var rect = null, flows = null, focused = false, i, tr, e;
      for (i = 0; i < step.tracks.length; i++) {
        tr = step.tracks[i];
        if (tr.kind !== 'flow') continue;
        e = tr.edge && self.edges[tr.edge];
        if (e && e.path) flows = union(flows, localRect(e.path));
      }
      function targetRect(id) {
        var el = anchor(id);
        if (!el) return null;
        var r = localRect(el);
        if (self.sequence && flows) return { x: r.x, y: flows.y, w: r.w, h: flows.h };
        return r;
      }
      for (i = 0; i < step.tracks.length; i++) {
        tr = step.tracks[i];
        if (tr.kind === 'focus' && tr.target) {
          if (!focused) { rect = null; focused = true; }
          rect = union(rect, targetRect(tr.target));
        }
      }
      if (focused) return self.sequence ? union(rect, flows) : rect;
      rect = flows;
      for (i = 0; i < step.tracks.length; i++) {
        tr = step.tracks[i];
        if (tr.kind === 'scene') continue;
        if (tr.kind === 'flow') {
          if (!self.sequence) {
            var from = tr.from && anchor(tr.from);
            var to = tr.to && anchor(tr.to);
            if (from) rect = union(rect, localRect(from));
            if (to) rect = union(rect, localRect(to));
          }
        } else if (tr.target) {
          rect = union(rect, targetRect(tr.target));
        }
      }
      return rect;
    }

    var poses = [];
    var prev = fit;
    for (var s = 0; s < sc.steps.length; s++) {
      var rect = rectOf(sc.steps[s]);
      if (!rect) {
        // No spatial targets (a storyboard beat, say): hold the previous
        // pose. The shared reference is what lets cameraApply skip the glide.
        poses.push(prev);
        continue;
      }
      var zT = CAM_MARGIN * Math.min(W / Math.max(1, rect.w), H / Math.max(1, rect.h));
      var z = clamp(zT, 1, CAM_MAX);
      var pose = z <= 1 + CAM_SNAP
        ? fit
        : { cx: rect.x + rect.w / 2, cy: rect.y + rect.h / 2, z: z };
      poses.push(pose);
      prev = pose;
    }

    this.camKeys = { o: o, W: W, H: H, poses: poses };
    return this.camKeys;
  };

  // cameraApply evaluates the pose at t and writes the transform. It is
  // called from inside apply(), so it must never call applyTransform — the
  // style write alone is enough, because everything positioned from client
  // rects is laid out later in the same apply() pass (chips excepted, which
  // apply() refreshes when _camMoved says the camera wrote).
  Player.prototype.cameraApply = function (t) {
    var K = this.camKeys || this.cameraKeys();
    if (!K || !K.poses.length) return;
    var steps = this.scenario().steps;

    var i = 0;
    for (var s = 0; s < steps.length; s++) {
      if (steps[s].start <= t) i = s; else break;
    }

    var cur = K.poses[i];
    var pose = cur;
    var prevP = i > 0 ? K.poses[i - 1] : cur;
    if (prevP !== cur) {
      var w = prefersReducedMotion()
        ? 0
        : Math.min(CAM_TRANS, (steps[i].end - steps[i].start) / 2);
      var u = w > 0 ? clamp((t - steps[i].start) / w, 0, 1) : 1;
      if (u < 1) {
        // Cosine in-out; zoom interpolates in log space so an equal-ratio
        // move reads uniform instead of lurching at the wide end.
        var e = (1 - Math.cos(Math.PI * u)) / 2;
        pose = {
          z: Math.exp((1 - e) * Math.log(prevP.z) + e * Math.log(cur.z)),
          cx: prevP.cx + e * (cur.cx - prevP.cx),
          cy: prevP.cy + e * (cur.cy - prevP.cy)
        };
      }
    }

    var z = pose.z;
    var oldX = this.panX, oldY = this.panY;
    this.centreOnLocal(pose.cx, pose.cy, z, K);
    if (Math.abs(z - this.zoom) < 1e-4 &&
        Math.abs(this.panX - oldX) < 1e-4 &&
        Math.abs(this.panY - oldY) < 1e-4) {
      // Close enough to be the same pose: put the exact values back, so a
      // glide that has settled stops writing rather than creeping by epsilons.
      this.panX = oldX;
      this.panY = oldY;
      return;
    }

    this.zoom = z;
    this.setTransform();
    this._camMoved = true;
  };

  // isEmbedded reads the query string rather than the hash, so it survives the
  // hash being rewritten by navigation.
  //
  // A host that mounts the player itself has no URL to say it with, so opts
  // answers first: inline implies embedded, since inline is embed with a
  // transport put back.
  function isEmbedded(opts) {
    if (opts && (opts.embed || opts.inline)) return true;
    return /(^|[?&])embed(=|&|$)/.test(location.search);
  }

  // isPresenter is the same trick for `?present`: a flag that says how the page
  // should behave, kept out of the hash so navigating between views cannot
  // silently drop it.
  function isPresenter(opts) {
    if (opts && opts.present) return true;
    // A detached player must not read the host's query string either: `?present`
    // meant for the page around it is not an instruction to a diagram inside it.
    if (opts && opts.hash === false) return false;
    return /(^|[?&])present(=|&|$)/.test(location.search);
  }

  // isReel is the same trick again for `?reel`, with the same two guards.
  function isReel(opts) {
    if (opts && opts.reel) return true;
    if (opts && opts.hash === false) return false;
    return /(^|[?&])reel(=|&|$)/.test(location.search);
  }

  // copyLink puts the deep link on the clipboard and says so beside the button
  // itself — a toast for a two-word confirmation would be more chrome than the
  // thing it is confirming.
  Player.prototype.copyLink = function () {
    var self = this;
    var link = this.shareLink();
    var done = function (ok) {
      self.flashShare(ok ? 'Copied' : 'Press ⌘C');
    };

    // The async clipboard API needs a secure context, which a page opened from
    // the filesystem is not. Falling back to a selection at least leaves the
    // link somewhere the reader can copy it from by hand.
    if (navigator.clipboard && navigator.clipboard.writeText) {
      navigator.clipboard.writeText(link).then(function () { done(true); }, function () { self.selectLink(link, done); });
      return;
    }
    this.selectLink(link, done);
  };

  // flashShare says "Copied" for a moment. The word goes in a label anchored to
  // the button rather than into the button, which on an icon in a rail would
  // resize the column out from under the cursor that just clicked it; the
  // aria-label changes too, because the old textContent swap was what a screen
  // reader heard. The timer is single: two clicks in quick succession must not
  // leave the first one's restore to fire over the second one's word.
  Player.prototype.flashShare = function (text) {
    var self = this;
    this.shareNote.textContent = text;
    this.shareNote.classList.add('is-on');
    this.shareBtn.setAttribute('aria-label', text);
    if (this._shareTimer) clearTimeout(this._shareTimer);
    this._shareTimer = setTimeout(function () {
      self._shareTimer = null;
      // The text stays behind as the label fades: clearing it would blank the
      // word a frame into a transition that is meant to carry it out.
      self.shareNote.classList.remove('is-on');
      self.shareBtn.setAttribute('aria-label', SHARE_LABEL);
    }, 1400);
  };

  Player.prototype.selectLink = function (link, done) {
    var box = document.createElement('input');
    box.className = 'dgm-linkbox';
    box.value = link;
    this.root.appendChild(box);
    box.select();
    var ok = false;
    try { ok = document.execCommand('copy'); } catch (e) { ok = false; }
    if (ok) box.remove();
    else {
      // The box stays selected so the reader can copy by hand — but only
      // while they are actually using it. Leaving, or Escape, dismisses it
      // rather than holding keyboard focus for a fixed eight seconds.
      box.addEventListener('blur', function () { box.remove(); });
      box.addEventListener('keyup', function (ev) { if (ev.key === 'Escape') box.remove(); });
      setTimeout(function () { box.remove(); }, 8000);
    }
    done(ok);
  };

  // parseHash reads the location hash in either form it may take.
  //
  // The long form is `#v=<view>&s=<scenario>&t=<ms>`, which is what a shared
  // link carries. The short form is a bare view id, which is what navigate()
  // writes for an ordinary drill-in and what older links contain — a hash with
  // no `=` in it is unambiguously that, so both keep working.
  function parseHash() {
    var raw = (location.hash || '').replace(/^#/, '');
    if (!raw) return {};
    if (raw.indexOf('=') < 0) return { v: decodeURIComponent(raw) };

    var out = {};
    raw.split('&').forEach(function (pair) {
      var eq = pair.indexOf('=');
      if (eq < 0) return;
      out[pair.slice(0, eq)] = decodeURIComponent(pair.slice(eq + 1));
    });
    return out;
  }

  // setTheme switches the palette from outside.
  //
  // A page discovers its theme, by reading the root element or asking the
  // system. A host does not have to be asked: an editor that has just gone
  // light knows, and nothing else on the page will tell the player. Without
  // this a diagram embedded in a document keeps the palette it was mounted with
  // for as long as its element survives, which is until the block's own text
  // changes.
  //
  // It is also the one way in: the attribute watcher in build() lands here too,
  // and the guard below is what keeps a player's own mirror-write from coming
  // back round through it.
  Player.prototype.setTheme = function (kind) {
    if (kind !== 'dark' && kind !== 'light') return;
    if (this.theme === kind) return;

    this.theme = kind;
    if (this.hostTheme) this.hostTheme = kind;
    // Only where nothing else writes it — see build(). On a page with chrome of
    // its own the attribute is what *told* us, and writing it back would stamp
    // the page of a reader who asked to follow the system.
    if (!this.pageChrome) document.documentElement.setAttribute('data-theme', kind);
    // mermaid's theme is chosen per render, so the diagram has to be redrawn
    // rather than merely restyled.
    this.render();
  };

  // usesHash reports whether this player owns the address bar.
  //
  // Exactly one player on a page can, so a host that mounts several says no and
  // gets the same navigation kept inside each player instead.
  Player.prototype.usesHash = function () { return this.opts.hash !== false; };

  // readHash is parseHash for a player that may not own the address.
  //
  // A detached player must read no fragment at all rather than the host's own —
  // a Markdown preview puts scroll anchors there, and a player treating one as
  // a view id would jump somewhere the reader never asked to go.
  Player.prototype.readHash = function () {
    return this.usesHash() ? parseHash() : {};
  };

  // hashView is the view id the current URL selects, defaulting to the root.
  Player.prototype.hashView = function () {
    var id = this.readHash().v;
    if (id && this.viewIndexOf(id) >= 0) return id;
    return this.timeline.root;
  };

  // applyHash moves to whatever the URL now names, keeping the back stack in
  // step: returning to the view we came from pops it, anything else pushes the
  // view being left.
  //
  // A link that also names a scenario and a time lands paused at that moment,
  // which is the whole point of being able to share one.
  Player.prototype.applyHash = function () {
    var want = this.readHash();
    this.enter(this.hashView());
    this.applyDeepLink(want);
  };

  // enter moves to a view and keeps the back stack in step: returning to the
  // view we came from pops it, anything else pushes the view being left.
  //
  // Split out of applyHash because a detached player performs the same move
  // without a URL to read it from — the stack is the part that is really doing
  // the work, and the hash was only ever how the page spelled it.
  Player.prototype.enter = function (id) {
    if (id === this.view().id) return;
    if (this.stack[this.stack.length - 1] === id) this.stack.pop();
    else this.stack.push(this.view().id);
    this.setView(id);
  };

  // applyDeepLink honours the scenario and time a long-form hash carries.
  Player.prototype.applyDeepLink = function (want) {
    if (!want) return;
    var moved = false;

    if (want.s) {
      var i = this.scenarioIndexOf(want.s);
      if (i >= 0 && i !== this.scenarioIndex) {
        this.picker.value = String(i);
        this.selectScenario(i);
        moved = true;
      }
    }
    if (want.t !== undefined) {
      var ms = parseInt(want.t, 10);
      if (!isNaN(ms)) {
        // Arriving at a named moment and then playing straight past it would
        // defeat the link, so a deep link always lands paused.
        this.pause();
        this.pendingAutoplay = false;
        this.seek(ms);
        moved = true;
      }
    }
    return moved;
  };

  Player.prototype.scenarioIndexOf = function (id) {
    var scenarios = this.view().scenarios;
    for (var i = 0; i < scenarios.length; i++) {
      if (scenarios[i].id === id) return i;
    }
    return -1;
  };

  // navigate drills into another view. It only moves the hash; applyHash does
  // the work, so a click and a browser history move follow the same path.
  Player.prototype.navigate = function (id) {
    if (this.viewIndexOf(id) < 0 || id === this.view().id) return;
    if (!this.usesHash()) {
      // No address to move, so make the move directly. Back still works: it is
      // the stack, not the history, that remembers where the reader came from.
      this.enter(id);
      return;
    }
    location.hash = id === this.timeline.root ? '' : '#' + encodeURIComponent(id);
  };

  // shareLink is the address of exactly what is on screen: this view, this
  // scenario, this moment.
  Player.prototype.shareLink = function () {
    return pageBase() + '#' + this.hashParts().concat(['t=' + Math.round(this.time)]).join('&');
  };

  // hashParts is what the address should say about the page, moment aside.
  Player.prototype.hashParts = function () {
    return [
      'v=' + encodeURIComponent(this.view().id),
      's=' + encodeURIComponent(this.scenario().id || '')
    ];
  };

  // pageBase is the address without its fragment.
  //
  // Built from href rather than origin + pathname: a page opened off the
  // filesystem (or in a sandboxed iframe) has the literal origin "null", which
  // would produce a link starting with the word null.
  function pageBase() {
    var base = location.href;
    var cut = base.indexOf('#');
    return cut >= 0 ? base.slice(0, cut) : base;
  }

  // syncHash rewrites the address to name the scenario now showing.
  //
  // It *replaces* rather than pushes, unlike navigate(): drilling into another
  // view is navigation and Back should undo it, where choosing a scenario is
  // changing a setting on the page you are already on. Replacing also fires no
  // hashchange, so this cannot re-enter applyHash and re-select what the reader
  // just picked.
  //
  // No `t`: the moment is what Copy link is for, and a hash that froze the time
  // at the instant of the pick would send anyone reloading to a stopped clock.
  //
  // Only a *human* picking from the selector calls this, which is why it hangs
  // off the change listener rather than off selectScenario. selectScenario also
  // runs while a deep link is being applied, and a hash written there would
  // overwrite the `t=` of the very link being followed — replaceState at that
  // moment destroys the shared moment in the address bar and in history both,
  // so the link would survive being opened but not being reloaded.
  Player.prototype.syncHash = function () {
    if (!this.usesHash()) return;
    var want = '#' + this.hashParts().join('&');
    if (want === location.hash) return;
    try {
      history.replaceState(null, '', pageBase() + want);
    } catch (e) {
      // Some sandboxed iframes forbid replaceState. A pushed hash is still
      // better than an address that has stopped describing the page.
      location.hash = want;
    }
  };

  // buildPicker fills the scenario selector for the current view, hiding it
  // when the view offers no choice.
  Player.prototype.buildPicker = function () {
    var self = this;
    var scenarios = this.view().scenarios;
    this.picker.innerHTML = '';
    scenarios.forEach(function (s, i) {
      var o = document.createElement('option');
      o.value = String(i);
      // A failure path is worth spotting before you pick it. The glyph is the
      // one the fail mark already uses, so the two read as the same idea.
      o.textContent = (s.name || 'scenario ' + (i + 1)) + (s.outcome === 'fail' ? ' ✕' : '');
      self.picker.appendChild(o);
    });
    this.picker.value = String(this.scenarioIndex);
    this.picker.style.display = scenarios.length > 1 ? '' : 'none';
  };

  Player.prototype.selectScenario = function (i) {
    this.scenarioIndex = i;
    this.time = this.restingTime();
    this.stopAt = null;
    this.adoptScenarioSpeed();
    this.buildSteps();
    // A new scenario is a new context; a dismissal does not carry into it.
    this.boardDismissed = false;
    this.syncBoard();
    this.apply(this.time);
    this.syncChrome();
  };

  // setView switches the stage to another diagram. Call navigate() rather than
  // this: the hash has to move too, or the URL stops describing the page.
  Player.prototype.setView = function (id) {
    var next = this.viewIndexOf(id);
    if (next < 0 || next === this.viewIndex) return;

    this.pause();
    this.viewIndex = next;
    this.scenarioIndex = 0;
    this.time = this.restingTime();
    this.stopAt = null;
    this.revealed = {};
    this.svg = null;
    this.pendingAutoplay = true;
    // A zoom that framed one diagram means nothing over the next — and
    // neither does a manual override of the camera.
    this.zoom = 1;
    this.panX = 0;
    this.panY = 0;
    this.camOverride = false;
    this.camKeys = null;
    this.mapKeys = null;

    this.buildPicker();
    this.adoptScenarioSpeed();
    this.render();
  };

  // back retraces one step. Going through history rather than straight to the
  // view keeps the forward button working.
  Player.prototype.back = function () {
    if (!this.stack.length) return;
    // A detached player must not touch the host's history — going back in a
    // Markdown preview would leave the document, not the diagram.
    if (!this.usesHash()) {
      this.enter(this.stack[this.stack.length - 1]);
      return;
    }
    history.back();
  };

  Player.prototype.syncNav = function () {
    this.backBtn.style.display = this.stack.length ? '' : 'none';
    this.crumb.textContent = this.stack.length
      ? this.stack.map(titleOf, this).concat([titleOf.call(this, this.view().id)]).join(' › ')
      : '';
  };

  // titleOf names a view for the breadcrumb, falling back to its id.
  function titleOf(id) {
    var i = this.viewIndexOf(id);
    var v = i >= 0 ? this.timeline.views[i] : null;
    return (v && (v.title || v.id)) || id;
  }

  Player.prototype.render = function () {
    var self = this;
    var v = this.view();
    var d = v.diagram || { mermaid: '' };

    this.title.textContent = v.title || this.timeline.title || 'Cinegram';
    this.syncNav();

    window.mermaid.initialize({
      startOnLoad: false,
      theme: this.theme === 'dark' ? 'dark' : 'default',
      securityLevel: 'loose',
      flowchart: { curve: 'basis', htmlLabels: true, padding: 12 }
    });

    // A fresh id each render keeps mermaid from reusing stale definitions.
    var id = 'dgm-svg-' + Math.floor(Math.random() * 1e9);
    var drawn = false;
    window.mermaid
      .render(id, d.mermaid)
      .then(function (out) {
        drawn = true;
        var holder = el('div', 'dgm-svg-holder');
        holder.innerHTML = out.svg;
        // Emptying the stage detaches the rail, and detaching the focused
        // element drops focus to <body>. A render is often something a rail
        // button just did — the theme button calls render() out of its own
        // handler — so a keyboard reader would press Enter on Dark and find
        // the focus gone, with no way back to the button that would undo it.
        // The rail is re-appended rather than rebuilt, so the same node is
        // still focusable afterwards.
        var refocus = document.activeElement;
        self.stage.innerHTML = '';
        self.stage.appendChild(holder);
        self.stage.appendChild(self.overlay);
        self.stage.appendChild(self.map);
        self.stage.appendChild(self.rail);
        if (refocus && self.rail.contains(refocus)) refocus.focus();
        self.holder = holder;
        self.mapClone = null;
        self.mapKeys = null;
        // The style write alone, not applyTransform: the id→element binds
        // still point into the svg the innerHTML above just detached, and
        // applyTransform would re-run apply() over them — an active flow then
        // asks a non-rendered path for getTotalLength, which throws. The full
        // apply comes a few lines down, after index() has rebound everything.
        self.setTransform();

        self.svg = holder.querySelector('svg');
        if (self.svg) {
          self.svg.removeAttribute('width');
          self.svg.removeAttribute('height');
          self.svg.style.maxWidth = '100%';
          self.svg.style.height = 'auto';
        }
        // Clone for the map before index() and apply() touch the live SVG, so
        // the thumbnail is the diagram at rest rather than mid-animation.
        self.buildMap();
        self.index();
        self.buildSteps();
        // A fresh render is a fresh context; a dismissal does not survive it.
        self.boardDismissed = false;
        self.syncBoard();
        self.apply(self.time);
        self.syncChrome();
        // Last, because buildSteps and the caption have both had a chance to
        // change the shape of the page since buildMap measured it: a scenario
        // with a taller step list leaves the stage a different size, and the
        // map is a share of the stage.
        self.syncMap();

        // Autoplay waits for a successful render: starting the clock over a
        // diagram mermaid failed to draw would just run it out invisibly.
        // A reel never autoplays — it opens at rest and each tap is a beat.
        if (self.pendingAutoplay && !self.reel) {
          self.pendingAutoplay = false;
          self.maybeAutoplay();
        }
      })
      .catch(function (err) {
        // The catch spans the whole chain, so name the half that failed:
        // blaming mermaid for an exception in the runtime's own post-render
        // work sent one debugging session entirely the wrong way.
        self.warn([(drawn
          ? 'Displaying the diagram failed: '
          : 'Mermaid failed to render the diagram: ') + err]);
      });
  };

  // makeLayer adds a group for particles above everything mermaid drew.
  // Without it particles are painted under g.nodes and their labels disappear
  // behind the node they are travelling towards.
  function makeLayer(svg) {
    var layer = document.createElementNS(SVG_NS, 'g');
    layer.setAttribute('class', 'dgm-layer');
    var edgePaths = svg.querySelector('.edgePaths');
    var host = (edgePaths && edgePaths.parentNode) || svg;
    host.appendChild(layer);
    return layer;
  }

  Player.prototype.index = function () {
    if (!this.svg) return;
    var v = this.view();

    // Indexing strategy is chosen by diagram type. Everything after this
    // point — states, flows, notes, pills, chips — works off the same three
    // maps, which is what lets a further diagram type cost one indexer instead
    // of a second runtime.
    //
    // There are two strategies, not three. A sequence diagram has neither
    // g.node nor .edgePaths and needs its own; a state diagram has both, so it
    // reuses the flowchart's — differing only in mermaid's id prefix and in
    // being able to match an edge that lands on a composite rather than a node.
    var type = (v.diagram && v.diagram.type) || '';
    this.sequence = type === 'sequenceDiagram';
    this.anchors = {};

    if (this.sequence) {
      this.layer = makeLayer(this.svg);
      this.clusters = {};
      this.nodes = indexActors(this.svg, v, this.anchors);
      this.edges = indexMessages(this.svg, v, this.anchors);
    } else if (type === 'stateDiagram') {
      this.nodes = indexStateNodes(this.svg);
      this.clusters = indexClusters(this.svg, v);
      this.layer = makeLayer(this.svg);
      // The cluster-aware lookup goes only to the state branch, so flowchart
      // edge matching stays byte-identical to what it has always done.
      this.edges = indexEdges(this.svg, v, withClusters(this.nodes, this.clusters));
    } else {
      this.nodes = indexNodes(this.svg);
      this.clusters = indexClusters(this.svg, v);
      this.layer = makeLayer(this.svg);
      this.edges = indexEdges(this.svg, v, this.nodes);
    }

    // Cache each path's transform into the overlay's coordinate system, so
    // positions stay exact even if a future mermaid release puts a transform
    // on the edge group.
    for (var id in this.edges) {
      this.edges[id].matrix = relativeMatrix(this.layer, this.edges[id].path);
    }
    this.particles = {};
    this.notes = {};
    // Pills live in the overlay, which is emptied just below, so the map has to
    // go with them or the next frame would reuse detached nodes.
    this.pills = {};
    this.nodeState = {};
    this.edgeState = {};
    this.overlay.innerHTML = '';

    // Surface anything that failed to bind. A silently half-animated diagram
    // is far more confusing than an explicit list of what could not be found.
    var problems = [];
    var self = this;
    // Resolved through elementFor, not this.nodes, because a name can be both.
    // Writing an edge into a composite state mentions the composite, which
    // registers a node placeholder beside the group of the same name — and the
    // thing mermaid actually drew for it is the cluster.
    v.nodes.forEach(function (n) {
      if (!self.elementFor(n.id)) problems.push('node "' + n.id + '" not found in the rendered SVG');
    });
    v.edges.forEach(function (e) {
      if (!self.edges[e.id]) problems.push('edge ' + e.from + ' → ' + e.to + ' could not be matched to a path');
    });

    // Click handlers go on elements inside the SVG, which render() replaces
    // wholesale on every view switch and theme change — so they are discarded
    // with it and never need removing.
    (v.bindings || []).forEach(function (b) {
      var target = self.elementFor(b.source);
      if (!target) {
        problems.push('clickable "' + b.source + '" not found in the rendered SVG');
        return;
      }
      target.setAttribute('class', baseClass(target) + ' dgm-clickable');
      if (b.label) tooltip(target, b.label);

      // Reachable by keyboard, and announced as the control it behaves like.
      // An SVG group is not focusable or actionable by default, so every part
      // of that has to be said explicitly.
      target.setAttribute('tabindex', '0');
      target.setAttribute('role', 'button');
      target.setAttribute('aria-label', b.label || describeBinding(b));

      target.addEventListener('click', function (ev) {
        ev.preventDefault();
        ev.stopPropagation();
        self.activate(b);
      });
      target.addEventListener('keydown', function (ev) {
        if (ev.key !== 'Enter' && ev.key !== ' ') return;
        ev.preventDefault();
        ev.stopPropagation();
        self.activate(b);
      });
    });

    this.buildChips();
    this.applyHidden();
    this.warn(problems);
  };

  // describeBinding is the fallback accessible name for a clickable element
  // that the author gave no label.
  function describeBinding(b) {
    if (b.kind === 'view') return 'Open the ' + (b.view || 'linked') + ' view';
    if (b.kind === 'url') return 'Open ' + (b.url || 'link') + ' in a new tab';
    if (b.kind === 'step') return 'Jump to step ' + (b.step || '');
    return 'Reveal hidden detail';
  }

  // elementFor resolves a binding source, which may name a node or a subgraph.
  Player.prototype.elementFor = function (id) {
    return this.nodes[id] || this.clusters[id] || null;
  };

  // activate runs what a click was bound to.
  Player.prototype.activate = function (b) {
    var self = this;
    if (b.kind === 'view') {
      this.navigate(b.view);
      return;
    }
    if (b.kind === 'url') {
      // noopener, because the opened page must not get a handle on this one.
      if (b.url) window.open(b.url, '_blank', 'noopener');
      return;
    }
    if (b.kind === 'step') {
      var steps = this.scenario().steps;
      for (var i = 0; i < steps.length; i++) {
        if (steps[i].id === b.step) {
          this.pause();
          this.seek(steps[i].start);
          return;
        }
      }
      return;
    }
    if (b.kind === 'reveal') {
      // Toggle, so a click both opens and closes the detail.
      (b.targets || []).forEach(function (id) {
        if (self.revealed[id]) delete self.revealed[id];
        else self.revealed[id] = true;
      });
      this.applyHidden();
    }
  };

  // applyHidden conceals every element that a reveal binding has not yet been
  // clicked to open, along with the edges that would otherwise dangle.
  Player.prototype.applyHidden = function () {
    var self = this;
    var v = this.view();
    var hidden = v.hidden || [];

    if (hidden.length) {
      // A group's children are listed in `hidden` individually, so testing
      // membership here is enough to decide an element's own state.
      var conceal = {};
      hidden.forEach(function (id) {
        if (!self.revealed[id]) conceal[id] = true;
      });

      hidden.forEach(function (id) {
        var element = self.elementFor(id);
        if (element) setCollapsed(element, !!conceal[id]);
      });

      v.edges.forEach(function (e) {
        var bind = self.edges[e.id];
        if (bind) setCollapsed(bind.path, !!(conceal[e.from] || conceal[e.to]));
      });
    }

    // The chips advertise what is still folded away, so they move with it.
    this.syncChips();
  };

  // buildChips gives clickable elements a visible affordance.
  //
  // A reveal is worth nothing if nobody knows it is there: the diagram looks
  // finished, and the detail behind it is only found by chance. The chip says
  // how much is hidden before you click, and what you are undoing after.
  Player.prototype.buildChips = function () {
    var self = this;
    this.chips = [];

    (this.view().bindings || []).forEach(function (b) {
      if (b.kind !== 'reveal' && b.kind !== 'view') return;
      var host = (self.anchors && self.anchors[b.source]) || self.elementFor(b.source);
      if (!host) return; // already reported by the binding pass in index()

      var chip = el('button', 'dgm-chip dgm-chip-' + b.kind);
      chip.type = 'button';
      chip.addEventListener('click', function (ev) {
        ev.preventDefault();
        ev.stopPropagation();
        self.activate(b);
      });
      self.overlay.appendChild(chip);
      self.chips.push({ binding: b, el: chip, host: host });
    });

    this.syncChips();
  };

  Player.prototype.syncChips = function () {
    var self = this;
    if (!this.chips || !this.chips.length) return;
    var stageRect = this.stage.getBoundingClientRect();

    this.chips.forEach(function (c) {
      var r = c.host.getBoundingClientRect();
      c.el.style.left = (r.right - stageRect.left) + 'px';
      c.el.style.top = (r.bottom - stageRect.top) + 'px';

      if (c.binding.kind === 'view') {
        c.el.textContent = '⤢';
        c.el.title = c.binding.label || 'Open this view';
        c.el.setAttribute('aria-label', c.el.title);
        return;
      }

      var folded = (c.binding.targets || []).filter(function (id) {
        return !self.revealed[id];
      }).length;
      c.el.textContent = folded ? '+' + folded : '–';
      c.el.title = folded ? 'Reveal ' + folded + ' hidden element' + (folded === 1 ? '' : 's')
                          : 'Fold this detail away again';
      c.el.setAttribute('aria-label', c.el.title);
      c.el.classList.toggle('is-open', !folded);
    });
  };

  function setCollapsed(element, on) {
    var base = baseClass(element).replace(/\s*dgm-collapsed\b/g, '');
    element.setAttribute('class', on ? base + ' dgm-collapsed' : base);
  }

  // tooltip gives an SVG element a native hover label.
  function tooltip(element, text) {
    var t = element.querySelector(':scope > title');
    if (!t) {
      t = document.createElementNS(SVG_NS, 'title');
      element.insertBefore(t, element.firstChild);
    }
    t.textContent = text;
  }

  Player.prototype.warn = function (problems) {
    if (!problems || !problems.length) {
      this.warning.style.display = 'none';
      return;
    }
    this.warning.style.display = '';
    this.warning.textContent = 'Animation binding problems: ' + problems.join('; ');
  };

  Player.prototype.buildSteps = function () {
    var self = this;
    var sc = this.scenario();
    this.steps.innerHTML = '';
    this.stepEls = [];
    sc.steps.forEach(function (st, i) {
      var li = el('li', 'dgm-step-row');
      // A real button, not a click handler on an <li>: it is a control, and
      // making it one is what puts it in the tab order and gives it Enter and
      // Space without any of that being reimplemented here.
      var btn = el('button', 'dgm-step');
      btn.type = 'button';
      btn.appendChild(elText('span', 'dgm-step-name', st.name || st.id));
      btn.appendChild(elText('span', 'dgm-step-time', fmt(st.start) + ' – ' + fmt(st.end)));
      // The list stays a compact index; the first line of the prose is enough
      // to tell two similarly named steps apart on hover.
      if (st.desc) btn.title = firstLine(st.desc);
      btn.addEventListener('click', function () { self.seek(st.start); });
      li.appendChild(btn);
      self.steps.appendChild(li);
      self.stepEls.push(btn);
    });

    this.scrub.max = String(sc.duration || 0);
    this.buildScrubMarks(sc);
    this.buildReelBar(sc);

    // The caption belongs to whichever scenario is showing, so a switch has to
    // let it redraw even if the new step happens to carry the same id.
    this.captionKey = null;

    // The buttons syncChrome was following are gone with the list; the next
    // pass finds the new one and scrolls to it.
    this._activeStepEl = null;

    // Camera keyframes are per (view, scenario, layout); this runs on every
    // render and scenario switch, which covers all three. The map's keys are
    // per layout alone, but the step list this just rebuilt is part of it: a
    // scenario with more steps than the last takes a taller column.
    this.camKeys = null;
    this.mapKeys = null;
  };

  // buildScrubMarks lays a tick over the scrubber at each step boundary. They
  // are real buttons: the scrubber tells you where you are, and the ticks tell
  // you where the beats are and take you to one.
  Player.prototype.buildScrubMarks = function (sc) {
    var self = this;
    this.marks.innerHTML = '';
    var duration = sc.duration || 0;
    if (!duration) return;

    sc.steps.forEach(function (st) {
      var label = st.name || st.id;
      var tick = el('button', 'dgm-scrub-mark');
      tick.type = 'button';
      tick.style.left = (100 * st.start / duration) + '%';
      tick.title = label;
      tick.setAttribute('aria-label', 'Jump to ' + label);
      tick.addEventListener('click', function () {
        self.pause();
        self.seek(st.start);
      });
      self.marks.appendChild(tick);
    });
  };

  // buildReelBar lays one equal-width segment per step, Instagram-style: the
  // bar answers "how many beats, which one am I on", not "how long is each" —
  // proportional widths would render a 300ms hop as an invisible sliver.
  // The fill inside the active segment is written by syncChrome as a plain
  // width with no transition, so it is a pure function of the clock and a
  // frame photographed at any paused millisecond is exact.
  Player.prototype.buildReelBar = function (sc) {
    this.reelbar.innerHTML = '';
    this.reelFills = [];
    for (var i = 0; i < sc.steps.length; i++) {
      var seg = el('div', 'dgm-reelbar-seg');
      var fill = el('div', 'dgm-reelbar-fill');
      seg.appendChild(fill);
      this.reelbar.appendChild(seg);
      this.reelFills.push(fill);
    }
  };

  function firstLine(s) {
    var i = s.indexOf('\n');
    return i < 0 ? s : s.slice(0, i);
  }

  // ---------------------------------------------------------------------
  // Playback
  // ---------------------------------------------------------------------

  Player.prototype.toggle = function () {
    if (this.playing) { this.pause(); return; }
    // In the one-beat transport, Play is the beat key by another name.
    if (this.stepwise()) { this.advanceStep(); return; }
    this.play();
  };

  Player.prototype.play = function () {
    var sc = this.scenario();
    if (this.time >= sc.duration) this.time = 0;
    this.playing = true;
    this.playBtn.textContent = 'Pause';
    this.lastFrame = 0;
    this.loopFrame();
  };

  Player.prototype.pause = function () {
    this.playing = false;
    this.playBtn.textContent = 'Play';
    if (this.raf) cancelAnimationFrame(this.raf);
    this.raf = null;
  };

  // dispose stops the clock and takes back every listener the player put
  // outside its own DOM. A host that mounts a fresh player per edit — the
  // playground, a webview re-rendering a block — needs it: element listeners
  // die with the root the host replaces, but the window, document and
  // matchMedia ones would stack up one set per mount, each of them still
  // holding the player it was built for.
  //
  // documentElement's `data-theme` is deliberately left as it is. It describes
  // the page, not this player, and clearing it would make the whole document
  // flash back to light between two mounts.
  Player.prototype.dispose = function () {
    this.pause();
    // A host tearing down a presenting player must not leave the page stuck
    // fullscreen on an element it is about to remove.
    if (fullscreenElement() === this.root) exitFull();
    for (var i = 0; i < this._unbind.length; i++) {
      // A remover cannot meaningfully fail, but a host disposing twice or
      // after its document went away should not throw at the caller.
      try { this._unbind[i](); } catch (e) { /* already gone */ }
    }
    this._unbind = [];

    // A pending "Copied" restore is a timer holding this player for another
    // 1400ms, and a host that mounts a fresh one per keystroke — the playground
    // — would keep every dead player the reader had clicked Copy link on.
    if (this._shareTimer) clearTimeout(this._shareTimer);
    this._shareTimer = null;

    // The minimap holds a second copy of the whole diagram. A host that keeps
    // a disposed player around — for its snapshot, say — should not be holding
    // that too, and the clone is the largest thing this object owns.
    if (this.mapBody) this.mapBody.innerHTML = '';
    this.mapClone = null;
    this.mapKeys = null;
  };

  Player.prototype.seek = function (ms) {
    // Clamped at both ends: a shared link can carry any `t=` at all, and a
    // time past the end would leave the clock and the scrubber disagreeing.
    var max = this.scenario().duration || 0;
    this.time = Math.min(max, Math.max(0, ms));
    // A pending presenter stop belonged to the step that was playing; jumping
    // somewhere else abandons it rather than stopping the new run early.
    this.stopAt = null;
    this.apply(this.time);
    this.syncChrome();
  };

  // --- presenter mode ---------------------------------------------------
  //
  // A demo is not a video. Talking over a diagram means playing one beat,
  // stopping, saying what happened, and then playing the next — so in presenter
  // mode Space plays exactly the next step and pauses at its end.
  //
  // The whole mechanism is `stopAt`: a moment in timeline milliseconds that
  // loopFrame checks. Because it lives in the same domain as the clock, the
  // speed multiplier, `seek`, deep links and the scrubber all keep working
  // without knowing presenter mode exists.

  Player.prototype.setPresenter = function (on) {
    this.present = !!on;
    this.root.classList.toggle('dgm-present', this.present);
    this.presentBtn.textContent = this.present ? 'Exit' : 'Present';
    this.presentBtn.classList.toggle('is-on', this.present);
    this.presentBtn.title = this.present
      ? 'Leave presenter mode'
      : 'Step through one beat at a time';

    // Either direction stops the clock. Entering, so the first Space plays the
    // first step rather than being spent halting a run already under way;
    // leaving, so a step that was mid-flight does not carry on to the end of
    // the scenario the moment its stop is dropped.
    this.stopAt = null;
    this.pause();

    // Either direction also changes the layout the camera measured against —
    // the step list and the transport come and go — so the keys are stale, and
    // a frame the reader took by hand in one mode should not be carried into
    // the other.
    this.camKeys = null;
    this.mapKeys = null;
    this.camOverride = false;
    // A dismissal was made in one mode's layout and does not survive into the
    // other: leaving must put the side panel back, and re-entering starts the
    // presentation with its thumbnail.
    if (this.boardDismissed) {
      this.boardDismissed = false;
      this.syncBoard();
    }

    // Presenting into one pane of a split view is not presenting, so the mode
    // asks the browser for the screen. Whether it gets it is not something the
    // caller can promise: this runs on page load for `?present`, with no user
    // gesture, and inside hosts that refuse element fullscreen outright. So
    // `dgm-present-fill` goes on unconditionally rather than as an answer to a
    // refusal: on the fullscreen element its position: fixed resolves against
    // the viewport and its z-index is inert in the top layer, so the box is
    // identical whether the request was granted or not — and there is no async
    // refusal round-trip left to race an enter/exit/enter against.
    var owned = fullscreenElement() === this.root;
    if (this.present) {
      this.root.classList.add('dgm-present-fill');
      // Already the fullscreen element: re-requesting it would be a rejected
      // promise and nothing else.
      if (!owned) requestFull(this.root);
      // The chrome has gone; reframe against what the stage is now. Entering
      // real fullscreen resizes the window too, and the resize and
      // fullscreenchange handlers in build() re-measure again when it lands.
      this.apply(this.time);
      this.syncChips();
      // The map is measured against the stage as well, and nothing here writes
      // the transform — so unlike the leaving branch, which reaches syncMap
      // through resetZoom, this one has to ask for it.
      this.syncMap();
    } else {
      this.root.classList.remove('dgm-present-fill');
      // Only when it is ours. A host with its own fullscreen — a webview, a
      // slide deck embedding the player — keeps it.
      if (owned) exitFull();
      // Leaving hands the whole diagram back. Without this the reader is left
      // staring at the last step's pose with the camera that framed it gone.
      this.resetZoom();
    }
  };

  // advanceStep plays the next beat and stops at its end.
  //
  // "Next" is the first step that has not *started*, which is the rule that
  // makes advancing always move forwards. Sitting exactly at a step's start
  // counts as not started, so the first press plays that step; anywhere inside
  // one, the press skips the rest of it and moves on.
  //
  // The alternative — the first step that has not finished — replays whatever
  // is mid-flight, and that is wrong for the case this control exists for.
  // Someone reaching for the key while a beat is still running is saying "yes,
  // yes, get on with it", and answering that by starting the beat over means
  // they cannot leave it until it has played out in full.
  //
  // Nothing is lost by skipping: a step's end state is a pure function of the
  // time, so seeking to the next step's start shows everything the one being
  // cut short would have finished by.
  Player.prototype.advanceStep = function () {
    // Moving to another beat hands the framing back to the camera: a manual
    // zoom is per-beat inspection, and on a phone — where an accidental drag
    // is easy and a double-click does not exist — the next tap is the only
    // hand-back gesture a viewer will ever find.
    if (this.follow) this.camOverride = false;
    var steps = this.scenario().steps;
    for (var i = 0; i < steps.length; i++) {
      // The millisecond of tolerance is the same one prevStep and nextStep
      // use: the clock rarely lands exactly on a boundary, and a step whose
      // start it has drifted a fraction past is still the step to play.
      if (steps[i].start < this.time - 1) continue;
      this.pause();
      this.seek(steps[i].start); // clears stopAt, so it is set after
      // One millisecond shy of the seam, not on it. A step's end IS the next
      // step's start, and every "which step is this?" answer — the caption,
      // the camera, the step list, the diagram's own per-step state — reads
      // the last step whose start the clock has reached. Pausing exactly on
      // the boundary therefore shows the NEXT step's narration over the beat
      // that just played. Stopping inside the step keeps the whole paused
      // world — notes included — on the beat the presenter is talking about,
      // and stays a pure function of the clock. The lost millisecond is
      // sub-pixel, and the next advance seeks the seam anyway.
      //
      // Only a step with a millisecond to spare, though. The scan above skips
      // a step whose start the clock is already more than a millisecond past —
      // the same tolerance, spent twice — so stopping inside a beat two
      // milliseconds long or shorter would leave the next press selecting this
      // same step again, and again. A beat that short is under one frame, so
      // it stops on its end and the seam costs nothing anyone can see.
      var span = steps[i].end - steps[i].start;
      this.stopAt = steps[i].end - (span > 2 ? 1 : 0);
      this.play();
      return;
    }
    // Past the last step there is nothing to advance into.
  };

  // prevStep goes back to the start of a beat and waits there, so Space then
  // replays it. Backing up and playing again is one mechanism, not two.
  Player.prototype.prevStep = function () {
    // Same hand-back as advanceStep: navigating beats restores the camera.
    if (this.follow) this.camOverride = false;
    var steps = this.scenario().steps;
    var target = 0;
    for (var i = 0; i < steps.length; i++) {
      if (steps[i].start < this.time - 1) target = steps[i].start;
    }
    this.pause();
    this.seek(target);
  };

  Player.prototype.nextStep = function (dir) {
    var sc = this.scenario();
    var steps = sc.steps;
    this.pause();
    for (var i = 0; i < steps.length; i++) {
      if (dir > 0 && steps[i].start > this.time + 1) return this.seek(steps[i].start);
      if (dir < 0 && steps[i].end >= this.time - 1 && steps[i].start < this.time - 1) {
        return this.seek(steps[i].start);
      }
    }
    this.seek(dir > 0 ? sc.duration : 0);
  };

  // adoptScenarioSpeed takes the playback rate the author compiled in. It runs
  // when the selected scenario changes — not on every render — so a theme
  // toggle does not throw away a rate the viewer chose with the button.
  Player.prototype.adoptScenarioSpeed = function () {
    // An authored rate outranks the remembered one: the saved key is written
    // by the sheet's speed menu on ANY diagram on this origin, and letting it
    // shadow a scenario that explicitly declares `speed:` would mean one 0.25x
    // picked on some other diagram permanently overrides every author's
    // pacing. A scenario speed of 1 is indistinguishable from "not declared",
    // so the remembered rate fills exactly that gap.
    var s = this.scenario().speed;
    var declared = (typeof s === 'number' && s > 0 && s !== 1) ? s : 0;
    var saved = parseFloat(prefGet('dgm.speed'));
    if (declared) this.speed = declared;
    else if (!isNaN(saved) && saved > 0) this.speed = saved;
    else this.speed = 1;
    this.syncSpeed();
  };

  // syncSpeed is the one place the sheet's menu is written from this.speed.
  // Everything that moves the rate — an authored `speed:`, the remembered
  // preference, a scenario change — arrives through adoptScenarioSpeed or
  // setSpeed, and both end here.
  Player.prototype.syncSpeed = function () {
    var sel = this.speedSel;
    if (!sel) return;

    // A scenario may declare a rate that is not on the menu: `speed: 0.8`. The
    // button this replaced stepped to the next preset *by value* rather than by
    // index precisely so that such a rate went somewhere sensible instead of
    // snapping to the slowest, and a <select> has no equivalent affordance — an
    // unlisted value leaves the menu showing its first option, so a 0.8x
    // scenario would read as 0.25x and be wrong rather than merely coarse. So
    // the property is kept the other way round: the declared rate joins the
    // menu, in rate order, for exactly as long as it is the rate in force.
    // Choosing a preset drops it again, because nothing is then holding it.
    var i, declared = null;
    for (i = 0; i < sel.options.length; i++) {
      if (sel.options[i].getAttribute('data-dgm-declared')) {
        declared = sel.options[i];
        break;
      }
    }
    if (declared) sel.removeChild(declared);

    var listed = false;
    for (i = 0; i < SPEED_PRESETS.length; i++) {
      if (Math.abs(SPEED_PRESETS[i] - this.speed) < 1e-9) { listed = true; break; }
    }
    if (!listed) {
      var opt = document.createElement('option');
      opt.setAttribute('data-dgm-declared', '1');
      opt.value = String(this.speed);
      opt.textContent = speedLabel(this.speed);
      // In rate order rather than tacked on the end: a menu reading 0.25x,
      // 0.5x, 1x, 1.5x, 2x, 0.8x is a menu nobody can scan. insertBefore(x,
      // null) appends, which is the right answer for a rate above them all.
      var at = null;
      for (i = 0; i < sel.options.length; i++) {
        if (parseFloat(sel.options[i].value) > this.speed) { at = sel.options[i]; break; }
      }
      sel.insertBefore(opt, at);
    }

    sel.value = String(this.speed);
    // The rate is deliberately no longer part of the accessible name. The
    // button this replaced had to carry it, because its label *was* the rate
    // and "1x" read out of context is a number attached to nothing; a select
    // announces its own value, so repeating it in the name would have a screen
    // reader say the rate twice and re-announce the control on every change.
    // What the name has to supply is the word the visible "Speed" leaves to its
    // heading.
    sel.setAttribute('aria-label', 'Playback speed');
  };

  // setSpeed is the reader's own pacing, and the one writer of the remembered
  // key. That key is global to the origin — 0.25x chosen here is 0.25x on every
  // cinegram this browser opens next — which is exactly why adoptScenarioSpeed
  // lets an authored `speed:` outrank it.
  Player.prototype.setSpeed = function (v) {
    if (!(v > 0)) return;
    this.speed = v;
    prefSet('dgm.speed', String(v));
    this.syncSpeed();
  };

  // maybeAutoplay starts playback once a view has actually rendered, unless the
  // author opted out or the viewer's system asks for reduced motion. Autoplay
  // is deliberately outside apply(): it moves the clock, it is not a frame.
  Player.prototype.maybeAutoplay = function () {
    var sc = this.scenario();
    if (this.playing || !sc.autoplay || !sc.duration) return;
    // Presenting means the room waits for the presenter, so the page must not
    // start without one — the same reasoning as reduced motion, arrived at from
    // the other direction.
    if (this.present || prefersReducedMotion()) return;
    this.play();
  };

  function prefersReducedMotion() {
    try {
      return window.matchMedia('(prefers-reduced-motion: reduce)').matches;
    } catch (e) {
      return false;
    }
  }

  // speedLabel prints the effective rate without trailing zeroes: 1x, 1.5x.
  function speedLabel(v) {
    return String(Math.round(v * 100) / 100) + 'x';
  }

  Player.prototype.loopFrame = function () {
    var self = this;
    this.raf = requestAnimationFrame(function (ts) {
      var sc = self.scenario();
      if (self.lastFrame) {
        self.time += (ts - self.lastFrame) * self.speed;
      }
      self.lastFrame = ts;

      // The presenter stop is checked before the end-of-scenario one, so the
      // last step of a looping scenario stops where it was told to instead of
      // wrapping round underneath the presenter.
      if (self.stopAt !== null && self.time >= self.stopAt) {
        self.time = self.stopAt;
        self.stopAt = null;
        self.apply(self.time);
        self.syncChrome();
        self.pause();
        return;
      }

      if (self.time >= sc.duration) {
        if (sc.loop) {
          self.time = sc.duration ? self.time % sc.duration : 0;
        } else {
          self.time = sc.duration;
          self.apply(self.time);
          self.syncChrome();
          self.pause();
          return;
        }
      }
      self.apply(self.time);
      self.syncChrome();
      if (self.playing) self.loopFrame();
    });
  };

  Player.prototype.syncChrome = function () {
    var sc = this.scenario();
    this.scrub.value = String(Math.round(this.time));
    this.clock.textContent = fmt(this.time) + ' / ' + fmt(sc.duration);

    var current = null;
    var kids = this.stepEls || [];
    var activeEl = null;
    for (var i = 0; i < kids.length; i++) {
      var st = sc.steps[i];
      var active = st && this.time >= st.start && this.time < st.end;
      var done = st && this.time >= st.end;
      if (active) { current = st; activeEl = kids[i]; }
      kids[i].classList.toggle('is-active', !!active);
      kids[i].classList.toggle('is-done', !!done);
      kids[i].setAttribute('aria-current', active ? 'step' : 'false');
    }

    // The list scrolls inside the bounded shell, so a scenario with more steps
    // than fit would run the highlight off the bottom and leave the reader
    // watching a stationary index. Only when the active step actually changes:
    // asking sixty times a second would fight a reader scrolling the list by
    // hand. `nearest` then scrolls the least that will do, which is nothing
    // while the step is already in view.
    //
    // And only when the list is the thing that scrolls. In an auto-height host
    // — an inline block in a Markdown preview — the list has no overflow of
    // its own, so the nearest scrollable ancestor is the reader's document,
    // and every beat would drag the page back to the diagram.
    if (activeEl !== this._activeStepEl) {
      this._activeStepEl = activeEl;
      if (activeEl && activeEl.scrollIntoView &&
          this.steps.scrollHeight > this.steps.clientHeight + 1) {
        activeEl.scrollIntoView({ block: 'nearest' });
      }
    }

    // The reel bar's fills ride the same loop shape: done segments read
    // full, future ones empty, the active one fills with its step.
    var fills = this.reelFills || [];
    for (var j = 0; j < fills.length; j++) {
      var s = sc.steps[j];
      if (!s) continue;
      fills[j].style.width =
        (100 * clamp((this.time - s.start) / ((s.end - s.start) || 1), 0, 1)) + '%';
    }

    this.syncCaption(current);
  };

  // syncCaption narrates the step the clock is inside, and blanks between
  // steps or before the start.
  //
  // It diffs on the step rather than rewriting every frame. That is not a
  // rendering optimisation: the caption is an aria-live region, and rewriting
  // it sixty times a second would have a screen reader announce the same
  // sentence continuously for the length of the step.
  Player.prototype.syncCaption = function (step) {
    var key = step ? step.id + '\x00' + (step.name || '') + '\x00' + (step.desc || '') : '';
    if (key === this.captionKey) return;
    this.captionKey = key;

    // The caption is a row of the bounded shell, so a step with three lines of
    // narration where the last had one takes its extra height off the stage.
    // The camera and the map both measured against the stage that was, and
    // this is the only place that height ever changes without a resize.
    var was = this.caption.offsetHeight;

    this.caption.innerHTML = '';
    this.caption.classList.toggle('is-on', !!step);
    if (step) {
      this.caption.appendChild(elText('span', 'dgm-caption-name', step.name || step.id));
      if (step.desc) this.caption.appendChild(elText('span', 'dgm-caption-desc', step.desc));
      // The box scrolls when narration outgrows its cap, and the browser keeps
      // an element's scroll position across a rewrite — so a step read to the
      // bottom would hand the next step over already scrolled past its opening.
      this.caption.scrollTop = 0;
    }

    // Both caches are keyed to a stage of a given size, and it just changed
    // under them. Nulling is all that is needed: each rebuilds itself the next
    // time it is asked for, which is this same frame.
    if (this.caption.offsetHeight !== was) {
      this.camKeys = null;
      this.mapKeys = null;
    }
  };

  // ---------------------------------------------------------------------
  // Frame application
  // ---------------------------------------------------------------------

  // WAYPOINT_LEAD is the tail of a hop during which its destination node
  // announces the arrival. A multi-hop flow is one track per hop, so this makes
  // each intermediate stop legible without the compiler emitting anything extra.
  var WAYPOINT_LEAD = 0.15;

  Player.prototype.apply = function (t) {
    if (!this.svg) return;
    // The camera writes the transform before anything is measured: notes,
    // pills and particles are all positioned from client rects later in this
    // same pass, so they see the framing they will be photographed under.
    if (this.follow && !this.camOverride) this.cameraApply(t);
    var sc = this.scenario();

    var wantNode = {};   // node id -> {cls, color, key}
    var wantEdge = {};   // edge id -> {cls, color, key}
    var wantFlow = {};   // track key -> {track, progress}
    var wantNote = {};   // note key -> {track, target}
    var wantWaypoint = {}; // node id -> true
    var focusRoots = null; // null while no focus track is open this frame

    for (var s = 0; s < sc.steps.length; s++) {
      var step = sc.steps[s];
      if (t < step.start || t > step.end) continue;

      for (var k = 0; k < step.tracks.length; k++) {
        var tr = step.tracks[k];
        // A scene is sticky and so is read outside this scan entirely: it has
        // to survive past its own step, and it paints a panel rather than a
        // diagram element, so it must not fall through to the node states.
        if (tr.kind === 'scene') continue;
        if (t < tr.start || t > tr.end) continue;
        var span = Math.max(1, tr.end - tr.start);
        var p = (t - tr.start) / span;
        var key = s + ':' + k;

        if (tr.kind === 'flow') {
          var progress = Math.min(1, Math.max(0, p));
          wantFlow[key] = { track: tr, progress: progress };
          if (tr.edge) wantEdge[tr.edge] = edgeState(tr);
          if (tr.to && progress >= 1 - WAYPOINT_LEAD) wantWaypoint[tr.to] = true;
        } else if (tr.kind === 'note') {
          wantNote[key] = tr;
          if (!wantNode[tr.target]) wantNode[tr.target] = nodeState('noted', '');
        } else if (tr.kind === 'hide') {
          wantNode[tr.target] = nodeState('hidden', '');
        } else if (tr.kind === 'focus') {
          if (!focusRoots) focusRoots = {};
          focusRoots[tr.target] = true;
        } else {
          wantNode[tr.target] = nodeState(
            tr.style ? tr.kind + ' ' + tr.style : tr.kind, tr.color);
        }
      }
    }

    // A waypoint rides on top of whatever else the node is doing rather than
    // replacing it: a node that is both highlighted and a hop destination
    // should show both. Merging after the loop keeps the result independent of
    // the order the two tracks happened to appear in.
    for (var id in wantWaypoint) {
      var was = wantNode[id];
      if (was && was.cls.indexOf('hidden') === 0) continue;
      wantNode[id] = was ? nodeState(was.cls + ' waypoint', was.color)
                         : nodeState('waypoint', '');
    }

    // An edge into a hidden node conceals itself, exactly as reveal's
    // applyHidden does for collapsed elements — an arrow pointing at nothing
    // reads as a rendering bug. An open flow keeps its edge: the flow decides
    // what its own path looks like.
    var ve = this.view().edges;
    for (var e = 0; e < ve.length; e++) {
      var ed = ve[e];
      if (wantEdge[ed.id]) continue;
      var fromHidden = wantNode[ed.from] && wantNode[ed.from].cls.indexOf('hidden') === 0;
      var toHidden = wantNode[ed.to] && wantNode[ed.to].cls.indexOf('hidden') === 0;
      if (fromHidden || toHidden) wantEdge[ed.id] = nodeState('hidden', '');
    }

    // Persistent state is scenario-scoped, so it is read from its own list
    // rather than from the steps — but it lands on the same nodes, so it has to
    // be folded in before the class attribute is written or the frame would
    // erase it.
    var standing = this.collectPersistent(t, sc);
    for (var sid in standing.states) {
      var prior = wantNode[sid];
      var cls = 'stated state-' + safeName(standing.states[sid].name);
      // The frame's own colour wins: a standing colour describes what a node
      // *is*, and a highlight describes what is happening to it now.
      var color = (prior && prior.color) || standing.states[sid].color;
      wantNode[sid] = prior ? nodeState(prior.cls + ' ' + cls, color)
                            : nodeState(cls, color);
    }

    // Focus is expressed as what recedes, and that is computed last so it
    // applies to whatever the rest of the frame decided — a highlighted node
    // outside the focus set still dims.
    if (focusRoots) this.applyFocus(focusRoots, wantNode, wantEdge);

    this.applyNodeStates(wantNode);
    this.applyEdgeStates(wantEdge);
    this.applyFlows(wantFlow);
    this.applyNotes(wantNote);
    this.applyPills(standing.pills);
    this.applyBoard(this.activeScene(t, sc));

    // Chips are the one overlay positioned outside apply(); when the camera
    // wrote the transform this pass, they have to follow it.
    if (this._camMoved) {
      this._camMoved = false;
      this.syncChips();
    }
  };

  // applyFocus marks everything outside the focus set. The track names only
  // what to look at; the containment tree lives in the view, so expanding a
  // group to its contents is the renderer's job and the timeline stays ID-only.
  Player.prototype.applyFocus = function (roots, wantNode, wantEdge) {
    var inFocus = this.focusSet(roots);
    var v = this.view();

    v.nodes.forEach(function (n) {
      if (inFocus[n.id]) return;
      var prior = wantNode[n.id];
      wantNode[n.id] = prior ? nodeState(prior.cls + ' unfocused', prior.color)
                             : nodeState('unfocused', '');
    });
    (v.groups || []).forEach(function (g) {
      if (inFocus[g.id]) return;
      var prior = wantNode[g.id];
      wantNode[g.id] = prior ? nodeState(prior.cls + ' unfocused', prior.color)
                             : nodeState('unfocused', '');
    });

    // An edge with one end still in focus is part of the story — it is how the
    // focused thing connects to the rest. Only edges wholly outside recede.
    v.edges.forEach(function (e) {
      if (inFocus[e.from] || inFocus[e.to]) return;
      var prior = wantEdge[e.id];
      wantEdge[e.id] = prior ? nodeState(prior.cls + ' unfocused', prior.color)
                             : nodeState('unfocused', '');
    });
  };

  // focusSet expands the named roots downwards into group contents and upwards
  // into containing groups.
  //
  // Downwards because focusing a subgraph plainly means its members. Upwards
  // because dimming the frame drawn around the very thing you asked to look at
  // would undo the effect.
  Player.prototype.focusSet = function (roots) {
    var v = this.view();
    var groups = {};
    (v.groups || []).forEach(function (g) { groups[g.id] = g; });
    var nodes = {};
    v.nodes.forEach(function (n) { nodes[n.id] = n; });

    var out = {};
    var stack = Object.keys(roots);
    while (stack.length) {
      var id = stack.pop();
      if (out[id]) continue;
      out[id] = true;
      var g = groups[id];
      if (g) {
        (g.children || []).forEach(function (child) { stack.push(child); });
      }
    }

    Object.keys(out).forEach(function (id) {
      var parent = (nodes[id] && nodes[id].group) || (groups[id] && groups[id].parent);
      while (parent && !out[parent]) {
        out[parent] = true;
        parent = groups[parent] && groups[parent].parent;
      }
    });
    return out;
  };

  // collectPersistent gathers the badges, gauges and state classes whose window
  // contains t.
  //
  // Windows are half-open. A write that replaces a badge closes the old window
  // at exactly the instant the new one opens, and treating both ends as
  // inclusive would show the old and new value together on that frame. The one
  // exception is the last window of a key, which the compiler runs to the
  // scenario duration — without letting that one include its end, every badge
  // would blink off on the final frame.
  Player.prototype.collectPersistent = function (t, sc) {
    var pills = {};   // target -> [{key, kind, label, value}]
    var states = {};  // target -> {name, color}
    var list = sc.persistent || [];

    // One winner per slot. The compiler writes one badge per node and one
    // reading per (node, label), but a rewrite at exactly the scenario's end
    // produces an old window closing at the duration and a zero-width new one
    // opening there, and the end-inclusive exception below admits both on the
    // final frame. Tracks arrive in write order, so the later entry is the
    // rewrite and wins its slot.
    var slots = {};
    for (var i = 0; i < list.length; i++) {
      var tr = list[i];
      if (t < tr.start) continue;
      if (t >= tr.end && !(tr.end >= sc.duration && t >= sc.duration)) continue;
      var slot = tr.kind === 'gauge'
        ? 'gauge|' + tr.target + '|' + (tr.label || '')
        : tr.kind + '|' + tr.target;
      slots[slot] = { i: i, tr: tr };
    }

    for (var slot2 in slots) {
      var tr2 = slots[slot2].tr;
      if (tr2.kind === 'set' && tr2.value) {
        states[tr2.target] = { name: tr2.value, color: tr2.color || '' };
      }
      var text = tr2.kind === 'gauge' ? tr2.value : tr2.label;
      if (!text) continue;

      if (!pills[tr2.target]) pills[tr2.target] = [];
      pills[tr2.target].push({
        key: sc.id + ':' + slots[slot2].i,
        kind: tr2.kind,
        label: tr2.kind === 'gauge' ? tr2.label : '',
        value: text,
        color: tr2.color || '',
        style: tr2.style || ''
      });
    }
    return { pills: pills, states: states };
  };

  // applyPills draws badges and gauge readings as HTML over the stage, anchored
  // to the top-right of the element they belong to and stacking downwards when
  // a node carries several.
  //
  // They are overlay HTML rather than SVG for the same reason notes are: text
  // that has to stay legible at any zoom is far easier to lay out in the DOM
  // than inside the diagram's coordinate system.
  Player.prototype.applyPills = function (groups) {
    var self = this;

    var live = {};
    Object.keys(groups).forEach(function (target) {
      groups[target].forEach(function (p) { live[p.key] = true; });
    });
    for (var key in this.pills) {
      if (!live[key]) {
        this.pills[key].remove();
        delete this.pills[key];
      }
    }

    var stageRect = this.stage.getBoundingClientRect();
    Object.keys(groups).forEach(function (target) {
      var host = (self.anchors && self.anchors[target]) || self.elementFor(target);
      if (!host) return;
      var r = host.getBoundingClientRect();

      groups[target].forEach(function (p, i) {
        var div = self.pills[p.key];
        if (!div) {
          div = makePill(p);
          self.overlay.appendChild(div);
          self.pills[p.key] = div;
        }
        // Pills are HTML, so they do not shrink with the SVG. On a large
        // diagram scaled to fit, a full-size badge dwarfs the node it
        // annotates and sits over its neighbours — over exactly the region
        // being animated. Scale with the host, with a floor that keeps the
        // text legible.
        var s = Math.max(0.55, Math.min(1, r.height / PILL_HOST_REF));
        div.style.transform = 'translate(-30%, -50%) scale(' + s + ')';
        div.style.transformOrigin = 'left center';
        div.style.left = (r.right - stageRect.left) + 'px';
        div.style.top = (r.top - stageRect.top + i * PILL_STACK * s) + 'px';
      });
    });
  };

  var PILL_STACK = 22;

  // PILL_HOST_REF is the on-screen node height at which a pill renders at
  // full size — roughly a mermaid node at natural scale.
  var PILL_HOST_REF = 44;

  function makePill(p) {
    var div = el('div', 'dgm-pill dgm-pill-' + p.kind + (p.style ? ' dgm-pill-' + safeName(p.style) : ''));
    if (p.color) div.style.setProperty('--dgm-color', p.color);
    if (p.kind === 'gauge') {
      div.appendChild(elText('span', 'dgm-pill-label', p.label));
      div.appendChild(elText('span', 'dgm-pill-value', p.value));
    } else {
      div.textContent = p.value;
    }
    return div;
  }

  // safeName keeps an author-chosen name usable as a CSS class. State and style
  // names come from source text, and a space in one would otherwise turn a
  // single class into two.
  function safeName(s) {
    return String(s).replace(/[^A-Za-z0-9_-]+/g, '-');
  }

  // nodeState packages what a frame wants an element to look like. `key` is the
  // whole thing flattened, so the diff against the previous frame is one string
  // comparison and a colour change repaints exactly like a class change.
  function nodeState(cls, color) {
    return { cls: cls, color: color || '', key: cls + '|' + (color || '') };
  }

  // edgeState is what an open flow track wants its edge path to look like.
  // `flow-active` is the general "something is travelling here" hook; the style
  // and status names ride alongside it so CSS can distinguish a response from a
  // failure without the runtime hard-coding either.
  // `flow-reverse` says the traffic is running against the arrow mermaid drew.
  // The stylesheet uses it to take that arrowhead off for as long as the flow
  // is open, because an arrowhead pointing one way while the particle travels
  // the other is a contradiction the viewer has to resolve every time.
  function edgeState(tr) {
    var cls = 'flow-active';
    if (tr.style) cls += ' flow-' + tr.style;
    if (tr.status === 'fail') cls += ' flow-fail';
    if (tr.reverse) cls += ' flow-reverse';
    return nodeState(cls, tr.color);
  }

  // applyStates diffs a wanted class/colour map against the previous frame and
  // rewrites only what changed. `lookup` turns an id into the element to paint,
  // which is all that differs between nodes and edge paths.
  function applyStates(prev, want, lookup) {
    var id, was;
    for (id in prev) {
      was = prev[id];
      if (want[id] && want[id].key === was.key) continue;
      var stale = lookup(id);
      if (stale) {
        stale.setAttribute('class', baseClass(stale));
        stale.style.removeProperty('--dgm-color');
      }
    }
    for (id in want) {
      var element = lookup(id);
      if (!element) continue;
      was = prev[id];
      if (was && was.key === want[id].key) continue;
      element.setAttribute('class',
        baseClass(element) + ' dgm-' + want[id].cls.split(' ').join(' dgm-'));
      // The colour rides in as a custom property rather than a direct fill or
      // stroke: runtime.css decides which parts of an element it tints, and
      // every rule that reads it falls back to the theme token when absent.
      if (want[id].color) element.style.setProperty('--dgm-color', want[id].color);
      else element.style.removeProperty('--dgm-color');
    }
    return want;
  }

  // applyNodeStates paints element state. It resolves through elementFor rather
  // than the node map alone, so a subgraph is as animatable as a node — which
  // `focus` relies on, and which `highlight <group>` gets for free.
  Player.prototype.applyNodeStates = function (want) {
    var self = this;
    this.nodeState = applyStates(this.nodeState, want, function (id) {
      return self.elementFor(id);
    });
  };

  // applyEdgeStates lights the path a flow is travelling along. It goes through
  // the same diff as node state — and so through baseClass, which means an edge
  // concealed by a reveal keeps its dgm-collapsed while it animates.
  Player.prototype.applyEdgeStates = function (want) {
    var self = this;
    this.edgeState = applyStates(this.edgeState, want, function (id) {
      var bind = self.edges[id];
      return bind ? bind.path : null;
    });
  };

  // EASINGS remap a track's linear progress. Each is a pure function of p over
  // [0,1] with f(0)=0 and f(1)=1, which is what keeps seeking equivalent to
  // playing: the runtime never integrates, it evaluates.
  var EASINGS = {
    linear: function (p) { return p; },
    'in': function (p) { return p * p; },
    out: function (p) { return p * (2 - p); },
    'in-out': function (p) { return p < 0.5 ? 2 * p * p : 1 - 2 * (1 - p) * (1 - p); }
  };

  // ease applies a named curve, falling back to linear for an unknown name so
  // an old page still animates a timeline compiled by a newer binary. The
  // compiler rejects unknown names, so this fallback is belt and braces.
  function ease(name, p) {
    var fn = (name && EASINGS[name]) || EASINGS.linear;
    return fn(Math.min(1, Math.max(0, p)));
  }

  // TRAIL_FRACTION and TRAIL_MAX size the comet behind a particle: a share of
  // the edge, capped so a long path does not turn into a lit-up hose.
  var TRAIL_FRACTION = 0.3;
  var TRAIL_MAX = 90;

  // FAIL_LEAD is the tail of a failing flow during which the ✕ is drawn at the
  // destination. Like the trail it is read off the frame's progress, never
  // accumulated, so scrubbing backwards takes it away again.
  var FAIL_LEAD = 0.2;

  Player.prototype.applyFlows = function (want) {
    var key;
    // Where the action is, this frame. Sequence-diagram notes anchor to it, and
    // it is recomputed from scratch every frame like everything else.
    this.flowY = null;
    for (key in this.particles) {
      if (!want[key]) {
        disposeFlow(this.particles[key]);
        delete this.particles[key];
      }
    }
    for (key in want) {
      var tr = want[key].track;
      var bind = this.edges[tr.edge];
      if (!bind) continue;

      var fx = this.particles[key];
      if (!fx) {
        fx = this.makeFlow(tr, bind);
        this.particles[key] = fx;
      }

      // Easing remaps progress before anything geometric happens, so it is a
      // pure function of the frame time and a seek lands where playback would.
      var eased = ease(tr.ease, want[key].progress);

      // `reverse` is what the source asked for; `flip` corrects for mermaid
      // having drawn the path from the other end. They compose. Easing is
      // applied first because it describes travel along the flow, not along
      // the path mermaid happened to draw.
      var backwards = !!tr.reverse !== !!bind.flip;
      var u = backwards ? 1 - eased : eased;
      var len = bind.path.getTotalLength();
      var along = u * len;
      var pt = pointAt(bind, along);
      fx.group.setAttribute('transform', 'translate(' + pt.x + ',' + pt.y + ')');
      if (fx.arrow) this.aimArrow(fx.arrow, bind, along, len, backwards);
      if (this.sequence) {
        var box = fx.group.getBoundingClientRect();
        this.flowY = box.top + box.height / 2;
      }

      if (fx.trail) this.drawTrail(fx.trail, along, len, backwards);
      if (tr.status === 'fail') {
        this.drawFailMark(fx, bind, len, backwards, want[key].progress >= 1 - FAIL_LEAD);
      }
    }
  };

  // pointAt is a position on a bound edge, in the overlay layer's coordinates.
  // The path's `d` is in whatever user space mermaid drew it in, so the bind's
  // matrix is what carries it over to the layer.
  function pointAt(bind, along) {
    var pt = bind.path.getPointAtLength(along);
    if (bind.matrix && pt.matrixTransform) pt = pt.matrixTransform(bind.matrix);
    return pt;
  }

  // AIM_STEP is how far either side of the particle the heading is sampled.
  // Small enough to follow a curve, large enough that two samples on a
  // straight run do not land on the same point and lose the angle.
  var AIM_STEP = 3;

  // aimArrow turns the chevron to face the way the flow is travelling, which is
  // the one thing the diagram underneath cannot say: a response runs against
  // the arrow mermaid drew, and without this the only cue that it is going the
  // other way is which side of the dot the trail is on.
  //
  // The heading is measured between two points *ordered by travel*, so it comes
  // out right whether the flow runs with or against the drawn path — no
  // separate 180° correction to keep in step with `backwards`.
  Player.prototype.aimArrow = function (arrow, bind, along, len, backwards) {
    var step = Math.min(AIM_STEP, len);
    var ahead = backwards ? Math.max(0, along - step) : Math.min(len, along + step);
    var behind = backwards ? Math.min(len, along + step) : Math.max(0, along - step);
    var from = pointAt(bind, behind);
    var to = pointAt(bind, ahead);
    var dx = to.x - from.x, dy = to.y - from.y;
    // A degenerate sample (a zero-length path) leaves the last angle standing
    // rather than snapping the chevron to 0°.
    if (!dx && !dy) return;
    arrow.setAttribute('transform',
      'rotate(' + (Math.atan2(dy, dx) * 180 / Math.PI) + ')');
  };

  // drawTrail slides a dash window along the cloned edge so it ends exactly
  // where the particle is. The whole state of the effect is the dasharray, and
  // that is recomputed from `along` every frame — there is nothing to reset on
  // a seek because nothing carries over from the last one.
  Player.prototype.drawTrail = function (trail, along, len, backwards) {
    var span = Math.min(TRAIL_MAX, len * TRAIL_FRACTION);
    // The comet sits behind the direction of travel, which is the far side of
    // the particle when the flow is running against how the path was drawn.
    var lo = backwards ? along : Math.max(0, along - span);
    var hi = backwards ? Math.min(len, along + span) : along;
    // dash 0, gap `lo`, dash `hi-lo`, gap `len`: exactly the arc [lo,hi] is
    // painted and the trailing gap is long enough that the pattern never
    // repeats within the path.
    trail.setAttribute('stroke-dasharray', '0 ' + lo + ' ' + (hi - lo) + ' ' + len);
  };

  // drawFailMark places the ✕ at the end of the path the flow was heading for,
  // which is length 0 when the flow runs against the drawn direction.
  Player.prototype.drawFailMark = function (fx, bind, len, backwards, show) {
    if (!fx.mark) {
      if (!show) return;
      fx.mark = makeFailMark(this.layer);
    }
    fx.mark.style.display = show ? '' : 'none';
    if (!show) return;

    var pt = pointAt(bind, backwards ? 0 : len);
    fx.mark.setAttribute('transform', 'translate(' + pt.x + ',' + pt.y + ')');
  };

  // makeFlow builds everything one open flow track draws: the trail underneath,
  // then the particle over it. The ✕ is added later, only if the flow gets far
  // enough to need one.
  // The two are built in painting order — the layer has no z-index, so the
  // trail has to be in the DOM before the particle to stay under it.
  Player.prototype.makeFlow = function (tr, bind) {
    var trail = makeTrail(tr, bind, this.layer);
    var group = this.makeParticle(tr, this.layer);
    return {
      trail: trail,
      group: group,
      arrow: group.querySelector('.dgm-particle-arrow'),
      mark: null
    };
  };

  function disposeFlow(fx) {
    fx.group.remove();
    if (fx.trail) fx.trail.remove();
    if (fx.mark) fx.mark.remove();
  }

  // makeTrail clones the edge path into the overlay layer so a dash window can
  // slide along it without touching the path mermaid drew — which still has to
  // render as an ordinary edge underneath.
  //
  // The clone is stamped with the path's transform *relative to the layer*,
  // because its `d` is in the original path's user space and the layer is
  // somewhere else in the tree.
  function makeTrail(tr, bind, parent) {
    var clone = bind.path.cloneNode(false);
    clone.removeAttribute('id');
    clone.removeAttribute('style');
    var cls = 'dgm-trail';
    // The comet takes the same style and status hooks as the edge under it, so
    // a response trails green wherever its edge lights green.
    if (tr.style) cls += ' dgm-trail-' + tr.style;
    if (tr.status === 'fail') cls += ' dgm-trail-fail';
    clone.setAttribute('class', cls);
    clone.setAttribute('fill', 'none');
    // The original's arrowhead would otherwise be drawn a second time, riding
    // the end of the dash window.
    clone.setAttribute('marker-end', 'none');
    clone.setAttribute('marker-start', 'none');
    // Start empty: the first frame sets the real window, and without this a
    // clone would flash the whole edge for one frame.
    clone.setAttribute('stroke-dasharray', '0 1');
    if (tr.color) clone.style.setProperty('--dgm-color', tr.color);
    if (bind.matrix) clone.setAttribute('transform', matrixString(bind.matrix));
    else clone.removeAttribute('transform');
    parent.appendChild(clone);
    return clone;
  }

  function matrixString(m) {
    return 'matrix(' + m.a + ',' + m.b + ',' + m.c + ',' + m.d + ',' + m.e + ',' + m.f + ')';
  }

  // makeFailMark draws the ✕ as two lines over a plate of page background, so
  // it stays legible on top of whatever node it lands on.
  function makeFailMark(parent) {
    var g = document.createElementNS(SVG_NS, 'g');
    g.setAttribute('class', 'dgm-fail-mark');

    var plate = document.createElementNS(SVG_NS, 'circle');
    plate.setAttribute('r', '10');
    g.appendChild(plate);

    var strokes = [[-5, -5, 5, 5], [5, -5, -5, 5]];
    for (var i = 0; i < strokes.length; i++) {
      var line = document.createElementNS(SVG_NS, 'line');
      line.setAttribute('x1', strokes[i][0]);
      line.setAttribute('y1', strokes[i][1]);
      line.setAttribute('x2', strokes[i][2]);
      line.setAttribute('y2', strokes[i][3]);
      g.appendChild(line);
    }

    parent.appendChild(g);
    return g;
  }

  Player.prototype.makeParticle = function (tr, parent) {
    var g = document.createElementNS(SVG_NS, 'g');
    var cls = 'dgm-particle';
    if (tr.style) cls += ' dgm-particle-' + tr.style;
    if (tr.status === 'fail') cls += ' dgm-particle-fail';
    g.setAttribute('class', cls);
    // A custom property rather than a fill, so the stylesheet keeps deciding
    // which of the head, halo and label the colour reaches.
    if (tr.color) g.style.setProperty('--dgm-color', tr.color);

    var halo = document.createElementNS(SVG_NS, 'circle');
    halo.setAttribute('r', '11');
    halo.setAttribute('class', 'dgm-particle-halo');
    g.appendChild(halo);

    // The head rides in its own group because only it turns: rotating the whole
    // particle would tip the label over with it, and a label upside down on a
    // right-to-left response is exactly the case this is here to fix. Drawn
    // along +x, so aimArrow's angle is the heading with no offset.
    var arrow = document.createElementNS(SVG_NS, 'g');
    arrow.setAttribute('class', 'dgm-particle-arrow');
    var head = document.createElementNS(SVG_NS, 'path');
    head.setAttribute('class', 'dgm-particle-head');
    // A dart rather than a plain triangle: the notch in the back gives it a
    // direction you can read at a glance even at this size. The tip leads the
    // point on the path, so the comet behind meets the notch and the two draw
    // as one arrow rather than as a shape with a line stuck to it.
    head.setAttribute('d', 'M10,0 L-6,-7.5 L-2,0 L-6,7.5 Z');
    arrow.appendChild(head);
    g.appendChild(arrow);

    if (tr.label) {
      var text = document.createElementNS(SVG_NS, 'text');
      text.setAttribute('class', 'dgm-particle-label');
      text.setAttribute('y', '-16');
      text.setAttribute('text-anchor', 'middle');
      text.textContent = tr.label;
      g.appendChild(text);
    }

    parent.appendChild(g);
    return g;
  };

  // NOTE_GAP is the breathing room between a note and the element it points at;
  // NOTE_SHOVE is how far a note is pushed clear of one already placed.
  var NOTE_GAP = 10;
  var NOTE_SHOVE = 6;

  Player.prototype.applyNotes = function (want) {
    var key;
    for (key in this.notes) {
      if (!want[key]) {
        this.notes[key].remove();
        delete this.notes[key];
      }
    }

    var stageRect = this.stage.getBoundingClientRect();
    var placed = [];

    for (key in want) {
      var tr = want[key];
      var hostRect = this.anchorRect(tr.target);
      if (!hostRect) continue;

      var div = this.notes[key];
      if (!div) {
        div = el('div', 'dgm-note');
        div.textContent = tr.text;
        this.overlay.appendChild(div);
        // Pin the width the note measures with the whole stage to itself.
        // An absolutely positioned box re-wraps against whatever room its
        // left edge leaves, so an unpinned note placed near the right edge
        // ends up taller than the box that was measured before placing it —
        // and escapes the stage clamp. Every record frame is a fresh page,
        // so recordings always take this first-placement path.
        var natural = div.getBoundingClientRect();
        div.style.width = Math.min(Math.ceil(natural.width),
          Math.max(120, Math.floor(this.stage.clientWidth * 0.75))) + 'px';
        this.notes[key] = div;
      }

      var side = sideOf(tr.side);
      div.className = 'dgm-note dgm-note-' + side;

      // Measured after the class is set, because the side decides the arrow and
      // an arrow is part of the box the layout has to fit.
      var box = div.getBoundingClientRect();
      var anchor = rectIn(hostRect, stageRect);
      var spot = place(side, anchor, box.width, box.height);

      var shoved = settle(spot, box.width, box.height, placed,
        this.stage.clientWidth, this.stage.clientHeight);
      placed.push({
        left: shoved.left, top: shoved.top,
        right: shoved.left + box.width, bottom: shoved.top + box.height
      });

      // A shoved note no longer points at anything, so it drops its arrow
      // rather than aiming it at whatever it landed next to.
      div.classList.toggle('is-adrift', shoved.moved);
      div.style.left = shoved.left + 'px';
      div.style.top = shoved.top + 'px';
    }
  };

  // anchorRect is where overlay content should point for an element.
  //
  // For a flowchart it is simply the node's box. For a sequence diagram it is
  // the actor's column at the height of whatever message is moving right now —
  // a note about a message belongs beside that message, not up at the actor
  // box where the whole conversation would pile up. It falls back to the box
  // when nothing is in flight.
  Player.prototype.anchorRect = function (id) {
    var host = (this.anchors && this.anchors[id]) || this.elementFor(id);
    if (!host) return null;

    var r = host.getBoundingClientRect();
    if (!this.sequence || this.flowY === null || this.flowY === undefined) return r;

    return {
      left: r.left, right: r.right, width: r.width,
      top: this.flowY, bottom: this.flowY, height: 0
    };
  };

  function sideOf(s) {
    return (s === 'below' || s === 'left' || s === 'right') ? s : 'above';
  }

  // rectIn re-expresses a client rect in the stage's coordinates.
  function rectIn(r, stage) {
    return {
      left: r.left - stage.left, top: r.top - stage.top,
      right: r.right - stage.left, bottom: r.bottom - stage.top,
      cx: r.left - stage.left + r.width / 2,
      cy: r.top - stage.top + r.height / 2
    };
  }

  function place(side, a, w, h) {
    if (side === 'below') return { left: a.cx - w / 2, top: a.bottom + NOTE_GAP };
    if (side === 'left') return { left: a.left - w - NOTE_GAP, top: a.cy - h / 2 };
    if (side === 'right') return { left: a.right + NOTE_GAP, top: a.cy - h / 2 };
    return { left: a.cx - w / 2, top: a.top - h - NOTE_GAP };
  }

  // settle clamps a note into the stage and pushes it clear of anything already
  // placed. One pass downwards, deliberately: this is a tidy-up, not a layout
  // solver, and a note that ends up somewhere odd is better than a frame spent
  // iterating to an optimum nobody asked for.
  function settle(spot, w, h, placed, stageW, stageH) {
    var left = clamp(spot.left, 0, Math.max(0, stageW - w));
    var top = clamp(spot.top, 0, Math.max(0, stageH - h));
    var moved = false;

    for (var i = 0; i < placed.length; i++) {
      var p = placed[i];
      var overlaps = left < p.right && left + w > p.left &&
                     top < p.bottom && top + h > p.top;
      if (!overlaps) continue;
      top = clamp(p.bottom + NOTE_SHOVE, 0, Math.max(0, stageH - h));
      moved = true;
    }
    return { left: left, top: top, moved: moved };
  }

  function clamp(v, lo, hi) {
    return v < lo ? lo : (v > hi ? hi : v);
  }

  // ---------------------------------------------------------------------
  // Helpers
  // ---------------------------------------------------------------------

  // STICKY are the dgm-* classes that describe an element rather than the
  // current animation frame, so baseClass must not strip them.
  //
  // This matters more than it looks. applyNodeStates rewrites an element's
  // whole class attribute on every frame that changes its state; without this
  // exemption a node would silently lose its click affordance, and a revealed
  // element would flicker back to hidden, the moment the clock touched it.
  var STICKY = { 'dgm-clickable': true, 'dgm-collapsed': true, 'dgm-actor': true };

  // baseClass strips the dgm-* state classes we previously added, so state
  // changes never accumulate on an element.
  function baseClass(el) {
    var cls = el.getAttribute('class') || '';
    return cls
      .split(/\s+/)
      .filter(function (c) { return c && (STICKY[c] || c.indexOf('dgm-') !== 0); })
      .join(' ');
  }

  function el(tag, cls) {
    var e = document.createElement(tag);
    if (cls) e.className = cls;
    return e;
  }

  function elText(tag, cls, text) {
    var e = el(tag, cls);
    e.textContent = text;
    return e;
  }

  function button(label, cls, fn) {
    var b = el('button', cls);
    b.type = 'button';
    b.textContent = label;
    b.addEventListener('click', fn);
    return b;
  }

  function fmt(ms) {
    return (ms / 1000).toFixed(1) + 's';
  }

  // --- fullscreen -------------------------------------------------------
  //
  // Both spellings, because Safari still ships only the webkit* ones. Every
  // path here is silent: build() asks for fullscreen on load whenever the page
  // opens in presenter mode, with no user gesture behind it, and a browser is
  // entitled to refuse that — as is a VS Code webview and an iframe without
  // `allowfullscreen`. Refusal costs nothing to detect anymore: the
  // `dgm-present-fill` class setPresenter puts on unconditionally draws the
  // same box either way, so the request is fire-and-forget.

  function fullscreenElement() {
    return document.fullscreenElement || document.webkitFullscreenElement || null;
  }

  function requestFull(node) {
    var fn = node.requestFullscreen || node.webkitRequestFullscreen;
    // No element fullscreen at all — iOS Safari, most notably.
    if (!fn) return;
    try {
      var p = fn.call(node);
      if (p && p.catch) p.catch(function () { /* refused; the fill has it */ });
    } catch (e) { /* refused; the fill has it */ }
  }

  function exitFull() {
    var fn = document.exitFullscreen || document.webkitExitFullscreen;
    if (!fn) return;
    try {
      var p = fn.call(document);
      if (p && p.catch) p.catch(function () { /* already out */ });
    } catch (e) { /* already out */ }
  }

  // normalize fills in the list-valued fields a timeline is entitled to omit.
  //
  // Go marshals a nil slice as `null`, so a flowchart with no subgraph arrives
  // with `groups: null` and a step whose actions all draw nothing arrives with
  // `tracks: null`. The compiler now emits `[]` for both, but the runtime is
  // handed timelines from elsewhere too — a webview, a hand-written fixture, a
  // file compiled by an older binary — and one missing list should not take the
  // whole page down with a TypeError. Done once here rather than as a `|| []`
  // at each of the twenty-odd reads, which is a guard that only has to be
  // forgotten once.
  function normalize(timeline) {
    var t = timeline || {};
    t.views = t.views || [];
    t.views.forEach(function (v) {
      v.nodes = v.nodes || [];
      v.groups = v.groups || [];
      v.edges = v.edges || [];
      v.bindings = v.bindings || [];
      v.hidden = v.hidden || [];
      v.scenarios = v.scenarios || [];
      v.scenarios.forEach(function (sc) {
        sc.steps = sc.steps || [];
        sc.persistent = sc.persistent || [];
        sc.steps.forEach(function (st) { st.tracks = st.tracks || []; });
      });
      if (v.storyboard) v.storyboard.frames = v.storyboard.frames || [];
    });
    return t;
  }

  window.Cinegram = {
    // opts is optional and documented on Player above. The page passes none;
    // an editor host embedding several players in one document passes
    // { inline: true, keys: 'scoped', hash: false, autoplay: false, theme }.
    mount: function (root, timeline, opts) {
      return new Player(root, normalize(timeline), opts);
    },
    // themeToggle hands page chrome the light/dark control, ready to drop
    // wherever the page keeps its own buttons. A page that would rather have
    // the control in its markup — so it is there before this file loads —
    // emits pkg/emit/html's ThemeToggleHTML instead and gets the same button,
    // upgraded by the sweep below.
    themeToggle: function () {
      var btn = el('button', 'dgm-page-theme');
      btn.type = 'button';
      btn.setAttribute('data-dgm-theme-toggle', '');
      btn.setAttribute('aria-live', 'polite');
      return wireThemeToggle(btn);
    }
  };

  // Adopt the placeholders the page emitted. This file is loaded at the end of
  // the body on every surface that emits one, so the elements are already
  // there — but a host free to load it from the head should get the control
  // too, hence the readyState branch rather than a bare call.
  if (document.readyState === 'loading') {
    document.addEventListener('DOMContentLoaded', upgradeThemeToggles);
  } else {
    upgradeThemeToggles();
  }
})();
