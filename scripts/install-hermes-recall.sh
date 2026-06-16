#!/usr/bin/env bash
set -euo pipefail

script_dir="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
repo_root="$(cd "$script_dir/.." && pwd)"

RECALL_PROJECT="${RECALL_PROJECT:-$HOME/brain}"
HERMES_HOME="${HERMES_HOME:-$HOME/.hermes}"
RECALL_BIN="${RECALL_BIN:-/usr/local/bin/recall}"
COPY_AGENT_TEMPLATE_TO="${COPY_AGENT_TEMPLATE_TO:-}"

skill_src="$repo_root/skills/recall-memory/SKILL.md"
skill_dest="$HERMES_HOME/skills/memory/recall-memory/SKILL.md"
agent_template_src="$repo_root/docs/templates/AGENTS.md"

printf 'Recall repo: %s\n' "$repo_root"
printf 'Recall project: %s\n' "$RECALL_PROJECT"
printf 'Hermes home: %s\n' "$HERMES_HOME"

if ! command -v hermes >/dev/null 2>&1; then
  printf 'WARN: hermes command not found; skipping hermes mcp add recall.\n' >&2
else
  # Keep exact command visible for docs/tests:
  # hermes mcp add recall --command /usr/local/bin/recall --args mcp --env RECALL_PROJECT=$HOME/brain
  if ! hermes mcp add recall --command "$RECALL_BIN" --args mcp --env "RECALL_PROJECT=$RECALL_PROJECT"; then
    printf 'WARN: hermes mcp add recall failed, possibly because server already exists. Continuing.\n' >&2
  fi
  hermes mcp test recall || printf 'WARN: hermes mcp test recall failed. Check binary path and RECALL_PROJECT.\n' >&2
fi

if [[ ! -f "$skill_src" ]]; then
  printf 'ERROR: missing skill source: %s\n' "$skill_src" >&2
  exit 1
fi

mkdir -p "$(dirname "$skill_dest")"
cp "$skill_src" "$skill_dest"
printf 'Installed recall-memory skill: %s\n' "$skill_dest"

if [[ -n "$COPY_AGENT_TEMPLATE_TO" ]]; then
  if [[ ! -d "$COPY_AGENT_TEMPLATE_TO" ]]; then
    printf 'ERROR: COPY_AGENT_TEMPLATE_TO is not a directory: %s\n' "$COPY_AGENT_TEMPLATE_TO" >&2
    exit 1
  fi
  cp "$agent_template_src" "$COPY_AGENT_TEMPLATE_TO/AGENTS.md"
  printf 'Copied agent template: %s\n' "$COPY_AGENT_TEMPLATE_TO/AGENTS.md"
fi

cat <<'MSG'

Next steps for running Hermes sessions:
/reload-mcp
/reload-skills

New Hermes sessions see Recall MCP and recall-memory automatically.
MSG
