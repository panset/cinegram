# Timing rules

All of this lives in `pkg/compile`. Nothing here may reach for geometry or for
diagram vocabulary.

- Actions inside a step start together; steps run in sequence. `seq { }` chains.
- A `flow` splits its `dur` across hops using `total*i/hops`, so hops tile the
  duration exactly regardless of remainder.
- A stateful action (`highlight`, `dim`, `note`, …) with no `dur` spans its
  whole step.
- A flow may run against an edge's drawn direction; `symbol.Table.FindEdge`
  matches reversed edges and the track records `Reverse`.
- `scene` is a stateful action like any other here, but its target is a
  **storyboard frame**, not a node: it is the one action `validate.go` resolves
  against `doc.Storyboards` instead of the symbol table, and the runtime shows
  the latest scene track with `Start <= t` rather than the ones open at `t`.
  That stickiness is what makes the panel hold a screen across the steps where
  nothing the user can see changes.
- A `scene` inside a `seq` costs **zero** of the chain (`seqSpan`), like a
  persistent action: it fires where the chain has reached and the panel then
  holds, so `seq { flow a -> b; scene x }` means "the screen changes when the
  arrow lands" without the author computing an `at:`.
- `scenario … { variant: "base", until: <step> }` is spliced in
  `resolveVariants` **before** any timing runs, so the merged scenario is an
  ordinary `ast.Scenario` and every rule above applies to it unchanged. Depth-1
  only; `until` is inclusive. Keep the splice at AST level — lowering it would
  mean reimplementing hop-occurrence and persistent-window resets.
