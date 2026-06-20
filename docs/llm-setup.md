# LLM setup for Recall MCP and skills

This guide gives LLM agents enough information to install and verify Recall as long-lived local memory.

Product stance: Recall is agent-written, human-readable memory. Use CLI/MCP/API for writes; use the web UI mainly to browse, search, view metadata, and audit what agents stored.

## Files in this repo

| Path | Purpose |
|---|---|
| `AGENTS.md` | Root bootstrap for agents working inside this repo. |
| `docs/agent-instructions.md` | Runtime policy: when/how agents should use Recall. |
| `docs/templates/AGENTS.md` | Copy into other repos so coding agents know to use Recall. |
| `docs/examples/hermes-mcp-recall.yaml` | Copyable Hermes `config.yaml` MCP snippet. |
| `docs/examples/mcp-recall.json` | Generic MCP client config snippet. |
| `skills/recall-memory/SKILL.md` | Hermes skill package for Recall memory behavior. |
| `scripts/install-hermes-recall.sh` | Installs/copies Hermes Recall MCP and skill setup. |
| `scripts/verify-agent-setup.sh` | Verifies docs, examples, skill, MCP, and Hermes visibility when available. |
| `README.md` | User-facing build, CLI, MCP, UI, and check commands. |

## Prerequisites

- Recall binary built from this repo or installed as `recall`.
- Hermes Agent installed if configuring Hermes.
- A Recall project directory such as `$HOME/brain`.

Build fast Go-only binary:

```bash
make build-nui
./bin/recall version
```

Optional full UI build:

```bash
make build
./bin/recall version
```

Create local command if needed:

```bash
ln -sf "$PWD/bin/recall" /usr/local/bin/recall
```

## Create or choose Recall project

Initialize a local project:

```bash
recall init --path "$HOME/brain"
```

For agents and services, set explicit project env:

```bash
export RECALL_PROJECT="$HOME/brain"
```

The project root contains:

```text
vault/              Markdown source of truth
db/recall.sqlite   Rebuildable SQLite index
```

## Hermes MCP setup

Use absolute command paths for long-lived Hermes/gateway processes.

Fast path from this repo:

```bash
scripts/install-hermes-recall.sh
```

Manual CLI setup:

```bash
hermes mcp add recall --command /usr/local/bin/recall --args mcp --env RECALL_PROJECT=$HOME/brain
hermes mcp test recall
hermes mcp configure recall
```

Copyable config examples:

- `docs/examples/hermes-mcp-recall.yaml` — paste under Hermes `config.yaml`.
- `docs/examples/mcp-recall.json` — generic MCP client config.

Expected `hermes mcp test recall` result:

```text
✓ Connected
✓ Tools discovered
```

Expected tools:

```text
recall_add
recall_get
recall_graph
recall_list_domains
recall_reindex
recall_search
recall_update
recall_use_project
```

For a running Hermes chat, reload MCP:

```text
/reload-mcp
```

Otherwise start a new session or restart the gateway.

## Hermes skill setup

Install the skill from this repo when supported by the current Hermes CLI:

```bash
hermes skills install ./skills/recall-memory/SKILL.md --name recall-memory
hermes skills list | grep recall-memory
```

Fallback copy method:

```bash
mkdir -p ~/.hermes/skills/memory/recall-memory
cp skills/recall-memory/SKILL.md ~/.hermes/skills/memory/recall-memory/SKILL.md
hermes skills list | grep recall-memory
```

For a running Hermes chat, reload skills:

```text
/reload-skills
```

Agents can explicitly load the skill in-session:

```text
/skill recall-memory
```

## Agent instruction setup for non-Hermes agents

Copy the template into any repo where coding agents should use Recall:

```bash
cp docs/templates/AGENTS.md /path/to/target-repo/AGENTS.md
```

Then configure that agent with Recall MCP using its MCP config format. Equivalent MCP server shape:

```json
{
  "mcpServers": {
    "recall": {
      "command": "/usr/local/bin/recall",
      "args": ["mcp"],
      "env": {
        "RECALL_PROJECT": "/home/you/brain"
      }
    }
  }
}
```

## Runtime policy summary

Use `docs/agent-instructions.md` as source policy. Minimum behavior:

1. Use Recall before answering questions about previous projects, decisions, people, opportunities, durable research, tools, or "remember/earlier/previous/what did we decide" prompts.
2. Read flow: `recall_search` then `recall_get`.
3. Write flow: `recall_list_domains` then `recall_add` or `recall_update`.
4. Never store secrets, raw chat logs, generic Q&A, or temporary task progress.
5. Report id + vault path after writes.

## Verification checklist

Run:

```bash
go test ./... -run TestAgentInstructionArtifactsExist -count=1
go test ./...
hermes mcp test recall
hermes skills list | grep recall-memory
scripts/verify-agent-setup.sh
```

Check manually:

- `docs/agent-instructions.md` has read/write policy.
- `docs/templates/AGENTS.md` has install/enable notes.
- `skills/recall-memory/SKILL.md` has valid frontmatter.
- Root `AGENTS.md` points agents to all setup files.
