-- +goose Up
-- +goose StatementBegin
ALTER TABLE s3_storages
    ADD COLUMN s3_prefix TEXT;

ALTER TABLE s3_storages
    ADD COLUMN s3_use_virtual_hosted_style BOOLEAN NOT NULL DEFAULT FALSE;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
ALTER TABLE s3_storages
    DROP COLUMN s3_use_virtual_hosted_style;

ALTER TABLE s3_storages
    DROP COLUMN s3_prefix;
-- +goose StatementEnd
