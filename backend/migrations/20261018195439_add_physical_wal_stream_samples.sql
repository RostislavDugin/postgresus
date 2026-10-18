-- +goose Up
-- +goose StatementBegin

-- One row per WAL-streaming database every 30 seconds, written by the WAL
-- streamer's lag monitor. Source-side columns are NULL whenever source_state is
-- not REACHABLE, so a NULL slot_safe_wal_size_bytes on a reachable row means the
-- source sets no limit.
CREATE TABLE physical_wal_stream_samples (
    id                         UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    database_id                UUID        NOT NULL,
    sampled_at                 TIMESTAMPTZ NOT NULL,
    session_started_at         TIMESTAMPTZ NOT NULL,
    source_state               TEXT        NOT NULL,
    source_unreachable_since   TIMESTAMPTZ,
    source_current_lsn         PG_LSN,
    archive_progress_lsn       PG_LSN,
    source_written_bytes       BIGINT,
    archived_segments          INTEGER     NOT NULL,
    archived_wal_bytes         BIGINT      NOT NULL,
    archived_stored_bytes      BIGINT      NOT NULL,
    wal_segment_size_bytes     BIGINT      NOT NULL,
    highest_concurrent_uploads INTEGER     NOT NULL,
    segment_write_avg_millis   INTEGER,
    segment_write_max_millis   INTEGER,
    slot_wal_status            TEXT,
    slot_extra_wal_since       TIMESTAMPTZ,
    slot_lag_bytes             BIGINT,
    slot_safe_wal_size_bytes   BIGINT,
    unarchived_wal_bytes       BIGINT,
    archive_stalled_since      TIMESTAMPTZ,
    local_queue_bytes          BIGINT      NOT NULL,
    local_queue_pause_bytes    BIGINT      NOT NULL,
    receiver_restarts          INTEGER     NOT NULL,
    receiver_pauses            INTEGER     NOT NULL,
    is_receiver_running        BOOLEAN     NOT NULL,
    is_forced_rotation_denied  BOOLEAN     NOT NULL
);

ALTER TABLE physical_wal_stream_samples
    ADD CONSTRAINT fk_physical_wal_stream_samples_database_id
    FOREIGN KEY (database_id)
    REFERENCES databases (id)
    ON DELETE CASCADE;

CREATE INDEX idx_physical_wal_stream_samples_database_id_sampled_at
    ON physical_wal_stream_samples (database_id, sampled_at);

-- The retention cleaner deletes expired samples of every database in one
-- statement, including databases that no longer stream.
CREATE INDEX idx_physical_wal_stream_samples_sampled_at
    ON physical_wal_stream_samples (sampled_at);

-- A session that fails before its first sample leaves no sample behind, so the
-- first failure of a crash loop is kept here until a session samples again.
ALTER TABLE physical_wal_streamers
    ADD COLUMN failing_since TIMESTAMPTZ;

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin

ALTER TABLE physical_wal_streamers
    DROP COLUMN IF EXISTS failing_since;

DROP INDEX IF EXISTS idx_physical_wal_stream_samples_sampled_at;
DROP INDEX IF EXISTS idx_physical_wal_stream_samples_database_id_sampled_at;

ALTER TABLE physical_wal_stream_samples
    DROP CONSTRAINT IF EXISTS fk_physical_wal_stream_samples_database_id;

DROP TABLE IF EXISTS physical_wal_stream_samples;

-- +goose StatementEnd
