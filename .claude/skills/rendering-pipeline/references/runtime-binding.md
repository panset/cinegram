# How the runtime binds to the SVG

`pkg/emit/html/assets/runtime.js` gets an SVG mermaid drew and has to find the
nodes and edges the timeline names in it.

## Defensively, not by id

Nodes come from mermaid's `<renderer>-<id>-<n>` id. **Edges are matched
geometrically**, by comparing path endpoints to node centres — mermaid's
edge-id format has changed between releases, and geometric matching
additionally detects paths mermaid drew from the far end (composed with
`Reverse` as `!reverse !== !flip`). A backwards reading pays `REVERSE_COST`, so
two arrows running between the same pair in opposite directions each take the
one drawn for it instead of scoring identically and swapping.

Unbound nodes, edges or click sources surface in a warning banner on the page
rather than silently failing.

## Two strategies, not one per diagram type

`index()` picks between them on `ir.Diagram.Type`.

**Flowchart, and state diagrams on the same apparatus.** A state diagram reuses
`indexNodesBy`, `indexClusters`, `indexEdges` and `makeLayer`, differing in
exactly two details: mermaid's id prefix is `state-` rather than `flowchart-`
(and its composite clusters are `g.statediagram-cluster` with no counter
suffix), and the node lookup handed to `indexEdges` is merged with the
clusters, because a transition into a composite stops at the cluster's border
rather than at any node. That merge is passed **only** in the state branch, so
flowchart matching is byte-identical.

**Sequence diagrams** genuinely need the second strategy: they have neither
`g.node` nor `.edgePaths`. Actors are recovered by **column** — the parts of
one actor are loose rects and texts sharing a lifeline x — and wrapped in a
`g.dgm-actor`, so every existing `.dgm-highlight rect` rule applies unchanged.
Messages are matched to edges by **order**, because mermaid draws them top to
bottom in message order and that is far more robust than recovering identity
from geometry. Only the direction a line was drawn in is read from geometry,
and it composes with `Reverse` exactly as the flowchart `flip` does.

## One Player hosts every view

`build()` installs a **document-level** keydown handler, so building the chrome
more than once would stack handlers that fight over Space and the arrow keys.
Views are swapped inside the one Player instead.

Click listeners attach in `index()` — the one place that runs per mermaid
render with the id→element maps in hand. They live on SVG elements that
`render()` replaces wholesale, so they never need removing.

## What survives a frame

`baseClass` strips `dgm-*` classes and `applyNodeStates` rewrites the whole
class attribute every frame. Anything that must outlive a frame — the click
affordance, reveal state — has to be listed in `STICKY`, or the clock erases it
the moment a node animates.

The storyboard panel is overlay-style HTML built once in `build()` and shown or
hidden by `syncBoard()`, which re-runs per render *and* per scenario change,
because scene usage is per scenario. Being outside the SVG, `baseClass` and
`STICKY` do not apply to it: `applyBoard` diffs on the frame id instead, and
has to — rewriting an `<img src>` every frame would restart the crossfade
transition forever.

## Navigation

`navigate()` only moves `location.hash`; `applyHash()` does the work. A click
and a browser history move therefore follow the same path, and the Back button
cannot disagree with the browser's own.
