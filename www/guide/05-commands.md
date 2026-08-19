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
cinegram sheet   <file.dgm> -o sheet.png    # a labelled grid, one cell per step
cinegram narrate <file.dgm> [--format=md|json]   # the animation, written out
cinegram lint    <file.dgm> [--format=text|json] [--strict] [--fix] # diagnostics only
cinegram mcp                                # the same tools over MCP, on stdio
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

### Contact sheets

A recording is legible to a reader who can watch it. `sheet` is the same
walkthrough for a reader who cannot — one PNG, one captioned cell per step:

```
cinegram sheet examples/02-storytelling/01-payment-checkout.dgm -o checkout.png
cinegram sheet examples/02-storytelling/01-payment-checkout.dgm --scenario "gateway outage" \
  -o outage.png --manifest outage.json
```

Each cell is photographed one millisecond before its step ends. Not at the
start, where its flows have not gone anywhere yet; and not at the end either,
because "which step is this" resolves to the last step whose start is at or
before the clock, so the end of a step already belongs to the next one and
would caption the cell with the wrong name. One millisecond earlier is the last
instant that is unambiguously this step, with everything it did already done.

The labels cost nothing to draw: the cells are shot in `?embed`, which hides
the toolbar and the step list but keeps the caption, so every cell is titled by
the document in the document's own words. Nothing in cinegram renders text into
an image — the browser was already doing it correctly.

`--manifest` writes the map from pixels back to the document: the grid, the
cell size, and for every cell its step id, name, description, the moment it
shows and the rectangle it occupies. That is what makes the sheet addressable —
spot something wrong in the third cell, read off its step and its `at`, and
re-shoot exactly that moment with `frame` for a close-up.

| Flag | Default | |
| --- | --- | --- |
| `-o` | *required* | Where the PNG goes. |
| `--manifest` | off | Also write the cell map as JSON. |
| `--cols` | from the step count | Columns in the grid: as square as the count allows, at most 4 across. |
| `--width`, `--height` | `900`, `600` | One cell, which is the viewport each still is shot at. |
| `--scenario`, `--view` | the first / the entry document | Which walkthrough to lay out. |

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
`[{"file","line","col","severity","message","hint"}]` on stdout, each entry
optionally carrying `"fix": {"line","col","old","new"}` — the same suggestion in
a form something other than a person can act on. Exit codes stay
out of the payload — warnings 0, errors 1 — so a caller can branch on the status
*and* read the detail instead of choosing between them. Adding `--strict` makes
a warning exit 1 too, leaving the document byte-identical: an agent can then
loop on the status alone until the array is `[]`.

The page opens at rest — on the scenario's `poster` moment if it names one —
and plays only when asked: pressing Play, or a scenario that declares
`autoplay: true` (still skipped when the reader's system asks for reduced
motion). `window.CINEGRAM_PLAYER` is the same player, so
`CINEGRAM_PLAYER.seek(2400)` lands on a moment deterministically.

### Fixing what lint finds

```
cinegram lint out.dgm --fix
```

The "did you mean" diagnostics — a misspelt node, frame, view alias, step id,
scenario name or attribute key — carry the correction as a structured edit, not
only as English. `--fix` applies them and prints one line per edit on stderr:

```
fixed out.dgm:11:15: ingres -> ing
applied 1 fix
```

An edit is verified against the file before it is spliced: the text has to
still be what the compiler saw, or the fix is skipped and said so. Fixes are
applied right to left within a line so earlier columns cannot move, and the
file is re-parsed between rounds — up to five — because one correction can
uncover the next.

Whether a fix exists at all is decided in the parser, by the same closeness
bound that decides whether to say "did you mean" in the first place. There is
no second, looser rule at the command layer, so `--fix` can never apply an edit
the diagnostic would not have suggested out loud.

Everything else is unchanged: `--fix` composes with `--strict` and
`--format=json`, the JSON still goes to stdout on its own, and the exit status
is exactly the one a plain lint of the repaired file would earn.

### The MCP server

```
cinegram mcp
```

The same commands, offered down a pipe. `mcp` speaks the Model Context Protocol
on stdin and stdout, so an agent host that already knows how to launch a server
gets the tools without a shell:

```json
{"mcpServers": {"cinegram": {"command": "cinegram", "args": ["mcp"]}}}
```

Five tools, named after the subcommands they are:

| Tool | Returns |
| --- | --- |
| `lint` | the `--format=json` array, fixes included |
| `narrate` | the walkthrough, `format` `md` or `json` |
| `mermaid` | the diagram half as plain Mermaid |
| `frame` | the PNG of one moment, plus the view, scenario and time it shows |
| `sheet` | the contact-sheet PNG, plus the manifest that maps cells to steps |

Every tool takes `path` **or** `source`, never both: `path` names a file on
disk, `source` carries the document itself for a draft that has not been saved,
with an optional `as` filename so that relative `view … from` and storyboard
paths have somewhere to resolve (default `inline.dgm`). The rule is stated in
each schema and enforced in the handler, because client support for JSON
Schema's `oneOf` is uneven and a rule only half of them check is not a rule.

The one resource is `cinegram://reference/language.md`, the complete authoring
reference — the same file the skill ships, served out of the binary for a model
that has no skill folder installed.

`frame` and `sheet` need a headless Chrome or Chromium, found on `PATH` or
named by `$CINEGRAM_CHROME`, exactly as the subcommands do. Their descriptions
say so, so a model without one can choose `narrate` instead rather than
discovering it by failing. A diagnostic is never a failed call: `lint` on a
broken document returns the diagnostics and succeeds, and only a document that
could not be read at all comes back as an error.

The CLI stays primary — every tool is one subcommand's code path called
in-process, so the two cannot disagree about what a document means, and nothing
here is reachable only through MCP.

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
| `?` | Show or hide the settings and shortcuts sheet |

Scroll to zoom the stage — anchored on the cursor, so the thing you are
pointing at stays put — drag to pan, and a double-click puts the framing back.
Past fit a minimap appears in the stage's corner: the whole diagram with a
rectangle around the part you are looking at. Click or drag it to move the
view, and double-click it — or press the **Fit** button in its corner — to put
the whole diagram back. The map and that button are on screen only while
something is zoomed, because that is the only time either has anything to say.
Notes, badges and gauges are positioned from live element rects, so they stay
glued to their nodes at any zoom. Zoom resets when you change view, since a
framing that suited one diagram means nothing over the next.

Every clickable element is reachable by `Tab` and activates with `Enter` or
`Space`; the step list is real buttons, not clickable rows.

Theme and speed persist in `localStorage`. The theme control is page chrome
rather than a diagram tool — top right on a page `cinegram preview` writes, in
the header of a `cinegram site` page, beside the other buttons in the playground
— because dark and light describe the page, and every player on it follows the
`data-theme` the control writes. It is a **light ⇄ dark** flip. Until you press
it a page has no theme of its own and simply shows whichever your OS is set to,
following it live through `prefers-color-scheme` with no script in the way; the
first press picks a side and keeps it, on that page and every other cinegram
page in that browser. Inside VS Code there is no control at all: the editor's
theme is the answer, and the preview follows it live.

Speed is a setting rather than a tool, so it lives in the sheet `?` and the
rail's last button open — settings above, shortcuts below. The menu offers
`0.25 → 0.5 → 1 → 1.5 → 2`, and a scenario that declares a rate of its own that
is not on that list shows it there too, for as long as it is the rate in effect.

The remembered speed is scoped the other way from the theme: it is one key for
every diagram on the origin, so a scenario that declares its own `speed` keeps
it — an author's pacing should not be overridden by a 0.25x you picked on some
other diagram last week. The remembered rate applies to scenarios that leave
`speed` unset.

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

Warnings never fail a build unless you ask (`lint --strict`); errors always do.
Diagnostics carry a line, a column, and usually a suggestion:

```
errors.dgm:11:15: error: "ingres" is not a node in this diagram
  hint: did you mean ing?
errors.dgm:15:20: error: no edge between "client" and "svc" to animate along
  hint: add `client --> svc` to the diagram, or route the flow through nodes that are connected
```

`cinegram lint --strict` moves only the exit status: the same diagnostics are
printed, and a run with nothing but warnings exits 1 instead of 0. It is for a
CI job, or an agent looping until its diagram is clean, that wants "animates
less than it was meant to" to stop the loop the same way a real error does.
