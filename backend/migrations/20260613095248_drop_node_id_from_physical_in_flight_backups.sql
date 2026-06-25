-- +goose Up
ALTER TABLE physical_in_flight_backups
    DROP COLUMN node_id;

-- +goose Down
ALTER TABLE physical_in_flight_backups
    ADD COLUMN node_id UUID;
