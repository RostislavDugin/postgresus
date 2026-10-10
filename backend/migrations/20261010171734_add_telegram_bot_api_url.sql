-- +goose Up
-- +goose StatementBegin

ALTER TABLE telegram_notifiers
    ADD COLUMN bot_api_url TEXT;

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin

ALTER TABLE telegram_notifiers
    DROP COLUMN IF EXISTS bot_api_url;

-- +goose StatementEnd
