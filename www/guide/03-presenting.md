<!-- Generated from README.md by `bazel run //site:sync`. Do not edit: //site:site_test fails while this file disagrees with its source, and the next sync overwrites it. -->

# Presenting and sharing

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
