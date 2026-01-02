-- +goose Up
-- +goose StatementBegin
ALTER TABLE backups DROP COLUMN type;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
ALTER TABLE backups ADD COLUMN type TEXT NOT NULL DEFAULT 'DEFAULT';
-- +goose StatementEnd
