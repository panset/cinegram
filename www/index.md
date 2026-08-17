# Cinegram

**Mermaid draws the system; Cinegram plays the story.**

Animated, narrated, explorable architecture diagrams from a Mermaid-compatible
DSL — for the humans who read them and the AI agents that write them.

A static diagram shows you that an Ingress sits in front of a Service. It
cannot show you what happens to a single `GET /api/orders` as it travels
LB → Ingress → Service → Pod and back. Cinegram adds a small animation
language on top of Mermaid so that path becomes something you can watch.

Press play.

<div class="cinegram" data-cinegram="01-basics/01-k8s-request" data-height="740"></div>

That is not a video and not a GIF. It is the diagram below, compiled to a
timeline and animated in your browser — scrubbable, and readable as text in
[the source](examples/01-basics/01-k8s-request.md).

## Where to go

<div class="grid cards" markdown>

- **[The examples tour](examples/index.md)**

    Twelve diagrams, front to back: a request path, a release, a failure and a
    recovery, a consensus round. Each one plays on the page and opens in the
    playground.

- **[The guide](guide/index.md)**

    What a `.dgm` file is made of, how to narrate one, how to present it, and
    every command the CLI has.

- **[Embedding cinegrams](embedding.md)**

    Putting a player inside a page of your own — a Zensical or MkDocs site, or
    anything else you serve. This site is built that way.

- **[The playground](playground/)**

    The compiler itself, built to WASM and running in the tab. Paste a
    diagram, watch it animate, share the link. Nothing is uploaded.

</div>

## Get it

A single static binary with no runtime dependencies — the compiler, the player
and its own copy of Mermaid are all inside it. No package manager involved:

```sh
# target: darwin-arm64 | darwin-x64 | linux-x64 | linux-arm64
#         (Windows: cinegram-win32-x64.exe)
TARGET="$(uname -s | tr A-Z a-z)-$(uname -m | sed -e s/x86_64/x64/ -e s/aarch64/arm64/)"
mkdir -p ~/.cinegram
curl -fsSL "https://github.com/panset/cinegram/releases/latest/download/cinegram-$TARGET" \
  -o ~/.cinegram/cinegram
chmod +x ~/.cinegram/cinegram
~/.cinegram/cinegram version
```

Then `cinegram preview diagram.dgm -o out.html` and open it. Later,
`cinegram upgrade` replaces the binary in place with the checksum-verified
latest release.

There is also a
[VS Code extension](https://marketplace.visualstudio.com/items?itemName=tejaspanse.cinegram)
with a live preview beside the editor — it carries the same binary — and the
[source is on GitHub](https://github.com/panset/cinegram).
