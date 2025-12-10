-- +goose Up
ALTER TABLE backup_configs ADD COLUMN IF NOT EXISTS cluster_id uuid NULL;
ALTER TABLE backup_configs ADD COLUMN IF NOT EXISTS managed_by_cluster boolean NOT NULL DEFAULT false;
CREATE INDEX IF NOT EXISTS idx_backup_configs_cluster_id ON backup_configs(cluster_id);

-- +goose Down
DROP INDEX IF EXISTS idx_backup_configs_cluster_id;
ALTER TABLE backup_configs DROP COLUMN IF EXISTS managed_by_cluster;
ALTER TABLE backup_configs DROP COLUMN IF EXISTS cluster_id;
