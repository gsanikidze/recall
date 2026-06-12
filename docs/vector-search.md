# Vector search

Recall supports local-first semantic search with cached embeddings in SQLite.

## Model/provider

Default provider is Ollama:

```text
provider: ollama
model: nomic-embed-text
base URL: http://127.0.0.1:11434
```

Override Ollama URL with:

```bash
export RECALL_OLLAMA_URL=http://127.0.0.1:11434
```

No cloud provider is used by default.

## Setup

Install and start Ollama, then pull the embedding model:

```bash
ollama pull nomic-embed-text
```

## Populate embedding cache

Embedding rows are a rebuildable SQLite cache. Markdown remains source of truth.

```bash
recall embed --provider ollama --model nomic-embed-text
```

JSON output:

```bash
recall embed --provider ollama --model nomic-embed-text --json
```

Example:

```json
{
  "provider": "ollama",
  "model": "nomic-embed-text",
  "embedded": 8,
  "skipped": 0,
  "failed": 0
}
```

`skipped` means an embedding already exists for the same provider, model, and content hash. It does not mean the DB is empty.

Force rebuild:

```bash
recall embed --provider ollama --model nomic-embed-text --force
```

## Search modes

Keyword search remains default:

```bash
recall search "phone sync"
```

Semantic-only search:

```bash
recall search "phone sync" --semantic --provider ollama --model nomic-embed-text
```

Hybrid search combines keyword + semantic scores:

```bash
recall search "phone sync" --hybrid --provider ollama --model nomic-embed-text
```

JSON exposes score fields:

```bash
recall search "phone sync" --hybrid --json
```

Fields:

```text
score: final lower-is-better rank score
keyword_score: keyword contribution when present
semantic_score: cosine similarity, higher means more semantically similar
```

## Web UI

The web UI includes:

```text
Keyword | Semantic | Hybrid
```

Semantic and Hybrid modes call the API with:

```text
mode=semantic|hybrid
provider=ollama
model=nomic-embed-text
```

Result cards show semantic/keyword badges when scores are present.

## API

```bash
curl 'http://127.0.0.1:8888/api/memories?q=phone%20sync&mode=hybrid&provider=ollama&model=nomic-embed-text'
```

`mode` values:

```text
keyword
semantic
hybrid
```

## MCP

`recall_search` accepts optional semantic fields:

```json
{
  "query": "phone sync",
  "mode": "hybrid",
  "provider": "ollama",
  "model": "nomic-embed-text"
}
```

Compatibility rule: default MCP search mode remains `keyword`.

## Troubleshooting

### `embedded: 0, skipped: N`

Embeddings already exist and content did not change. Use `--force` to rebuild.

### Semantic search returns empty results

Check:

```bash
recall embed --provider ollama --model nomic-embed-text --json
recall search "test query" --semantic --provider ollama --model nomic-embed-text --json
```

Also verify Ollama model exists:

```bash
ollama list
```

### Ollama unavailable

Start Ollama and retry. If Ollama listens somewhere else, set:

```bash
export RECALL_OLLAMA_URL=http://host:11434
```

or pass:

```bash
recall embed --base-url http://host:11434
```
