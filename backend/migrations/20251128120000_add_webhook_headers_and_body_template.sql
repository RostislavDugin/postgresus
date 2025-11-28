-- +goose Up
-- +goose StatementBegin

ALTER TABLE webhook_notifiers
    ADD COLUMN body_template TEXT,
    ADD COLUMN headers       TEXT DEFAULT '[]';

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin

ALTER TABLE webhook_notifiers
    DROP COLUMN body_template,
    DROP COLUMN headers;

-- +goose StatementEnd

