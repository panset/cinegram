/* Mounts cinegram players into the pages of a Material-family static site.
 *
 * Written for Zensical and Material for MkDocs, and used by cinegram's own
 * site. Nothing here is specific to either beyond the palette hooks, which
 * degrade rather than break on a theme that does not have them.
 *
 * Markup contract, written by hand in a Markdown page:
 *
 *   <div class="cinegram" data-cinegram="01-basics/k8s-request" data-height="900"></div>
 *
 * `data-cinegram` names a timeline under `<this script's folder>/timelines/`,
 * with `.json` appended — subfolders and all. Compile one with
 * `cinegram compile path/to.dgm -o timelines/name.json`.
 *
 * `data-height` is the space the box reserves before the player mounts, and
 * only until then: the player replaces it with its own height, measured from
 * the diagram, the step list and the caption. Set it to roughly what the
 * mounted diagram measures. Too small and the page jumps as each one arrives,
 * which on a page of five is the whole article moving under the reader; too
 * large costs nothing once the player is up, so guess high.
 *
 * Paths are resolved against this script's own URL rather than the site root,
 * so the kit works unchanged under a path prefix — a GitHub Pages project
 * site, or an enterprise install serving from /pages/org/repo/.
 *
 * Nothing is fetched on a page that holds no host. That is the whole reason
 * this file exists rather than listing mermaid and the runtime directly in the
 * theme's `extra_javascript`: mermaid alone is 2.6 MB, and every page of the
 * site would pay for it to serve the handful that show a diagram.
 */
(function () {
  'use strict';

  var HOSTS = '.cinegram[data-cinegram]';

  var BASE = (function () {
    var s = document.currentScript;
    return s ? s.src.replace(/[^/]*$/, '') : 'assets/cinegram/';
  })();

  /** Every mounted player, so a theme or fullscreen change reaches all of them. */
  var players = [];

  /** True when the page carries exactly one diagram — see `present` below. */
  var solo = false;

  function load(tag, attrs) {
    return new Promise(function (resolve, reject) {
      var el = document.createElement(tag);
      Object.keys(attrs).forEach(function (k) { el[k] = attrs[k]; });
      el.onload = resolve;
      el.onerror = function () { reject(new Error('failed to load ' + (attrs.src || attrs.href))); };
      document.head.appendChild(el);
    });
  }

  /** The three runtime files, fetched once for the page and only on demand. */
  var runtime = null;
  function loadRuntime() {
    if (!runtime) {
      runtime = load('link', { rel: 'stylesheet', href: BASE + 'runtime.css' })
        // mermaid first: runtime.js reads window.mermaid when it renders.
        .then(function () { return load('script', { src: BASE + 'mermaid.min.js' }); })
        .then(function () { return load('script', { src: BASE + 'runtime.js' }); });
    }
    return runtime;
  }

  /**
   * 'slate' is Material's dark palette and 'default' its light one. A theme
   * that sets neither is not necessarily a light site, so fall back to what
   * the reader's system asks for rather than assuming.
   */
  function theme() {
    var scheme = document.body.getAttribute('data-md-color-scheme');
    if (scheme) return scheme === 'slate' ? 'dark' : 'light';
    return window.matchMedia && window.matchMedia('(prefers-color-scheme: dark)').matches
      ? 'dark'
      : 'light';
  }

  function fail(host, message) {
    host.classList.add('cinegram--failed');
    host.textContent = message;
  }

  // --- fullscreen, spelled for Safari as well ------------------------------

  function fullscreenEl() {
    return document.fullscreenElement || document.webkitFullscreenElement || null;
  }

  function enterFullscreen(el) {
    var req = el.requestFullscreen || el.webkitRequestFullscreen;
    // A rejected request is not a failure worth reporting: presenter mode is
    // perfectly usable in the flow of the page, it is just smaller.
    if (req) try { Promise.resolve(req.call(el)).catch(function () {}); } catch (e) {}
  }

  function leaveFullscreen() {
    var ex = document.exitFullscreen || document.webkitExitFullscreen;
    if (ex) try { Promise.resolve(ex.call(document)).catch(function () {}); } catch (e) {}
  }

  /**
   * Presenter mode is unreachable in a document unless we put the control
   * back. Two separate things hide it, both deliberate in the runtime:
   *
   *   - `.dgm-inline` blanks the whole control bar and restores only Play and
   *     the scenario picker, on the grounds that the rest is for authoring a
   *     diagram rather than reading one;
   *   - a player mounted with `hash: false` ignores `?present` in the page's
   *     URL, because a query string meant for the page is not an instruction
   *     to a diagram inside it.
   *
   * Both are right for an embed in general and wrong for a page whose whole
   * job is to show the diagram. So we un-hide the runtime's own button rather
   * than building one: Exit, the Escape key and the label are then all still
   * the runtime's, and if a future version renames the property presenter mode
   * simply stays hidden instead of breaking. What we add is the fullscreen the
   * mode implies — a 60vh stage is worth little inside a box on an article.
   */
  function enablePresenter(host, player) {
    var btn = player.presentBtn;
    if (!btn) return;
    btn.style.display = 'inline-block';

    // Runs after the runtime's own click handler, so `present` is already the
    // state we are moving to. Being inside the click keeps the user gesture
    // that requestFullscreen needs.
    btn.addEventListener('click', function () {
      if (player.present) enterFullscreen(host);
      else if (fullscreenEl() === host) leaveFullscreen();
    });
  }

  // Leaving fullscreen by any route — Escape, F11, the browser's own chrome —
  // leaves presenter mode with it. Otherwise the page is left holding a stage
  // sized for a projector and a transport that only steps one beat at a time.
  function watchFullscreen() {
    var sync = function () {
      // A class rather than the :fullscreen pseudo, so the stylesheet states
      // the fullscreen layout once instead of twice for Safari's prefix —
      // one unknown pseudo-class drops a whole selector list.
      var fs = fullscreenEl();
      [].forEach.call(document.querySelectorAll(HOSTS), function (h) {
        h.classList.toggle('cinegram--fullscreen', h === fs);
      });
      if (fs) return;
      players.forEach(function (p) { if (p.present) p.setPresenter(false); });
    };
    document.addEventListener('fullscreenchange', sync);
    document.addEventListener('webkitfullscreenchange', sync);
  }

  function mount(host) {
    var name = host.getAttribute('data-cinegram');
    host.classList.add('cinegram--loading');

    Promise.all([
      loadRuntime(),
      fetch(BASE + 'timelines/' + name + '.json').then(function (r) {
        if (!r.ok) throw new Error(r.status + ' ' + r.statusText);
        return r.json();
      })
    ]).then(function (both) {
      host.classList.remove('cinegram--loading');
      // Release the reservation. data-height is a guess at what the player
      // will want, and it stops being a guess the moment the player is up.
      // Held any longer it would leave a permanent band of empty space under
      // every diagram whose author guessed high — and guessing high is the
      // direction that avoids the scroll jump, so it has to be the cheap one.
      host.style.minHeight = '';
      host.classList.add('cinegram--mounted');
      host.textContent = '';
      // Mount into a child rather than the host itself: the player owns its
      // root's className outright and would drop the .cinegram class the
      // page's own framing hangs off.
      var stage = document.createElement('div');
      host.appendChild(stage);
      var player = window.Cinegram.mount(stage, both[1], {
        // The reader's transport — play, scrub, caption — without the
        // authoring chrome the standalone page carries.
        inline: true,
        // Listen on this element only. Without it the player would swallow
        // the arrow keys and `/` that the site's own search and navigation
        // want, on every page that shows a diagram.
        keys: 'scoped',
        // The address bar belongs to the page: the theme writes heading
        // anchors there, and a player treating one as a view id would jump
        // somewhere the reader never asked to go. Drill-down and Back stay
        // inside the player instead.
        hash: false,
        // Several diagrams starting to move as the page renders is noise.
        autoplay: false,
        // A page carrying exactly one diagram may open straight into presenter
        // mode with `?present`, the way the standalone page does. Withheld on a
        // page of several, where it would start every one of them at once.
        present: solo && /(^|[?&])present(=|&|$)/.test(location.search),
        theme: theme()
      });
      players.push(player);
      enablePresenter(host, player);
    }).catch(function (err) {
      fail(host, 'Could not load the diagram "' + name + '": ' + err.message);
    });
  }

  function init() {
    var hosts = [].slice.call(document.querySelectorAll(HOSTS));
    if (!hosts.length) return;
    solo = hosts.length === 1;
    watchFullscreen();

    hosts.forEach(function (host) {
      var h = host.getAttribute('data-height');
      if (h) host.style.minHeight = /^\d+$/.test(h) ? h + 'px' : h;
    });

    // A diagram is mounted when it is close to being looked at. Mounting all
    // of them up front would run mermaid over five layouts before the reader
    // has scrolled past the first.
    if (!('IntersectionObserver' in window)) {
      hosts.forEach(mount);
      return;
    }
    var seen = new IntersectionObserver(function (entries) {
      entries.forEach(function (e) {
        if (!e.isIntersecting) return;
        seen.unobserve(e.target);
        mount(e.target);
      });
    }, { rootMargin: '400px' });
    hosts.forEach(function (host) { seen.observe(host); });

    // mermaid picks its palette per render, so a palette switch has to redraw
    // rather than merely restyle. setTheme does both.
    new MutationObserver(function () {
      var t = theme();
      players.forEach(function (p) { p.setTheme(t); });
    }).observe(document.body, {
      attributes: true,
      attributeFilter: ['data-md-color-scheme']
    });
  }

  if (document.readyState === 'loading') {
    document.addEventListener('DOMContentLoaded', init);
  } else {
    init();
  }
})();
