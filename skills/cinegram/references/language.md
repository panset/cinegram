# The .dgm language

A `.dgm` file is a Mermaid diagram followed by animation blocks:

```
<mermaid body — flowchart/graph, sequenceDiagram or stateDiagram-v2, untouched>

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
  `flowchart`/`graph`, `sequenceDiagram` and `stateDiagram-v2`
  (`stateDiagram` too).
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
| `retells` | Re-explain another scenario's steps in different words — see Retellings. |
| `audience` | Free label for who the words are written for (`kid`, `newcomer`, …). Steers no timing. |
| `pace` | Only `voice`: stretch each step to fit its recorded narration. Inert until clips exist. |

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

## State diagrams

States are nodes and each `A --> B` is its own edge. Composite `state X { }`
blocks behave exactly like subgraphs — target the composite to `highlight`,
`dim` or `reveal` everything inside it, at any nesting depth. `<<choice>>`,
`<<fork>>` and `<<join>>` pseudostates are ordinary targets. `note`,
`direction`, `classDef` and the `--` concurrency divider round-trip verbatim.
Rules:

- **`[*]` is addressed by a synthesized name**, because `[*]` itself is not a
  legal identifier. Use `root_start` and `root_end` at the top level, and
  `<Composite>_start` / `<Composite>_end` inside a composite — so
  `flow root_start -> CLOSED` animates the opening arrow. (Markers inside a
  composite split by `--` into concurrent regions are the exception: Mermaid
  names them after the region and they cannot be targeted.)
- **Use `msg:` when the same pair has several transitions**, exactly as in a
  sequence diagram. Two `A --> B` lines are two arrows.
- **Leave `label` off flows** — the diagram already prints the event text.
- A `flow` into a composite (`flow ESTABLISHED -> Teardown`) is fine; it lands
  on the composite's border, which is where Mermaid draws that arrow.

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

## Retellings (reading levels)

The same animation explained to a different reader. Use this — never a second
copy of the scenario — when a diagram needs both a plain-English telling and a
precise one.

```
scenario "like you're 5" { retells: "authorization code flow", audience: "kid" }
  step exchange "The app shows the ticket and its own badge" {
    desc: "It hands over the ticket together with its own secret badge. Neither is enough alone."
  }
```

- `retells:` names the base **by its name**. The retelling adopts all of its
  steps, actions and timing; each of its own `step`s names an existing step **by
  its effective id** and replaces that step's prose.
- A retelling's steps carry **`desc` and an optional title, and no actions** — an
  action in one is an error. Changing what happens is what `variant` is for, and
  a scenario that is both is an error.
- **Every step of a retelling needs an explicit id.** An anonymous step would
  fall back to `step<index>` and override whichever beat sat at that position.
- Steps the retelling says nothing about keep the base's prose. Write only the
  rungs that differ.
- Scenario attributes are **inherited then overridden** (the opposite of a
  variant), so `speed`, `loop` and `outcome` carry — a retold failure path is
  still a failure path.
- Retelling a `variant` is fine: variants resolve first, so the base is already
  spliced. Depth-1 among retellings: a retelling of a retelling is an error.
- `skills/explain-diagram/SKILL.md` is the guidance for *writing* the rungs.

## Narration out loud

`cinegram voice <file>` records each step's `desc` as speech into
`<file>.voice/`, and the player's **Voice** button speaks it. With no recording
the button falls back to the browser's own synthesizer, so a page can talk with
nothing installed.

- The synthesizer is `$CINEGRAM_TTS_COMMAND` (must write a WAV to `{out}`, read
  its line from stdin); macOS `say` is the default. Nothing is built in.
- Clips are keyed by **the words**, so rewording re-records one line and
  renaming or reordering re-records nothing.
- Add `pace: voice` to the scenario, or narration is cut off — a walkthrough
  written to be watched is far shorter than one read aloud.
- `--with-voice` is needed to carry it: on `preview` and `compile` to inline it,
  on `record` to mix it into an mp4 or webm (a GIF has no audio track).
- Never commit `*.voice/`; it is a build product.

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

`cinegram lint file.dgm --fix --format=json --strict` →
`[{"file","line","col","severity","message","hint"}]`, each entry optionally
carrying `"fix": {"line","col","old","new"}`; with `--strict` exit 0 only when
that is `[]`. Without it, warnings exit 0 and errors exit 1, and the JSON is the
same either way. `--fix` applies the did-you-mean edits to the file first —
misspelled node, frame, view, step, scenario and attribute names — and prints
`fixed file:line:col: old -> new` per edit on stderr; the exit status is still
the one the repaired file earns. Fix the rest by hand — hints usually name the
fix (a missing edge suggests what to add or reroute).

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
  they carry the protocol; omit them in sequence and state diagrams, which
  already draw the text themselves.
