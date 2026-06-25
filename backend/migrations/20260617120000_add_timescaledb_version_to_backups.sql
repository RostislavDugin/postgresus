-- +goose Up
ALTER TABLE logical_backups
    ADD COLUMN timescaledb_version TEXT NOT NULL DEFAULT '';

-- +goose Down
ALTER TABLE logical_backups
    DROP COLUMN timescaledb_version;
