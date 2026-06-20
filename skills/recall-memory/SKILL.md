---
name: recall-memory
description: Use Recall MCP as local-first long-lived memory for projects, decisions, people, tools, research, and opportunities.
version: 1.0.0
author: Recall
license: MIT
platforms: [linux, macos, windows]
---

# Recall Memory

## Purpose

Recall MCP gives agents tool access to local-first Markdown + SQLite memory. This skill tells agents when and how to use those tools.

Use Recall before answering questions about previous projects, decisions, people, opportunities, durable research, tools, or phrases such as "remember", "earlier", "previous", and "what did we decide".

## Read flow

1. Call `recall_search` with a natural-language query.
2. Call `recall_get` for the top relevant ids.
3. Answer from retrieved Markdown, not vague model memory.
4. Mention id/path when useful for verification.
5. If no hit exists, say Recall had no matching memory.

## Write flow

1. Store only durable user/project-specific facts, decisions, research, opportunities, people, tools, and reusable lessons.
2. Call `recall_list_domains` before `recall_add` unless the correct domain is already known.
3. Use `lifecycle: evergreen` for durable facts, tools, people, project context, and decisions.
4. Use `lifecycle: expires` with `expires_on` for time-bound opportunities/news.
5. Use `recall_update` for corrections, added relationships, or changed metadata.
6. After write/update, report id and path briefly.

## Do not store

Never store secrets, raw chat logs, generic Q&A, or temporary task progress.

Skip credentials, tokens, one-off command output, commit SHAs, PR numbers, issue numbers, and facts likely to become stale within a week.

## Expected MCP tools

- `recall_search`
- `recall_get`
- `recall_add`
- `recall_update`
- `recall_graph`
- `recall_list_domains`
- `recall_reindex`

Optional current/future tools may include `recall_doctor`, `recall_list`, and `recall_use_project`.

## Fallback

If MCP is unavailable but CLI exists, use JSON CLI fallback:

```bash
recall search "query" --json
recall get <id> --json
recall domain list --json
```

Prefer MCP whenever available.
