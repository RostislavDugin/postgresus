-- +goose Up
-- +goose StatementBegin
ALTER TABLE postgresql_logical_databases
    ADD COLUMN IF NOT EXISTS ssh_is_enabled BOOLEAN NOT NULL DEFAULT FALSE;

ALTER TABLE postgresql_logical_databases
    ADD COLUMN IF NOT EXISTS ssh_host TEXT NOT NULL DEFAULT '';

ALTER TABLE postgresql_logical_databases
    ADD COLUMN IF NOT EXISTS ssh_port INTEGER NOT NULL DEFAULT 22;

ALTER TABLE postgresql_logical_databases
    ADD COLUMN IF NOT EXISTS ssh_username TEXT NOT NULL DEFAULT '';

ALTER TABLE postgresql_logical_databases
    ADD COLUMN IF NOT EXISTS ssh_password TEXT NOT NULL DEFAULT '';

ALTER TABLE postgresql_logical_databases
    ADD COLUMN IF NOT EXISTS ssh_private_key TEXT NOT NULL DEFAULT '';

ALTER TABLE postgresql_logical_databases
    ADD COLUMN IF NOT EXISTS ssh_private_key_passphrase TEXT NOT NULL DEFAULT '';

ALTER TABLE mysql_databases
    ADD COLUMN IF NOT EXISTS ssh_is_enabled BOOLEAN NOT NULL DEFAULT FALSE;

ALTER TABLE mysql_databases
    ADD COLUMN IF NOT EXISTS ssh_host TEXT NOT NULL DEFAULT '';

ALTER TABLE mysql_databases
    ADD COLUMN IF NOT EXISTS ssh_port INTEGER NOT NULL DEFAULT 22;

ALTER TABLE mysql_databases
    ADD COLUMN IF NOT EXISTS ssh_username TEXT NOT NULL DEFAULT '';

ALTER TABLE mysql_databases
    ADD COLUMN IF NOT EXISTS ssh_password TEXT NOT NULL DEFAULT '';

ALTER TABLE mysql_databases
    ADD COLUMN IF NOT EXISTS ssh_private_key TEXT NOT NULL DEFAULT '';

ALTER TABLE mysql_databases
    ADD COLUMN IF NOT EXISTS ssh_private_key_passphrase TEXT NOT NULL DEFAULT '';

ALTER TABLE mariadb_databases
    ADD COLUMN IF NOT EXISTS ssh_is_enabled BOOLEAN NOT NULL DEFAULT FALSE;

ALTER TABLE mariadb_databases
    ADD COLUMN IF NOT EXISTS ssh_host TEXT NOT NULL DEFAULT '';

ALTER TABLE mariadb_databases
    ADD COLUMN IF NOT EXISTS ssh_port INTEGER NOT NULL DEFAULT 22;

ALTER TABLE mariadb_databases
    ADD COLUMN IF NOT EXISTS ssh_username TEXT NOT NULL DEFAULT '';

ALTER TABLE mariadb_databases
    ADD COLUMN IF NOT EXISTS ssh_password TEXT NOT NULL DEFAULT '';

ALTER TABLE mariadb_databases
    ADD COLUMN IF NOT EXISTS ssh_private_key TEXT NOT NULL DEFAULT '';

ALTER TABLE mariadb_databases
    ADD COLUMN IF NOT EXISTS ssh_private_key_passphrase TEXT NOT NULL DEFAULT '';

ALTER TABLE mongodb_databases
    ADD COLUMN IF NOT EXISTS ssh_is_enabled BOOLEAN NOT NULL DEFAULT FALSE;

ALTER TABLE mongodb_databases
    ADD COLUMN IF NOT EXISTS ssh_host TEXT NOT NULL DEFAULT '';

ALTER TABLE mongodb_databases
    ADD COLUMN IF NOT EXISTS ssh_port INTEGER NOT NULL DEFAULT 22;

ALTER TABLE mongodb_databases
    ADD COLUMN IF NOT EXISTS ssh_username TEXT NOT NULL DEFAULT '';

ALTER TABLE mongodb_databases
    ADD COLUMN IF NOT EXISTS ssh_password TEXT NOT NULL DEFAULT '';

ALTER TABLE mongodb_databases
    ADD COLUMN IF NOT EXISTS ssh_private_key TEXT NOT NULL DEFAULT '';

ALTER TABLE mongodb_databases
    ADD COLUMN IF NOT EXISTS ssh_private_key_passphrase TEXT NOT NULL DEFAULT '';
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
ALTER TABLE mongodb_databases
    DROP COLUMN IF EXISTS ssh_private_key_passphrase;

ALTER TABLE mongodb_databases
    DROP COLUMN IF EXISTS ssh_private_key;

ALTER TABLE mongodb_databases
    DROP COLUMN IF EXISTS ssh_password;

ALTER TABLE mongodb_databases
    DROP COLUMN IF EXISTS ssh_username;

ALTER TABLE mongodb_databases
    DROP COLUMN IF EXISTS ssh_port;

ALTER TABLE mongodb_databases
    DROP COLUMN IF EXISTS ssh_host;

ALTER TABLE mongodb_databases
    DROP COLUMN IF EXISTS ssh_is_enabled;

ALTER TABLE mariadb_databases
    DROP COLUMN IF EXISTS ssh_private_key_passphrase;

ALTER TABLE mariadb_databases
    DROP COLUMN IF EXISTS ssh_private_key;

ALTER TABLE mariadb_databases
    DROP COLUMN IF EXISTS ssh_password;

ALTER TABLE mariadb_databases
    DROP COLUMN IF EXISTS ssh_username;

ALTER TABLE mariadb_databases
    DROP COLUMN IF EXISTS ssh_port;

ALTER TABLE mariadb_databases
    DROP COLUMN IF EXISTS ssh_host;

ALTER TABLE mariadb_databases
    DROP COLUMN IF EXISTS ssh_is_enabled;

ALTER TABLE mysql_databases
    DROP COLUMN IF EXISTS ssh_private_key_passphrase;

ALTER TABLE mysql_databases
    DROP COLUMN IF EXISTS ssh_private_key;

ALTER TABLE mysql_databases
    DROP COLUMN IF EXISTS ssh_password;

ALTER TABLE mysql_databases
    DROP COLUMN IF EXISTS ssh_username;

ALTER TABLE mysql_databases
    DROP COLUMN IF EXISTS ssh_port;

ALTER TABLE mysql_databases
    DROP COLUMN IF EXISTS ssh_host;

ALTER TABLE mysql_databases
    DROP COLUMN IF EXISTS ssh_is_enabled;

ALTER TABLE postgresql_logical_databases
    DROP COLUMN IF EXISTS ssh_private_key_passphrase;

ALTER TABLE postgresql_logical_databases
    DROP COLUMN IF EXISTS ssh_private_key;

ALTER TABLE postgresql_logical_databases
    DROP COLUMN IF EXISTS ssh_password;

ALTER TABLE postgresql_logical_databases
    DROP COLUMN IF EXISTS ssh_username;

ALTER TABLE postgresql_logical_databases
    DROP COLUMN IF EXISTS ssh_port;

ALTER TABLE postgresql_logical_databases
    DROP COLUMN IF EXISTS ssh_host;

ALTER TABLE postgresql_logical_databases
    DROP COLUMN IF EXISTS ssh_is_enabled;
-- +goose StatementEnd
