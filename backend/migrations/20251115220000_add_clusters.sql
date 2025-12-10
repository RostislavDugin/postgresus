-- +goose Up
-- +goose StatementBegin

CREATE TABLE clusters (
    id                      UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    workspace_id            UUID NOT NULL REFERENCES workspaces(id) ON DELETE CASCADE,
    name                    TEXT NOT NULL,

    is_backups_enabled      BOOLEAN NOT NULL DEFAULT FALSE,
    store_period            TEXT NOT NULL DEFAULT 'WEEK',
    backup_interval_id      UUID,
    storage_id              UUID REFERENCES storages(id),
    send_notifications_on   TEXT NOT NULL DEFAULT 'BACKUP_FAILED,BACKUP_SUCCESS',
    cpu_count               INT NOT NULL DEFAULT 1
);

CREATE TABLE postgresql_clusters (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    cluster_id  UUID NOT NULL REFERENCES clusters(id) ON DELETE CASCADE,
    version     TEXT NOT NULL,
    host        TEXT NOT NULL,
    port        INT  NOT NULL,
    username    TEXT NOT NULL,
    password    TEXT NOT NULL,
    is_https    BOOLEAN NOT NULL DEFAULT FALSE
);

CREATE TABLE cluster_notifiers (
    cluster_id  UUID NOT NULL REFERENCES clusters(id) ON DELETE CASCADE,
    notifier_id UUID NOT NULL REFERENCES notifiers(id) ON DELETE CASCADE,
    PRIMARY KEY (cluster_id, notifier_id)
);

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE IF EXISTS cluster_notifiers;
DROP TABLE IF EXISTS postgresql_clusters;
DROP TABLE IF EXISTS clusters;
-- +goose StatementEnd
