-- +goose Up
-- +goose StatementBegin

-- Add workspace_id column to databases table
ALTER TABLE databases ADD COLUMN workspace_id UUID;

-- Create default workspace only if there are databases, storages, or notifiers
INSERT INTO workspaces (id, name, created_at)
SELECT 
    gen_random_uuid(),
    'Default Workspace',
    NOW()
WHERE NOT EXISTS (SELECT 1 FROM workspaces)
  AND (
    EXISTS (SELECT 1 FROM databases WHERE user_id IS NOT NULL)
    OR EXISTS (SELECT 1 FROM storages)
    OR EXISTS (SELECT 1 FROM notifiers)
  );

-- Update databases that HAVE user_id to get workspace_id
-- Databases without user_id (restore databases) remain with workspace_id = null
UPDATE databases
SET workspace_id = (SELECT id FROM workspaces ORDER BY created_at ASC LIMIT 1)
WHERE workspace_id IS NULL AND user_id IS NOT NULL;

-- Add foreign key constraint
ALTER TABLE databases
    ADD CONSTRAINT fk_databases_workspace_id
    FOREIGN KEY (workspace_id)
    REFERENCES workspaces (id);

-- Add index for workspace_id lookups
CREATE INDEX idx_databases_workspace_id ON databases (workspace_id);

-- Now drop the user_id column
ALTER TABLE databases DROP COLUMN user_id;

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin

-- Restore user_id column (cannot restore original values)
ALTER TABLE databases ADD COLUMN user_id UUID;

-- Drop index
DROP INDEX IF EXISTS idx_databases_workspace_id;

-- Drop foreign key constraint
ALTER TABLE databases DROP CONSTRAINT IF EXISTS fk_databases_workspace_id;

-- Drop workspace_id column
ALTER TABLE databases DROP COLUMN workspace_id;

-- +goose StatementEnd

