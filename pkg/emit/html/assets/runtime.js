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
      // The list stays a compact index; the first line of the prose is enough
      // to tell two similarly named steps apart on hover.
      if (st.desc) li.title = firstLine(st.desc);
      li.addEventListener('click', function () { self.seek(st.start); });
      self.steps.appendChild(li);
    });

    this.scrub.max = String(sc.duration || 0);
    this.buildScrubMarks(sc);

    // The caption belongs to whichever scenario is showing, so a switch has to
    // let it redraw even if the new step happens to carry the same id.
    this.captionKey = null;
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

  function firstLine(s) {
    var i = s.indexOf('\n');
    return i < 0 ? s : s.slice(0, i);
  }

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

    var current = null;
    var kids = this.steps.children;
    for (var i = 0; i < kids.length; i++) {
      var st = sc.steps[i];
      var active = st && this.time >= st.start && this.time < st.end;
      var done = st && this.time >= st.end;
      if (active) current = st;
      kids[i].classList.toggle('is-active', !!active);
      kids[i].classList.toggle('is-done', !!done);
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
    var key = step ? step.id + ' ' + (step.name || '') + ' ' + (step.desc || '') : '';
    if (key === this.captionKey) return;
    this.captionKey = key;

    this.caption.innerHTML = '';
    this.caption.classList.toggle('is-on', !!step);
    if (!step) return;

    this.caption.appendChild(elText('span', 'dgm-caption-name', step.name || step.id));
    if (step.desc) this.caption.appendChild(elText('span', 'dgm-caption-desc', step.desc));
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
    var sc = this.scenario();

    var wantNode = {};   // node id -> {cls, color, key}
    var wantEdge = {};   // edge id -> {cls, color, key}
    var wantFlow = {};   // track key -> {track, progress}
    var wantNote = {};   // note key -> {track, target}
    var wantWaypoint = {}; // node id -> true

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
          var progress = Math.min(1, Math.max(0, p));
          wantFlow[key] = { track: tr, progress: progress };
          if (tr.edge) wantEdge[tr.edge] = edgeState(tr);
          if (tr.to && progress >= 1 - WAYPOINT_LEAD) wantWaypoint[tr.to] = true;
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

    this.applyNodeStates(wantNode);
    this.applyEdgeStates(wantEdge);
    this.applyFlows(wantFlow);
    this.applyNotes(wantNote);
    this.applyPills(standing.pills);
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

    for (var i = 0; i < list.length; i++) {
      var tr = list[i];
      if (t < tr.start) continue;
      if (t >= tr.end && !(tr.end >= sc.duration && t >= sc.duration)) continue;

      if (tr.kind === 'set' && tr.value) {
        states[tr.target] = { name: tr.value, color: tr.color || '' };
      }
      var text = tr.kind === 'gauge' ? tr.value : tr.label;
      if (!text) continue;

      if (!pills[tr.target]) pills[tr.target] = [];
      pills[tr.target].push({
        key: sc.id + ':' + i,
        kind: tr.kind,
        label: tr.kind === 'gauge' ? tr.label : '',
        value: text,
        color: tr.color || '',
        style: tr.style || ''
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
      var host = self.elementFor(target);
      if (!host) return;
      var r = host.getBoundingClientRect();

      groups[target].forEach(function (p, i) {
        var div = self.pills[p.key];
        if (!div) {
          div = makePill(p);
          self.overlay.appendChild(div);
          self.pills[p.key] = div;
        }
        div.style.left = (r.right - stageRect.left) + 'px';
        div.style.top = (r.top - stageRect.top + i * PILL_STACK) + 'px';
      });
    });
  };

  var PILL_STACK = 22;

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
  function edgeState(tr) {
    var cls = 'flow-active';
    if (tr.style) cls += ' flow-' + tr.style;
    if (tr.status === 'fail') cls += ' flow-fail';
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

  Player.prototype.applyNodeStates = function (want) {
    var self = this;
    this.nodeState = applyStates(this.nodeState, want, function (id) {
      return self.nodes[id];
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
      var pt = bind.path.getPointAtLength(along);
      if (bind.matrix && pt.matrixTransform) {
        pt = pt.matrixTransform(bind.matrix);
      }
      fx.group.setAttribute('transform', 'translate(' + pt.x + ',' + pt.y + ')');

      if (fx.trail) this.drawTrail(fx.trail, along, len, backwards);
      if (tr.status === 'fail') {
        this.drawFailMark(fx, bind, len, backwards, want[key].progress >= 1 - FAIL_LEAD);
      }
    }
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

    var pt = bind.path.getPointAtLength(backwards ? 0 : len);
    if (bind.matrix && pt.matrixTransform) {
      pt = pt.matrixTransform(bind.matrix);
    }
    fx.mark.setAttribute('transform', 'translate(' + pt.x + ',' + pt.y + ')');
  };

  // makeFlow builds everything one open flow track draws: the trail underneath,
  // then the particle over it. The ✕ is added later, only if the flow gets far
  // enough to need one.
  Player.prototype.makeFlow = function (tr, bind) {
    return {
      trail: makeTrail(tr, bind, this.layer),
      group: this.makeParticle(tr, this.layer),
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
