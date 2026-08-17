<!-- Generated from README.md by `bazel run //site:sync`. Do not edit: //site:site_test fails while this file disagrees with its source, and the next sync overwrites it. -->

# The language

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
| `speed` | scenario | Initial playback rate, e.g. `1.5`. The player starts here; the speed button cycles from it. |
| `loop` | scenario | Restart at the end. |
| `autoplay` | scenario | Start playing once the diagram has rendered. Defaults to **false** — a page opens at rest — and is skipped when the system asks for reduced motion. |
| `poster` | scenario | The moment the page rests at before anyone presses play, e.g. `1600ms`. Defaults to the start. A shared link's `t=` wins over it. |
| `stepwise` | scenario | Play advances exactly one step and stops at its end — the presenter transport without presenter mode. |
| `variant`, `until` | scenario | Inherit another scenario's opening steps — see [Failure paths](02-storytelling.md#failure-paths). |
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
