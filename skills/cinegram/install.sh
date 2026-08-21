#!/bin/sh
# Installs the cinegram authoring skill for Claude Code or Cursor.
#
#   curl -fsSL https://raw.githubusercontent.com/panset/cinegram/main/skills/cinegram/install.sh | sh
#   curl -fsSL .../install.sh | sh -s -- cursor
#
# `claude` (the default) installs into ~/.claude/skills/cinegram/ for every
# project; `cursor` installs into ./.cursor/ of the current project, because
# Cursor rules are per-project. POSIX sh throughout — /bin/sh is dash on
# Debian-family systems.
set -eu

REF="${CINEGRAM_SKILL_REF:-main}"
BASE="https://raw.githubusercontent.com/panset/cinegram/$REF/skills/cinegram"
MODE="${1:-claude}"

fetch() {
  mkdir -p "$(dirname "$2")"
  curl -fsSL "$BASE/$1" -o "$2"
  echo "  $2"
}

case "$MODE" in
  claude)
    DEST="$HOME/.claude/skills/cinegram"
    echo "Installing the cinegram skill for Claude Code:"
    fetch SKILL.md "$DEST/SKILL.md"
    fetch references/language.md "$DEST/references/language.md"
    SKILL_MD="~/.claude/skills/cinegram/SKILL.md"
    ;;
  cursor)
    echo "Installing the cinegram skill for Cursor into $(pwd):"
    fetch SKILL.md .cursor/skills/cinegram/SKILL.md
    fetch references/language.md .cursor/skills/cinegram/references/language.md
    fetch cinegram.mdc .cursor/rules/cinegram.mdc
    SKILL_MD=".cursor/skills/cinegram/SKILL.md"
    ;;
  *)
    echo "usage: install.sh [claude|cursor]" >&2
    exit 2
    ;;
esac

cat <<EOF

Done. Start a new session and try one of these:

  "Animate this Mermaid diagram with cinegram and give me the HTML: <paste your diagram>"
  "Understand the request flow in src/api/ and create a cinegram of it"
  "Read .github/workflows/deploy.yml, draw the pipeline, and animate a deploy with cinegram"
  "Add a failure path to this cinegram where the payment provider times out"

The agent writes the .dgm, lints it, and previews it as one self-contained
HTML file you can open, share or commit. It fetches the cinegram binary by
itself if you do not have one.

In this same session the skill is not loaded yet — first say:
  "Read $SKILL_MD and follow it."
EOF
