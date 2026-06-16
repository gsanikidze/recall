# Recall agent instructions

Use this file as the bootstrap policy for any LLM agent that has access to Recall through MCP. MCP gives the agent memory tools; this policy tells the agent when and how to use them.

## Core rule

Use Recall before answering questions about previous projects, decisions, people, opportunities, durable research, tools, or phrases such as "remember", "earlier", "previous", "what did we decide", and "what do we know about".

MCP is the primary interface. Do not shell out to `recall` CLI when Recall MCP tools are available.

## Read flow

1. Call `recall_search` with a natural-language query and any useful filters.
2. Call `recall_get` for the top relevant ids before relying on details.
3. Answer from the retrieved Markdown content, not from vague memory.
4. Mention source path/id when it helps user verify or continue editing.
5. If search is empty, say Recall had no matching memory and continue from available context.

## Write flow

1. Store only durable user/project-specific facts, decisions, research, opportunities, people, tools, and reusable lessons.
2. Call `recall_list_domains` before `recall_add` unless the correct domain is already known from current context.
3. Use `lifecycle: evergreen` for durable tools, people, preferences, project facts, and decisions.
4. Use `lifecycle: expires` plus `expires_on` for time-bound opportunities, news, offers, or temporary facts.
5. Prefer concise Markdown bodies with context, source, and why the fact matters.
6. After writing or updating, report the memory id and vault path briefly.

## Do not store

Never store secrets, raw chat logs, generic Q&A, or temporary task progress.

Also skip:

- API keys, passwords, tokens, private credentials.
- One-off command output that will be stale soon.
- Completed task logs such as commit SHAs, PR numbers, issue numbers, or "phase done" notes.
- Generic facts easily found on the web.
- User intent that was not confirmed as durable.

## Domain hints

| Domain | Use for |
|---|---|
| `tools` | Stable tool configs, schedulers, integrations, agent setup, operational notes. |
| `projects` | Durable project facts, architecture, repo paths, product context. |
| `decisions` | Choices made, rationale, rejected alternatives. |
| `people` | Durable contact/user/person facts and preferences. |
| `research` | Findings with sources, comparisons, technical or market research. |
| `opportunities` | Business, money, hiring, grant, or market opportunities. |
| `inbox` | Useful but unsorted notes needing later cleanup. |

If domain names differ, trust `recall_list_domains`.

## Tool mapping

Expected MCP tools:

- `recall_search` — lightweight search over title/body/metadata.
- `recall_get` — full Markdown content by id.
- `recall_add` — create durable memory.
- `recall_update` — edit fields/relationships.
- `recall_list_domains` — list domain folders and descriptions.
- `recall_reindex` — rebuild SQLite index after manual vault edits.
- Optional: `recall_doctor`, `recall_list`, `recall_use_project` if exposed by current server.

## Reporting pattern

After write:

```text
Recall updated.
- ID: <id>
- Path: <vault/path.md>
- Stored: <one-line summary>
```

After read:

```text
Recall found: <title> (<id>, <path>)
```

Use compact reporting. Do not dump long Markdown unless user asks.

## Non-MCP fallback

If MCP is not available but CLI is installed, use CLI only as fallback:

```bash
recall search "query" --json
recall get <id> --json
recall domain list --json
```

Prefer environment-scoped projects for agents:

```bash
RECALL_PROJECT=/path/to/brain recall mcp
```
