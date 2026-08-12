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
    ;;
  cursor)
    echo "Installing the cinegram skill for Cursor into $(pwd):"
    fetch SKILL.md .cursor/skills/cinegram/SKILL.md
    fetch references/language.md .cursor/skills/cinegram/references/language.md
    fetch cinegram.mdc .cursor/rules/cinegram.mdc
    ;;
  *)
    echo "usage: install.sh [claude|cursor]" >&2
    exit 2
    ;;
esac

echo "Done. Start a new session and ask: 'animate this mermaid diagram: ...'"
echo "(In this same session, ask the agent to read the installed SKILL.md and follow it.)"
