-- +goose Up
-- +goose StatementBegin

ALTER TABLE postgresql_databases
    ADD COLUMN schemas TEXT;

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin

ALTER TABLE postgresql_databases
    DROP COLUMN IF EXISTS schemas;

-- +goose StatementEnd
