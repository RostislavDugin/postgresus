-- +goose Up
-- +goose StatementBegin

-- Add workspace_id column to storages table
ALTER TABLE storages ADD COLUMN workspace_id UUID;

-- Add workspace_id column to notifiers table
ALTER TABLE notifiers ADD COLUMN workspace_id UUID;

-- Update storages to get workspace_id from first workspace
UPDATE storages
SET workspace_id = (SELECT id FROM workspaces ORDER BY created_at ASC LIMIT 1)
WHERE workspace_id IS NULL;

-- Update notifiers to get workspace_id from first workspace
UPDATE notifiers
SET workspace_id = (SELECT id FROM workspaces ORDER BY created_at ASC LIMIT 1)
WHERE workspace_id IS NULL;

-- Add foreign key constraint for storages
ALTER TABLE storages
    ADD CONSTRAINT fk_storages_workspace_id
    FOREIGN KEY (workspace_id)
    REFERENCES workspaces (id);

-- Add foreign key constraint for notifiers
ALTER TABLE notifiers
    ADD CONSTRAINT fk_notifiers_workspace_id
    FOREIGN KEY (workspace_id)
    REFERENCES workspaces (id);

-- Add index for workspace_id lookups on storages
CREATE INDEX idx_storages_workspace_id ON storages (workspace_id);

-- Add index for workspace_id lookups on notifiers
CREATE INDEX idx_notifiers_workspace_id ON notifiers (workspace_id);

-- Now drop the user_id column from storages
ALTER TABLE storages DROP COLUMN user_id;

-- Now drop the user_id column from notifiers
ALTER TABLE notifiers DROP COLUMN user_id;

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin

-- Restore user_id column for storages (cannot restore original values)
ALTER TABLE storages ADD COLUMN user_id UUID;

-- Restore user_id column for notifiers (cannot restore original values)
ALTER TABLE notifiers ADD COLUMN user_id UUID;

-- Drop indexes
DROP INDEX IF EXISTS idx_storages_workspace_id;
DROP INDEX IF EXISTS idx_notifiers_workspace_id;

-- Drop foreign key constraints
ALTER TABLE storages DROP CONSTRAINT IF EXISTS fk_storages_workspace_id;
ALTER TABLE notifiers DROP CONSTRAINT IF EXISTS fk_notifiers_workspace_id;

-- Drop workspace_id columns
ALTER TABLE storages DROP COLUMN workspace_id;
ALTER TABLE notifiers DROP COLUMN workspace_id;

-- +goose StatementEnd

