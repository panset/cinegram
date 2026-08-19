<!-- Generated from README.md by `bazel run //site:sync`. Do not edit: //site:site_test fails while this file disagrees with its source, and the next sync overwrites it. -->

# Interaction

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

**A bound subgraph is clicked on its border or its title, not its middle.** A
subgraph's box is mostly the space around the things inside it, and clicking
there means the diagram, not the box — most of all while presenting, where a
click on the stage is how you get to the next step. The chip in its corner is a
target for the binding wherever a border is awkward to hit.

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
