-- +goose Up
-- +goose StatementBegin
ALTER TABLE backup_configs DROP COLUMN cpu_count;
-- +goose StatementEnd

-- +goose StatementBegin
ALTER TABLE postgresql_databases ADD COLUMN cpu_count INT NOT NULL DEFAULT 1;
-- +goose StatementEnd

-- +goose StatementBegin
ALTER TABLE mongodb_databases ADD COLUMN cpu_count INT NOT NULL DEFAULT 1;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
ALTER TABLE backup_configs ADD COLUMN cpu_count INT NOT NULL DEFAULT 1;
-- +goose StatementEnd

-- +goose StatementBegin
ALTER TABLE postgresql_databases DROP COLUMN cpu_count;
-- +goose StatementEnd

-- +goose StatementBegin
ALTER TABLE mongodb_databases DROP COLUMN cpu_count;
-- +goose StatementEnd
