# Cinegram authoring skill

Teach your AI coding agent to animate Mermaid diagrams. Paste your existing
`flowchart` or `sequenceDiagram`, say what should move — *"walk a request
through the load balancer, then show the retry when the pod dies"* — and the
agent writes, validates and previews the [Cinegram](https://github.com/panset/cinegram)
`.dgm` file for you. You never have to learn the format.

The skill is self-contained: it carries the full language reference, and it
can fetch the `cinegram` binary by itself from the project's
[GitHub releases](https://github.com/panset/cinegram/releases) — a single
static binary per platform, no package manager, no repo clone.

## Install — Claude Code

Personal (all projects):

```sh
mkdir -p ~/.claude/skills/cinegram/references
curl -fsSL https://raw.githubusercontent.com/panset/cinegram/main/skills/cinegram/SKILL.md \
  -o ~/.claude/skills/cinegram/SKILL.md
curl -fsSL https://raw.githubusercontent.com/panset/cinegram/main/skills/cinegram/references/language.md \
  -o ~/.claude/skills/cinegram/references/language.md
```

Per project: same two files under `<project>/.claude/skills/cinegram/` —
commit them and the whole team's agents pick the skill up.

Then just ask: *"animate this mermaid diagram: …"*.

## Install — Cursor

Put the same two files in the project and add the pointer rule:

```sh
mkdir -p .cursor/skills/cinegram/references .cursor/rules
curl -fsSL https://raw.githubusercontent.com/panset/cinegram/main/skills/cinegram/SKILL.md \
  -o .cursor/skills/cinegram/SKILL.md
curl -fsSL https://raw.githubusercontent.com/panset/cinegram/main/skills/cinegram/references/language.md \
  -o .cursor/skills/cinegram/references/language.md
curl -fsSL https://raw.githubusercontent.com/panset/cinegram/main/skills/cinegram/cinegram.mdc \
  -o .cursor/rules/cinegram.mdc
```

## What's in the folder

| File | Purpose |
| --- | --- |
| `SKILL.md` | The workflow: find or install the binary, author, lint until clean, preview, narrate. |
| `references/language.md` | The complete `.dgm` language reference the agent reads before writing. |
| `cinegram.mdc` | A Cursor rule that routes diagram-animation requests to the skill. |
