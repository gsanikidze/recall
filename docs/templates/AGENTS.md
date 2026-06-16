# Agent memory policy

This project uses Recall as local-first long-lived memory for agents.

## Install / enable

If Hermes Agent is available, verify Recall MCP and skill setup:

```bash
hermes mcp test recall
hermes skills list | grep recall-memory
```

If not configured yet, from the Recall repo use:

```bash
hermes mcp add recall --command /usr/local/bin/recall --args mcp --env RECALL_PROJECT=$HOME/brain
hermes mcp test recall
hermes mcp configure recall
hermes skills install ./skills/recall-memory/SKILL.md --name recall-memory
```

In running Hermes sessions, reload changed integrations:

```text
/reload-mcp
/reload-skills
```

For non-Hermes agents, configure equivalent MCP server:

```json
{
  "mcpServers": {
    "recall": {
      "command": "/usr/local/bin/recall",
      "args": ["mcp"],
      "env": {"RECALL_PROJECT": "/home/you/brain"}
    }
  }
}
```

## Recall MCP

If Recall MCP tools are available, use them as the primary memory interface. MCP provides tool access; this policy provides behavior.

Use Recall before answering questions about:

- previous or earlier work
- project decisions
- people and preferences
- opportunities
- durable research
- tool/scheduler/integration context
- anything phrased as "remember", "what did we decide", or "what do we know about"

## Read flow

1. `recall_search` with a natural query.
2. `recall_get` for relevant hits.
3. Answer from retrieved content.
4. Include id/path only when useful.

## Write flow

1. Save only durable, user/project-specific knowledge.
2. Call `recall_list_domains` before adding unless domain is obvious.
3. Use `recall_add` for new durable notes.
4. Use `recall_update` for corrections or relationships.
5. Use `evergreen` for lasting facts and `expires` for time-bound notes.
6. Report id/path after writes.

## Never store

Never store secrets, raw chat logs, generic Q&A, or temporary task progress.

Do not store credentials, tokens, one-off outputs, commit SHAs, PR numbers, issue numbers, or short-lived task logs.

## Fallback

If MCP is unavailable but the CLI exists, prefer JSON commands:

```bash
recall search "query" --json
recall get <id> --json
recall domain list --json
```
