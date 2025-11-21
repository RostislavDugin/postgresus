-- +goose Up
-- +goose StatementBegin

ALTER TABLE postgresql_databases
    DROP CONSTRAINT IF EXISTS fk_postgresql_databases_restore_id;

DROP INDEX IF EXISTS idx_postgresql_databases_restore_id;

ALTER TABLE postgresql_databases
    DROP COLUMN IF EXISTS restore_id;

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin

ALTER TABLE postgresql_databases
    ADD COLUMN restore_id UUID;

CREATE INDEX idx_postgresql_databases_restore_id ON postgresql_databases (restore_id);

ALTER TABLE postgresql_databases
    ADD CONSTRAINT fk_postgresql_databases_restore_id
    FOREIGN KEY (restore_id)
    REFERENCES restores (id)
    ON DELETE CASCADE;

-- +goose StatementEnd
