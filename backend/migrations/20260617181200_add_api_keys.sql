-- +goose Up
-- +goose StatementBegin

CREATE TABLE api_keys (
    id                 UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name               TEXT NOT NULL,
    hashed_token       TEXT NOT NULL,
    token_prefix       TEXT NOT NULL,
    role               TEXT NOT NULL,
    created_by_user_id UUID NOT NULL,
    last_used_at       TIMESTAMPTZ,
    expires_at         TIMESTAMPTZ,
    revoked_at         TIMESTAMPTZ,
    created_at         TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE api_key_workspaces (
    api_key_id   UUID NOT NULL,
    workspace_id UUID NOT NULL
);

ALTER TABLE api_keys
    ADD CONSTRAINT fk_api_keys_created_by_user_id
    FOREIGN KEY (created_by_user_id)
    REFERENCES users (id)
    ON DELETE CASCADE;

ALTER TABLE api_key_workspaces
    ADD CONSTRAINT pk_api_key_workspaces
    PRIMARY KEY (api_key_id, workspace_id);

ALTER TABLE api_key_workspaces
    ADD CONSTRAINT fk_api_key_workspaces_api_key_id
    FOREIGN KEY (api_key_id)
    REFERENCES api_keys (id)
    ON DELETE CASCADE;

ALTER TABLE api_key_workspaces
    ADD CONSTRAINT fk_api_key_workspaces_workspace_id
    FOREIGN KEY (workspace_id)
    REFERENCES workspaces (id)
    ON DELETE CASCADE;

CREATE UNIQUE INDEX idx_api_keys_hashed_token ON api_keys (hashed_token);
CREATE INDEX idx_api_keys_created_at ON api_keys (created_at DESC);
CREATE INDEX idx_api_key_workspaces_api_key_id ON api_key_workspaces (api_key_id);
CREATE INDEX idx_api_key_workspaces_workspace_id ON api_key_workspaces (workspace_id);

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin

DROP INDEX IF EXISTS idx_api_key_workspaces_workspace_id;
DROP INDEX IF EXISTS idx_api_key_workspaces_api_key_id;
DROP INDEX IF EXISTS idx_api_keys_created_at;
DROP INDEX IF EXISTS idx_api_keys_hashed_token;

ALTER TABLE api_key_workspaces DROP CONSTRAINT IF EXISTS fk_api_key_workspaces_workspace_id;
ALTER TABLE api_key_workspaces DROP CONSTRAINT IF EXISTS fk_api_key_workspaces_api_key_id;
ALTER TABLE api_key_workspaces DROP CONSTRAINT IF EXISTS pk_api_key_workspaces;
ALTER TABLE api_keys DROP CONSTRAINT IF EXISTS fk_api_keys_created_by_user_id;

DROP TABLE IF EXISTS api_key_workspaces;
DROP TABLE IF EXISTS api_keys;

-- +goose StatementEnd
