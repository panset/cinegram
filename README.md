# Cinegram

**Mermaid draws the system; Cinegram plays the story.**

Animated, narrated, explorable architecture diagrams from a Mermaid-compatible
DSL — for the humans who read them and the AI agents that write them.

A static diagram shows you that an Ingress sits in front of a Service. It cannot
show you what happens to a single `GET /api/orders` as it travels
LB → Ingress → Service → Pod and back. Cinegram adds a small animation
language on top of Mermaid so that path becomes something you can watch.

```
bazel run //cmd/cinegram -- preview examples/01-basics/01-k8s-request.dgm -o /tmp/k8s.html
open /tmp/k8s.html
```

Or skip the build entirely: every example is playable at
<https://panset.github.io/cinegram/>.

`examples/` is a tour, and the site mirrors it folder for folder:
`01-basics/` is a request path and a pipeline, `02-storytelling/` adds
failure paths, narration and storyboards, `03-interaction/` is clicking and
drilling in, `04-diagram-types/` proves the same language over `sequence-`
and `stateDiagram`, and `05-ai-systems/` animates what people are building
with models — an agent's tool loop, a multi-agent fan-out, a RAG pipeline with
its cache cold, and the MCP handshake `cinegram mcp` performs. The numeric
prefixes are ordering: they show in a page's
URL and nowhere a reader looks, since titles come from the diagram itself.

**Try it in the browser.** <https://panset.github.io/cinegram/playground/> is
the compiler itself, built to WASM and running in the tab — paste a diagram,
watch it animate, share the link. Drop in a whole folder of `.dgm` files (or
*Add folder…*) and the *Files* view browses it the way `cinegram site` does,
numeric prefixes ordering the tree. Nothing is uploaded; the document lives in the
URL fragment. Locally that page is `bazel run //web/playground:site --
--serve`.

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
have a file any Mermaid renderer will draw. `cinegram mermaid` does exactly
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
| `scene` | `scene idp_login` | Show a storyboard frame beside the diagram — see below. |
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
| `msg` | `flow` | Which arrow to use, 1-based, when several run between the same pair. |
| `repeat` | `flow`, `pulse` | Repeat count. Parsed and reserved; the runtime does not read it yet. |
| `bidi` | `flow` | Travel both ways. Parsed and reserved; the runtime does not read it yet. |
| `side` | `note` | `above` (default), `below`, `left`, `right`. |
| `badge`, `state` | `set` | Pill text, and a standing `dgm-state-<name>` class. |
| `label`, `value` | `gauge` | What the reading is called and what it currently says. Both required. |
| `desc` | step | Prose narration for the step. Shown in the caption; `\n` works. |
| `speed` | scenario | Initial playback rate, e.g. `1.5`. The player starts here; the reader can change it in the settings sheet. |
| `loop` | scenario | Restart at the end. |
| `autoplay` | scenario | Start playing once the diagram has rendered. Defaults to **false** — a page opens at rest — and is skipped when the system asks for reduced motion. |
| `poster` | scenario | The moment the page rests at before anyone presses play, e.g. `1600ms`. Defaults to the start. A shared link's `t=` wins over it. |
| `stepwise` | scenario | Play advances exactly one step and stops at its end — the presenter transport without presenter mode. |
| `variant`, `until` | scenario | Inherit another scenario's opening steps — see [Failure paths](#failure-paths). |
| `outcome` | scenario | `ok` or `fail`. A failure is marked `✕` in the scenario picker. |
| `img`, `caption` | storyboard frame | The picture to show and the line under it. At least one is required. |

`color` reaches the page as a `--dgm-color` custom property on the particle or
the node, which `runtime.css` reads with the theme colour as its fallback — so a
colour tints the same parts the default would have, in both light and dark.

`ease` is a remap of the flow's progress, not a CSS transition: the runtime
evaluates it at the current time rather than integrating between frames, so
scrubbing to a moment shows exactly what playing to it would.

`examples/01-basics/02-deploy-pipeline.dgm` puts them together — a green deploy easing into
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

The picture keeps up: the particle is an arrowhead turned to the way it is
actually travelling, and while a reversed track is open the edge's own
arrowhead is taken off — a static arrow pointing one way while a `200 OK`
slides the other is a contradiction the viewer would have to resolve on every
hop. The arrowhead comes back the moment the track closes.

### What a flow draws

A flow is not just a mark sliding along a line. While its track is open the
edge underneath lights up, a comet trails the particle along the real path
geometry, and the last
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

`examples/02-storytelling/01-payment-checkout.dgm` runs the same checkout twice — one scenario
where the gateway answers and one where it times out and traffic fails over to
a backup provider. Failure paths are first-class, not an afterthought in red.

## Sequence diagrams

`sequenceDiagram` bodies work too, and nothing else changed to make them:

```
sequenceDiagram
  participant C as Browser
  participant S as WebSocket Server

  C->>S: GET /socket (Upgrade: websocket)
  S-->>C: 101 Switching Protocols
  C->>S: ping
  S-->>C: pong

scenario "the upgrade"
  step keepalive "Frames travel both ways" {
    desc: "Ping and pong are how each side finds out the other is still there."
    focus S
    flow C -> S { dur: 500ms, msg: 1 }
    flow S -> C { dur: 500ms, style: response, msg: 2 }
  }
```

Participants become nodes and each message becomes its own edge; `Note`,
`activate`, `loop`/`alt`/`opt`/`end` and `box` round-trip verbatim as
unmodelled syntax. Everything on top — `desc`, `set`/`gauge`, `focus`,
`click … -> view`, `narrate`, `frame` — is the same code it was for flowcharts.
A bundle can mix the two: `examples/04-diagram-types/01-websocket-handshake.dgm` drills from an
actor into a flowchart of the server's internals.

Two things are worth knowing when animating one:

**`msg` picks between repeated messages.** A flowchart usually draws one arrow
between two nodes; a sequence diagram draws one per message, and `ping` then
`close` from the same participant are two different arrows. Flows consume them
in order and lint says so when it had to guess, so `msg: 2` is how you say
which one you meant. A reply is matched as its own message rather than as the
request run backwards.

**Leave `label` off.** The diagram already draws the message text, so a flow
label prints a second copy on top of it.

## State diagrams

`stateDiagram-v2` is the diagram type Cinegram's premise fits best. A state
chart is static, but "what happens when this event fires in this state" is
exactly what a scenario is:

```
stateDiagram-v2
  [*] --> CLOSED
  CLOSED --> SYN_SENT : connect() / send SYN
  ESTABLISHED --> Teardown : close begins
  state Teardown {
    state who_closes <<choice>>
    [*] --> who_closes
    who_closes --> FIN_WAIT_1 : we sent the first FIN
  }

scenario "orderly close"
  step choose "Someone has to hang up first" {
    highlight Teardown
    seq {
      flow ESTABLISHED -> Teardown { dur: 500ms }
      flow Teardown_start -> who_closes { dur: 400ms }
      flow who_closes -> FIN_WAIT_1 { dur: 400ms }
    }
  }
```

States, transitions, `<<choice>>`/`<<fork>>`/`<<join>>` pseudostates and
composite `state X { }` blocks are all addressable animation targets; a
composite behaves like a subgraph, so `highlight`, `dim` and `reveal` on one
cover everything inside it, however deeply nested. `note`, `direction`,
`classDef` and the `--` concurrency divider round-trip verbatim as unmodelled
syntax. The plain `stateDiagram` header works too and renders identically.
`examples/04-diagram-types/02-tcp-connection.dgm` is the worked example.

Three things are worth knowing when animating one:

**`[*]` answers to a name.** It is not a legal identifier, so a scenario could
never reference it — yet the first arrow into a machine and the last one out are
usually the first things worth animating. Mermaid already names these nodes
internally and Cinegram adopts its spelling: `root_start` and `root_end` at the
top level, `<Composite>_start` and `<Composite>_end` inside a composite. So
`flow root_start -> CLOSED` animates the opening arrow, and lint's did-you-mean
knows the names. (One gap: inside a composite split into concurrent regions by
`--`, Mermaid names the markers after the region it generated rather than after
the composite, and those names are not predictable from the source.)

**`msg` picks between repeated transitions**, exactly as it does between a
sequence diagram's repeated messages. Two `A --> B` lines are two arrows on the
page and two edges here, and lint warns when a flow had to guess.

**Leave `label` off.** The diagram already draws each transition's event text,
so a flow label prints a second copy on top of it.

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

`examples/02-storytelling/03-oauth-login.dgm` is the worked example: an OAuth 2.0 authorization
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

`examples/04-diagram-types/03-raft-election.dgm` is the worked example: a leader dies, an election
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

`examples/03-interaction/02-layered-arch.dgm` walks a request down four layers, focusing one at a
time, with the cross-cutting concerns folded away behind a chip.

## Storyboard

A diagram says what the system does. It rarely says what the *person* sees, and
in an authentication flow that is half the story: eighteen messages fly and the
user looks at four screens. A `storyboard` block gives them a panel beside the
stage, and a `scene` says which one is in front of them.

```
storyboard "What the person signing in sees" {
  frame app_signin { img: "frames/app-signin.svg", caption: "One button, no password field." }
  frame idp_form   { img: "frames/idp-login.svg",  caption: "The provider's domain." }
  frame consent    { caption: "Caption-only frames are allowed." }
}

step redirect "The browser is handed to the IdP" {
  flow E -> A { dur: 500ms, style: response }
  scene idp_form
}
```

Cinegram supplies the synchronisation, not the content. There is no browser
bezel, no window chrome, no drawing tools — a frame is an image you wrote and a
caption, which is what keeps the feature bounded: a screenshot of an email is as
valid a frame as a login form.

**Where a scene sits inside its step matters as much as which step it is in.**
A `scene` is an action like any other, so on its own it fires the instant the
step begins — but a screen changes when an arrow *lands*. Put it in a `seq`
after the hop that causes it and it fires there:

```
step redirect "The IdP sends back a code" {
  seq {
    flow IdP -> U { dur: 700ms }
    scene app_waiting      %% the redirect lands; the address bar changes
    flow U -> A { dur: 600ms }
  }
}
```

A scene consumes none of the chain, so it means "here" without stretching the
step, and it keeps meaning it when a hop's duration changes. `at:` and `delay:`
work too — `scene x { at: 1100ms }` — but they are arithmetic you have to redo
by hand later. Several scenes in one step is normal: the panel changes and the
diagram barely moves, which for an authentication hop is the honest picture.

A step that *ends* on a new screen needs a `dur` long enough to look at it,
since a scene on the last instant of a scenario flashes for one frame and stops.

**A scene is sticky.** At any moment the panel shows the last scene to have
started, not the scene belonging to the current step. That is what lets six
steps of server-side verification sit under one motionless "Signing you in…"
interstitial, which is exactly what those six steps look like from a chair. It
is also a pure function of the time, so scrubbing backwards lands where playing
forwards would.

Frame names are flat across every `storyboard` block in a file — there is one
panel, so qualifying a name would buy nothing — and blocks merge. Images are
read by the loader and inlined as `data:` URIs, so the emitted page stays
self-contained; `.svg`, `.png`, `.jpg`, `.gif` and `.webp` are understood, and
paths resolve relative to the file that declares them. The panel appears only
for scenarios that actually use a scene, so a document can storyboard its happy
path and give the failure path the full width.

`examples/02-storytelling/04-oidc-login.dgm` is the worked example, with its frames in
`examples/02-storytelling/frames/`.

## Failure paths

The interesting scenario is usually the happy path right up to the moment it
stops being one. Writing that prefix twice means maintaining it twice, so a
scenario can inherit it:

```
scenario "happy path" { speed: 1.0 }
  step submit "Shopper submits the order" { … }
  step authorize "The primary gateway authorises the card" { … }
  …

scenario "gateway outage" { variant: "happy path", until: submit, outcome: fail }
  step attempt "The primary gateway never answers" { … }
```

`until` is **inclusive**: the variant replays the base *through* that step and
then diverges, because "X happened, and then things went wrong" is the sentence
the pair is meant to read as. With no `until` the whole base is inherited and
the variant's steps are appended.

Inheritance is depth-1 — a variant of a variant is an error. A chain would make
a step's addressable id a function of an arbitrarily long ancestry, and
`click … -> step` has to stay something you can work out by looking. For the
same reason, an id that would collide with an inherited one is rejected by name
rather than silently shadowed.

The splice happens before any timing runs, so every rule applies to the merged
scenario unchanged: step spans, `seq` chaining, which of several parallel
messages a hop takes, and the windows persistent state holds for. All of those
reset per scenario, so an inherited prefix genuinely replays rather than
continuing the base's bookkeeping.

`outcome: fail` marks the scenario `✕` in the picker, using the same glyph a
failing flow draws — a reader should be able to see that one of the alternatives
ends badly before choosing it. Choosing a scenario also rewrites the address, so
the link you copy names what you are looking at.

`examples/02-storytelling/01-payment-checkout.dgm` tells checkout twice: the path everyone draws,
and the one that costs money.

## Presenter mode

Talking over a diagram is not watching a video. `?present` — or the **Present**
button, which toggles in place without reloading — switches Space from "play"
to "play exactly the next step, then stop":

```sh
cinegram preview examples/02-storytelling/04-oidc-login.dgm --serve
# then open http://127.0.0.1:8731/?present
```

Space or → plays one beat and pauses at its end. Pressing it again while a beat
is still running **skips the rest of it** rather than starting it over —
reaching for the key mid-animation means "get on with it", and answering that by
replaying the beat would trap you in it until it had played out in full. Nothing
is lost by skipping, because a step's end state is a pure function of the time.
← backs up to the start of a beat, so Space replays it when you do want that; a
click on the stage advances, because presenting from a lectern means a clicker
and a clicker sends a click. The step list, the
scrubber and the authoring controls go away, the stage grows, and the step's
`desc` becomes speaker text at a size that survives a projector. The storyboard
panel stays — a demo is exactly when someone wants to point at what the user
sees. Escape leaves.

Nothing about the timeline changes: the stop is a moment in the same
milliseconds the clock already runs in, so the playback speed, deep links and
`CINEGRAM_PLAYER.seek()` all keep working.

## Reels

`?reel` turns the page into a vertical story — the diagram as something you
tap through on a phone. The chrome goes away entirely; an Instagram-style
segmented bar says how many beats the story has and which one you are on; the
caption narrates at arm's-length size; a tap (or Space) plays exactly the next
step and stops, the same one-beat transport presenter mode uses. Steps map
one-to-one onto story segments, which is the honest presentation of a
step-structured walkthrough: the bar promises "five beats", not "ninety
seconds".

An **auto-follow camera** frames each step's action: one pose per step,
computed from what the step's tracks touch — a `focus` names the frame
outright, otherwise the flows, highlights and notes define it — with a short
glide at each boundary. Drag or scroll to take the framing over; advancing to
the next beat hands it back — a manual zoom is per-beat inspection, and on a
phone the next tap is the gesture you already have (double-click works too).
The pose is a function of the clock, evaluated at the current
moment rather than accumulated between frames, so scrubbing, deep links and
recording all see exactly what playing does — mid-glide included.

```
cinegram record examples/02-storytelling/04-oidc-login.dgm --reel -o login.mp4
cinegram frame  examples/02-storytelling/04-oidc-login.dgm --reel --at 6500ms -o beat.png
```

`--reel` shoots the `?reel` page at 1080×1920 — the portrait clip LinkedIn,
Shorts and a phone-first Slack want — and an explicit `--width`/`--height`
still wins, per axis. Prefer mp4 for reel-sized recordings: the GIF encoder
holds every frame in memory, and at 1080×1920 that is ~8 MB a frame (the
recording line says so when it applies).

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

**While the transport plays one beat at a time, a bound subgraph is clicked on
its border or its title, not its middle.** A subgraph's box is mostly the space
around the things inside it, and in presenter mode, a reel or a `stepwise`
scenario a click there is how a reader asks for the next step. At rest nothing
is competing for it and the whole box stays clickable, as before. The chip in
its corner works either way.

**Sub-diagrams are ordinary `.dgm` files.** `pod-a.dgm` previews and lints on
its own; `from` paths resolve relative to the file that declares them. `preview`
follows every reference and bundles the whole set into one self-contained page,
so drilling in swaps the stage rather than loading anything. The current view is
in `location.hash`, which makes browser back and forward work as expected.

**`reveal` is not `show`/`hide`.** Those are timeline state: the clock owns them
and a seek resets them. Reveal is interaction state that persists until the
viewer leaves the view. Being the target of a reveal is what makes an element
start hidden — there is no separate declaration.

`examples/02-storytelling/02-blue-green-deploy.dgm` is the timeline side of that distinction: the
green pods are hidden while blue serves, appear when the controller starts them
(a `seq` chains the launches, a `wait` stands in for each readiness probe), and
scrubbing backwards removes them again. Edges into a hidden node conceal
themselves with it.

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

`examples/03-interaction/01-incident-triage.dgm` is the worked example — a service map where
every service links out to its dashboard or runbook, because a diagram
consulted during an incident is a navigation surface rather than an
illustration.

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

Layout is delegated to mermaid.js. Cinegram emits clean Mermaid, mermaid.js
produces the SVG, and the runtime animates particles along the real edge
geometry using `getPointAtLength`. There is no layout engine here to maintain.

Two properties follow from that shape. The timeline holds **no geometry** —
tracks are node and edge IDs with absolute start/end times — so a renderer is a
clock and a scrubber. And the animation layer **never mentions flowcharts**: a
diagram parser's only obligation is to produce a `symbol.Table`, so adding
`sequenceDiagram` or `architecture-beta` costs one parser and nothing else.

The runtime binds to the rendered SVG defensively, and anything that fails to
bind is reported in a banner on the page instead of silently not animating.

Contributing? `.claude/skills/rendering-pipeline/` is the working reference for
all of it.

## Building

Bazel with bzlmod. Go is not required locally — `rules_go` fetches a hermetic
SDK, and Gazelle runs through Bazel.

```
bazel build //...
bazel test  //...
bazel run   //:gazelle          # after adding or moving packages
```

There are no third-party Go dependencies, and the intent is to keep it that
way: a hand-rolled lexer and recursive-descent parser need nothing beyond the
standard library. The website at <https://panset.github.io/cinegram/> is the
one exception, and it is held at arm's length: Zensical renders `www/` in the
`pages` workflow at a pinned version, and Bazel's job is to guarantee the
input. `bazel run //site:sync` writes the generated half of `www/` — the
examples tour and this README, cut into guide pages — and `//site:site_test`
fails while the committed copy is stale, so the renderer is never handed
anything out of date.

`.claude/skills/build/` has the rest: golden fixtures, formatting, and the
invariants a change has to preserve.

## Status

Working today: flowcharts (`flowchart` / `graph`) with every Mermaid node shape
and link form, `sequenceDiagram`, `stateDiagram-v2` with composites and
pseudostates, nested subgraphs, frontmatter, scenarios with
narration and persistent state, focus, deep links and embedding, the timeline
compiler, clickable drill-down between diagrams of different types, the
animated HTML preview, a serve/watch authoring loop, `narrate`, PNG frame
capture, GIF/mp4/webm recording, and the VS Code extension — ```dgm blocks
animate inside the built-in Markdown preview, and `.dgm` files get syntax
highlighting, a preview panel, an *Open With… → Cinegram Animation* editor, and
`Cinegram: Export Animation…` to record one straight to a GIF (see
`editors/vscode/`).

Not built yet: `architecture-beta` and the other Mermaid diagram types (the
registry seam exists for them).

## The VS Code extension

`editors/vscode/README.md` is the Marketplace listing and is written for someone
installing the extension. Building, packaging and publishing it are in
[`editors/vscode/CONTRIBUTING.md`](editors/vscode/CONTRIBUTING.md).

Each package carries the `cinegram` binary for one platform, because the
extension shells out to the compiler rather than reimplementing any of it, and
VS Code installs the package matching the machine.

## License

[MIT](LICENSE).
