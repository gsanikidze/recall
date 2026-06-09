-- +goose Up
CREATE TABLE memory_relationships (
    source_id TEXT NOT NULL REFERENCES memories(id) ON DELETE CASCADE,
    target_id TEXT NOT NULL,
    type      TEXT NOT NULL CHECK (type IN (
        'related_to',
        'about_project',
        'uses_tool',
        'depends_on',
        'decided_by',
        'supersedes',
        'contradicts',
        'references_person'
    )),
    note      TEXT NOT NULL DEFAULT '' CHECK (length(note) <= 300),
    CHECK (source_id <> target_id),
    PRIMARY KEY (source_id, target_id, type)
);

CREATE INDEX memory_relationships_target_idx ON memory_relationships(target_id);
CREATE INDEX memory_relationships_type_idx ON memory_relationships(type);

-- +goose Down
DROP INDEX IF EXISTS memory_relationships_type_idx;
DROP INDEX IF EXISTS memory_relationships_target_idx;
DROP TABLE IF EXISTS memory_relationships;
