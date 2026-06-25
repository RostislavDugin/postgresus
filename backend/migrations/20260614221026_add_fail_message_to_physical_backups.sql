-- +goose Up
ALTER TABLE physical_full_backups
    ADD COLUMN fail_message TEXT;

ALTER TABLE physical_incremental_backups
    ADD COLUMN fail_message TEXT;

-- +goose Down
ALTER TABLE physical_full_backups
    DROP COLUMN fail_message;

ALTER TABLE physical_incremental_backups
    DROP COLUMN fail_message;
