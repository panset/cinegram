# The .dgm language

A `.dgm` file is a Mermaid diagram followed by animation blocks:

```
<mermaid body — flowchart/graph or sequenceDiagram, untouched>

view <id> "<title>" from "<path.dgm>"     (optional, repeatable)
interact { … }                            (optional)
storyboard "<title>" { … }                (optional, repeatable)
scenario "<name>" { <attrs> }             (one or more)
  step <id> "<title>" { <actions> }
  …
```

- The diagram half is **ordinary Mermaid** — every node shape, link form,
  `subgraph`, `classDef`, frontmatter and `%%` comment round-trips verbatim.
  Never rewrite it; only append blocks after it. Supported diagram types:
  `flowchart`/`graph` and `sequenceDiagram`.
- The four top-level keywords are `scenario`, `storyboard`, `view`,
  `interact` — any order, any number of each.
- `%%` comments work in the animation half too. **Never emit an empty `%%`
  line** — it breaks Mermaid's own comment stripping.

### Syntax ground rules

- **Strings live on one line**: `"…"` with `\n` and `\t` escapes. Wrapping
  the source line is an unterminated-string error.
- **An opening `{` sits on the same line** as its construct
  (`scenario "x" {`, `step id "n" {`, `flow a -> b {`, `seq {`); once open,
  the block may span lines. A `flow` hop chain and a comma-separated target
  list must each sit on one line.
- Identifiers may contain letters, digits, `_` and `-`, but **cannot start
  with a digit** — a Mermaid node id like `3rd` cannot be targeted.
- Durations: `600ms`, `1.2s`, `2m`, or a bare number meaning milliseconds.
- Attribute values are strings, bare identifiers, numbers, durations, or
  booleans (`true/false/yes/no/on/off/1/0`). Quote anything else — `#`, `/`
  and `%` are not valid bare tokens, so `color: "#22c55e"` and
  `value: "3 / 5"` need quotes.

## scenario

```
scenario "GET /api/orders" { speed: 1.0, loop: true }
```

The name may be a quoted string or a bare identifier, and may be omitted; the
`{ … }` attribute block may be omitted. **Steps follow at top level — there
are no braces around them.** The scenario ends at the first thing that is not
a `step`.

| Attribute | Meaning |
| --- | --- |
| `speed` | Initial playback rate (default `1.0`). |
| `loop` | Restart at the end (default `false`). |
| `autoplay` | Start once rendered. **Default `true`**; skipped under reduced motion. |
| `variant`, `until` | Inherit another scenario's opening steps — see Variants. |
| `outcome` | `ok` or `fail`; `fail` marks the scenario ✕ in the picker. |

## step

```
step route "Ingress rule matches host and path" {
  desc: "Prose narration shown in the caption. \n makes a paragraph."
  note ing "host: api.example.com\npath: /api/*"
  flow ing -> svc { dur: 500ms }
}
```

- Both the id and the title are optional (`step { … }`, `step route { … }`,
  `step "Title" { … }` all parse). A step with no id gets the effective id
  `step<index>` (`step0`, `step1`, …) — that is what `click … -> step` and
  `until:` address, so **give ids to steps you'll refer to**. Ids must be
  unique within the scenario; hyphens are fine (`outage-confirm`).
- Step attributes: `dur` (force the step's length), `delay` (gap after the
  previous step), `desc` (narration — use it generously; it becomes the
  caption, presenter speaker notes, and `narrate` output).
- **All actions in a step start together; steps run one after another.** A
  step lasts as long as its longest action, unless it has its own `dur:`.
  A step whose actions have no durations lasts 800ms.

## Actions

| Action | Form | Notes |
| --- | --- | --- |
| `flow` | `flow a -> b -> c { … }` | A packet travelling the hops (≥2 nodes, one line); each hop is its own track. |
| `highlight` | `highlight a, b` | Emphasise nodes or subgraphs. |
| `dim` | `dim a, b` | Fade back. |
| `note` | `note a "text"` | Callout on a node; exactly one target, text must be quoted. |
| `pulse` | `pulse a` | Repeating pulse. |
| `focus` | `focus domain` | Everything outside recedes; a subgraph focuses its contents, its frame stays lit. |
| `show` / `hide` | `show a` | Timeline-owned reveal/conceal (a seek resets them). |
| `set` | `set n1 { badge: "leader", state: leader }` | Standing state; outlives its step. |
| `gauge` | `gauge n1 { label: "term", value: 2 }` | Named persistent reading; `label` and `value` both required. |
| `unset` | `unset n1` | Retire everything `set`/`gauge` put on that target. |
| `scene` | `scene idp_form` | Show a storyboard frame (one frame per `scene`). |
| `wait` | `wait 500ms` | Bare duration; consumes time, draws nothing. |
| `seq` | `seq { … }` | Chain the contained actions in sequence. |

Targets are diagram ids: node ids or subgraph ids, never labels. `scene`
targets are storyboard frame ids. **Edges are never named** — a flow names
its endpoint nodes and the compiler finds the edge between each pair.

## Attributes per action

Written `{ key: value, … }` on the action's line (the block may span lines;
pairs separate with commas or newlines). An unknown key is a warning with a
did-you-mean hint; a known key with a bad value is an error.

Every non-persistent action takes `dur`, `delay`, `at`, `label`, `style`.
**`delay` and `at` are summed**, not alternatives: both offset the action's
start within its step. `style` names a CSS hook; `response` and `busy` ship
styled.

| Action | Extra attributes |
| --- | --- |
| `flow` | `color` (CSS colour), `ease` (`linear`\|`in`\|`out`\|`in-out`), `status` (`ok`\|`fail`), `msg` (1-based count). Do not use `repeat`/`bidi` — parsed but not yet rendered. |
| `highlight` | `color` |
| `pulse` | `color` |
| `note` | `side` (`above` default \| `below` \| `left` \| `right`) — a preference, not a coordinate |
| `set` | `badge` (pill text; `badge: ""` retires it), `state` (standing `dgm-state-<name>` class) |
| `gauge` | `label`, `value` (both required, strings) |

**Persistent actions (`set`, `gauge`, `unset`) take `delay`, `at`, `style`,
`color` — but not `dur`** (they fire at an instant and hold; a `dur` is an
unknown-attribute warning). Click bindings take `label` and `style` only.

`status: fail` is semantic, not cosmetic: the particle takes an error look,
the edge is marked failed, and a ✕ lands at the destination. Use it whenever
the flow genuinely fails; use `style`/`color` for mere emphasis.

## Timing rules

- A `flow` with no `dur` takes **600ms per hop**; with one, the total is
  split evenly across hops and sums exactly.
- A stateful action (`highlight`, `dim`, `note`, `focus`, `pulse`, `show`,
  `hide`) with no `dur` spans its **whole step** — `highlight ing` beside a
  flow does the obvious thing. With a `dur`, it runs `[start, start+dur]`.
- `set`/`gauge` persist **beyond** their step until overwritten (same badge
  slot, or same gauge `label`), `unset`, or scenario end. Different gauge
  labels coexist on one node. Windows are exact, so scrubbing to any moment
  shows the state the prior actions imply.
- Inside `seq { … }`: children chain; `scene`, `set`, `gauge` and `unset`
  consume **zero** of the chain (they fire where it has reached); a stateful
  action with no `dur` costs 800ms there (it does not stretch to the step).
  A child's `delay:`/`at:` pushes the chain. Seqs nest. A `seq`'s own
  attributes go as `key: value` lines *inside* its braces (`dur` is ignored —
  a seq's span is the sum of its children).
- `wait` inside a `seq` models a gap (a readiness probe, a timeout).

## Flows against the arrow

A flow may travel **against** the drawn edge: `flow pod1 -> svc` reuses
`svc --> pod1` and the runtime turns the arrowhead around while the track is
open. Response paths need no extra edges — never ask for reverse edges. If no
edge connects a pair in either direction, lint errors with a hint.

## Sequence diagrams

Participants are nodes — target the **short id** (`participant C as Browser`
→ `C`); each message is its own edge. `Note`, `activate`,
`loop`/`alt`/`opt`/`end`, `box` round-trip verbatim. Rules:

- **Use `msg:` when the same pair exchanges several messages.** Flows consume
  parallel same-direction edges in written order and lint warns when it
  guessed; `msg: 2` picks the second message between that pair (and moves the
  implicit cursor for later flows). A reply is its own message, not the
  request reversed.
- **Leave `label` off flows** — the diagram already prints the message text.
- `focus` the busy participant to direct attention.

## Storyboard

```
storyboard "What the person signing in sees" {
  frame app_signin { img: "frames/app-signin.svg", caption: "One button." }
  frame consent    { caption: "Caption-only frames are allowed." }
}
```

- A frame needs at least one of `img`/`caption`. Image paths resolve relative
  to the declaring file; `.svg` `.png` `.jpg` `.jpeg` `.gif` `.webp` only
  (they are inlined into the page as `data:` URIs).
- Frame ids are flat across all storyboard blocks in a file; blocks merge.
- `scene <frame_id>` shows a frame. **A scene is sticky**: the panel shows
  the last scene started, across steps, until another fires.
- Placement matters: a bare `scene` fires when its step begins, but a screen
  changes when an arrow *lands* — so put the scene in a `seq` right after the
  hop that causes it:
  `seq { flow IdP -> U { dur: 700ms }; scene app_waiting }`
  (it costs the chain nothing). A step that ends on a new screen needs enough
  `dur` to look at it.

## Variants (failure paths)

```
scenario "gateway outage" { variant: "happy path", until: submit, outcome: fail }
```

- `variant:` names the base scenario **by its name**; `until:` names a step
  **by its effective id** and is **inclusive** — the variant replays the base
  *through* that step, then diverges into its own steps. No `until` → the
  whole base is inherited and these steps are appended.
- Depth-1 only: a variant of a variant is an error. A step id colliding with
  an inherited one is an error — rename it.
- Write the shared prefix once, in the base; tell the failure as a variant.

## view and interact

```
view podA "Inside Pod A" from "pod-a.dgm"

interact {
  click pod1    -> view podA { label: "Zoom into Pod A" }
  click cluster -> reveal cp
  click pod2    -> step balance
  click svc     -> url "https://grafana.example.com/d/svc"
}
```

- `view <alias> ["Title"] from "<path>"` — the path is quoted, relative to
  the declaring file, and must point at an ordinary `.dgm` (which lints and
  previews on its own). `preview` bundles the whole set into one page.
- Click sources are nodes or subgraphs; **one binding per element**. Verbs:
  `view <alias>`, `reveal <id>[, <id>…]`, `step <effective-step-id>`,
  `url "https://…"` (only `url` takes a quoted target; only `reveal` takes
  several).
- **Being a reveal target is what makes an element start hidden** — there is
  no separate declaration; a revealed subgraph brings its contents. Reveal is
  interaction state (persists until the viewer leaves the view);
  `show`/`hide` are timeline state (a seek resets them).
- A `click … -> step` whose id only exists in a later scenario warns: it does
  nothing until that scenario is chosen.

## The lint loop

`cinegram lint file.dgm --format=json` →
`[{"file","line","col","severity","message","hint"}]`; warnings exit 0,
errors exit 1. Fix everything — hints usually name the fix (misspelled ids
get a did-you-mean; a missing edge suggests what to add or reroute).

## Pitfalls (all verified against the parser)

- **Do not route a `flow` through a subgraph id.** It validates clean and
  then silently produces no animation. Flow endpoints must be nodes.
- A node id starting with a digit cannot be targeted by any action.
- `delay` and `at` add together if you set both.
- `dur` on `set`/`gauge`/`unset` is ignored (warning) — persistence is ended
  by the next write or `unset`, not by a duration.
- `repeat` and `bidi` on flows are reserved and unread — don't emit them.
- An element declared but never referenced (by an edge, action, or click)
  warns; a `scene` naming a same-named frame does not count as a reference.
- Two scenarios with the same name: the picker can't tell them apart (warning).

## Compact grammar

```
document    := [frontmatter] diagram-header diagram-body { toplevel }
toplevel    := scenario | storyboard | viewdecl | interact
scenario    := "scenario" [ident|string] ["{" attrs "}"] { step }
step        := "step" [ident] [string] "{" { key ":" value | action } "}"
action      := "flow" ident {"->" ident} [attrblock]          // ≥2 idents, one line
             | ("highlight"|"dim"|"pulse"|"show"|"hide"|"focus"
               |"set"|"gauge"|"unset"|"scene") ident {"," ident} [attrblock]
             | "note" ident string [attrblock]
             | "wait" duration
             | "seq" "{" { key ":" value | action } "}"       // attrs INSIDE braces
attrblock   := "{" { ident ":" value [","] } "}"              // opens on the action's line
storyboard  := "storyboard" [string] "{" { "frame" ident "{" ("img"|"caption") ":" string … "}" } "}"
viewdecl    := "view" ident [string] "from" string
interact    := "interact" "{" { "click" ident "->" ("view" ident | "reveal" ident {"," ident}
                              | "step" ident | "url" string) [attrblock] } "}"
```

## Authoring judgement (what makes a good animation)

- One story per scenario; tell the failure path as a `variant` rather than
  cramming both into one timeline.
- 4–8 steps is the sweet spot; give each a `desc` sentence or two — the
  narration is half the value of the output.
- Prefer step-level parallelism (a `flow` plus a `highlight`/`note` starting
  together) over `seq`; reach for `seq` only for genuine cause-then-effect
  within one beat.
- Use `set`/`gauge` for facts a scrubbing reader must see (who is leader, how
  many votes), not for transient emphasis.
- Give flows `label`s in flowcharts (`"POST /token + secret"`, `"200 OK"`) —
  they carry the protocol; omit them in sequence diagrams.
