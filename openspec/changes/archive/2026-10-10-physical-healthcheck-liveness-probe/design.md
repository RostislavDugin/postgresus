## Context

See proposal.md for the motivation and the reproduction.

The healthcheck reaches the source through one chain, and every link except the last is shared with the user-facing connection test:

- `CheckDatabaseHealthUseCase.healthcheckDatabase` calls `DatabaseService.TestTrustedDatabaseConnection` (`backend/internal/features/healthcheck/attempt/check_database_health_uc.go:157`), declared on the healthcheck's own interface (`backend/internal/features/healthcheck/attempt/interfaces.go:24`) and mocked in `backend/internal/features/healthcheck/attempt/mocks.go:30`.
- `TestTrustedDatabaseConnection` (`backend/internal/features/databases/service.go:488`) is called by the healthcheck only. It delegates to the private `testDatabaseConnection` (`service.go:965`), which opens the SSH tunnel when one is configured and calls `Database.TestConnection` on the tunneled copy. The workspace-authorized connection test shares `testDatabaseConnection` (`service.go:485`).
- `Database.TestConnection` (`backend/internal/features/databases/model.go:108`) dispatches by type. For a physical source it calls `TestReplicationConnection` (`backend/internal/features/databases/databases/postgresql/physical/model.go:306`), which runs `checkReplicationReadiness` (`model.go:836`). That check opens a replication connection, then an ordinary one, and on it runs up to four queries covering seven checks: custom tablespaces, `wal_level`, `max_wal_senders`, `max_replication_slots` and `summarize_wal` from one settings read, the `pg_switch_wal` privilege (WAL streaming only), and the cluster system identifier (only when one is stored).

An ordinary connection to the source already exists as `openConn` (`backend/internal/features/databases/databases/postgresql/physical/connection.go:18`). It is the same connection the readiness check opens with the same credentials and TLS files, so any source that passes readiness today also accepts it.

Constraints from `backend/AGENTS.md`: test names follow `Test_WhatWeDo_WhichConditions_WhatWeExpect`; tests live next to their source, not in `internal/features/tests/`, which is reserved for backup and restore cycles; errors go to logs as key-value pairs, and passwords never do.

## Goals / Non-Goals

**Goals:**

- A physical source healthcheck that cannot be turned red by WAL sender exhaustion and cannot exhaust WAL senders itself.
- The probe's failure reason reaches the existing `database healthcheck probe failed` log entry.

**Non-Goals:**

- Any change to the probe of other database types, which keep running their full `TestConnection`.
- Any change to the readiness check or to its error classification.

## Decisions

### Decision 1: The physical healthcheck probes liveness with an ordinary connection and `SELECT 1`

A new `PostgresqlPhysicalDatabase.ProbeLiveness(ctx, logger, encryptor)` in `postgresql/physical/model.go`, next to `TestReplicationConnection`, opens the connection through `openConn`, runs `SELECT 1`, and closes the connection. It bounds itself with a 30-second timeout derived from the caller's context, matching the readiness check's budget, so a hung source cannot stall a healthcheck goroutine past the next tick.

Alternatives rejected:

- **Keep the replication probe and treat the `max_wal_senders` refusal as available.** PostgreSQL refuses a surplus WAL sender before it authenticates the client, so such a pass would not even prove the credentials still work. The probe would also keep taking a WAL sender slot when one is free, which is the half of the problem that can fail a backup.
- **Skip the healthcheck while a backup or WAL stream runs.** Continuous WAL streaming runs permanently, so the healthcheck would never run on a streaming database. It would also couple the healthcheck to backup scheduling state it does not own today.
- **Keep the readiness checks but run them over the ordinary connection only.** This drops the WAL sender, but keeps re-validating configuration every minute. A transient failure of any of the inspection queries, or a configuration change already reported through the backups, would still paint the database red. The healthcheck exists to answer whether the source is up. Configuration drift that matters breaks the next backup, which notifies on its own.

### Decision 2: The type dispatch gains `Database.ProbeLiveness`, which overrides only the physical case

`Database.ProbeLiveness(ctx, logger, encryptor)` calls `PostgresqlPhysical.ProbeLiveness` for a physical source, with the same nil-config guard `TestConnection` has ("postgresql physical config is not set"). For every other type it runs the full `Database.TestConnection`, so for those types "liveness" still means today's connection test, privileges included. The method's doc comment says so, so the name does not overpromise. The service's trusted path calls `ProbeLiveness` instead of `TestConnection`, keeping the tunnel handling and the embedded-target refusal that `OpenTunnel` already applies (`backend/internal/features/databases/tunneled_database.go:39`).

Alternatives rejected:

- **Branch on the physical type inside the service method.** The model already owns per-type dispatch (`model.go:108`). A second, partial dispatch in the service would be the one place a reader must remember to check when a type is added.
- **Give every type its own lightweight liveness probe.** Logical PostgreSQL, MySQL, MariaDB and MongoDB have no reported false alarms and no shared replication capacity to protect. Changing them would widen the change with no observed benefit.

### Decision 3: Name the healthcheck path `Probe...`, not `Test...`

`TestTrustedDatabaseConnection` becomes `ProbeDatabaseLiveness`, and the two new model methods are `ProbeLiveness`, not `TestLiveness`. After this change the path no longer tests a connection the way the user-facing "Test connection" does, and a `Test...` name beside `TestConnection` would invite a future caller to use it as one. The healthcheck interface, its mock, the use case and its tests are renamed in the same change. The old name is deleted, per the project rule against unrequested backward compatibility. The private `testDatabaseConnection` stays for the workspace-authorized path, and the trusted path gets its own short body: open the tunnel, call `ProbeLiveness`.

Alternative rejected:

- **Keep the name.** That makes a smaller diff, but the name would then describe behavior the method no longer has, and the healthcheck is its only caller, so the rename touches one feature.

### Decision 4: The probe returns the driver error unclassified

`ProbeLiveness` returns the `pgx` connect or query error wrapped with context. It does not map it to a `ConnectionTestError` code, because no UI consumes the healthcheck's error, and only the log does. The use case already logs it as `"error", err`. `pgx` reports a connect failure as ``failed to connect to `user=... database=...` `` followed by the server's message, without the password, which keeps the spec's no-password requirement without ad hoc redaction.

Alternative rejected:

- **Reuse `classifyReplicationConnectError`.** Its default branch reduces most failures to a bare `CONNECTION_FAILED` code. That is the opacity that made this bug hard to diagnose.

### Decision 5: Pin the behavior with a source whose WAL senders are exhausted

The physical PostgreSQL test container gains a `WithMaxWalSenders(n)` option beside the existing `WithMaxConnections` (`backend/internal/util/testing/containers/postgres_physical.go`). The test starts a source with `max_wal_senders=1`, holds that one slot with an open replication connection, and asserts two things: `ProbeLiveness` succeeds, and `TestReplicationConnection` fails. The second assertion proves the fixture really exhausted the slots, so the first one is meaningful. A probe that took a WAL sender could not pass this test, which also covers the "backup starts while the healthcheck runs" scenario without a timing race.

Alternative rejected:

- **Reproduce with a real `pg_basebackup`.** That is slower and timing-dependent, and it proves nothing beyond what holding the slot proves.

## Risks / Trade-offs

- [The healthcheck stops noticing a removed `host replication` rule or a revoked `REPLICATION` attribute] → The next backup or WAL stream reconnect fails and sends its own failure notification. The user-facing connection test still reports both.
- [A source that allows the ordinary connection but not replication now shows green while backups fail] → Same mitigation. The healthcheck promises reachability, and backup health has its own status and notifications.
- [The healthcheck stops noticing a source swapped for a different cluster, for example after a DNS change, because it no longer compares the cluster system identifier (`backend/internal/features/databases/databases/postgresql/physical/model.go:910-920`); it also stops noticing drift in `wal_level` or `summarize_wal` and a newly created custom tablespace] → Backup preflight still compares the identity and refuses to write a backup on a mismatch, pausing scheduling and reporting `SYSTEM_IDENTIFIER_MISMATCH` (`openspec/specs/postgresql-physical-backup-failover/spec.md`). Configuration drift fails the next backup, which notifies.
- [The rename breaks an out-of-tree caller] → The healthcheck is the only caller in the repository, and the method is internal to the backend binary.

## Migration Plan

No data or schema migration. Healthcheck attempts recorded before the deploy stay as they are, and the next tick after the deploy uses the new probe. Rollback is a plain revert.
