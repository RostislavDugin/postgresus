-- +goose Up
ALTER TABLE intervals ADD COLUMN cron_expression TEXT;

-- +goose Down
ALTER TABLE intervals DROP COLUMN cron_expression;
