-- +goose Up
-- +goose StatementBegin
CREATE TABLE mariadb_databases (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    database_id UUID REFERENCES databases(id) ON DELETE CASCADE,
    version     TEXT NOT NULL,
    host        TEXT NOT NULL,
    port        INT NOT NULL,
    username    TEXT NOT NULL,
    password    TEXT NOT NULL,
    database    TEXT,
    is_https    BOOLEAN NOT NULL DEFAULT FALSE
);
-- +goose StatementEnd

-- +goose StatementBegin
CREATE INDEX idx_mariadb_databases_database_id ON mariadb_databases(database_id);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP INDEX IF EXISTS idx_mariadb_databases_database_id;
-- +goose StatementEnd

-- +goose StatementBegin
DROP TABLE IF EXISTS mariadb_databases;
-- +goose StatementEnd

