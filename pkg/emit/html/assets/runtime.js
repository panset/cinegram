/*
 * Diagramator animation runtime.
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
  // Nodes are found by id: mermaid builds them as `flowchart-<id>-<counter>`.
  // Edges are matched geometrically instead, by comparing each path's
  // endpoints against node centres. That avoids depending on mermaid's edge-id
  // format, which has changed between releases, and as a bonus tells us
  // whether a path happens to be drawn against the direction we declared.
  // ---------------------------------------------------------------------

  function indexNodes(svg) {
    var map = {};
    var groups = svg.querySelectorAll('g.node');
    for (var i = 0; i < groups.length; i++) {
      var g = groups[i];
      var m = /^flowchart-(.+)-\d+$/.exec(g.id || '');
      if (m) {
        map[m[1]] = g;
      }
    }
    return map;
  }

  function indexClusters(svg, view) {
    var map = {};
    var els = svg.querySelectorAll('g.cluster');
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
        var rev = dist(ends[k].start, b) + dist(ends[k].end, a);
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
  // Player
  // ---------------------------------------------------------------------

  function Player(root, timeline) {
    this.root = root;
    this.timeline = timeline;
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
    this.pendingAutoplay = true;
    this.raf = null;
    this.lastFrame = 0;
    this.nodeState = {};
    this.particles = {};
    this.notes = {};
    // The trail of view ids drilled through, so Back knows where to return.
    this.stack = [];
    // Elements a reveal binding has opened in the current view. Unlike a
    // track this is not owned by the clock: it persists across seeks and is
    // cleared only when the view changes.
    this.revealed = {};
    this.theme = matchMedia('(prefers-color-scheme: dark)').matches ? 'dark' : 'light';
    this.build();
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
    });
    controls.appendChild(this.picker);

    this.playBtn = button('Play', 'dgm-btn dgm-btn-primary', function () { self.toggle(); });
    controls.appendChild(this.playBtn);
    controls.appendChild(button('Restart', 'dgm-btn', function () { self.seek(0); }));

    this.speedBtn = button(speedLabel(this.speed), 'dgm-btn', function () { self.cycleSpeed(); });
    controls.appendChild(this.speedBtn);

    this.themeBtn = button(this.theme === 'dark' ? 'Light' : 'Dark', 'dgm-btn', function () {
      self.theme = self.theme === 'dark' ? 'light' : 'dark';
      self.themeBtn.textContent = self.theme === 'dark' ? 'Light' : 'Dark';
      document.documentElement.setAttribute('data-theme', self.theme);
      self.render();
    });
    controls.appendChild(this.themeBtn);

    bar.appendChild(controls);
    this.root.appendChild(bar);

    this.warning = el('div', 'dgm-warning');
    this.warning.style.display = 'none';
    this.root.appendChild(this.warning);

    var body = el('div', 'dgm-body');
    this.stage = el('div', 'dgm-stage');
    this.overlay = el('div', 'dgm-overlay');
    this.stage.appendChild(this.overlay);
    body.appendChild(this.stage);

    this.steps = el('ol', 'dgm-steps');
    body.appendChild(this.steps);
    this.root.appendChild(body);

    var foot = el('div', 'dgm-foot');
    this.scrub = document.createElement('input');
    this.scrub.type = 'range';
    this.scrub.min = '0';
    this.scrub.step = '10';
    this.scrub.className = 'dgm-scrub';
    this.scrub.addEventListener('input', function () {
      self.pause();
      self.seek(parseInt(self.scrub.value, 10));
    });
    foot.appendChild(this.scrub);
    this.clock = el('div', 'dgm-clock');
    foot.appendChild(this.clock);
    this.root.appendChild(foot);

    document.documentElement.setAttribute('data-theme', this.theme);
    // One document-level handler for the whole page: the player swaps views
    // rather than being replaced, so these never stack up.
    document.addEventListener('keydown', function (ev) {
      if (ev.target && /input|select|textarea/i.test(ev.target.tagName)) return;
      if (ev.key === ' ') { ev.preventDefault(); self.toggle(); }
      if (ev.key === 'ArrowRight') { ev.preventDefault(); self.nextStep(1); }
      if (ev.key === 'ArrowLeft') { ev.preventDefault(); self.nextStep(-1); }
      if (ev.key === 'Escape' && self.stack.length) { ev.preventDefault(); self.back(); }
    });

    // The hash is the single source of truth for which view is showing, and
    // every navigation goes through it. That is what keeps the Back button
    // and the browser's own history from ever disagreeing.
    window.addEventListener('hashchange', function () { self.applyHash(); });

    this.viewIndex = Math.max(0, this.viewIndexOf(this.hashView()));
    this.buildPicker();
    this.adoptScenarioSpeed();
    this.render();
  };

  // hashView is the view id the current URL selects, defaulting to the root.
  Player.prototype.hashView = function () {
    var id = decodeURIComponent((location.hash || '').replace(/^#/, ''));
    if (id && this.viewIndexOf(id) >= 0) return id;
    return this.timeline.root;
  };

  // applyHash moves to whatever view the URL now names, keeping the back
  // stack in step: returning to the view we came from pops it, anything else
  // pushes the view being left.
  Player.prototype.applyHash = function () {
    var id = this.hashView();
    if (id === this.view().id) return;

    if (this.stack[this.stack.length - 1] === id) this.stack.pop();
    else this.stack.push(this.view().id);

    this.setView(id);
  };

  // navigate drills into another view. It only moves the hash; applyHash does
  // the work, so a click and a browser history move follow the same path.
  Player.prototype.navigate = function (id) {
    if (this.viewIndexOf(id) < 0 || id === this.view().id) return;
    location.hash = id === this.timeline.root ? '' : '#' + encodeURIComponent(id);
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
      o.textContent = s.name || 'scenario ' + (i + 1);
      self.picker.appendChild(o);
    });
    this.picker.value = String(this.scenarioIndex);
    this.picker.style.display = scenarios.length > 1 ? '' : 'none';
  };

  Player.prototype.selectScenario = function (i) {
    this.scenarioIndex = i;
    this.time = 0;
    this.adoptScenarioSpeed();
    this.buildSteps();
    this.apply(0);
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
    this.time = 0;
    this.revealed = {};
    this.svg = null;
    this.pendingAutoplay = true;

    this.buildPicker();
    this.adoptScenarioSpeed();
    this.render();
  };

  // back retraces one step. Going through history rather than straight to the
  // view keeps the forward button working.
  Player.prototype.back = function () {
    if (this.stack.length) history.back();
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

    this.title.textContent = v.title || this.timeline.title || 'Diagramator';
    this.syncNav();

    window.mermaid.initialize({
      startOnLoad: false,
      theme: this.theme === 'dark' ? 'dark' : 'default',
      securityLevel: 'loose',
      flowchart: { curve: 'basis', htmlLabels: true, padding: 12 }
    });

    // A fresh id each render keeps mermaid from reusing stale definitions.
    var id = 'dgm-svg-' + Math.floor(Math.random() * 1e9);
    window.mermaid
      .render(id, d.mermaid)
      .then(function (out) {
        var holder = el('div', 'dgm-svg-holder');
        holder.innerHTML = out.svg;
        self.stage.innerHTML = '';
        self.stage.appendChild(holder);
        self.stage.appendChild(self.overlay);

        self.svg = holder.querySelector('svg');
        if (self.svg) {
          self.svg.removeAttribute('width');
          self.svg.removeAttribute('height');
          self.svg.style.maxWidth = '100%';
          self.svg.style.height = 'auto';
        }
        self.index();
        self.buildSteps();
        self.apply(self.time);
        self.syncChrome();

        // Autoplay waits for a successful render: starting the clock over a
        // diagram mermaid failed to draw would just run it out invisibly.
        if (self.pendingAutoplay) {
          self.pendingAutoplay = false;
          self.maybeAutoplay();
        }
      })
      .catch(function (err) {
        self.warn(['Mermaid failed to render the diagram: ' + err]);
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
    this.nodes = indexNodes(this.svg);
    this.clusters = indexClusters(this.svg, v);
    this.layer = makeLayer(this.svg);
    this.edges = indexEdges(this.svg, v, this.nodes);

    // Cache each path's transform into the overlay's coordinate system, so
    // positions stay exact even if a future mermaid release puts a transform
    // on the edge group.
    for (var id in this.edges) {
      this.edges[id].matrix = relativeMatrix(this.layer, this.edges[id].path);
    }
    this.particles = {};
    this.notes = {};
    this.nodeState = {};
    this.overlay.innerHTML = '';

    // Surface anything that failed to bind. A silently half-animated diagram
    // is far more confusing than an explicit list of what could not be found.
    var problems = [];
    var self = this;
    v.nodes.forEach(function (n) {
      if (!self.nodes[n.id]) problems.push('node "' + n.id + '" not found in the rendered SVG');
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
      target.addEventListener('click', function (ev) {
        ev.preventDefault();
        ev.stopPropagation();
        self.activate(b);
      });
    });

    this.applyHidden();
    this.warn(problems);
  };

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
    if (!hidden.length) return;

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
    sc.steps.forEach(function (st, i) {
      var li = el('li', 'dgm-step');
      li.appendChild(elText('span', 'dgm-step-name', st.name || st.id));
      li.appendChild(elText('span', 'dgm-step-time', fmt(st.start) + ' – ' + fmt(st.end)));
      li.addEventListener('click', function () { self.seek(st.start); });
      self.steps.appendChild(li);
    });
    this.scrub.max = String(sc.duration || 0);
  };

  // ---------------------------------------------------------------------
  // Playback
  // ---------------------------------------------------------------------

  Player.prototype.toggle = function () { this.playing ? this.pause() : this.play(); };

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

  Player.prototype.seek = function (ms) {
    this.time = Math.max(0, ms);
    this.apply(this.time);
    this.syncChrome();
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
    var s = this.scenario().speed;
    this.speed = (typeof s === 'number' && s > 0) ? s : 1;
    this.syncSpeed();
  };

  Player.prototype.syncSpeed = function () {
    if (this.speedBtn) this.speedBtn.textContent = speedLabel(this.speed);
  };

  // cycleSpeed steps to the next preset above the current rate, wrapping at the
  // top. Picking by value rather than by index means a scenario speed that is
  // not itself a preset (0.8, say) still cycles somewhere sensible instead of
  // snapping to the slowest.
  Player.prototype.cycleSpeed = function () {
    var order = [0.25, 0.5, 1, 1.5, 2];
    var next = order[0];
    for (var i = 0; i < order.length; i++) {
      if (order[i] > this.speed + 1e-9) { next = order[i]; break; }
    }
    this.speed = next;
    this.syncSpeed();
  };

  // maybeAutoplay starts playback once a view has actually rendered, unless the
  // author opted out or the viewer's system asks for reduced motion. Autoplay
  // is deliberately outside apply(): it moves the clock, it is not a frame.
  Player.prototype.maybeAutoplay = function () {
    var sc = this.scenario();
    if (this.playing || !sc.autoplay || !sc.duration) return;
    if (prefersReducedMotion()) return;
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

    var kids = this.steps.children;
    for (var i = 0; i < kids.length; i++) {
      var st = sc.steps[i];
      var active = st && this.time >= st.start && this.time < st.end;
      var done = st && this.time >= st.end;
      kids[i].classList.toggle('is-active', !!active);
      kids[i].classList.toggle('is-done', !!done);
    }
  };

  // ---------------------------------------------------------------------
  // Frame application
  // ---------------------------------------------------------------------

  Player.prototype.apply = function (t) {
    if (!this.svg) return;
    var sc = this.scenario();

    var wantNode = {};   // node id -> {cls, color, key}
    var wantFlow = {};   // track key -> {track, progress}
    var wantNote = {};   // note key -> {track, target}

    for (var s = 0; s < sc.steps.length; s++) {
      var step = sc.steps[s];
      if (t < step.start || t > step.end) continue;

      for (var k = 0; k < step.tracks.length; k++) {
        var tr = step.tracks[k];
        if (t < tr.start || t > tr.end) continue;
        var span = Math.max(1, tr.end - tr.start);
        var p = (t - tr.start) / span;
        var key = s + ':' + k;

        if (tr.kind === 'flow') {
          wantFlow[key] = { track: tr, progress: Math.min(1, Math.max(0, p)) };
        } else if (tr.kind === 'note') {
          wantNote[key] = tr;
          if (!wantNode[tr.target]) wantNode[tr.target] = nodeState('noted', '');
        } else if (tr.kind === 'hide') {
          wantNode[tr.target] = nodeState('hidden', '');
        } else {
          wantNode[tr.target] = nodeState(
            tr.style ? tr.kind + ' ' + tr.style : tr.kind, tr.color);
        }
      }
    }

    this.applyNodeStates(wantNode);
    this.applyFlows(wantFlow);
    this.applyNotes(wantNote);
  };

  // nodeState packages what a frame wants an element to look like. `key` is the
  // whole thing flattened, so the diff against the previous frame is one string
  // comparison and a colour change repaints exactly like a class change.
  function nodeState(cls, color) {
    return { cls: cls, color: color || '', key: cls + '|' + (color || '') };
  }

  Player.prototype.applyNodeStates = function (want) {
    // Diff against the previous frame so we only touch the DOM on change.
    var id, was;
    for (id in this.nodeState) {
      was = this.nodeState[id];
      if (want[id] && want[id].key === was.key) continue;
      var prev = this.nodes[id];
      if (prev) {
        prev.setAttribute('class', baseClass(prev));
        prev.style.removeProperty('--dgm-color');
      }
    }
    for (id in want) {
      var g = this.nodes[id];
      if (!g) continue;
      was = this.nodeState[id];
      if (was && was.key === want[id].key) continue;
      g.setAttribute('class', baseClass(g) + ' dgm-' + want[id].cls.split(' ').join(' dgm-'));
      // The colour rides in as a custom property rather than a direct fill or
      // stroke: runtime.css decides which parts of a node it tints, and every
      // rule that reads it falls back to the theme token when it is absent.
      if (want[id].color) g.style.setProperty('--dgm-color', want[id].color);
      else g.style.removeProperty('--dgm-color');
    }
    this.nodeState = want;
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

  Player.prototype.applyFlows = function (want) {
    var key;
    for (key in this.particles) {
      if (!want[key]) {
        this.particles[key].remove();
        delete this.particles[key];
      }
    }
    for (key in want) {
      var tr = want[key].track;
      var bind = this.edges[tr.edge];
      if (!bind) continue;

      var group = this.particles[key];
      if (!group) {
        group = this.makeParticle(tr, this.layer);
        this.particles[key] = group;
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
      var pt = bind.path.getPointAtLength(u * len);
      if (bind.matrix && pt.matrixTransform) {
        pt = pt.matrixTransform(bind.matrix);
      }
      group.setAttribute('transform', 'translate(' + pt.x + ',' + pt.y + ')');
    }
  };

  Player.prototype.makeParticle = function (tr, parent) {
    var g = document.createElementNS(SVG_NS, 'g');
    g.setAttribute('class', 'dgm-particle' + (tr.style ? ' dgm-particle-' + tr.style : ''));
    // A custom property rather than a fill, so the stylesheet keeps deciding
    // which of the dot, halo and label the colour reaches.
    if (tr.color) g.style.setProperty('--dgm-color', tr.color);

    var halo = document.createElementNS(SVG_NS, 'circle');
    halo.setAttribute('r', '11');
    halo.setAttribute('class', 'dgm-particle-halo');
    g.appendChild(halo);

    var dot = document.createElementNS(SVG_NS, 'circle');
    dot.setAttribute('r', '5.5');
    dot.setAttribute('class', 'dgm-particle-dot');
    g.appendChild(dot);

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

  Player.prototype.applyNotes = function (want) {
    var key;
    for (key in this.notes) {
      if (!want[key]) {
        this.notes[key].remove();
        delete this.notes[key];
      }
    }
    var stageRect = this.stage.getBoundingClientRect();
    for (key in want) {
      var tr = want[key];
      var node = this.nodes[tr.target];
      if (!node) continue;

      var div = this.notes[key];
      if (!div) {
        div = el('div', 'dgm-note');
        div.textContent = tr.text;
        this.overlay.appendChild(div);
        this.notes[key] = div;
      }
      var r = node.getBoundingClientRect();
      div.style.left = (r.left - stageRect.left + r.width / 2) + 'px';
      div.style.top = (r.top - stageRect.top - 12) + 'px';
    }
  };

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
  var STICKY = { 'dgm-clickable': true, 'dgm-collapsed': true };

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

  window.Diagramator = {
    mount: function (root, timeline) {
      return new Player(root, timeline);
    }
  };
})();
