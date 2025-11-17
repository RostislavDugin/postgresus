-- +goose Up
-- +goose StatementBegin

ALTER TABLE backup_configs
    ADD COLUMN encryption TEXT NOT NULL DEFAULT 'NONE';

ALTER TABLE backups
    ADD COLUMN encryption_salt TEXT,
    ADD COLUMN encryption_iv   TEXT,
    ADD COLUMN encryption      TEXT NOT NULL DEFAULT 'NONE';

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin

ALTER TABLE backups
    DROP COLUMN IF EXISTS encryption,
    DROP COLUMN IF EXISTS encryption_iv,
    DROP COLUMN IF EXISTS encryption_salt;

ALTER TABLE backup_configs
    DROP COLUMN IF EXISTS encryption;

-- +goose StatementEnd
