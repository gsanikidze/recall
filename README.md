# Recall

Recall is a local-first memory CLI and web UI for humans and AI agents. It stores durable facts as Markdown files in a vault, indexes them in local SQLite, and exposes the same memory through CLI commands, an MCP server, and a loopback-only web UI/API.

## Status

Recall is early-stage. Storage is local device only: no cloud sync, no hosted service, no remote database.

## What it stores

A Recall project contains:

- `vault/` — source of truth. Markdown memories grouped by domain folders.
- `db/recall.sqlite` — rebuildable SQLite search/index database.
- `vault/README.md` and per-domain `README.md` files — human/agent guidance for what belongs where.

Default domains include `tools`, `inbox`, `people`, `projects`, `decisions`, `research`, and `goals`. Add custom domains with `recall domain add` or from the UI.

## Install / build

Prerequisites:

- Go matching `go.mod`.
- Node.js 22+ and npm for UI builds.
- Git.

Build full binary with embedded UI:

```bash
make build
./bin/recall version
```

Build CLI/API without embedded UI assets:

```bash
make build-nui
./bin/recall version
```

Difference:

- `make build` runs `npm --prefix ui ci`, builds React assets, then builds Go with `-tags ui`. `recall ui` serves embedded UI.
- `make build-nui` builds only Go without Node. `recall ui` still starts API, but SPA serving returns `503` with “recall UI not built.”

## Initialize workspace

```bash
recall init --path ~/brain
```

Config path is OS-dependent. On Linux it is usually:

```text
~/.config/recall/config.json
```

Useful environment overrides:

- `RECALL_PROJECT=/path/to/project` — project root containing `vault/` and `db/`.
- `RECALL_HOME=/path/to/config-dir` — config directory used by Recall.

Environment variables override saved config and are useful for tests, temporary projects, and agent sandboxes.

## CLI examples

Add memory:

```bash
recall add --title "SQLite WAL note" --domain tools --tags sqlite,go --importance 4 --body "Use WAL plus busy_timeout for local concurrent reads/writes."
```

Add typed relationships as graph edges:

```bash
recall add \
  --title "Hermes uses Recall MCP" \
  --domain tools \
  --relationships '[{"target_id":"01PROJECT...","type":"uses_tool","note":"stdio MCP"}]' \
  --body "Hermes stores durable memory in Recall over MCP."
```

Pipe body from stdin:

```bash
printf 'Decision rationale here\n' | recall add --title "Use local-only storage" --domain decisions --project recall
```

Search:

```bash
recall search sqlite --domain tools --limit 10
recall search --tag go --project recall --json
```

Populate local embedding cache with Ollama:

```bash
ollama pull nomic-embed-text
recall embed --provider ollama --model nomic-embed-text
```

Semantic and hybrid search:

```bash
recall search "phone sync" --semantic --provider ollama --model nomic-embed-text
recall search "phone sync" --hybrid --provider ollama --model nomic-embed-text --json
```

`embedded: 0, skipped: N` means vectors already exist for unchanged memories. Use `--force` to rebuild.

Importance is an integer from 1–5. `3` is default durable memory; `5` is critical operating context such as stable paths, preferences, and integration configs. Keyword search ranking blends full-text relevance, recency, and importance. Semantic and hybrid modes use the SQLite embedding cache.

Relationships are typed directed edges from one memory to another. Supported types: `related_to`, `about_project`, `uses_tool`, `depends_on`, `decided_by`, `supersedes`, `contradicts`, `references_person`. Markdown frontmatter is source of truth; SQLite stores `memory_relationships` as rebuildable graph index rows.

Get memory:

```bash
recall get 01ABCDEF
recall get 01ABCDEF --json
```

Delete memory:

```bash
recall delete 01ABCDEF --yes
```

Manage domains:

```bash
recall domain list
recall domain add personal-notes --desc "Private notes and observations."
```

Rebuild index from Markdown vault:

```bash
recall reindex
```

Check workspace health:

```bash
recall doctor
```

Open TUI:

```bash
recall
```

Start web UI/API:

```bash
recall ui --port 8888
recall ui --port 8888 --no-browser
```

## MCP setup

Run MCP server over stdio:

```bash
recall mcp
```

Example MCP server config shape for an agent:

```json
{
  "mcpServers": {
    "recall": {
      "command": "recall",
      "args": ["mcp"],
      "env": {
        "RECALL_PROJECT": "/home/you/brain"
      }
    }
  }
}
```

Use absolute paths when configuring long-lived agent processes.

## Web UI / API development

Run API/UI server from Go:

```bash
make build
recall ui --no-browser --port 8888
```

Run Vite dev server for frontend work:

```bash
cd ui
npm ci
npm run dev
```

The REST API lives under `/api/` and includes:

- `GET /api/domains`
- `POST /api/domains`
- `GET /api/memories` (`mode=keyword|semantic|hybrid`, `provider`, and `model` are supported for vector search)
- `GET /api/memories/:id`
- `POST /api/memories`
- `PUT /api/memories/:id`
- `DELETE /api/memories/:id`
- `POST /api/reindex`

## Checks

Full local check:

```bash
make check
```

Individual checks:

```bash
make fmt-check
make tidy-check
make vet
make test
make race
make cover
make generate-check
make install-ui
make lint-ui
make test-ui
make build-ui
make audit-ui
make build-nui
make build
make test-ui-tag
```

Generated sqlc code freshness:

```bash
make generate-check
```

`make generate-check` runs sqlc `v1.30.0` and fails if generated files under `internal/index/db/` differ from committed output.

## Local security model

Recall is local-first and assumes local device trust, not hostile multi-user hosting.

- The web API is unauthenticated and intended for loopback use only.
- `recall ui` listens on `localhost`.
- API middleware rejects non-loopback hostnames to reduce DNS-rebinding risk.
- CORS allowlist is limited to local Vite dev origins (`localhost:5173`, `127.0.0.1:5173`).
- Do not expose Recall’s API port to LAN/WAN or run it behind public reverse proxies without adding authentication.

## More docs

- [Vector search](docs/vector-search.md)
- [Development guide](docs/development.md)
- [Code quality roadmap](docs/plans/2026-06-08-code-quality-roadmap.md)
