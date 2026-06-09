-- +goose Up
-- +goose StatementBegin
CREATE UNIQUE INDEX idx_memories_path_unique ON memories(path);
-- +goose StatementEnd

-- +goose StatementBegin
CREATE TABLE tags_new (
    memory_id TEXT NOT NULL REFERENCES memories(id) ON DELETE CASCADE,
    tag       TEXT NOT NULL,
    UNIQUE(memory_id, tag)
);
INSERT OR IGNORE INTO tags_new (memory_id, tag)
SELECT tags.memory_id, tags.tag
FROM tags
JOIN memories ON memories.id = tags.memory_id;
DROP TABLE tags;
ALTER TABLE tags_new RENAME TO tags;
CREATE INDEX idx_tags_tag       ON tags(tag);
CREATE INDEX idx_tags_memory_id ON tags(memory_id);
-- +goose StatementEnd

-- +goose StatementBegin
CREATE TABLE links_new (
    memory_id TEXT NOT NULL REFERENCES memories(id) ON DELETE CASCADE,
    target_id TEXT NOT NULL,
    UNIQUE(memory_id, target_id)
);
INSERT OR IGNORE INTO links_new (memory_id, target_id)
SELECT links.memory_id, links.target_id
FROM links
JOIN memories ON memories.id = links.memory_id;
DROP TABLE links;
ALTER TABLE links_new RENAME TO links;
CREATE INDEX idx_links_memory_id ON links(memory_id);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP INDEX IF EXISTS idx_memories_path_unique;
-- +goose StatementEnd

-- +goose StatementBegin
CREATE TABLE tags_old (
    memory_id TEXT NOT NULL,
    tag       TEXT NOT NULL
);
INSERT INTO tags_old (memory_id, tag) SELECT memory_id, tag FROM tags;
DROP TABLE tags;
ALTER TABLE tags_old RENAME TO tags;
CREATE INDEX idx_tags_tag       ON tags(tag);
CREATE INDEX idx_tags_memory_id ON tags(memory_id);
-- +goose StatementEnd

-- +goose StatementBegin
CREATE TABLE links_old (
    memory_id TEXT NOT NULL,
    target_id TEXT NOT NULL
);
INSERT INTO links_old (memory_id, target_id) SELECT memory_id, target_id FROM links;
DROP TABLE links;
ALTER TABLE links_old RENAME TO links;
CREATE INDEX idx_links_memory_id ON links(memory_id);
-- +goose StatementEnd
