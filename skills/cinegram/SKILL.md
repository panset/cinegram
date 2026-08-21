---
name: cinegram
description: >
  Author, preview and record Cinegram .dgm files — animated, narrated Mermaid
  diagrams. Use when the user wants to animate or narrate a Mermaid diagram
  (flowchart, sequenceDiagram or stateDiagram), or to create, edit, lint,
  preview, record or export a .dgm file.
---

# Animating Mermaid diagrams with Cinegram

A `.dgm` file is a Mermaid diagram followed by animation blocks. You write the
animation; the `cinegram` CLI validates it, previews it, and records it. The
user does not need to know the format — you are the author, they direct.

**Before writing any `.dgm` content, read [references/language.md](references/language.md)
in this skill folder.** It is the complete authoring reference — write every
line from it rather than from memory of Mermaid.

## Step 0 — get a working binary

Try these in order; use the first that works. Verify with `cinegram version`
(any subcommand error output means the candidate exists — keep it).

1. `$CINEGRAM_BIN` if set, else `cinegram` on `PATH`.
2. A package manager the machine already has, which needs no install step of
   its own: `npx cinegram version` (Node ≥ 18) or `uvx cinegram version` (uv).
   Both are launchers — they download the release binary for this platform,
   check it against the release's `SHA256SUMS`, cache it under
   `~/.cinegram/bin/v<version>/`, and run it. Prefix any command below with
   `npx cinegram` / `uvx cinegram` the same way. After the first run the
   resolved binary sits at `~/.cinegram/bin/v<version>/cinegram` and can be
   called directly, which is faster than paying the launcher every command. One caveat: the cache entry is
   version-pinned, so `cinegram upgrade` refuses to run under either — get a
   newer version with `npx cinegram@latest …` or `uvx cinegram@latest …`.
3. The copy bundled with the VS Code / Cursor extension:
   ```sh
   ls ~/.vscode/extensions/tejaspanse.cinegram-*/bin/*/cinegram \
      ~/.cursor/extensions/tejaspanse.cinegram-*/bin/*/cinegram 2>/dev/null | tail -1
   ```
4. A workspace build, if the cinegram repo itself is the workspace:
   `bazel-bin/cmd/cinegram/cinegram_/cinegram` (build with `bazel build //cmd/cinegram`).
5. **Install it** (no package manager needed) from the tested, versioned
   GitHub release — a single static binary:

   ```sh
   # target: darwin-arm64 | darwin-x64 | linux-x64 | linux-arm64 (Windows: cinegram-win32-x64.exe)
   TARGET="$(uname -s | tr A-Z a-z)-$(uname -m | sed -e s/x86_64/x64/ -e s/aarch64/arm64/)"
   mkdir -p ~/.cinegram
   curl -fsSL "https://github.com/panset/cinegram/releases/latest/download/cinegram-$TARGET" \
     -o ~/.cinegram/cinegram
   chmod +x ~/.cinegram/cinegram
   ~/.cinegram/cinegram version
   ```

   The binary is then `~/.cinegram/cinegram`. Offer to link it onto `PATH`
   (`ln -sf ~/.cinegram/cinegram /usr/local/bin/cinegram` or the user's
   preferred bin dir) but don't require it — an absolute path works for
   everything below. Ask before installing if the user has not already
   agreed to tools being installed.

   Fallback if the release is unreachable: the VS Code Marketplace package
   `tejaspanse.cinegram` carries the same binary — download
   `https://marketplace.visualstudio.com/_apis/public/gallery/publishers/tejaspanse/vsextensions/cinegram/<version>/vspackage?targetPlatform=$TARGET`
   with `curl --compressed`, `unzip` it, and use `extension/bin/*/cinegram`.

A binary that exists but is stale can update itself: `cinegram upgrade
--check` reports whether a newer release exists (exit 1 when one does), and
`cinegram upgrade` replaces the binary in place with the checksum-verified
latest release. Three exceptions: a workspace Bazel build, which `upgrade`
refuses — rebuild that with `bazel build //cmd/cinegram`; a binary reached
through `npx`/`uvx`, which `upgrade` also refuses, since that cache entry
belongs to one released version — ask for a newer one (`npx cinegram@latest`,
`uvx cinegram@latest`); and a binary that answers `unknown command "upgrade"`,
which predates 0.2.0: replace that one by re-running the install download
above over it (the URL always fetches the latest release).

If your harness speaks MCP, the same binary is also a server: `cinegram mcp`
offers `lint`, `narrate`, `mermaid`, `frame` and `sheet` as tools, each taking
a `path` or an inline `source`, and serves this language reference as the
resource `cinegram://reference/language.md`. Use it when the host already has
it configured — the results come back in the conversation rather than as files
you then have to read. The CLI below stays the primary path: it is what these
instructions are written against, and it needs no host support at all.

Below, `cinegram` means whichever path you found: a single static binary with
no dependencies of its own. The delivery table in step 5 marks what an
individual output format additionally needs — Chrome/Chromium on `PATH` or
named by `$CINEGRAM_CHROME`, or ffmpeg on `PATH` or `$CINEGRAM_FFMPEG`.

## Workflow

### 1. Start from the user's Mermaid — and never change it

A `.dgm` is the user's Mermaid source **verbatim**, with `scenario` (and
optionally `storyboard`, `view`, `interact`) blocks appended after it. Do not
reformat, re-indent, rename ids, or "clean up" their diagram. If the diagram
lacks an edge the animation needs, say so and ask — don't silently add one.

To check you kept it intact: `cinegram mermaid out.dgm` prints the diagram
half back out; it should match what the user gave you.

**If there is no diagram yet** — the ask is "understand this workflow / request
path / codebase and create a cinegram" — draw it first, then treat that drawing
as the user's. Read the code, config or docs the ask points at; write a small
Mermaid diagram of it (5–9 nodes, real ids, the edges the story will travel —
`flowchart LR` for a request path or pipeline, `sequenceDiagram` for a
protocol, `stateDiagram-v2` for a lifecycle); show it and say what you left
out; only then append the animation. From that point the rule above applies
unchanged: the diagram half is fixed, and anything the animation turns out to
need is a change you ask about, not one you slip in.

### 2. Write the scenario

Read `references/language.md` first. The rules that most often trip authors:

- Actions inside a `step` start **together**; steps run in sequence. Use
  `seq { … }` only when actions inside one step must chain.
- A `flow` needs a matching edge in the diagram (direction may be reversed —
  response paths reuse the request edge). No edge → lint error.
- Strings live on one line; use `\n` inside them, never wrap the source line.
- Every step gets a short id and a human title: `step route "Ingress matches" { … }`.
- Use `desc:` liberally — the narration is half the value of the output.
- A step id may equal a node id (`step auth` beside a node `auth` is fine) —
  steps and diagram ids are separate namespaces; only steps within one
  scenario must be unique.
- `%%` comments placed before the first `scenario`/`view`/`interact`/
  `storyboard` keyword belong to the diagram half and come back out of
  `cinegram mermaid`; keep your own notes inside a scenario or after that
  first keyword, or the round-trip check below will look like a mismatch.
- Group with subgraphs freely and target them with `highlight`/`focus` —
  never with `flow`. A subgraph nothing ever targets warns as unreferenced.
- A matrix or fan-out (three platforms, N shards, M replicas) is one node
  carrying a `set … { badge: "3 platforms" }`, not three nodes.
- Two or three labelled flows in one step is the practical ceiling; drop the
  labels the diagram already implies, or the text piles up.

### 3. Lint until clean — this loop is mandatory

```sh
cinegram lint out.dgm --fix --format=json --strict
```

`--fix` is the first move: it rewrites the file with the corrections the
"did you mean" diagnostics already carry — a misspelled node, frame, view
alias, step id, scenario name or attribute key — and reports each edit on
stderr as `fixed file:line:col: old -> new`. It never guesses: an edit is
applied only where the parser was confident enough to name the right spelling,
and only while the text on disk still matches. Then read the array for what is
left.

The array is `[{"file","line","col","severity","message","hint"}]` on stdout,
each entry optionally carrying `"fix": {"line","col","old","new"}`; with
`--strict` the exit code is 0 only when that output is `[]`, so warnings stop
the loop exactly as errors do. Fix **every** remaining diagnostic by hand — the
hints usually name the fix. Warnings matter too: they describe animations that
compile but won't do what was meant. Re-run until the output is `[]`.

(Without `--strict` the exit code is 0 with warnings and 1 with errors, and the
JSON is identical — an older binary that rejects either flag still lints, you
just have to read the array rather than the status, and fix by hand.)

### 4. Show the user the result

```sh
cinegram preview out.dgm -o out.html     # self-contained page, open it
cinegram preview out.dgm --serve --watch # live-reload while iterating
cinegram narrate out.dgm                 # the animation as written prose
```

Paste the `narrate` output (or summarize it) when asking the user to confirm
the animation does what they asked — they can check it in English without
reading the `.dgm`.

### 5. Deliver in whatever form the user needs

| Ask | Command |
| --- | --- |
| Interactive page (works from `file://`, self-contained, shareable) | `cinegram preview out.dgm -o out.html` |
| GIF for a PR / README | `cinegram record out.dgm -o out.gif --fps 10` — Chrome |
| Video | `cinegram record out.dgm -o out.mp4` (or `.webm`) — Chrome + ffmpeg |
| Vertical story clip for LinkedIn/Shorts/Slack (9:16, big captions, auto-follow camera) | `cinegram record out.dgm --reel -o out.mp4` — Chrome + ffmpeg (mp4 preferred at this size; `?reel` on the HTML page is the live version) |
| One scenario / one sub-view | add `--scenario s1` / `--view <id>` to `record`/`frame` |
| Still of one moment | `cinegram frame out.dgm --at 1620ms -o still.png` — Chrome |
| Every step at a glance, for a PR review or an agent to read | `cinegram sheet out.dgm -o sheet.png --manifest sheet.json` — Chrome (one captioned cell per step; `--cols N` to reshape the grid) |
| Plain Mermaid back out | `cinegram mermaid out.dgm` |
| Written walkthrough for docs | `cinegram narrate out.dgm -o walkthrough.md` |
| Timeline as data | `cinegram compile out.dgm -o timeline.json` |
| A browsable site from a folder of `.dgm` files | `cinegram site diagrams/ -o out/` |
| A diagram **inside** a page they already have (MkDocs/Zensical, or any site) | the embed kit — see below |

To put a player in the middle of someone's existing page rather than give it a
page of its own: `cinegram assets -o <site>/assets/cinegram` installs the
loader, its stylesheet and the player; `cinegram compile x.dgm -o
<site>/assets/cinegram/timelines/x.json` supplies what it plays; the page gets
`<div class="cinegram" data-cinegram="x" data-height="900"></div>`, and the
site loads `cinegram-embed.css` and `cinegram-embed.js` once, site-wide. They
are inert on pages with no diagram, so mermaid's 2.6 MB is only fetched where
it is needed. Full contract: <https://panset.github.io/cinegram/embedding/>.

Useful page facts: `out.html#v=<view>&s=<scenario>&t=<ms>` deep-links a
paused moment (the **Copy link** button writes it); `?embed` strips the
chrome for an `<iframe>`; `?present` is presenter mode (Space plays one step
at a time). Scenarios are addressed as `s0`, `s1`, … in declaration order.
`record --width/--height` default to 1280×720, `--fps` to 12.

### 6. Verify visually when it matters

**Read the sheet first, then use `frame` for close-ups.**

Both need a headless Chrome or Chromium on `PATH` or named by
`$CINEGRAM_CHROME` (the table in step 5). And both show the **first** scenario
unless you pass `--scenario "<name>"` — check every variant you wrote, since
the failure path is usually the one worth verifying.

```sh
cinegram sheet out.dgm -o sheet.png --manifest sheet.json
```

`sheet` is one PNG with a captioned cell per step — the whole scenario checked
in a single image read, instead of one screenshot per beat. Each cell is shot a
millisecond before its step ends, so its flows have landed and the caption
names the step it belongs to. `--manifest` maps every rectangle back to its
step id and the moment it shows.

Then `cinegram frame out.dgm --at 1620ms -o still.png` for anything the sheet
made you suspicious of: it captures that one exact, deterministic moment at
full size — e.g. that a `status: fail` ✕ lands where intended.
