-- +goose Up
-- +goose StatementBegin

CREATE TABLE sftp_storages (
    storage_id           UUID PRIMARY KEY,
    host                 TEXT NOT NULL,
    port                 INTEGER NOT NULL DEFAULT 22,
    username             TEXT NOT NULL,
    password             TEXT,
    private_key          TEXT,
    path                 TEXT,
    skip_host_key_verify BOOLEAN NOT NULL DEFAULT FALSE
);

ALTER TABLE sftp_storages
    ADD CONSTRAINT fk_sftp_storages_storage
    FOREIGN KEY (storage_id)
    REFERENCES storages (id)
    ON DELETE CASCADE DEFERRABLE INITIALLY DEFERRED;

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin

DROP TABLE IF EXISTS sftp_storages;

-- +goose StatementEnd
