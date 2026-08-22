<!-- Generated from examples/07-from-a-trace by `bazel run //site:sync`. Do not edit: //site:site_test fails while this file disagrees with its source, and the next sync overwrites it. -->

# from a trace

## [the slow checkout, 14:02](01-checkout-architecture.md)

The checkout path as the team draws it on a whiteboard, animated by a request that actually happened. The scenario below was appended by `cinegram trace` from `checkout-slow.otlp.json`, so every duration in it is a measurement rather than a guess: a card-network authorisation timed out after a full second, the retry took another 1.14s, and a checkout that should be under half a second took 2.6. Delete the scenario, re-run the command, and it comes back byte for byte. Two details are worth reading the source for: `inventory` and `payments` overlapped, so they share one step and keep their real offsets via `at:`; and the trace caught `inventory` querying the database directly, which this diagram does not draw — the command says so rather than inventing an edge.

