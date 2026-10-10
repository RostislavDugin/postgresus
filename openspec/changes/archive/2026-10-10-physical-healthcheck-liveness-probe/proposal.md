## Why

The scheduled healthcheck of a PostgreSQL physical database turns red while a backup is running, even though the backup itself succeeds. The probe runs the full replication-readiness check, and that check opens a replication connection, which takes a WAL sender slot exactly as `pg_basebackup` and `pg_receivewal` do. On a source whose `max_wal_senders` leaves no spare slot during a backup, PostgreSQL rejects the probe with `number of requested standby connections exceeds "max_wal_senders"`, and the attempt is recorded as unavailable.

This was reproduced on PostgreSQL 17 with `max_wal_senders=2`: the probe passed with only `pg_receivewal` attached, failed while `pg_basebackup` ran, and the backup exited 0. The same contention also works in reverse: a probe that holds the last slot when a backup or a `pg_receivewal` restart begins makes the backup fail. A false alarm that can also break the thing it watches is worth fixing now.

## What Changes

- The scheduled healthcheck of a physical database checks liveness only: it opens an ordinary connection with the stored credentials and runs a trivial query. It no longer opens a replication connection and no longer re-validates the source's replication configuration.
- The full replication-readiness check stays where it is used today, in the user-facing connection test and when a database is created or updated.
- The healthcheck of logical PostgreSQL, MySQL, MariaDB and MongoDB databases is unchanged.
- When the liveness probe fails, the underlying error reaches the healthcheck log instead of being reduced to a bare code.
- **BREAKING (internal only):** the database service method the healthcheck calls is renamed from `TestTrustedDatabaseConnection` to `ProbeDatabaseLiveness`, along with the healthcheck's service interface and mock. The healthcheck is its only caller; the old name is deleted, not kept as an alias. No HTTP API, schema or configuration changes.

## Capabilities

### New Capabilities

- `physical-backups/source-healthcheck`: what the scheduled healthcheck of a PostgreSQL physical source checks, and that it never competes with backups for replication capacity on the source.

### Modified Capabilities

None. The `internal-postgresql-protection` requirement that healthchecks use a trusted internal path and still run embedded-target protection keeps holding unchanged.

## Out of scope

- Changing the healthcheck of any database type other than PostgreSQL physical.
- Changing the user-facing connection test or the readiness checks run on create and update.
- Preserving the underlying error in the readiness check's connect-error classifier, which today reduces most failures to a bare code. That is a separate observability fix for a follow-up change.
- A dedicated error code or UI hint for an exhausted `max_wal_senders` in the connection test.
- Storing the failure reason on healthcheck attempts or showing it in the attempts UI.
- Notifications, healthcheck intervals, and the attempts-before-down logic.

## Impact

- Backend only: `backend/internal/features/databases` (database model and service), `backend/internal/features/databases/databases/postgresql/physical` (new liveness probe), `backend/internal/features/healthcheck/attempt` (interface, mock, use case and its tests), and the physical PostgreSQL test container helper, which gains a `max_wal_senders` option.
- Answers to `AGENTS.md` and `backend/AGENTS.md`.
- Trade-off: the healthcheck stops noticing a removed `host replication` rule in `pg_hba.conf`, a revoked `REPLICATION` attribute, drift in `wal_level` or `summarize_wal`, a new custom tablespace, and a source swapped for a different cluster. Backup preflight still refuses a cluster whose system identifier changed, and the rest surface as failed backups with their own notifications.
