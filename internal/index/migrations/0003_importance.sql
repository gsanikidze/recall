-- +goose Up
ALTER TABLE memories ADD COLUMN importance INTEGER NOT NULL DEFAULT 3 CHECK (importance BETWEEN 1 AND 5);

-- +goose Down
-- SQLite cannot drop a column without rebuilding the table; keep importance on downgrade.
