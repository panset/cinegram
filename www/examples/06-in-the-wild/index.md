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

