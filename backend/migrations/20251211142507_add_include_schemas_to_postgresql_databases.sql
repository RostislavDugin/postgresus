-- +goose Up
-- +goose StatementBegin
ALTER TABLE postgresql_databases
    ADD COLUMN include_schemas TEXT NOT NULL DEFAULT '';
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
ALTER TABLE postgresql_databases
    DROP COLUMN include_schemas;
-- +goose StatementEnd
