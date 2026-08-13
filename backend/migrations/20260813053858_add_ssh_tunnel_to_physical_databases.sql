-- +goose Up
-- +goose StatementBegin
ALTER TABLE postgresql_physical_databases
    ADD COLUMN IF NOT EXISTS ssh_is_enabled BOOLEAN NOT NULL DEFAULT FALSE;

ALTER TABLE postgresql_physical_databases
    ADD COLUMN IF NOT EXISTS ssh_host TEXT NOT NULL DEFAULT '';

ALTER TABLE postgresql_physical_databases
    ADD COLUMN IF NOT EXISTS ssh_port INTEGER NOT NULL DEFAULT 22;

ALTER TABLE postgresql_physical_databases
    ADD COLUMN IF NOT EXISTS ssh_username TEXT NOT NULL DEFAULT '';

ALTER TABLE postgresql_physical_databases
    ADD COLUMN IF NOT EXISTS ssh_password TEXT NOT NULL DEFAULT '';

ALTER TABLE postgresql_physical_databases
    ADD COLUMN IF NOT EXISTS ssh_private_key TEXT NOT NULL DEFAULT '';

ALTER TABLE postgresql_physical_databases
    ADD COLUMN IF NOT EXISTS ssh_private_key_passphrase TEXT NOT NULL DEFAULT '';
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
ALTER TABLE postgresql_physical_databases
    DROP COLUMN IF EXISTS ssh_private_key_passphrase;

ALTER TABLE postgresql_physical_databases
    DROP COLUMN IF EXISTS ssh_private_key;

ALTER TABLE postgresql_physical_databases
    DROP COLUMN IF EXISTS ssh_password;

ALTER TABLE postgresql_physical_databases
    DROP COLUMN IF EXISTS ssh_username;

ALTER TABLE postgresql_physical_databases
    DROP COLUMN IF EXISTS ssh_port;

ALTER TABLE postgresql_physical_databases
    DROP COLUMN IF EXISTS ssh_host;

ALTER TABLE postgresql_physical_databases
    DROP COLUMN IF EXISTS ssh_is_enabled;
-- +goose StatementEnd
