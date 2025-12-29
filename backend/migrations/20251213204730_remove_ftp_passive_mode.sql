-- +goose Up
-- +goose StatementBegin

ALTER TABLE ftp_storages
    DROP COLUMN passive_mode;

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin

ALTER TABLE ftp_storages
    ADD COLUMN passive_mode BOOLEAN NOT NULL DEFAULT TRUE;

-- +goose StatementEnd
