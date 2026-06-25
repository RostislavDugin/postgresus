-- +goose Up
-- +goose StatementBegin

-- --- Drop legacy WAL / agent-token properties (forward-only) ---
DROP INDEX IF EXISTS idx_backups_pg_wal_segment_name;
DROP INDEX IF EXISTS idx_backups_pg_wal_backup_type_created;

ALTER TABLE backups
    DROP COLUMN IF EXISTS pg_wal_backup_type,
    DROP COLUMN IF EXISTS pg_wal_start_segment,
    DROP COLUMN IF EXISTS pg_wal_stop_segment,
    DROP COLUMN IF EXISTS pg_version,
    DROP COLUMN IF EXISTS pg_wal_segment_name,
    DROP COLUMN IF EXISTS upload_completed_at;

UPDATE postgresql_databases SET host     = 'localhost'    WHERE host     IS NULL OR host     = '';
UPDATE postgresql_databases SET port     = 5432           WHERE port     IS NULL OR port     = 0;
UPDATE postgresql_databases SET username = 'postgres'     WHERE username IS NULL OR username = '';
UPDATE postgresql_databases SET password = 'stubpassword' WHERE password IS NULL OR password = '';

ALTER TABLE postgresql_databases
    DROP COLUMN IF EXISTS backup_type;

ALTER TABLE postgresql_databases
    ALTER COLUMN host     SET NOT NULL,
    ALTER COLUMN port     SET NOT NULL,
    ALTER COLUMN username SET NOT NULL,
    ALTER COLUMN password SET NOT NULL;

DROP INDEX IF EXISTS idx_databases_agent_token;

ALTER TABLE databases
    DROP COLUMN IF EXISTS agent_token,
    DROP COLUMN IF EXISTS is_agent_token_generated;

-- --- Rename tables for logical/physical split ---
ALTER TABLE backup_configs       RENAME TO logical_backup_configs;
ALTER TABLE backups              RENAME TO logical_backups;
ALTER TABLE postgresql_databases RENAME TO postgresql_logical_databases;

UPDATE databases SET type = 'POSTGRES_LOGICAL' WHERE type = 'POSTGRES';

ALTER TABLE postgresql_logical_databases
    RENAME CONSTRAINT postgresql_databases_pkey TO postgresql_logical_databases_pkey;

ALTER TABLE postgresql_logical_databases
    RENAME CONSTRAINT uk_postgresql_databases_database_id TO uk_postgresql_logical_databases_database_id;

ALTER INDEX idx_postgresql_databases_database_id RENAME TO idx_postgresql_logical_databases_database_id;

-- --- Physical database connection table ---
CREATE TABLE postgresql_physical_databases (
    id                       UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    database_id              UUID,
    version                  TEXT NOT NULL,
    host                     TEXT NOT NULL,
    port                     INT NOT NULL,
    username                 TEXT NOT NULL,
    password                 TEXT NOT NULL,
    ssl_mode                 TEXT NOT NULL DEFAULT 'disable',
    ssl_client_cert          TEXT NOT NULL DEFAULT '',
    ssl_client_key           TEXT NOT NULL DEFAULT '',
    ssl_root_cert            TEXT NOT NULL DEFAULT '',
    replication_slot_name    TEXT NOT NULL,
    system_identifier        TEXT,
    backup_type              TEXT NOT NULL DEFAULT 'FULL',
    wal_segment_size_bytes   BIGINT
);

ALTER TABLE postgresql_physical_databases
    ADD CONSTRAINT uk_postgresql_physical_databases_database_id
    UNIQUE (database_id);

ALTER TABLE postgresql_physical_databases
    ADD CONSTRAINT fk_postgresql_physical_databases_database_id
    FOREIGN KEY (database_id)
    REFERENCES databases (id)
    ON DELETE CASCADE;

CREATE INDEX idx_postgresql_physical_databases_database_id
    ON postgresql_physical_databases (database_id);

-- --- Physical backup configuration ---
CREATE TABLE physical_backup_configs (
    database_id                    UUID PRIMARY KEY,

    is_backups_enabled             BOOLEAN NOT NULL DEFAULT FALSE,

    full_interval_type             TEXT NOT NULL DEFAULT '',
    full_time_of_day               TEXT,
    full_weekday                   INT,
    full_day_of_month              INT,
    full_cron_expression           TEXT,

    incremental_interval_type      TEXT NOT NULL DEFAULT '',
    incremental_time_of_day        TEXT,
    incremental_weekday            INT,
    incremental_day_of_month       INT,
    incremental_cron_expression    TEXT,

    retention                              TEXT NOT NULL DEFAULT 'FULL_BACKUPS',

    chains_retention_count                 INT  NOT NULL DEFAULT 0,

    full_backups_retention_policy          TEXT NOT NULL DEFAULT '',
    full_backups_retention_count           INT  NOT NULL DEFAULT 0,
    full_backups_retention_gfs_hours       INT  NOT NULL DEFAULT 0,
    full_backups_retention_gfs_days        INT  NOT NULL DEFAULT 0,
    full_backups_retention_gfs_weeks       INT  NOT NULL DEFAULT 0,
    full_backups_retention_gfs_months      INT  NOT NULL DEFAULT 0,
    full_backups_retention_gfs_years       INT  NOT NULL DEFAULT 0,

    wal_lag_threshold_bytes        BIGINT NOT NULL DEFAULT 0,

    storage_id                     UUID,
    encryption                     TEXT NOT NULL DEFAULT 'NONE',
    send_notifications_on          TEXT NOT NULL DEFAULT '',

    force_full_requested_at        TIMESTAMPTZ,
    force_incremental_requested_at TIMESTAMPTZ
);

ALTER TABLE physical_backup_configs
    ADD CONSTRAINT fk_physical_backup_configs_database_id
    FOREIGN KEY (database_id)
    REFERENCES databases (id)
    ON DELETE CASCADE;

ALTER TABLE physical_backup_configs
    ADD CONSTRAINT fk_physical_backup_configs_storage_id
    FOREIGN KEY (storage_id)
    REFERENCES storages (id);

-- --- Physical full backups ---
CREATE TABLE physical_full_backups (
    id                        UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    database_id               UUID NOT NULL,
    storage_id                UUID NOT NULL,
    timeline_id               INT  NOT NULL,
    status                    TEXT NOT NULL,
    error_reason              TEXT,
    file_name                 TEXT,
    start_lsn                 PG_LSN,
    stop_lsn                  PG_LSN,
    backup_size_mb            DOUBLE PRECISION,
    raw_size_mb               DOUBLE PRECISION,
    backup_duration_ms        BIGINT,
    compression               TEXT NOT NULL DEFAULT 'ZSTD',
    encryption                TEXT NOT NULL DEFAULT 'NONE',
    encryption_salt           TEXT,
    encryption_iv             TEXT,
    manifest_file_name        TEXT,
    manifest_encryption_salt  TEXT,
    manifest_encryption_iv    TEXT,
    created_at                TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    completed_at              TIMESTAMPTZ
);

ALTER TABLE physical_full_backups
    ADD CONSTRAINT fk_physical_full_backups_database_id
    FOREIGN KEY (database_id)
    REFERENCES databases (id)
    ON DELETE CASCADE;

ALTER TABLE physical_full_backups
    ADD CONSTRAINT fk_physical_full_backups_storage_id
    FOREIGN KEY (storage_id)
    REFERENCES storages (id);

CREATE INDEX idx_physical_full_backups_database_id_status_created_at
    ON physical_full_backups (database_id, status, created_at DESC);

CREATE INDEX idx_physical_full_backups_database_id_timeline_id
    ON physical_full_backups (database_id, timeline_id);

-- --- Physical incremental backups ---
CREATE TABLE physical_incremental_backups (
    id                            UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    database_id                   UUID NOT NULL,
    storage_id                    UUID NOT NULL,
    timeline_id                   INT  NOT NULL,
    status                        TEXT NOT NULL,
    error_reason                  TEXT,
    file_name                     TEXT,
    start_lsn                     PG_LSN,
    stop_lsn                      PG_LSN,
    backup_size_mb                DOUBLE PRECISION,
    raw_size_mb                   DOUBLE PRECISION,
    backup_duration_ms            BIGINT,
    compression                   TEXT NOT NULL DEFAULT 'ZSTD',
    encryption                    TEXT NOT NULL DEFAULT 'NONE',
    encryption_salt               TEXT,
    encryption_iv                 TEXT,
    manifest_file_name            TEXT,
    manifest_encryption_salt      TEXT,
    manifest_encryption_iv        TEXT,
    created_at                    TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    completed_at                  TIMESTAMPTZ,
    root_full_backup_id           UUID NOT NULL,
    parent_incremental_backup_id  UUID
);

ALTER TABLE physical_incremental_backups
    ADD CONSTRAINT fk_physical_incremental_backups_database_id
    FOREIGN KEY (database_id)
    REFERENCES databases (id)
    ON DELETE CASCADE;

ALTER TABLE physical_incremental_backups
    ADD CONSTRAINT fk_physical_incremental_backups_storage_id
    FOREIGN KEY (storage_id)
    REFERENCES storages (id);

ALTER TABLE physical_incremental_backups
    ADD CONSTRAINT fk_physical_incremental_backups_root_full_backup_id
    FOREIGN KEY (root_full_backup_id)
    REFERENCES physical_full_backups (id)
    ON DELETE RESTRICT;

ALTER TABLE physical_incremental_backups
    ADD CONSTRAINT fk_physical_incremental_backups_parent_incremental_backup_id
    FOREIGN KEY (parent_incremental_backup_id)
    REFERENCES physical_incremental_backups (id)
    ON DELETE RESTRICT;

CREATE INDEX idx_physical_incremental_backups_root_full_start_lsn
    ON physical_incremental_backups (root_full_backup_id, start_lsn);

CREATE INDEX idx_physical_incremental_backups_database_id_status_created_at
    ON physical_incremental_backups (database_id, status, created_at DESC);

CREATE INDEX idx_physical_incremental_backups_parent_incremental_backup_id
    ON physical_incremental_backups (parent_incremental_backup_id);

-- --- In-flight backup claims ---
CREATE TABLE physical_in_flight_backups (
    database_id  UUID        PRIMARY KEY,
    backup_type  TEXT        NOT NULL,
    backup_id    UUID        NOT NULL,
    claimed_at   TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    node_id      UUID
);

-- node_id records which backup node owns the in-flight backup, so a restarted
-- primary can tell a still-running backup (live owner) from an orphaned one
-- (dead owner) instead of failing every IN_PROGRESS row blindly. Nullable: a
-- claim exists briefly before the node is assigned, and pre-existing rows have
-- no owner — both are treated as "unknown owner" and failed on recovery.

ALTER TABLE physical_in_flight_backups
    ADD CONSTRAINT chk_physical_in_flight_backups_backup_type
    CHECK (backup_type IN ('FULL', 'INCREMENTAL'));

ALTER TABLE physical_in_flight_backups
    ADD CONSTRAINT fk_physical_in_flight_backups_database_id
    FOREIGN KEY (database_id)
    REFERENCES databases (id)
    ON DELETE CASCADE;

-- --- WAL segments ---
CREATE TABLE physical_wal_segments (
    id                 UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    database_id        UUID NOT NULL,
    storage_id         UUID NOT NULL,
    timeline_id        INT  NOT NULL,
    file_name          TEXT,
    wal_filename       TEXT   NOT NULL,
    start_lsn          PG_LSN NOT NULL,
    end_lsn            PG_LSN NOT NULL,
    compressed_size_mb DOUBLE PRECISION NOT NULL DEFAULT 0,
    received_at        TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    claimed_at         TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    encryption         TEXT NOT NULL DEFAULT 'NONE',
    encryption_salt    TEXT,
    encryption_iv      TEXT
);

ALTER TABLE physical_wal_segments
    ADD CONSTRAINT fk_physical_wal_segments_database_id
    FOREIGN KEY (database_id)
    REFERENCES databases (id)
    ON DELETE CASCADE;

ALTER TABLE physical_wal_segments
    ADD CONSTRAINT fk_physical_wal_segments_storage_id
    FOREIGN KEY (storage_id)
    REFERENCES storages (id);

ALTER TABLE physical_wal_segments
    ADD CONSTRAINT uk_physical_wal_segments_database_id_timeline_id_start_lsn
    UNIQUE (database_id, timeline_id, start_lsn);

CREATE INDEX idx_physical_wal_segments_database_id_received_at
    ON physical_wal_segments (database_id, received_at);

-- --- WAL history files ---
CREATE TABLE physical_wal_history_files (
    id                  UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    database_id         UUID NOT NULL,
    storage_id          UUID NOT NULL,
    timeline_id         INT  NOT NULL,
    file_name           TEXT NOT NULL,
    history_filename    TEXT NOT NULL,
    compressed_size_mb  DOUBLE PRECISION NOT NULL DEFAULT 0,
    created_at          TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

ALTER TABLE physical_wal_history_files
    ADD CONSTRAINT fk_physical_wal_history_files_database_id
    FOREIGN KEY (database_id)
    REFERENCES databases (id)
    ON DELETE CASCADE;

ALTER TABLE physical_wal_history_files
    ADD CONSTRAINT fk_physical_wal_history_files_storage_id
    FOREIGN KEY (storage_id)
    REFERENCES storages (id);

ALTER TABLE physical_wal_history_files
    ADD CONSTRAINT uk_physical_wal_history_files_database_id_timeline_id
    UNIQUE (database_id, timeline_id);

-- --- WAL streamers ---
CREATE TABLE physical_wal_streamers (
    database_id        UUID PRIMARY KEY,
    started_at         TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    last_heartbeat_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    status             TEXT NOT NULL
);

ALTER TABLE physical_wal_streamers
    ADD CONSTRAINT fk_physical_wal_streamers_database_id
    FOREIGN KEY (database_id)
    REFERENCES databases (id)
    ON DELETE CASCADE;

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
SELECT 'no-op: physical backup + logical/physical split is forward-only';
-- +goose StatementEnd
