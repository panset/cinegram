# Diagramator

Animated architecture diagrams from a Mermaid-like DSL.

A static diagram shows you that an Ingress sits in front of a Service. It cannot
show you what happens to a single `GET /api/orders` as it travels
LB → Ingress → Service → Pod and back. Diagramator adds a small animation
language on top of Mermaid so that path becomes something you can watch.

```
bazel run //cmd/diagramator -- preview examples/k8s-request.dgm -o /tmp/k8s.html
open /tmp/k8s.html
```

## The language

A `.dgm` file is a Mermaid diagram followed by one or more `scenario` blocks,
plus optional `view` and `interact` blocks that make elements clickable.

```
flowchart LR
  client[External Client]
  lb[(Cloud Load Balancer)]

  subgraph cluster[Kubernetes Cluster]
    ing[Ingress Controller]
    subgraph ns[namespace: prod]
      svc[ClusterIP Service]
      pod1[Pod A]
      pod2[Pod B]
    end
  end

  client --> lb
  lb --> ing
  ing --> svc
  svc --> pod1
  svc --> pod2

scenario "GET /api/orders" { speed: 1.0, loop: true }

  step route "Ingress rule matches host and path" {
    note ing "host: api.example.com\npath: /api/*"
    flow ing -> svc { dur: 500ms }
  }

  step respond "Response travels back to the client" {
    flow pod1 -> svc -> ing -> lb -> client {
      label: "200 OK", dur: 1400ms, style: response
    }
  }
```

Two properties shape everything else:

**The diagram half is untouched Mermaid.** Delete the scenario blocks and you
have a file any Mermaid renderer will draw. `diagramator mermaid` does exactly
that, and it is lossless by construction — statements are reprinted from their
original source text, so `classDef`, `click`, and any Mermaid syntax added after
this parser was written all survive verbatim.

**Actions inside a step start together; steps run one after another.** That
single rule covers most real scenarios. Reach for `seq { … }` when you need
actions inside one step to chain instead.

### Actions

| Action | Form | Notes |
| --- | --- | --- |
| `flow` | `flow a -> b -> c` | A packet travelling the hops. Each hop becomes its own track. |
| `highlight` | `highlight a, b` | Emphasise nodes. |
| `note` | `note a "text"` | A callout anchored to a node. `\n` works. |
| `dim` | `dim a` | Fade a node back. |
| `pulse` | `pulse a` | Repeating pulse. |
| `focus` | `focus domain` | Hold attention here; everything else recedes. |
| `show` / `hide` | `show a` | Reveal or conceal. |
| `set` | `set n1 { badge: "leader" }` | Standing state. Outlives its step — see below. |
| `gauge` | `gauge n1 { label: "term", value: 2 }` | A named reading that persists and updates. |
| `unset` | `unset n1` | Retire everything `set` or `gauge` put on a node. |
| `wait` | `wait 500ms` | Consume time, draw nothing. |
| `seq` | `seq { … }` | Run the contained actions in sequence. |

### Attributes

Written as `{ key: value, … }` after an action, or as bare `key: value` lines
inside a step or scenario body. Blocks may span lines.

Every action understands `label`, `dur`, `delay`, `at` and `style`; the rest are
per action, and an attribute an action does not understand is a warning with a
suggestion rather than a silent no-op.

| Attribute | Where | Meaning |
| --- | --- | --- |
| `label` | any action | Text carried by a flow, or a caption. |
| `dur` | any action | `600ms`, `1.2s`, or a bare number of milliseconds. |
| `delay`, `at` | any action | Offset the start within the step. |
| `style` | any action | A name the renderer maps to CSS (`response` and `busy` ship styled). |
| `color` | `flow`, `highlight`, `pulse` | Any CSS colour, e.g. `"#22c55e"` or `green`. Quote anything starting with `#`. |
| `ease` | `flow` | `linear` (default), `in`, `out`, `in-out`. |
| `status` | `flow` | `ok` (default) or `fail`. Semantic, unlike `style` — see below. |
| `repeat` | `flow`, `pulse` | Repeat count. Parsed and reserved; the runtime does not read it yet. |
| `bidi` | `flow` | Travel both ways. Parsed and reserved; the runtime does not read it yet. |
| `side` | `note` | `above` (default), `below`, `left`, `right`. |
| `badge`, `state` | `set` | Pill text, and a standing `dgm-state-<name>` class. |
| `label`, `value` | `gauge` | What the reading is called and what it currently says. Both required. |
| `desc` | step | Prose narration for the step. Shown in the caption; `\n` works. |
| `speed` | scenario | Initial playback rate, e.g. `1.5`. The player starts here; the speed button cycles from it. |
| `loop` | scenario | Restart at the end. |
| `autoplay` | scenario | Start playing once the diagram has rendered. Defaults to **true**, and is skipped when the system asks for reduced motion. |

`color` reaches the page as a `--dgm-color` custom property on the particle or
the node, which `runtime.css` reads with the theme colour as its fallback — so a
colour tints the same parts the default would have, in both light and dark.

`ease` is a remap of the flow's progress, not a CSS transition: the runtime
evaluates it at the current time rather than integrating between frames, so
scrubbing to a moment shows exactly what playing to it would.

`examples/deploy-pipeline.dgm` puts them together — a green deploy easing into
production, a red rollback coming back out:

```
scenario "ship a release" { speed: 1.5, autoplay: true }

  step promote "Promote to production, slowly at first" {
    flow staging -> prod {
      label: "canary 10% then 100%", dur: 1100ms, color: "#22c55e", ease: in-out
    }
    highlight prod { color: "#22c55e" }
  }

  step rollback "Roll back to the last good image" {
    flow prod -> reg { label: "rollback", dur: 900ms, color: "#ef4444", ease: out }
  }
```

Durations behave predictably:

- A `flow` with no `dur` takes 600ms per hop; with one, the total is split
  evenly across hops and always sums exactly to what you asked for.
- A stateful action (`highlight`, `dim`, `note`, …) with no `dur` spans its
  whole step, which is what makes `highlight ing` beside a flow do the obvious
  thing.
- A step lasts as long as its longest action unless given an explicit `dur`.

A flow may travel **against** the direction an edge was drawn. Response paths
are the normal case, so `flow pod1 -> svc` reuses the `svc --> pod1` edge and
marks the track reversed rather than demanding you draw a second arrow.

### What a flow draws

A flow is not just a dot. While its track is open the edge underneath lights
up, a comet trails the particle along the real path geometry, and the last
moments before it lands pulse the node it is arriving at — so a multi-hop
`flow a -> b -> c` reads as two arrivals rather than one long slide.

All of it is computed from the current time and nothing else. The trail is a
dash window whose position is a function of `t`; the arrival pulse is on
whenever `t` falls in the tail of a hop. Scrub backwards and everything
un-happens, because there was never any accumulated state to unwind.

**`status` is not `style`.** `style: error` is a CSS hook — it names a class
and the stylesheet decides what that looks like. `status: fail` says the flow
*did not succeed*, which the runtime draws rather than merely recolours: the
particle takes an error appearance, the edge is marked failed, and a ✕ lands at
the destination end of the path for the last fifth of the track. The set is
closed to `ok` and `fail`, so an unrecognised value is an error rather than a
class nobody styles.

```
step attempt "The primary gateway never answers" {
  flow checkout -> gw { label: "authorize $84.10", dur: 1100ms, status: fail }
  note gw "no response in 8s\nconnect timeout"
}
```

`examples/payment-checkout.dgm` runs the same checkout twice — one scenario
where the gateway answers and one where it times out and traffic fails over to
a backup provider. Failure paths are first-class, not an afterthought in red.

## Narration

An animation shows you *what* moves. It cannot tell you *why*, and a diagram
whose point is the reasoning — a protocol, a failover, a consensus round —
is mostly reasoning. `desc` is where that goes:

```
step exchange "The app trades the code for tokens" {
  desc: "Now the application speaks to the provider directly, back channel, server to server. It sends the code together with its client secret, and that pairing is what proves the exchange is genuine."
  flow app -> auth { label: "POST /token + secret", dur: 700ms }
}
```

A string lives on one line — use `\n` for a paragraph break rather than
wrapping the source, which the scanner reads as an unterminated string.

The player shows the active step's name and prose in a caption under the
stage, and marks each step boundary with a tick on the scrubber — click one to
jump to that beat. The step list beside the diagram stays the table of
contents; the caption is the "you are here".

The caption is an `aria-live` region, so a screen reader hears the walkthrough
rather than only seeing it. It is rewritten when the step changes, not when the
frame does — otherwise the same sentence would be announced sixty times a
second for the length of the step.

`examples/oauth-login.dgm` is the worked example: an OAuth 2.0 authorization
code flow where every step explains what the protocol is buying with it.

## Persistent state

Most actions describe a moment. Some facts are not moments: which node is
leader, what term the cluster is in, how many votes have come back. You cannot
infer those from a particle that has already gone past, and a reader who
scrubs to the middle of a scenario deserves to see them.

```
step timeout "n3's election timer fires first" {
  set   n3 { badge: "candidate", state: candidate, color: "#d97706" }
  gauge n3 { label: "term",  value: 2 }
  gauge n3 { label: "votes", value: "1 / 5" }
}

step votes "A majority answers" {
  gauge n3 { label: "votes", value: "3 / 5" }
}
```

`set` writes a badge and an optional `state` name, which becomes a standing
`dgm-state-<name>` class on the element. `gauge` writes a named reading;
writing the same `label` again replaces it, and a different label sits
alongside. `set n3 { badge: "" }` retires the badge; `unset n3` retires
everything on that node.

**The compiler works out the windows, not the renderer.** A badge is not "on
from here" — it is on over a closed interval that ends when something else
writes the same slot, or when the scenario does. Working that out means looking
ahead at every later action, and the compiler is already walking them in order.
So it closes each window as it goes and hands the renderer a list of intervals
to test `t` against, which is why scrubbing to any moment shows exactly the
state the actions before it imply.

The windows are half-open. When one reading replaces another they share an
instant, and treating both ends as inclusive would show the old and new value
together on that one frame.

`examples/raft-election.dgm` is the worked example: a leader dies, an election
runs, and the badges and gauges say what is true at whatever moment you stop.

## Attention

A diagram big enough to be worth animating is usually too big to read all at
once. `focus` narrows it to one thing per step:

```
step decide "The domain layer decides" {
  focus domain
  flow pricing -> rules { label: "apply discounts", dur: 500ms }
  note rules "pure functions of the basket" { side: left }
}
```

Naming a subgraph focuses everything inside it, and the frames around it stay
lit — dimming the box drawn around the very thing you asked to look at would
undo the effect. An edge with one end still in focus stays lit too, because
that is how the focused thing connects to the rest; only edges wholly outside
recede. The timeline still carries nothing but IDs: the track lists what to
look at, and the renderer expands it against the containment tree it already
has.

Notes take a `side` — `above` (the default), `below`, `left` or `right` — and
the runtime treats it as a preference rather than a coordinate. It clamps every
note inside the stage and shoves one clear of another if they would overlap. A
note that had to move drops its arrow instead of pointing at whatever it landed
beside.

Clickable elements advertise themselves. A `reveal` source carries a chip
saying how much is folded away (`+3`), which flips to `–` once it is open; a
`view` source carries a `⤢`. Both are clickable, and both do exactly what
clicking the element does. A reveal nobody can see is a reveal nobody finds.

`examples/layered-arch.dgm` walks a request down four layers, focusing one at a
time, with the cross-cutting concerns folded away behind a chip.

## Interaction

One diagram can only say so much. A cluster-level view has to either omit what
happens inside a pod or clutter the main picture with it. An `interact` block
makes elements clickable so the detail has somewhere to live:

```
view podA "Inside Pod A" from "pod-a.dgm"

interact {
  click pod1    -> view podA { label: "Zoom into Pod A" }
  click cluster -> reveal cp
  click pod2    -> step balance
}
```

| Click target | Form | Notes |
| --- | --- | --- |
| `view` | `click pod1 -> view podA` | Drill into another diagram, declared by a `view` line. |
| `reveal` | `click cluster -> reveal cp` | Toggle elements that start hidden. A subgraph brings its contents. |
| `step` | `click pod2 -> step balance` | Seek the current scenario to that step. |
| `url` | `click svc -> url "https://…"` | Open a dashboard or runbook in a new tab. Quoted, unlike the others. |

Bindings take `label` (a hover tooltip) and `style`. Nodes and subgraphs are
both clickable, and each element may carry one binding.

**Sub-diagrams are ordinary `.dgm` files.** `pod-a.dgm` previews and lints on
its own; `from` paths resolve relative to the file that declares them. `preview`
follows every reference and bundles the whole set into one self-contained page,
so drilling in swaps the stage rather than loading anything. The current view is
in `location.hash`, which makes browser back and forward work as expected.

**`reveal` is not `show`/`hide`.** Those are timeline state: the clock owns them
and a seek resets them. Reveal is interaction state that persists until the
viewer leaves the view. Being the target of a reveal is what makes an element
start hidden — there is no separate declaration.

## Sharing a moment

"Look at *this* step" is most of why anyone sends a diagram to a colleague, and
describing a moment in prose never reproduces it. The page's address carries
the whole state:

```
page.html#v=<view>&s=<scenario>&t=<ms>
```

Opening one lands on that view and scenario, seeks to that millisecond, and
stays **paused** — arriving at a named moment and then playing straight past it
would defeat the point. The **Copy link** button writes the link for whatever
is on screen right now.

The old short form still works: a hash with no `=` in it is a bare view id, so
existing links and the hash that ordinary drill-in navigation writes are
unaffected.

Add `?embed` to the query string for an iframe: the header bar and the step
list go away, the stage, the caption and the scrubber stay, and the keyboard
still works when the frame has focus.

```html
<iframe src="incident-triage.html?embed#v=incident-triage&s=s0&t=2750"
        width="100%" height="520" style="border:0" title="Incident cascade"></iframe>
```

`examples/incident-triage.dgm` is the worked example — a service map where
every service links out to its dashboard or runbook, because a diagram
consulted during an incident is a navigation surface rather than an
illustration.

## Commands

```
diagramator compile <file.dgm> [-o out.json]   # animation timeline JSON
diagramator mermaid <file.dgm> [-o out.mmd]    # the diagram as plain Mermaid
diagramator preview <file.dgm> [-o out.html]   # self-contained animated page
diagramator narrate <file.dgm> [--format=md|json]   # the animation, written out
diagramator lint    <file.dgm> [--format=text|json] # diagnostics only
```

### Authoring

```
diagramator preview examples/k8s-request.dgm --serve --watch
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
diagramator frame examples/payment-checkout.dgm --at 1620ms --scenario s1 -o fail.png
```

That is `examples/payment-checkout.fail.png` — the gateway timing out, ✕ and
all. It works by opening the Phase-6 deep link in a headless Chrome, and
because a deep link lands *paused*, the screenshot is deterministic rather than
a race against the animation. The browser is found on `PATH`
(`google-chrome`, `chromium`, …) or named by `$DIAGRAMATOR_CHROME`; shelling
out to one the machine already has is what keeps the no-dependencies rule.

`--frames N -o dir/` captures N evenly spaced moments. Turning those into an
animation is a job for tools that already do it well:

```
ffmpeg -framerate 12 -i dir/frame-%04d.png -vf palettegen palette.png
ffmpeg -framerate 12 -i dir/frame-%04d.png -i palette.png -lavfi paletteuse out.gif

# or, with ImageMagick
magick -delay 8 -loop 0 dir/frame-*.png out.gif
```

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

`examples/oauth-login.narrate.md` is that output, committed. `--format=json`
emits the same walkthrough as data — each event carries both the sentence and
the fields it was built from, so filtering for "every failing flow" does not
mean parsing the prose back apart.

`lint --format=json` emits
`[{"file","line","col","severity","message","hint"}]` on stdout. Exit codes are
unchanged — warnings 0, errors 1 — so a caller can branch on the status *and*
read the detail instead of choosing between them.

The preview page plays itself: after a view renders, a scenario with
`autoplay` (the default) starts, unless the reader's system asks for reduced
motion. `window.DIAGRAMATOR_PLAYER` is the same player, so
`DIAGRAMATOR_PLAYER.seek(2400)` lands on a moment deterministically.

### Driving the player

Press `?` in the page for this list.

| Key | Does |
| --- | --- |
| `Space` | Play or pause |
| `←` / `→` | Previous or next step |
| `Home` / `End` | Jump to the start or the end |
| `1`–`9` | Jump to step *n* |
| `Esc` | Back out of a drilled-in view, or close the help |
| `?` | Show or hide the shortcut list |

Scroll to zoom the stage — anchored on the cursor, so the thing you are
pointing at stays put — drag to pan, and `⌂` or a double-click resets. Notes,
badges and gauges are positioned from live element rects, so they stay glued to
their nodes at any zoom. Zoom resets when you change view, since a framing that
suited one diagram means nothing over the next.

Every clickable element is reachable by `Tab` and activates with `Enter` or
`Space`; the step list is real buttons, not clickable rows.

Theme and speed persist in `localStorage`. The theme follows your system until
you press the button, and then stops — an explicit choice is exactly the thing
that should not be overridden later. A speed you chose outranks a scenario's
`speed` for the same reason; untouched, the scenario's own rate still wins. The
speed button cycles `0.25 → 0.5 → 1 → 1.5 → 2` and its label always shows the
rate actually in effect.

With `prefers-reduced-motion: reduce`, nothing autoplays, the decorative
animation stops, and the help says so — stepping through with the arrow keys is
the intended way to read it. The particle still travels, because that is the
content rather than an effect.

Relative paths are resolved against the directory you ran the command from,
including under `bazel run` — the binary executes from its runfiles tree, so
Diagramator honours Bazel's `BUILD_WORKING_DIRECTORY` to get this right.
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

## How it fits together

```
source.dgm
   │
   ├─ loader ──────────────► the file and every `view` it references
   │
   ├─ parser ──────────────► ast.Document + symbol.Table   (per file, no I/O)
   │    ├── flowchart.go        the diagram half (pluggable per diagram type)
   │    ├── scenario.go         the animation half (diagram-agnostic)
   │    └── interact.go         the interaction half (diagram-agnostic)
   │
   ├─ compile ────────────► ir.Timeline      one View per diagram,
   │                                         absolute-millisecond tracks
   └─ emit
        ├── mermaid ───────► plain Mermaid source
        └── html ─────────► self-contained animated page
```

Layout is delegated to mermaid.js. Diagramator emits clean Mermaid, mermaid.js
produces the SVG, and the runtime animates particles along the real edge
geometry using `getPointAtLength`. There is no layout engine here to maintain.

Two design decisions are load-bearing and worth preserving:

- **The timeline holds no geometry.** Tracks reference node and edge IDs with
  absolute start/end times, so a renderer is a clock and a scrubber. The same
  timeline could drive a Go-native SVG backend later without recompilation.

- **The animation layer never mentions flowcharts.** A diagram parser's only
  obligation is to produce a `symbol.Table`; scenario parsing, interaction
  bindings, validation, and compilation work against that alone. Adding
  `sequenceDiagram` or `architecture-beta` costs one parser in `pkg/parser`
  registered via `pkg/registry`, and zero changes anywhere else.

- **Parsing does no I/O.** `parser.Parse` takes content and returns a tree, so
  it stays usable from a webview or a WASM build with no filesystem. Resolving
  the paths in `view` declarations is `pkg/loader`'s job, and it takes its read
  function as an argument.

The runtime binds to the rendered SVG defensively: nodes by mermaid's
`flowchart-<id>-<n>` id, and edges by matching path endpoints against node
centres rather than by parsing mermaid's edge-id format, which has changed
between releases. Anything that fails to bind is reported in a banner on the
page instead of silently not animating.

## Building

Bazel with bzlmod. Go is not required locally — `rules_go` fetches a hermetic
SDK, and Gazelle runs through Bazel.

```
bazel build //...
bazel test  //...
bazel run   //:gazelle          # after adding or moving packages
```

Golden fixtures are regenerated through `bazel run` (which sets
`BUILD_WORKSPACE_DIRECTORY`), not `bazel test`:

```
bazel run //pkg/parser:parser_test   -- -update
bazel run //pkg/compile:compile_test -- -update
```

There are no third-party Go dependencies, and the intent is to keep it that
way: a hand-rolled lexer and recursive-descent parser need nothing beyond the
standard library.

`pkg/emit/html/assets/` holds the vendored `mermaid.min.js` plus the runtime's
`runtime.js` and `runtime.css`. They live inside the package because `go:embed`
cannot reach outside it; a future VS Code plugin should consume them from there
rather than keeping a second copy. The runtime is a classic script, not an ES
module, because module scripts are blocked on `file://` and awkward in webviews.

## Status

Working today: flowcharts (`flowchart` / `graph`) with every Mermaid node shape
and link form, nested subgraphs, frontmatter, scenarios, the timeline compiler,
clickable drill-down between diagrams, and the animated HTML preview.

Not built yet: sequence diagrams and `architecture-beta` (the registry seam
exists for them), the VS Code plugin, WASM builds, and animated SVG/GIF export.
