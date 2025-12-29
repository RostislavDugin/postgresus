-- +goose Up
-- +goose StatementBegin

CREATE TABLE rclone_storages (
    storage_id     UUID PRIMARY KEY,
    config_content TEXT NOT NULL,
    remote_path    TEXT
);

ALTER TABLE rclone_storages
    ADD CONSTRAINT fk_rclone_storages_storage
    FOREIGN KEY (storage_id)
    REFERENCES storages (id)
    ON DELETE CASCADE DEFERRABLE INITIALLY DEFERRED;

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin

DROP TABLE IF EXISTS rclone_storages;

-- +goose StatementEnd
