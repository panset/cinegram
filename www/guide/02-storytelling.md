<!-- Generated from README.md by `bazel run //site:sync`. Do not edit: //site:site_test fails while this file disagrees with its source, and the next sync overwrites it. -->

# Telling the story

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
