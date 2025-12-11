-- +goose Up
-- +goose StatementBegin
ALTER TABLE s3_storages
    ADD COLUMN skip_tls_verify BOOLEAN NOT NULL DEFAULT FALSE;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
ALTER TABLE s3_storages
    DROP COLUMN skip_tls_verify;
-- +goose StatementEnd
