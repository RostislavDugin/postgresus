-- +goose Up
-- +goose StatementBegin
ALTER TABLE postgresql_logical_databases
    ADD COLUMN exclude_table_data TEXT NOT NULL DEFAULT '';
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
ALTER TABLE postgresql_logical_databases
    DROP COLUMN exclude_table_data;
-- +goose StatementEnd
