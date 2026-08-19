<!-- Generated from examples/05-ai-systems by `bazel run //site:sync`. Do not edit: //site:site_test fails while this file disagrees with its source, and the next sync overwrites it. -->

# ai systems

## [one question, four turns](01-agent-tool-loop.md)

One question through an agent's tool loop: think, call a tool, get a call wrong, retry, answer. The loop is not a metaphor here — the same agent node is entered four times, and the token and iteration counters say how far in you are at any moment.

## [fan out, barrier, synthesise](02-multi-agent-fanout.md)

An orchestrator splits one research brief across three worker agents, waits for the slowest of them, and writes the answer itself. Fan-out is the easy half; the barrier — sitting still while two workers are already done — is the half that decides how long the whole thing takes.

## [rag pipeline](03-rag-pipeline.md)

Retrieval-augmented generation, told twice: once with a warm cache, and once with the cache missing and a live web search standing in for the corpus. The second telling ends with an answer whose citations do not come from the documents anyone approved.

## [launch, negotiate, call](04-mcp-handshake.md)

The handshake `cinegram mcp` performs. An agent host launches the server on stdin and stdout, agrees a protocol version with it, asks what it can do, and only then calls a tool — here `sheet`, which comes back as a picture of a whole scenario.

