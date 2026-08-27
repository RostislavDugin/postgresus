-- +goose Up
-- +goose StatementBegin
ALTER TABLE physical_backup_configs
    ADD COLUMN IF NOT EXISTS summarizer_lag_threshold_bytes BIGINT NOT NULL DEFAULT 0;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
ALTER TABLE physical_backup_configs
    DROP COLUMN IF EXISTS summarizer_lag_threshold_bytes;
-- +goose StatementEnd
