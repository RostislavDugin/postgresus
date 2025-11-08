-- +goose Up
-- +goose StatementBegin
ALTER TABLE email_notifiers
    ADD COLUMN from_email VARCHAR(255);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
ALTER TABLE email_notifiers
    DROP COLUMN from_email;
-- +goose StatementEnd
