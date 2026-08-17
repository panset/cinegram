<!-- Generated from README.md by `bazel run //site:sync`. Do not edit: //site:site_test fails while this file disagrees with its source, and the next sync overwrites it. -->

# Commands

## Commands

```
cinegram compile <file.dgm> [-o out.json]   # animation timeline JSON
cinegram mermaid <file.dgm> [-o out.mmd]    # the diagram as plain Mermaid
cinegram preview <file.dgm> [-o out.html]   # self-contained animated page
cinegram site    <folder>   -o out/         # a browsable site from a folder tree
cinegram assets             -o dir/         # the embed kit, for a site of your own
cinegram frame   <file.dgm> --at 1620ms -o still.png   # one exact moment
cinegram record  <file.dgm> -o out.gif      # a GIF, mp4 or webm of one scenario
cinegram narrate <file.dgm> [--format=md|json]   # the animation, written out
cinegram lint    <file.dgm> [--format=text|json] # diagnostics only
```

### Authoring

```
cinegram preview examples/01-basics/01-k8s-request.dgm --serve --watch
```

Serves the page at `http://127.0.0.1:8731/`, compiled from source on every
request, and reloads the browser within about a second of a save. The watched
set is the bundle's own file list, so adding a `view` starts watching its
target without a restart.

The reload script is injected by the server, never by the renderer — the file
`preview -o` writes stays byte-identical to what it always was. A document that
stops parsing shows the error in the page rather than going blank, which with
the reload loop is the fastest way to read it.

For a still of one exact moment:

```
cinegram frame examples/02-storytelling/01-payment-checkout.dgm --at 1620ms --scenario s1 -o fail.png
```

That is `examples/02-storytelling/01-payment-checkout.fail.png` — the gateway timing out, ✕ and
all. It works by opening the Phase-6 deep link in a headless Chrome, and
because a deep link lands *paused*, the screenshot is deterministic rather than
a race against the animation. The browser is found on `PATH`
(`google-chrome`, `chromium`, …) or named by `$CINEGRAM_CHROME`; shelling
out to one the machine already has is what keeps the no-dependencies rule.

`--frames N -o dir/` captures N evenly spaced moments as a numbered sequence.

### A site from a folder

```
cinegram site diagrams/ --serve --watch     # browse a folder tree, live
cinegram site diagrams/ -o site/            # or export it as static files
```

Point `site` at a folder of `.dgm` files and it becomes a browsable site: one
page per document at the same relative path, an index per folder, a sidebar
carrying the whole tree, breadcrumbs, and prev/next arrows walking the site in
depth-first order. Folders are the hierarchy; an optional numeric filename
prefix (`01-intro.dgm`) forces order within one and is stripped from
everything a reader sees. A file another document pulls in via `view … from`
is reached by drill-down rather than listed.

Unlike `preview`, site pages share one copy of the runtime from `assets/`
instead of inlining 2.8 MB each — a 30-diagram site is a few megabytes, not
ninety. The export still works over `file://`.

Presentation is flags: `--title`, `--link Name=URL` for header links,
`--playground URL` to give every page an *Edit in playground* button whose
link carries the whole document (the playground's own share-link format,
minted at build time — nothing is uploaded), and `--hero` for the card on the
root index.

### A diagram inside a page you already wrote

`site` owns the whole page. When the diagram belongs in the middle of an
architecture doc or an RFC — a paragraph, not the point — install the **embed
kit** into a folder your site serves and write a `<div>`:

```
cinegram assets -o docs/assets/cinegram
cinegram compile diagrams/failover.dgm -o docs/assets/cinegram/timelines/failover.json
```

```html
<div class="cinegram" data-cinegram="failover" data-height="900"></div>
```

Five files: the loader, its stylesheet, and the three runtime files. The
loader finds its siblings and the `timelines/` folder relative to its own URL,
so the kit works unchanged under a path prefix, and it is **inert on a page
with no diagram** — which is why the 2.6 MB of mermaid can be listed site-wide
without every page paying for it. The player arrives as a guest: keys scoped to
itself, the address bar left to the theme, lazy mount, and the palette
following the site's own light/dark toggle.

This website is built that way, so the path is the one it documents:
<https://panset.github.io/cinegram/embedding/>.

### Recording

`record` is `frame` in a loop, encoded:

```
cinegram record examples/02-storytelling/01-payment-checkout.dgm -o checkout.gif --fps 10
cinegram record examples/02-storytelling/04-oidc-login.dgm -o login.mp4 --scenario s0
```

Because every frame is an independent deep link that lands paused, the
recording is a sequence of deterministic stills rather than a capture of
playback — the output does not depend on how fast the machine is. Frames are
shot in parallel, four browsers at a time, and taken in `?embed` mode so the
page furniture is not in the picture.

**GIF needs nothing installed.** The encoder is in `pkg/gifenc`: median-cut
quantization onto one 256-colour palette shared by every frame, and delays
tiled so 12fps stays 12fps instead of drifting a centisecond a frame. One
palette rather than one per frame is deliberate — per-frame palettes cost
several times as much and make the result shimmer, because a colour quantized
one way in frame 3 and another way in frame 4 flickers even where nothing moved.

`--format mp4` and `--format webm` shell out to **ffmpeg**, which is found on
`PATH` or named by `$CINEGRAM_FFMPEG`. It stays strictly out of the GIF path, so
"export this diagram for a pull request" never turns into "install a package
first".

| Flag | Default | |
| --- | --- | --- |
| `-o` | *required* | Where the recording goes. |
| `--format` | from the `-o` extension | `gif`, `mp4` or `webm`. |
| `--fps` | `12` | |
| `--width`, `--height` | `1280`, `720` | Rounded up to even, which yuv420p requires and a GIF does not mind. |
| `--reel` | off | The `?reel` story page at 1080×1920, auto-follow camera included. Explicit dimensions still win, per axis. |
| `--scenario`, `--view` | the first / the entry document | Which walkthrough to record. |
| `--progress` | off | `cinegram-progress capture <i> <n>` per frame and one `cinegram-progress encode`, on stderr, for a host drawing a progress bar. Purely additive: the human-readable lines are unchanged. |

### What an agent sees

An animation is legible to something with eyes. A timeline is precise but it is
a list of intervals. `narrate` is the third form — the same facts in the order a
reader meets them, with each track stated as a sentence:

```
#### 5. The app trades the code for tokens

2.9s–3.7s

Now the application speaks to the provider directly, back channel, server to
server. It sends the code together with its client secret, and that pairing is
what proves the exchange is genuine.

- **app → auth** carries "POST /token + secret" (2.9s–3.6s)
- **auth → tokens** carries "mint & record" (3.2s–3.7s)
```

`examples/02-storytelling/03-oauth-login.narrate.md` is that output, committed. `--format=json`
emits the same walkthrough as data — each event carries both the sentence and
the fields it was built from, so filtering for "every failing flow" does not
mean parsing the prose back apart.

`lint --format=json` emits
`[{"file","line","col","severity","message","hint"}]` on stdout. Exit codes are
unchanged — warnings 0, errors 1 — so a caller can branch on the status *and*
read the detail instead of choosing between them.

The page opens at rest — on the scenario's `poster` moment if it names one —
and plays only when asked: pressing Play, or a scenario that declares
`autoplay: true` (still skipped when the reader's system asks for reduced
motion). `window.CINEGRAM_PLAYER` is the same player, so
`CINEGRAM_PLAYER.seek(2400)` lands on a moment deterministically.

### Driving the player

Press `?` in the page for this list.

| Key | Does |
| --- | --- |
| `Space` | Play or pause — in presenter mode, play exactly the next step |
| `←` / `→` | Previous or next step |
| `Home` / `End` | Jump to the start or the end |
| `1`–`9` | Jump to step *n* |
| Click stage | In presenter mode, advance one step |
| `Esc` | Leave presenter mode, back out of a drilled-in view, or close the help |
| `?` | Show or hide the shortcut list |

Scroll to zoom the stage — anchored on the cursor, so the thing you are
pointing at stays put — drag to pan, and a double-click, or **Reset zoom** on
the tool rail, puts the framing back. Notes,
badges and gauges are positioned from live element rects, so they stay glued to
their nodes at any zoom. Zoom resets when you change view, since a framing that
suited one diagram means nothing over the next.

Every clickable element is reachable by `Tab` and activates with `Enter` or
`Space`; the step list is real buttons, not clickable rows.

Theme and speed persist in `localStorage`. The theme follows your system until
you press the button, and then stops — an explicit choice is exactly the thing
that should not be overridden later. The remembered speed is scoped the other
way: it is one key for every diagram on the origin, so a scenario that declares
its own `speed` keeps it — an author's pacing should not be overridden by a
0.25x you picked on some other diagram last week. The remembered rate applies
to scenarios that leave `speed` unset. The speed button cycles
`0.25 → 0.5 → 1 → 1.5 → 2` and its label always shows the rate actually in
effect.

With `prefers-reduced-motion: reduce`, nothing autoplays, the decorative
animation stops, and the help says so — stepping through with the arrow keys is
the intended way to read it. The particle still travels, because that is the
content rather than an effect.

Relative paths are resolved against the directory you ran the command from,
including under `bazel run` — the binary executes from its runfiles tree, so
Cinegram honours Bazel's `BUILD_WORKING_DIRECTORY` to get this right.
`preview` with no `-o` writes beside its input.

Some warnings are about omission rather than error — the mistakes that leave a
document compiling perfectly and animating less than its author meant:

- an element declared and then never referenced by an edge, a scenario or a
  click;
- a `click … -> step` whose target only exists in a later scenario, so it
  resolves and then does nothing until that scenario is chosen;
- two scenarios sharing a name, which the picker shows nothing else of;
- an empty `%%` comment line, which breaks Mermaid's own comment stripping and
  takes the whole diagram down with it.

Warnings never fail a build; errors do. Diagnostics carry a line, a column, and
usually a suggestion:

```
errors.dgm:11:15: error: "ingres" is not a node in this diagram
  hint: did you mean ing?
errors.dgm:15:20: error: no edge between "client" and "svc" to animate along
  hint: add `client --> svc` to the diagram, or route the flow through nodes that are connected
```
