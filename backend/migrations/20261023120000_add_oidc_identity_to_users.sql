-- +goose Up
-- +goose StatementBegin

-- One OpenID Connect provider per instance, but the subject is only unique
-- within its issuer, so an operator who repoints the instance at another
-- provider cannot have the new provider's subjects collide with the old ones.
ALTER TABLE users
    ADD COLUMN oidc_issuer  TEXT,
    ADD COLUMN oidc_subject TEXT;

ALTER TABLE users
    ADD CONSTRAINT ck_users_oidc_identity_complete
    CHECK ((oidc_issuer IS NULL) = (oidc_subject IS NULL));

CREATE UNIQUE INDEX idx_users_oidc_identity
    ON users (oidc_issuer, oidc_subject)
    WHERE oidc_issuer IS NOT NULL;

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin

DROP INDEX IF EXISTS idx_users_oidc_identity;

ALTER TABLE users
    DROP CONSTRAINT IF EXISTS ck_users_oidc_identity_complete;

ALTER TABLE users
    DROP COLUMN IF EXISTS oidc_subject,
    DROP COLUMN IF EXISTS oidc_issuer;

-- +goose StatementEnd
