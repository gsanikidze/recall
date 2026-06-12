-- +goose Up
CREATE TABLE IF NOT EXISTS memory_embeddings (
    memory_id TEXT NOT NULL,
    provider TEXT NOT NULL,
    model TEXT NOT NULL,
    dim INTEGER NOT NULL,
    vector BLOB NOT NULL,
    content_hash TEXT NOT NULL,
    embedded_at TEXT NOT NULL DEFAULT (datetime('now')),
    PRIMARY KEY (memory_id, provider, model),
    FOREIGN KEY (memory_id) REFERENCES memories(id) ON DELETE CASCADE
);

CREATE INDEX IF NOT EXISTS idx_memory_embeddings_model
ON memory_embeddings(provider, model, dim);

CREATE INDEX IF NOT EXISTS idx_memory_embeddings_hash
ON memory_embeddings(content_hash);

-- +goose Down
DROP INDEX IF EXISTS idx_memory_embeddings_hash;
DROP INDEX IF EXISTS idx_memory_embeddings_model;
DROP TABLE IF EXISTS memory_embeddings;
