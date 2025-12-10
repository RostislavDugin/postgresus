-- +goose Up
-- +goose StatementBegin

CREATE TABLE IF NOT EXISTS cluster_excluded_databases (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    cluster_id  UUID NOT NULL REFERENCES clusters(id) ON DELETE CASCADE,
    name        TEXT NOT NULL
);

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE IF EXISTS cluster_excluded_databases;
-- +goose StatementEnd
