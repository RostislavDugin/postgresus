-- +goose Up
-- +goose StatementBegin
ALTER TABLE mysql_databases
    ADD COLUMN IF NOT EXISTS is_use_extended_insert BOOLEAN NOT NULL DEFAULT FALSE;

ALTER TABLE mariadb_databases
    ADD COLUMN IF NOT EXISTS is_use_extended_insert BOOLEAN NOT NULL DEFAULT FALSE;

ALTER TABLE mariadb_databases
    ADD COLUMN IF NOT EXISTS is_skip_galera_disable BOOLEAN NOT NULL DEFAULT FALSE;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
ALTER TABLE mysql_databases
    DROP COLUMN IF EXISTS is_use_extended_insert;

ALTER TABLE mariadb_databases
    DROP COLUMN IF EXISTS is_use_extended_insert;

ALTER TABLE mariadb_databases
    DROP COLUMN IF EXISTS is_skip_galera_disable;
-- +goose StatementEnd
