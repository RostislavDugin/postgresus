-- +goose Up
-- +goose StatementBegin

-- Create users_settings table
CREATE TABLE users_settings (
    id                                     UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    is_allow_external_registrations        BOOLEAN NOT NULL DEFAULT TRUE,
    is_allow_member_invitations            BOOLEAN NOT NULL DEFAULT TRUE,
    is_member_allowed_to_create_workspaces BOOLEAN NOT NULL DEFAULT TRUE
);

-- Add new columns to users table
ALTER TABLE users ADD COLUMN name TEXT;
ALTER TABLE users ADD COLUMN status TEXT;
ALTER TABLE users ADD COLUMN github_oauth_id TEXT;
ALTER TABLE users ADD COLUMN google_oauth_id TEXT;

-- Update existing user(s)
-- Set email to 'admin', name to 'Admin', and status to 'ACTIVE'
UPDATE users
SET
    email = 'admin',
    name = 'Admin',
    status = 'ACTIVE'
WHERE id IN (SELECT id FROM users LIMIT 1);

-- Remove all users except the admin user (should not exist, just to be sure)
DELETE FROM users
WHERE id NOT IN (SELECT id FROM users WHERE email = 'admin' LIMIT 1);

-- Make name and status NOT NULL after data migration
ALTER TABLE users ALTER COLUMN name SET NOT NULL;
ALTER TABLE users ALTER COLUMN status SET NOT NULL;

-- Make hashed_password nullable to support OAuth-only accounts
ALTER TABLE users ALTER COLUMN hashed_password DROP NOT NULL;


-- Create indexes for new columns
CREATE INDEX idx_users_status ON users (status);
CREATE UNIQUE INDEX idx_users_github_oauth_id ON users (github_oauth_id) WHERE github_oauth_id IS NOT NULL;
CREATE UNIQUE INDEX idx_users_google_oauth_id ON users (google_oauth_id) WHERE google_oauth_id IS NOT NULL;

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin

-- Drop indexes
DROP INDEX IF EXISTS idx_users_google_oauth_id;
DROP INDEX IF EXISTS idx_users_github_oauth_id;
DROP INDEX IF EXISTS idx_users_status;

-- Remove new columns from users table
ALTER TABLE users DROP COLUMN IF EXISTS google_oauth_id;
ALTER TABLE users DROP COLUMN IF EXISTS github_oauth_id;
ALTER TABLE users DROP COLUMN IF EXISTS status;
ALTER TABLE users DROP COLUMN IF EXISTS name;

-- Restore hashed_password NOT NULL constraint
ALTER TABLE users ALTER COLUMN hashed_password SET NOT NULL;

-- Drop new tables
DROP TABLE IF EXISTS users_settings;

-- +goose StatementEnd
