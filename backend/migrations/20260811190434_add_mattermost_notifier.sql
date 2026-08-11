-- +goose Up
-- +goose StatementBegin

CREATE TABLE mattermost_notifiers (
    notifier_id             UUID PRIMARY KEY,
    delivery_mode           TEXT    NOT NULL,
    webhook_url             TEXT    NOT NULL DEFAULT '',
    server_url              TEXT    NOT NULL DEFAULT '',
    bot_token               TEXT    NOT NULL DEFAULT '',
    target_channel_name     TEXT    NOT NULL DEFAULT '',
    target_channel_id       TEXT    NOT NULL DEFAULT '',
    override_username       TEXT    NOT NULL DEFAULT '',
    override_icon_url       TEXT    NOT NULL DEFAULT '',
    is_insecure_skip_verify BOOLEAN NOT NULL DEFAULT FALSE
);

ALTER TABLE mattermost_notifiers
    ADD CONSTRAINT fk_mattermost_notifiers_notifier
    FOREIGN KEY (notifier_id)
    REFERENCES notifiers (id)
    ON DELETE CASCADE DEFERRABLE INITIALLY DEFERRED;

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE IF EXISTS mattermost_notifiers;
-- +goose StatementEnd
