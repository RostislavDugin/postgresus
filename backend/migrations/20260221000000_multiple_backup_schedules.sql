-- +goose Up
-- +goose StatementBegin

-- Add new ID column
ALTER TABLE backup_configs ADD COLUMN id UUID DEFAULT gen_random_uuid();

-- Populate ID for existing rows
UPDATE backup_configs SET id = gen_random_uuid() WHERE id IS NULL;

-- Set NOT NULL on id column
ALTER TABLE backup_configs ALTER COLUMN id SET NOT NULL;

-- Drop existing primary key constraint
ALTER TABLE backup_configs DROP CONSTRAINT backup_configs_pkey;

-- Set new primary key
ALTER TABLE backup_configs ADD PRIMARY KEY (id);

-- Create index on database_id for faster lookups
CREATE INDEX idx_backup_configs_database_id ON backup_configs(database_id);

-- Add name column for human-readable schedule identification
ALTER TABLE backup_configs ADD COLUMN name TEXT NOT NULL DEFAULT 'Default';

-- Add unique constraint for backup config naming within database
ALTER TABLE backup_configs ADD CONSTRAINT backup_configs_name_unique_per_database 
    UNIQUE (database_id, name);

-- Add backup_config_id to backups table
ALTER TABLE backups ADD COLUMN backup_config_id UUID;

-- Populate backup_config_id for existing backups
UPDATE backups b
SET backup_config_id = bc.id
FROM backup_configs bc
WHERE b.database_id = bc.database_id;

-- Add foreign key constraint for backup_config_id
ALTER TABLE backups
    ADD CONSTRAINT fk_backups_backup_config_id
    FOREIGN KEY (backup_config_id)
    REFERENCES backup_configs (id)
    ON DELETE SET NULL;

-- Create index for backups lookup by config
CREATE INDEX idx_backups_backup_config_id ON backups(backup_config_id);

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin

-- Remove index from backups
DROP INDEX IF EXISTS idx_backups_backup_config_id;

-- Remove foreign key from backups
ALTER TABLE backups DROP CONSTRAINT IF EXISTS fk_backups_backup_config_id;

-- Remove backup_config_id from backups
ALTER TABLE backups DROP COLUMN IF EXISTS backup_config_id;

-- Remove unique constraint from backup_configs
ALTER TABLE backup_configs DROP CONSTRAINT IF EXISTS backup_configs_name_unique_per_database;

-- Remove name column
ALTER TABLE backup_configs DROP COLUMN IF EXISTS name;

-- Remove index
DROP INDEX IF EXISTS idx_backup_configs_database_id;

-- Drop new primary key
ALTER TABLE backup_configs DROP CONSTRAINT backup_configs_pkey;

-- Restore old primary key
ALTER TABLE backup_configs ADD PRIMARY KEY (database_id);

-- Remove id column
ALTER TABLE backup_configs DROP COLUMN id;

-- +goose StatementEnd