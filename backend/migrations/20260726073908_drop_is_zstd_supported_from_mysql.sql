-- +goose Up
-- +goose StatementBegin
ALTER TABLE mysql_databases
    DROP COLUMN IF EXISTS is_zstd_supported;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
ALTER TABLE mysql_databases
    ADD COLUMN IF NOT EXISTS is_zstd_supported BOOLEAN NOT NULL DEFAULT TRUE;
-- +goose StatementEnd
