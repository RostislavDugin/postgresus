-- +goose Up
-- +goose StatementBegin

-- Create audit_logs table
CREATE TABLE audit_logs (
    id         UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id    UUID,
    workspace_id UUID,
    message    TEXT NOT NULL,
    created_at TIMESTAMP NOT NULL DEFAULT NOW()
);

-- Add foreign key constraints
ALTER TABLE audit_logs
    ADD CONSTRAINT fk_audit_logs_user_id
    FOREIGN KEY (user_id)
    REFERENCES users (id)
    ON DELETE SET NULL;

-- Create indexes for efficient querying
CREATE INDEX idx_audit_logs_user_id ON audit_logs (user_id);
CREATE INDEX idx_audit_logs_workspace_id ON audit_logs (workspace_id);
CREATE INDEX idx_audit_logs_created_at ON audit_logs (created_at);

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin

-- Drop indexes
DROP INDEX IF EXISTS idx_audit_logs_created_at;
DROP INDEX IF EXISTS idx_audit_logs_workspace_id;
DROP INDEX IF EXISTS idx_audit_logs_user_id;

-- Drop foreign key constraints
ALTER TABLE audit_logs DROP CONSTRAINT IF EXISTS fk_audit_logs_workspace_id;
ALTER TABLE audit_logs DROP CONSTRAINT IF EXISTS fk_audit_logs_user_id;

-- Drop table
DROP TABLE IF EXISTS audit_logs;

-- +goose StatementEnd
