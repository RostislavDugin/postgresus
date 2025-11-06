-- +goose Up
-- +goose StatementBegin

ALTER TABLE webhook_notifiers
    ADD COLUMN custom_template TEXT;

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin

ALTER TABLE webhook_notifiers
    DROP COLUMN IF EXISTS custom_template;

-- +goose StatementEnd
