-- +goose Up
-- +goose StatementBegin
ALTER TABLE postgresql_databases ADD COLUMN is_strict_tls BOOLEAN NOT NULL DEFAULT FALSE;
-- +goose StatementEnd

-- +goose StatementBegin
ALTER TABLE mysql_databases ADD COLUMN is_strict_tls BOOLEAN NOT NULL DEFAULT FALSE;
-- +goose StatementEnd

-- +goose StatementBegin
ALTER TABLE mariadb_databases ADD COLUMN is_strict_tls BOOLEAN NOT NULL DEFAULT FALSE;
-- +goose StatementEnd

-- +goose StatementBegin
ALTER TABLE mongodb_databases ADD COLUMN is_strict_tls BOOLEAN NOT NULL DEFAULT FALSE;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
ALTER TABLE postgresql_databases DROP COLUMN IF EXISTS is_strict_tls;
-- +goose StatementEnd

-- +goose StatementBegin
ALTER TABLE mysql_databases DROP COLUMN IF EXISTS is_strict_tls;
-- +goose StatementEnd

-- +goose StatementBegin
ALTER TABLE mariadb_databases DROP COLUMN IF EXISTS is_strict_tls;
-- +goose StatementEnd

-- +goose StatementBegin
ALTER TABLE mongodb_databases DROP COLUMN IF EXISTS is_strict_tls;
-- +goose StatementEnd
