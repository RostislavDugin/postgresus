-- +goose Up
-- +goose StatementBegin

CREATE TABLE azure_blob_storages (
    storage_id        UUID PRIMARY KEY,
    auth_method       TEXT NOT NULL,
    connection_string TEXT,
    account_name      TEXT,
    account_key       TEXT,
    container_name    TEXT NOT NULL,
    endpoint          TEXT,
    prefix            TEXT
);

ALTER TABLE azure_blob_storages
    ADD CONSTRAINT fk_azure_blob_storages_storage
    FOREIGN KEY (storage_id)
    REFERENCES storages (id)
    ON DELETE CASCADE DEFERRABLE INITIALLY DEFERRED;

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin

DROP TABLE IF EXISTS azure_blob_storages;

-- +goose StatementEnd
