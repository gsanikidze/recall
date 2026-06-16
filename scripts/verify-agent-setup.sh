#!/usr/bin/env bash
set -euo pipefail

script_dir="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
repo_root="$(cd "$script_dir/.." && pwd)"
cd "$repo_root"

required_files=(
  "AGENTS.md"
  "docs/llm-setup.md"
  "docs/agent-instructions.md"
  "docs/templates/AGENTS.md"
  "docs/examples/hermes-mcp-recall.yaml"
  "docs/examples/mcp-recall.json"
  "skills/recall-memory/SKILL.md"
  "scripts/install-hermes-recall.sh"
  "scripts/verify-agent-setup.sh"
)

for path in "${required_files[@]}"; do
  if [[ ! -f "$path" ]]; then
    printf 'ERROR: missing required file: %s\n' "$path" >&2
    exit 1
  fi
done

require_text() {
  local path="$1"
  local text="$2"
  if ! grep -Fq "$text" "$path"; then
    printf 'ERROR: %s missing required text: %s\n' "$path" "$text" >&2
    exit 1
  fi
}

require_text "docs/examples/hermes-mcp-recall.yaml" "mcp_servers:"
require_text "docs/examples/hermes-mcp-recall.yaml" "RECALL_PROJECT: /home/you/brain"
require_text "docs/examples/mcp-recall.json" '"mcpServers"'
require_text "docs/examples/mcp-recall.json" '"args": ["mcp"]'
require_text "scripts/install-hermes-recall.sh" "hermes mcp add recall"
require_text "scripts/install-hermes-recall.sh" "recall-memory"
require_text "docs/llm-setup.md" "scripts/install-hermes-recall.sh"
require_text "docs/llm-setup.md" "scripts/verify-agent-setup.sh"

# Keep exact test name visible for agent_docs_test.go coverage.
go test ./... -run TestAgentInstructionArtifactsExist -count=1

if command -v hermes >/dev/null 2>&1; then
  hermes mcp test recall
  hermes skills list | grep -F "recall-memory"
else
  printf 'WARN: hermes command not found; skipped live Hermes checks.\n' >&2
fi

printf 'Recall agent setup verification passed.\n'
