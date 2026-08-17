# The playground

A static page that runs the compiler in the tab. `wasm/` builds
`pkg/{loader,compile,emit/html}` to js/wasm, and `playground.js` calls it
through `cinegramCompile` / `cinegramRenderHTML`.

## It is assembled, never committed

Nothing in this folder is a site. `bazel run //web/playground:site -- -o DIR`
(or `--serve` to develop against it) assembles one out of the page here, the
canonical browser assets in `pkg/emit/html/assets/`, and the `.wasm` plus
`wasm_exec.js` from the same Go SDK that compiled it. So the 6.4 MB `.wasm` is
never in git, and there is no fourth copy of `mermaid.min.js` to drift.

`.github/workflows/pages.yml` assembles it at `/playground/`, beside the
committed `docs/` — see `.claude/skills/publish-site/`.

## Three things that are easy to break

- **`BUILD.bazel` here and in `wasm/` are hand-written**, each with a leading
  `# gazelle:ignore` and a comment saying why. Re-running `bazel run //:gazelle`
  must leave both untouched.
- **The envelope wire format lives in `pkg/envelope`**, not in
  `cmd/cinegram/main.go`, because the WASM main and the CLI hand the same JSON
  to the same consumers.
- **`pkg/emit/html.Render` is compiled into the `.wasm`** for exactly one
  reason: Download HTML must be byte-identical to `cinegram preview -o`.
