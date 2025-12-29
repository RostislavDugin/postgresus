-- +goose Up
-- +goose StatementBegin

CREATE TABLE ftp_storages (
    storage_id      UUID PRIMARY KEY,
    host            TEXT NOT NULL,
    port            INTEGER NOT NULL DEFAULT 21,
    username        TEXT NOT NULL,
    password        TEXT NOT NULL,
    path            TEXT,
    use_ssl         BOOLEAN NOT NULL DEFAULT FALSE,
    skip_tls_verify BOOLEAN NOT NULL DEFAULT FALSE,
    passive_mode    BOOLEAN NOT NULL DEFAULT TRUE
);

ALTER TABLE ftp_storages
    ADD CONSTRAINT fk_ftp_storages_storage
    FOREIGN KEY (storage_id)
    REFERENCES storages (id)
    ON DELETE CASCADE DEFERRABLE INITIALLY DEFERRED;

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin

DROP TABLE IF EXISTS ftp_storages;

-- +goose StatementEnd
