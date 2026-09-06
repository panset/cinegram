<!-- Generated from examples/06-in-the-wild by `bazel run //site:sync`. Do not edit: //site:site_test fails while this file disagrees with its source, and the next sync overwrites it. -->

# in the wild

## [build it up](01-progressive-reveal.md)

A `<v-click>`-style build-up of one Mermaid flowchart: seven steps, each revealing exactly one more piece of the architecture, with the narration a speaker would say over it. Asked for in slidevjs/slidev#1498 (https://github.com/slidevjs/slidev/issues/1498), where duplicating the diagram per slide made elements shift and `visibility:hidden` left the arrows behind.

## [walk the pipeline](02-stepwise-greyed-out.md)

Step-by-step presentation of one diagram, greyed out rather than hidden: the whole pipeline is on screen from the first beat, and the parts the story has not reached yet are faded back instead of removed. Asked for in mermaid-js/mermaid#7710 (https://github.com/mermaid-js/mermaid/issues/7710): advance on click or space, grey out what is still to come, and have a way to highlight specific elements.

## [one polling cycle](03-poller-sequence.md)

A twenty-message poller-and-webhook sequence diagram, exactly as posted in mermaidjs/mermaid-live-editor#53 (https://github.com/mermaidjs/mermaid-live-editor/issues/53) — the 2019 "Animated Diagrams" request whose author found larger diagrams "a bit overwhelming" and built a proof-of-concept to animate this one.

## [auth refresh edge cases](04-auth-refresh-edge-cases.md)

One diagram, three scenarios: a token refresh that works, a refresh token that was already rotated by a concurrent login, and an identity provider that does not answer. The happy path is the only thing drawn at full strength; the two edge cases are `variant` scenarios that replay it up to the exact hop where reality diverges and then tell their own ending. The shape of the question comes from r/webdev (reddit.com/r/webdev/comments/1u87ptj): "the happy path is clean but the edge cases are killing the diagram."

## [claude code tool call](05-claude-code-tool-call.md)

What actually happens when Claude Code runs a single tool call, end to end: one prompt, one round trip through the model, one Bash invocation against the repo, and the answer that comes back. Drawn for the people in anthropics/claude-code#14375 (https://github.com/anthropics/claude-code/issues/14375) who use Claude Code to map out codebases and wanted real diagrams instead of ASCII art.

## [conference signup flow](06-calm-flow-over-architecture.md)

A CALM flow playing over the architecture it runs on. An attendee signs up for a conference, and the request travels the very nodes and relationships the architecture declares: an actor, a webclient, a load balancer, an attendee service, a notification service and an attendee database — the last four `deployed-in` one Kubernetes cluster. Asked for in finos/architecture-as-code#2999 (https://github.com/finos/architecture-as-code/issues/2999), which proposes picking a published flow in CALM Hub and watching it as a live diagram with play, pause, step and scrub.

## [escaping the self-loop note hack](07-flowchart-node-notes.md)

Mermaid has no way to attach a note to a flowchart node — only sequence diagrams get `note right of X`. mermaid-js/mermaid#2712 (https://github.com/mermaid-js/mermaid/issues/2712) and its much older, much bigger sibling #821 (https://github.com/mermaid-js/mermaid/issues/821, open since 2019, 150 thumbs-up) both ask for it. The workaround people reach for — drawn verbatim in #2712 — is a fake self-loop edge that carries the note text, hidden afterwards with `linkStyle`. Its own author lists the cost: the fake edges are "hard to compute the number of ... when there are many edges", they "seriously affect the layout", and they are "very not convenient to distinguish" from the real ones.

## [two questions, answered in order](08-overlapping-activations.md)

Alice fires two questions at John before he answers either, so both of his activation bars are open at once. The animation plays the four messages in the order they happened — first question, second question, first answer, second answer — and a gauge on John counts the open questions live, so the pairing the static diagram cannot show is simply watched happening. See mermaid-js/mermaid #1765: https://github.com/mermaid-js/mermaid/issues/1765

## [order placed: payment approved](09-animate-the-flow.md)

A single POST /orders call unwinds through a service layer — client, controller, service, payment gateway, repository — one message at a time, with the reader holding the scrubber instead of a playback timer deciding the pace. It answers mermaid-js/zenuml-core #268, "animate the diagram": https://github.com/mermaid-js/zenuml-core/issues/268

## [replay a finished run](10-agent-run-replay.md)

A finished multi-agent run, replayed. A main agent spawns three subagents — research, design doc, test runner — one of those spawns a subagent of its own, the test run dies on a fixture database nobody migrated, and the run recovers and says so. Scrub to any second and every agent shows the context, tokens and cost it had consumed by then, and how many subagents were alive at that instant. Drawn for anthropics/claude-code#24537 (https://github.com/anthropics/claude-code/issues/24537), the Agent Hierarchy Dashboard proposal, which asks for historical replay of a multi-agent run with transport controls, a timeline scrubber, per-agent resource numbers, and an interactive HTML export.

