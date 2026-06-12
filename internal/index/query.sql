-- query.sql holds static, type-safe queries sqlc generates Go for.
-- Dynamic full-text search (combinable optional filters + FTS5 MATCH) is
-- hand-written in search.go, which is why no search query appears here.

-- name: UpsertMemory :exec
INSERT INTO memories (
    id, path, title, domain, project, source, lifecycle, expires_on, created, updated, importance, body
) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
ON CONFLICT(id) DO UPDATE SET
    path       = excluded.path,
    title      = excluded.title,
    domain     = excluded.domain,
    project    = excluded.project,
    source     = excluded.source,
    lifecycle  = excluded.lifecycle,
    expires_on = excluded.expires_on,
    created    = excluded.created,
    updated    = excluded.updated,
    importance = excluded.importance,
    body       = excluded.body;

-- name: GetMemory :one
SELECT * FROM memories WHERE id = ?;

-- name: DeleteMemory :exec
DELETE FROM memories WHERE id = ?;

-- name: ListMemoryIDs :many
SELECT id FROM memories;

-- name: DeleteTagsForMemory :exec
DELETE FROM tags WHERE memory_id = ?;

-- name: InsertTag :exec
INSERT INTO tags (memory_id, tag) VALUES (?, ?);

-- name: GetTagsForMemory :many
SELECT tag FROM tags WHERE memory_id = ? ORDER BY tag;

-- name: DeleteLinksForMemory :exec
DELETE FROM links WHERE memory_id = ?;

-- name: InsertLink :exec
INSERT INTO links (memory_id, target_id) VALUES (?, ?);

-- name: GetLinksForMemory :many
SELECT target_id FROM links WHERE memory_id = ? ORDER BY target_id;

-- name: DeleteRelationshipsForMemory :exec
DELETE FROM memory_relationships WHERE source_id = ?;

-- name: DeleteRelationshipsTouchingMemory :exec
DELETE FROM memory_relationships WHERE source_id = ? OR target_id = ?;

-- name: InsertRelationship :exec
INSERT INTO memory_relationships (source_id, target_id, type, note) VALUES (?, ?, ?, ?);

-- name: GetRelationshipsForMemory :many
SELECT target_id, type, note FROM memory_relationships WHERE source_id = ? ORDER BY target_id, type;

-- name: DeleteFTSForMemory :exec
DELETE FROM memories_fts WHERE id = ?;

-- name: InsertFTSForMemory :exec
INSERT INTO memories_fts (id, title, body) VALUES (?, ?, ?);

-- name: UpsertEmbedding :exec
INSERT INTO memory_embeddings (memory_id, provider, model, dim, vector, content_hash, embedded_at)
VALUES (?, ?, ?, ?, ?, ?, datetime('now'))
ON CONFLICT(memory_id, provider, model) DO UPDATE SET
    dim = excluded.dim,
    vector = excluded.vector,
    content_hash = excluded.content_hash,
    embedded_at = excluded.embedded_at;

-- name: GetEmbeddingForMemory :one
SELECT memory_id, provider, model, dim, vector, content_hash, embedded_at
FROM memory_embeddings
WHERE memory_id = ? AND provider = ? AND model = ?;

-- name: ListEmbeddingsForModel :many
SELECT memory_id, provider, model, dim, vector, content_hash, embedded_at
FROM memory_embeddings
WHERE provider = ? AND model = ?
ORDER BY memory_id;

-- name: DeleteEmbeddingForMemory :exec
DELETE FROM memory_embeddings WHERE memory_id = ? AND provider = ? AND model = ?;
