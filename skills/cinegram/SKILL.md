---
name: cinegram
description: >
  Author, edit, preview and record Cinegram .dgm files — animated, narrated
  Mermaid diagrams. Use when the user wants to animate a Mermaid diagram
  (flowchart or sequenceDiagram), add an animated walkthrough / scenario to a
  diagram, or create, edit, lint, preview or export a .dgm file.
---

# Animating Mermaid diagrams with Cinegram

A `.dgm` file is a Mermaid diagram followed by animation blocks. You write the
animation; the `cinegram` CLI validates it, previews it, and records it. The
user does not need to know the format — you are the author, they direct.

**Before writing any `.dgm` content, read [references/language.md](references/language.md)
in this skill folder.** It is the complete authoring reference. Do not write
from memory of Mermaid or guesses about the syntax.

## Step 0 — get a working binary

Try these in order; use the first that works. Verify with `cinegram version`
(any subcommand error output means the candidate exists — keep it).

1. `$CINEGRAM_BIN` if set, else `cinegram` on `PATH`.
2. The copy bundled with the VS Code / Cursor extension:
   ```sh
   ls ~/.vscode/extensions/tejaspanse.cinegram-*/bin/*/cinegram \
      ~/.cursor/extensions/tejaspanse.cinegram-*/bin/*/cinegram 2>/dev/null | tail -1
   ```
3. A workspace build, if the cinegram repo itself is the workspace:
   `bazel-bin/cmd/cinegram/cinegram_/cinegram` (build with `bazel build //cmd/cinegram`).
4. **Install it** (no package manager needed) from the tested, versioned
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
latest release. Two exceptions: a workspace Bazel build, which `upgrade`
refuses — rebuild that with `bazel build //cmd/cinegram` — and a binary that
answers `unknown command "upgrade"`, which predates 0.2.0: replace that one
by re-running the install download above over it (the URL always fetches the
latest release).

Below, `cinegram` means whichever path you found. It is a single static
binary with no dependencies; GIF recording needs nothing else installed
(mp4/webm need ffmpeg, PNG frames and GIF need a Chrome/Chromium on `PATH`
or named by `$CINEGRAM_CHROME`).

## Workflow

### 1. Start from the user's Mermaid — and never change it

A `.dgm` is the user's Mermaid source **verbatim**, with `scenario` (and
optionally `storyboard`, `view`, `interact`) blocks appended after it. Do not
reformat, re-indent, rename ids, or "clean up" their diagram. If the diagram
lacks an edge the animation needs, say so and ask — don't silently add one.

To check you kept it intact: `cinegram mermaid out.dgm` prints the diagram
half back out; it should match what the user gave you.

### 2. Write the scenario

Read `references/language.md` first. The rules that most often trip authors:

- Actions inside a `step` start **together**; steps run in sequence. Use
  `seq { … }` only when actions inside one step must chain.
- A `flow` needs a matching edge in the diagram (direction may be reversed —
  response paths reuse the request edge). No edge → lint error.
- Strings live on one line; use `\n` inside them, never wrap the source line.
- Every step gets a short id and a human title: `step route "Ingress matches" { … }`.
- Use `desc:` liberally — the narration is half the value of the output.

### 3. Lint until clean — this loop is mandatory

```sh
cinegram lint out.dgm --format=json
```

Emits `[{"file","line","col","severity","message","hint"}]` on stdout; exit
code 0 with warnings, 1 with errors. Fix **every** diagnostic — the hints
usually name the fix (e.g. a misspelled node id suggests the right one).
Warnings matter too: they describe animations that compile but won't do what
was meant. Re-run until the output is `[]`.

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
| GIF for a PR / README (needs Chrome only) | `cinegram record out.dgm -o out.gif --fps 10` |
| Video | `cinegram record out.dgm -o out.mp4` (or `.webm`; needs ffmpeg) |
| One scenario / one sub-view | add `--scenario s1` / `--view <id>` to `record`/`frame` |
| Still of one moment | `cinegram frame out.dgm --at 1620ms -o still.png` |
| Plain Mermaid back out | `cinegram mermaid out.dgm` |
| Written walkthrough for docs | `cinegram narrate out.dgm -o walkthrough.md` |
| Timeline as data | `cinegram compile out.dgm -o timeline.json` |

Useful page facts: `out.html#v=<view>&s=<scenario>&t=<ms>` deep-links a
paused moment (the **Copy link** button writes it); `?embed` strips the
chrome for an `<iframe>`; `?present` is presenter mode (Space plays one step
at a time). Scenarios are addressed as `s0`, `s1`, … in declaration order.
`record --width/--height` default to 1280×720, `--fps` to 12.

### 6. Verify visually when it matters

`cinegram frame` captures one exact, deterministic moment (needs Chrome on
`PATH` or `$CINEGRAM_CHROME`). Read the PNG to confirm a key beat looks
right — e.g. that a `status: fail` ✕ lands where intended.
