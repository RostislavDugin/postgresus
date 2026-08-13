-- +goose Up
-- +goose StatementBegin
ALTER TABLE postgresql_logical_databases
    ADD COLUMN IF NOT EXISTS ssh_auth_type TEXT NOT NULL DEFAULT 'PASSWORD';

ALTER TABLE postgresql_physical_databases
    ADD COLUMN IF NOT EXISTS ssh_auth_type TEXT NOT NULL DEFAULT 'PASSWORD';

ALTER TABLE mysql_databases
    ADD COLUMN IF NOT EXISTS ssh_auth_type TEXT NOT NULL DEFAULT 'PASSWORD';

ALTER TABLE mariadb_databases
    ADD COLUMN IF NOT EXISTS ssh_auth_type TEXT NOT NULL DEFAULT 'PASSWORD';

ALTER TABLE mongodb_databases
    ADD COLUMN IF NOT EXISTS ssh_auth_type TEXT NOT NULL DEFAULT 'PASSWORD';

-- A row holding both secrets was logging in with the password, which the old auth builder offered
-- first, so only a key-only row is migrated to key auth.
UPDATE postgresql_logical_databases
SET ssh_auth_type = 'PRIVATE_KEY'
WHERE ssh_private_key <> '' AND ssh_password = '';

UPDATE postgresql_physical_databases
SET ssh_auth_type = 'PRIVATE_KEY'
WHERE ssh_private_key <> '' AND ssh_password = '';

UPDATE mysql_databases
SET ssh_auth_type = 'PRIVATE_KEY'
WHERE ssh_private_key <> '' AND ssh_password = '';

UPDATE mariadb_databases
SET ssh_auth_type = 'PRIVATE_KEY'
WHERE ssh_private_key <> '' AND ssh_password = '';

UPDATE mongodb_databases
SET ssh_auth_type = 'PRIVATE_KEY'
WHERE ssh_private_key <> '' AND ssh_password = '';

-- The secret of the type not chosen is a dormant second way into the bastion that no screen shows
-- again, so it is cleared in the same migration that picks the type.
UPDATE postgresql_logical_databases
SET ssh_private_key = '', ssh_private_key_passphrase = ''
WHERE ssh_auth_type = 'PASSWORD'
  AND (ssh_private_key <> '' OR ssh_private_key_passphrase <> '');

UPDATE postgresql_physical_databases
SET ssh_private_key = '', ssh_private_key_passphrase = ''
WHERE ssh_auth_type = 'PASSWORD'
  AND (ssh_private_key <> '' OR ssh_private_key_passphrase <> '');

UPDATE mysql_databases
SET ssh_private_key = '', ssh_private_key_passphrase = ''
WHERE ssh_auth_type = 'PASSWORD'
  AND (ssh_private_key <> '' OR ssh_private_key_passphrase <> '');

UPDATE mariadb_databases
SET ssh_private_key = '', ssh_private_key_passphrase = ''
WHERE ssh_auth_type = 'PASSWORD'
  AND (ssh_private_key <> '' OR ssh_private_key_passphrase <> '');

UPDATE mongodb_databases
SET ssh_private_key = '', ssh_private_key_passphrase = ''
WHERE ssh_auth_type = 'PASSWORD'
  AND (ssh_private_key <> '' OR ssh_private_key_passphrase <> '');

UPDATE postgresql_logical_databases
SET ssh_password = ''
WHERE ssh_auth_type = 'PRIVATE_KEY'
  AND ssh_password <> '';

UPDATE postgresql_physical_databases
SET ssh_password = ''
WHERE ssh_auth_type = 'PRIVATE_KEY'
  AND ssh_password <> '';

UPDATE mysql_databases
SET ssh_password = ''
WHERE ssh_auth_type = 'PRIVATE_KEY'
  AND ssh_password <> '';

UPDATE mariadb_databases
SET ssh_password = ''
WHERE ssh_auth_type = 'PRIVATE_KEY'
  AND ssh_password <> '';

UPDATE mongodb_databases
SET ssh_password = ''
WHERE ssh_auth_type = 'PRIVATE_KEY'
  AND ssh_password <> '';
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
ALTER TABLE mongodb_databases
    DROP COLUMN IF EXISTS ssh_auth_type;

ALTER TABLE mariadb_databases
    DROP COLUMN IF EXISTS ssh_auth_type;

ALTER TABLE mysql_databases
    DROP COLUMN IF EXISTS ssh_auth_type;

ALTER TABLE postgresql_physical_databases
    DROP COLUMN IF EXISTS ssh_auth_type;

ALTER TABLE postgresql_logical_databases
    DROP COLUMN IF EXISTS ssh_auth_type;
-- +goose StatementEnd
