# CLAUDE.md

Cinegram compiles a `.dgm` source — a Mermaid diagram plus `scenario`, `view`
and `interact` blocks — into an animated, self-contained HTML page.

**Read the `build` skill before any code change.** It carries the commands, and
the hermetic invariants that no domain may break.

Then the area you are working in:

| Area | Where |
| --- | --- |
| `pkg/{loader,parser,compile,ir,emit}`, `runtime.js` | the `rendering-pipeline` skill |
| `www/`, `zensical.toml`, `examples/`, `pkg/{sitegen,embedkit}`, `site/` | the `publish-site` skill |
| the VS Code extension | [`editors/vscode/CONTRIBUTING.md`](editors/vscode/CONTRIBUTING.md) |
| the WASM playground | [`web/playground/README.md`](web/playground/README.md) |
| cutting a release | [`RELEASING.md`](RELEASING.md) |

`skills/` is not for any of that: it holds the **cinegram authoring skill that
ships to users**, installed by `skills/cinegram/install.sh` and verified by
`.github/workflows/release.yml`. Repository guidance lives in `.claude/skills/`.
