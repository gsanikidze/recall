# Recall agent setup

This repository contains Recall, a local-first memory CLI/web UI/MCP server for humans and LLM agents. Use this file as the first thing an agent reads when setting up or working on Recall.

## What Recall provides

- Markdown vault as source of truth.
- SQLite index for keyword/metadata search.
- MCP stdio server so agents can use memory tools directly.
- Hermes skill template so agents know when/how to use Recall.
- Copyable `AGENTS.md` template for other repos.

## Build Recall

Fast Go-only build:

```bash
make build-nui
./bin/recall version
```

Full build with embedded UI:

```bash
make build
./bin/recall version
```

Install or refresh local binary/symlink as needed:

```bash
go build -o bin/recall .
ln -sf "$PWD/bin/recall" /usr/local/bin/recall
```

## Initialize a project

```bash
recall init --path ~/brain
```

For agent processes, prefer explicit project env:

```bash
export RECALL_PROJECT="$HOME/brain"
```

## Hermes MCP setup

Add Recall MCP to Hermes default profile:

```bash
hermes mcp add recall --command /usr/local/bin/recall --args mcp --env RECALL_PROJECT=$HOME/brain
hermes mcp test recall
hermes mcp configure recall
```

Expected tools include:

```text
recall_add
recall_get
recall_list_domains
recall_reindex
recall_search
recall_update
recall_use_project
```

Running Hermes sessions need:

```text
/reload-mcp
```

New sessions see MCP tools automatically.

## Hermes skill setup

Install the Recall memory skill from this repo:

```bash
hermes skills install ./skills/recall-memory/SKILL.md --name recall-memory
hermes skills list | grep recall-memory
```

If local path install is not supported by the active Hermes version, copy the skill directory into the profile skills folder:

```bash
mkdir -p ~/.hermes/skills/memory/recall-memory
cp skills/recall-memory/SKILL.md ~/.hermes/skills/memory/recall-memory/SKILL.md
```

Running Hermes sessions need:

```text
/reload-skills
```

New sessions see the skill automatically.

## Agent behavior policy

Read and follow:

- `docs/agent-instructions.md` — full policy for LLM agents using Recall.
- `docs/llm-setup.md` — setup guide for MCP + skills.
- `docs/templates/AGENTS.md` — template to copy into other repos.
- `skills/recall-memory/SKILL.md` — Hermes skill package content.

Core rule: use Recall before answering questions about previous projects, decisions, people, opportunities, durable research, tools, or "remember/earlier/previous/what did we decide" prompts.

Never store secrets, raw chat logs, generic Q&A, or temporary task progress.

## Verification

```bash
go test ./... -run TestAgentInstructionArtifactsExist -count=1
go test ./...
hermes mcp test recall
```

If docs or setup templates change, update `agent_docs_test.go` so required bootstrapping text stays covered.
