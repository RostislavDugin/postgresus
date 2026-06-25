-- +goose Up
-- +goose StatementBegin
ALTER TABLE postgresql_logical_databases
    ADD COLUMN is_skip_user_mappings BOOLEAN NOT NULL DEFAULT FALSE;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
ALTER TABLE postgresql_logical_databases
    DROP COLUMN is_skip_user_mappings;
-- +goose StatementEnd
