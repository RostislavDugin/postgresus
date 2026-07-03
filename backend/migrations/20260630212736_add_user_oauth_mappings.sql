-- +goose Up
-- +goose StatementBegin

CREATE TABLE user_oauth_mappings (
    id         UUID        NOT NULL DEFAULT gen_random_uuid(),
    user_id    UUID        NOT NULL,
    provider   TEXT        NOT NULL,
    oauth_id   TEXT        NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

ALTER TABLE user_oauth_mappings
    ADD CONSTRAINT pk_user_oauth_mappings
    PRIMARY KEY (id);

ALTER TABLE user_oauth_mappings
    ADD CONSTRAINT fk_user_oauth_mappings_user_id
    FOREIGN KEY (user_id)
    REFERENCES users (id)
    ON DELETE CASCADE;

CREATE UNIQUE INDEX idx_user_oauth_mappings_unique
    ON user_oauth_mappings (user_id, provider, oauth_id);

INSERT INTO user_oauth_mappings (user_id, provider, oauth_id, created_at)
SELECT id, 'github', github_oauth_id, NOW()
FROM users
WHERE github_oauth_id IS NOT NULL;

INSERT INTO user_oauth_mappings (user_id, provider, oauth_id, created_at)
SELECT id, 'google', google_oauth_id, NOW()
FROM users
WHERE google_oauth_id IS NOT NULL;

ALTER TABLE users
    DROP COLUMN github_oauth_id,
    DROP COLUMN google_oauth_id;

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin

ALTER TABLE users
    ADD COLUMN github_oauth_id TEXT,
    ADD COLUMN google_oauth_id TEXT;

UPDATE users
SET github_oauth_id = m.oauth_id
FROM user_oauth_mappings m
WHERE m.user_id = users.id AND m.provider = 'github';

UPDATE users
SET google_oauth_id = m.oauth_id
FROM user_oauth_mappings m
WHERE m.user_id = users.id AND m.provider = 'google';

CREATE UNIQUE INDEX idx_users_github_oauth_id
    ON users (github_oauth_id)
    WHERE github_oauth_id IS NOT NULL;

CREATE UNIQUE INDEX idx_users_google_oauth_id
    ON users (google_oauth_id)
    WHERE google_oauth_id IS NOT NULL;

DROP TABLE user_oauth_mappings;

-- +goose StatementEnd
