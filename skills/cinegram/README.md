# Cinegram authoring skill

Teach your AI coding agent to animate Mermaid diagrams. Paste your existing
`flowchart`, `sequenceDiagram` or `stateDiagram-v2`, say what should move — *"walk a request
through the load balancer, then show the retry when the pod dies"* — and the
agent writes, validates and previews the [Cinegram](https://github.com/panset/cinegram)
`.dgm` file for you. You never have to learn the format.

The skill is self-contained: it carries the full language reference, and it
can fetch the `cinegram` binary by itself from the project's
[GitHub releases](https://github.com/panset/cinegram/releases) — a single
static binary per platform, no package manager, no repo clone.

## Install

Claude Code (personal, all projects):

```sh
curl -fsSL https://raw.githubusercontent.com/panset/cinegram/main/skills/cinegram/install.sh | sh
```

Cursor (per project — run from the project root):

```sh
curl -fsSL https://raw.githubusercontent.com/panset/cinegram/main/skills/cinegram/install.sh | sh -s -- cursor
```

## Try asking

Start a new session and say what you want in plain words — the agent does the
rest (writes the `.dgm`, lints it until clean, previews it as one self-contained
HTML file, fetches the `cinegram` binary if you have none):

- *"Animate this Mermaid diagram with cinegram and give me the HTML: …"* — you
  have a diagram; it gets a walkthrough.
- *"Understand the request flow in `src/api/` and create a cinegram of it"* — you
  have code; the agent draws the Mermaid first, then animates it.
- *"Read `.github/workflows/deploy.yml`, draw the pipeline, and animate a
  deploy"* — same, for a workflow.
- *"Add a failure path where the payment provider times out"* — an existing
  cinegram gets a `variant` scenario.
- *"Narrate it"* / *"record it as a GIF"* / *"make a contact sheet"* — the
  other outputs, once there is a `.dgm`.

In the very session that installed it, add *"read the installed SKILL.md and
follow it"* — new skills are discovered at session start.

To share the skill with a project's whole team instead, commit `SKILL.md`
and `references/language.md` under `<project>/.claude/skills/cinegram/`
(Claude Code) or keep the `.cursor/` files the installer wrote (Cursor).

## What's in the folder

| File | Purpose |
| --- | --- |
| `SKILL.md` | The workflow: find or install the binary, author, lint until clean, preview, narrate. |
| `references/language.md` | The complete `.dgm` language reference the agent reads before writing. |
| `cinegram.mdc` | A Cursor rule that routes diagram-animation requests to the skill. |
| `install.sh` | The one-line installer above (`sh install.sh [claude|cursor]`). |
