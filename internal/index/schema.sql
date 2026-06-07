-- schema.sql defines the relational tables for sqlc code generation only.
-- The authoritative, runtime schema lives in migrations/ (applied by goose) and
-- additionally creates the memories_fts FTS5 virtual table, which sqlc does not
-- need to know about (full-text search is hand-written in search.go). Keep the
-- table/column definitions here in sync with migrations/0001_init.sql.

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

CREATE TABLE tags (
    memory_id TEXT NOT NULL,
    tag       TEXT NOT NULL
);

CREATE TABLE links (
    memory_id TEXT NOT NULL,
    target_id TEXT NOT NULL
);
