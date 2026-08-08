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

  function indexClusters(svg, timeline) {
    var map = {};
    var els = svg.querySelectorAll('g.cluster');
    for (var i = 0; i < els.length; i++) {
      var el = els[i];
      var id = (el.id || '').replace(/-\d+$/, '');
      map[id] = el;
    }
    // Clusters are keyed by the subgraph id in every version we support, but
    // fall back to declaration order if that ever stops being true.
    var missing = timeline.groups.filter(function (g) { return !map[g.id]; });
    if (missing.length === els.length && els.length === timeline.groups.length) {
      for (var j = 0; j < timeline.groups.length; j++) {
        map[timeline.groups[j].id] = els[j];
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

  function indexEdges(svg, timeline, nodes) {
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
    for (var e = 0; e < timeline.edges.length; e++) {
      var edge = timeline.edges[e];
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
    this.scenarioIndex = 0;
    this.time = 0;
    this.playing = false;
    this.speed = 1;
    this.raf = null;
    this.lastFrame = 0;
    this.nodeState = {};
    this.particles = {};
    this.notes = {};
    this.theme = matchMedia('(prefers-color-scheme: dark)').matches ? 'dark' : 'light';
    this.build();
  }

  Player.prototype.scenario = function () {
    return this.timeline.scenarios[this.scenarioIndex] || { steps: [], duration: 0 };
  };

  Player.prototype.build = function () {
    var self = this;
    this.root.innerHTML = '';
    this.root.className = 'dgm';

    var bar = el('div', 'dgm-bar');
    this.title = el('div', 'dgm-title');
    bar.appendChild(this.title);

    var controls = el('div', 'dgm-controls');

    if (this.timeline.scenarios.length > 1) {
      this.picker = el('select', 'dgm-select');
      this.timeline.scenarios.forEach(function (s, i) {
        var o = document.createElement('option');
        o.value = String(i);
        o.textContent = s.name || 'scenario ' + (i + 1);
        self.picker.appendChild(o);
      });
      this.picker.addEventListener('change', function () {
        self.selectScenario(parseInt(self.picker.value, 10));
      });
      controls.appendChild(this.picker);
    }

    this.playBtn = button('Play', 'dgm-btn dgm-btn-primary', function () { self.toggle(); });
    controls.appendChild(this.playBtn);
    controls.appendChild(button('Restart', 'dgm-btn', function () { self.seek(0); }));

    this.speedBtn = button('1x', 'dgm-btn', function () { self.cycleSpeed(); });
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
    document.addEventListener('keydown', function (ev) {
      if (ev.target && /input|select|textarea/i.test(ev.target.tagName)) return;
      if (ev.key === ' ') { ev.preventDefault(); self.toggle(); }
      if (ev.key === 'ArrowRight') { ev.preventDefault(); self.nextStep(1); }
      if (ev.key === 'ArrowLeft') { ev.preventDefault(); self.nextStep(-1); }
    });

    this.render();
  };

  Player.prototype.selectScenario = function (i) {
    this.scenarioIndex = i;
    this.time = 0;
    this.buildSteps();
    this.apply(0);
    this.syncChrome();
  };

  Player.prototype.render = function () {
    var self = this;
    var d = this.timeline.diagram;

    this.title.textContent = this.timeline.title || 'Diagramator';

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
    this.nodes = indexNodes(this.svg);
    this.clusters = indexClusters(this.svg, this.timeline);
    this.layer = makeLayer(this.svg);
    this.edges = indexEdges(this.svg, this.timeline, this.nodes);

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
    this.timeline.nodes.forEach(function (n) {
      if (!self.nodes[n.id]) problems.push('node "' + n.id + '" not found in the rendered SVG');
    });
    this.timeline.edges.forEach(function (e) {
      if (!self.edges[e.id]) problems.push('edge ' + e.from + ' → ' + e.to + ' could not be matched to a path');
    });
    this.warn(problems);
  };

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

  Player.prototype.cycleSpeed = function () {
    var order = [0.25, 0.5, 1, 1.5, 2];
    var i = order.indexOf(this.speed);
    this.speed = order[(i + 1) % order.length];
    this.speedBtn.textContent = this.speed + 'x';
  };

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

    var wantNode = {};   // node id -> state class
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
          wantNode[tr.target] = wantNode[tr.target] || 'noted';
        } else if (tr.kind === 'hide') {
          wantNode[tr.target] = 'hidden';
        } else {
          wantNode[tr.target] = tr.style ? tr.kind + ' ' + tr.style : tr.kind;
        }
      }
    }

    this.applyNodeStates(wantNode);
    this.applyFlows(wantFlow);
    this.applyNotes(wantNote);
  };

  Player.prototype.applyNodeStates = function (want) {
    // Diff against the previous frame so we only touch the DOM on change.
    var id;
    for (id in this.nodeState) {
      if (want[id] === this.nodeState[id]) continue;
      var prev = this.nodes[id];
      if (prev) prev.setAttribute('class', baseClass(prev));
    }
    for (id in want) {
      var g = this.nodes[id];
      if (!g) continue;
      if (this.nodeState[id] === want[id]) continue;
      g.setAttribute('class', baseClass(g) + ' dgm-' + want[id].split(' ').join(' dgm-'));
    }
    this.nodeState = want;
  };

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

      // `reverse` is what the source asked for; `flip` corrects for mermaid
      // having drawn the path from the other end. They compose.
      var backwards = !!tr.reverse !== !!bind.flip;
      var u = backwards ? 1 - want[key].progress : want[key].progress;
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

  // baseClass strips any dgm-* classes we previously added, so state changes
  // never accumulate on an element.
  function baseClass(el) {
    var cls = el.getAttribute('class') || '';
    return cls
      .split(/\s+/)
      .filter(function (c) { return c && c.indexOf('dgm-') !== 0; })
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
