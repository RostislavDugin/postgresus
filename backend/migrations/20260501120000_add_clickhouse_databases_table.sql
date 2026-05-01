-- +goose Up
-- +goose StatementBegin
CREATE TABLE clickhouse_databases (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    database_id UUID REFERENCES databases(id) ON DELETE CASCADE,
    version     TEXT NOT NULL,
    host        TEXT NOT NULL,
    port        INT NOT NULL,
    username    TEXT NOT NULL,
    password    TEXT NOT NULL DEFAULT '',
    database    TEXT NOT NULL,
    is_https      BOOLEAN NOT NULL DEFAULT FALSE,
    is_strict_tls BOOLEAN NOT NULL DEFAULT FALSE
);
-- +goose StatementEnd

-- +goose StatementBegin
CREATE INDEX idx_clickhouse_databases_database_id ON clickhouse_databases(database_id);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP INDEX IF EXISTS idx_clickhouse_databases_database_id;
-- +goose StatementEnd

-- +goose StatementBegin
DROP TABLE IF EXISTS clickhouse_databases;
-- +goose StatementEnd
