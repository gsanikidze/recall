-- +goose Up
-- +goose StatementBegin
CREATE TABLE memories (
    id          TEXT PRIMARY KEY,
    path        TEXT NOT NULL,
    title       TEXT NOT NULL,
    domain      TEXT NOT NULL,
    project     TEXT NOT NULL DEFAULT '',
    source      TEXT NOT NULL DEFAULT '',
    lifecycle   TEXT NOT NULL,
    expires_on  TEXT NOT NULL DEFAULT '',
    created     TEXT NOT NULL,
    updated     TEXT NOT NULL,
    body        TEXT NOT NULL
);
-- +goose StatementEnd

-- +goose StatementBegin
CREATE INDEX idx_memories_domain  ON memories(domain);
-- +goose StatementEnd

-- +goose StatementBegin
CREATE INDEX idx_memories_project ON memories(project);
-- +goose StatementEnd

-- +goose StatementBegin
CREATE TABLE tags (
    memory_id TEXT NOT NULL,
    tag       TEXT NOT NULL
);
-- +goose StatementEnd

-- +goose StatementBegin
CREATE INDEX idx_tags_tag       ON tags(tag);
-- +goose StatementEnd

-- +goose StatementBegin
CREATE INDEX idx_tags_memory_id ON tags(memory_id);
-- +goose StatementEnd

-- +goose StatementBegin
CREATE TABLE links (
    memory_id TEXT NOT NULL,
    target_id TEXT NOT NULL
);
-- +goose StatementEnd

-- +goose StatementBegin
CREATE INDEX idx_links_memory_id ON links(memory_id);
-- +goose StatementEnd

-- +goose StatementBegin
CREATE VIRTUAL TABLE memories_fts USING fts5(
    id UNINDEXED,
    title,
    body
);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE IF EXISTS memories_fts;
-- +goose StatementEnd
-- +goose StatementBegin
DROP TABLE IF EXISTS links;
-- +goose StatementEnd
-- +goose StatementBegin
DROP TABLE IF EXISTS tags;
-- +goose StatementEnd
-- +goose StatementBegin
DROP TABLE IF EXISTS memories;
-- +goose StatementEnd
