-- +goose Up
-- +goose StatementBegin
ALTER TABLE mongodb_databases ADD COLUMN direct_connection BOOLEAN NOT NULL DEFAULT FALSE;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
ALTER TABLE mongodb_databases DROP COLUMN direct_connection;
-- +goose StatementEnd
