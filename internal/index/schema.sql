-- schema.sql defines the tables for sqlc code generation only.
-- The authoritative, runtime schema lives in migrations/ (applied by goose) and
-- creates memories_fts as an FTS5 virtual table. For sqlc, memories_fts is
-- declared as a normal table with matching writable columns so static FTS
-- maintenance queries can be generated. Dynamic FTS search remains hand-written
-- in search.go. Keep table/column definitions here in sync with migrations/.

CREATE TABLE memories (
    id          TEXT PRIMARY KEY,
    path        TEXT NOT NULL UNIQUE,
    title       TEXT NOT NULL,
    domain      TEXT NOT NULL,
    project     TEXT NOT NULL DEFAULT '',
    source      TEXT NOT NULL DEFAULT '',
    lifecycle   TEXT NOT NULL,
    expires_on  TEXT NOT NULL DEFAULT '',
    created     TEXT NOT NULL,
    updated     TEXT NOT NULL,
    importance  INTEGER NOT NULL DEFAULT 3,
    body        TEXT NOT NULL
);

CREATE TABLE tags (
    memory_id TEXT NOT NULL REFERENCES memories(id) ON DELETE CASCADE,
    tag       TEXT NOT NULL,
    UNIQUE(memory_id, tag)
);

CREATE TABLE links (
    memory_id TEXT NOT NULL REFERENCES memories(id) ON DELETE CASCADE,
    target_id TEXT NOT NULL,
    UNIQUE(memory_id, target_id)
);

CREATE TABLE memory_relationships (
    source_id TEXT NOT NULL REFERENCES memories(id) ON DELETE CASCADE,
    target_id TEXT NOT NULL,
    type      TEXT NOT NULL,
    note      TEXT NOT NULL DEFAULT '',
    PRIMARY KEY (source_id, target_id, type)
);

CREATE TABLE memories_fts (
    id    TEXT PRIMARY KEY,
    title TEXT NOT NULL,
    body  TEXT NOT NULL
);
