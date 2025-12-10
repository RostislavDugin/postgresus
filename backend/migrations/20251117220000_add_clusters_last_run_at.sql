-- +goose Up
ALTER TABLE clusters ADD COLUMN IF NOT EXISTS last_run_at timestamptz NULL;

-- +goose Down
ALTER TABLE clusters DROP COLUMN IF EXISTS last_run_at;
