-- +goose Up
-- +goose StatementBegin

CREATE TABLE workspaces (
    id         UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name       TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE workspace_memberships (
    id           UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id      UUID NOT NULL,
    workspace_id UUID NOT NULL,
    role         TEXT NOT NULL,
    created_at   TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

ALTER TABLE workspace_memberships
    ADD CONSTRAINT fk_workspace_memberships_user_id
    FOREIGN KEY (user_id)
    REFERENCES users (id)
    ON DELETE CASCADE;

ALTER TABLE workspace_memberships
    ADD CONSTRAINT fk_workspace_memberships_workspace_id
    FOREIGN KEY (workspace_id)
    REFERENCES workspaces (id)
    ON DELETE CASCADE;

ALTER TABLE workspace_memberships
    ADD CONSTRAINT uk_workspace_memberships_user_workspace
    UNIQUE (user_id, workspace_id);

CREATE INDEX idx_workspaces_created_at ON workspaces (created_at DESC);

CREATE INDEX idx_workspace_memberships_user_id ON workspace_memberships (user_id);
CREATE INDEX idx_workspace_memberships_workspace_id ON workspace_memberships (workspace_id);
CREATE INDEX idx_workspace_memberships_user_workspace ON workspace_memberships (user_id, workspace_id);
CREATE INDEX idx_workspace_memberships_workspace_user ON workspace_memberships (workspace_id, user_id);

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin

DROP INDEX IF EXISTS idx_workspace_memberships_workspace_user;
DROP INDEX IF EXISTS idx_workspace_memberships_user_workspace;
DROP INDEX IF EXISTS idx_workspace_memberships_workspace_id;
DROP INDEX IF EXISTS idx_workspace_memberships_user_id;

DROP INDEX IF EXISTS idx_workspaces_created_at;

ALTER TABLE workspace_memberships DROP CONSTRAINT IF EXISTS fk_workspace_memberships_workspace_id;
ALTER TABLE workspace_memberships DROP CONSTRAINT IF EXISTS fk_workspace_memberships_user_id;

DROP TABLE IF EXISTS workspace_memberships;
DROP TABLE IF EXISTS workspaces;

-- +goose StatementEnd
