---
name: explain-diagram
description: >
  Explain a diagram at several reading levels at once — like the reader is five,
  like they are new to the team, and like they are the engineer on call. Use when
  the user wants a diagram explained simply, wants an ELI5 / plain-English
  version of an architecture or protocol, wants onboarding narration for a
  system diagram, or wants one diagram to serve both a non-technical audience
  and an expert one.
---

# Explaining a diagram at several reading levels

A Cinegram scenario already carries prose: every `step` takes a `desc` that the
page shows as narration while the animation runs. This skill is about *what to
write in it*, and about writing it more than once — one animation, several
tellings, so the same diagram serves a curious five-year-old and the engineer
who has to fix it at 3am.

**Prerequisite: read the [`cinegram`](../cinegram/SKILL.md) skill first**, and
its [`references/language.md`](../cinegram/references/language.md). That is how
you get a working binary and how you author a `.dgm` at all. This skill adds
only the narration layer on top and assumes you can already write and lint one.

## The ladder

Three rungs, and they are genuinely different jobs rather than the same text at
three temperatures:

| Rung | `audience` | What it does | What it must never do |
| --- | --- | --- | --- |
| Engineer | *(the base)* | States the mechanism precisely, and argues *why this design and not the obvious cheaper one*. | Skip the tradeoff. If it reads like a description, it is not finished. |
| Newcomer | `newcomer` | Plain English, real names for real things, no metaphor. What happens, in order, with nothing assumed. | Use a term the diagram itself introduced without saying what it is. |
| Child | `kid` | One sustained metaphor made of things you could film. Carries *why it works*, not the vocabulary. | Contain a single domain word. Not one. |

**Write the engineer rung first, always.** It becomes the base the others
retell, and the order is not arbitrary: you cannot simplify something you have
not yet stated precisely, and an ELI5 written first tends to be a confident
explanation of something subtly wrong. Precision first, then strip.

## One animation, several tellings

Do not copy the scenario per rung. `retells` adopts the base's steps, actions
and timing wholesale and replaces only the words:

```
scenario "authorization code flow" { speed: 1.0 }

  step exchange "The app trades the code for tokens" {
    desc: "It sends the code together with its client secret, and that pairing is what proves the exchange is genuine."
    flow app -> auth { label: "POST /token + secret", dur: 700ms }
  }

scenario "like you're 5" { retells: "authorization code flow", audience: "kid" }

  step exchange "The app shows the ticket and its own badge" {
    desc: "It hands over the ticket together with its own secret badge. Neither is enough alone — the two arriving together is what proves this is really the app."
  }
```

Rules that follow from that, and that `cinegram lint` will hold you to:

- A retelling names an **existing step id** and carries **only** `desc` (and
  optionally a new title). No actions — a retelling changes what is *said*, never
  what happens.
- Steps you say nothing about keep the base's prose. A rung only has to differ
  where it needs to; a step whose engineer prose is already plain can be left
  alone, and leaving it alone is better than paraphrasing it worse.
- A retelling inherits `speed`, `loop` and `outcome`, so a failure path stays a
  failure path in every telling.
- To change *what happens*, that is a `variant`, not a retelling. The two are
  different features and lint rejects doing both in one scenario.

The reader switches rungs in the scenario picker, and the diagram does not move
when they do — which is the whole point. An explanation a child can follow and a
reference an engineer trusts cannot contradict each other about the mechanics
when there is only one set of mechanics.

## Writing the child rung

This is the part that is actually hard. The failure mode is not being too
technical, it is being *cute and empty* — friendly words wrapped around a
sentence that no longer explains anything.

**One metaphor, sustained.** Pick it once and carry it through every step. A
front desk, a paper ticket, a wristband. A metaphor that changes per step is
jargon with a costume on: the reader has to learn eight unrelated pictures
instead of one, which is harder than the real thing. If the metaphor cannot
stretch to step six, get a different metaphor — do not bolt a second one on.

**Only things you could film.** Concrete nouns and physical verbs: a note, a
door, a stub, a badge; hand, walk, knock, show, tear up. If you cannot picture
someone doing it, rewrite it.

**Say why, not what.** The animation already shows what moves where; the reader
can see the arrow. The prose is for the part that is invisible — why it goes the
long way round, why that thing is useless on its own, what would break if it
were done the obvious way.

**Zero domain words.** Not softened, not introduced-then-used — absent. If a
word only exists inside the subject you are explaining, it is out: token,
endpoint, payload, session, handshake, cache, replica, quorum, authenticate,
validate, asynchronous. A good check: every noun in the child rung should be one
a five-year-old has personally held or seen.

**Second person, present tense.** "You tell the app you want to come in" lands;
"the user then initiates a request" does not.

**One to three sentences, and read it out loud.** The caption renders large in
presenter and reel modes, and it may well be spoken by a narrator — if you
stumble reading it aloud, it is wrong. Around forty words is the ceiling before
a portrait reel starts to overflow.

**Never say "simply", "just", "basically", or "all you have to do".** They
inform nobody and they make a reader who is confused feel stupid.

**Do not lie to simplify.** The single rule that outranks the others. If the
metaphor forces a claim that is false, change the metaphor, not the truth — a
child who later learns the real mechanism should find the story was a smaller
version of it, not a different one.

## Verify

`lint` is mandatory and is not the interesting check. These are:

```sh
cinegram lint    diagram.dgm            # must be clean
cinegram narrate diagram.dgm            # read every rung back, top to bottom
cinegram preview diagram.dgm --serve    # switch rungs in the picker
```

Three things to look for in `narrate` output, which no linter can catch:

1. **Each rung reads as one coherent piece on its own.** Read it as a reader
   would, start to finish, without the other rungs for context. A retelling that
   only makes sense next to the base has not replaced enough.
2. **Every rung reports the same duration.** `narrate` prints it per scenario.
   Identical durations are the proof that you retold the animation instead of
   accidentally rewriting it.
3. **The metaphor survives to the last step.** Skim the child rung for the
   moment it quietly reverts to real vocabulary; that is the step where the
   metaphor broke and you stopped noticing.

`examples/oauth-login.dgm` in this repository is a worked three-rung example.

## Delivering it

The rungs are ordinary scenarios, so everything downstream already works — and
`--reel` is worth knowing about here specifically:

```sh
cinegram record diagram.dgm -o eli5.mp4 --scenario s1 --reel --format mp4
```

That is the child rung as a portrait video, which is the form an explanation
like this usually wants to travel in. `--scenario` takes the compiled id
(`s0`, `s1`, …) as `cinegram compile` reports it.
