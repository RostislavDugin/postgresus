## Why

The backend suite fails in CI often enough that a red run no longer means a regression: over the last two months most failed attempts passed on a rerun of the same commit. Three local full runs on an idle machine passed, so these failures depend on runner load rather than on the code under test. Two of the causes are also defects users see. A restore or backup whose database tool exits quickly can lose the tool's error output, and a backup whose storage fails at once reports `context canceled` instead of the storage error.

## What Changes

- Test containers for the SSH bastion, Samba and FTP count as ready only once the service inside answers its protocol, not as soon as Docker's proxy accepts on the mapped port. Today they wait for an open port only, which produced about a dozen `connection reset by peer` and `EOF` failures, plus the FTP `connection refused` after an egress delay is added.
- Every place that runs a database client tool hands the tool's error output to a buffer that the process wait itself fills to the end, with a bounded wait for that output. Today restores for all four engines, logical backups for all four engines and the WAL receiver read the output through a pipe in a goroutine and wait for the process first, and waiting closes the pipe the output is still in. Restores and backups then report a failure without the tool's message, and the WAL receiver can treat a fatal error as one worth retrying.
- A PostgreSQL restore whose tool exits before reading the whole backup reports the tool's error instead of the failure to pass the remaining data to it.
- A logical backup whose storage fails at once, before reading any backup data, reports the storage error. Today the backup context is already cancelled when the tool starts, so the backup fails with `start <tool>: context canceled`.
- Tests that wait a fixed time for another goroutine stop doing so: the telemetry test that calls `Run` twice, and the idle-timeout reader tests in the verification agent.
- The container's supervisor writes `Databasus application exited with status <n>` to standard error, the same stream PostgreSQL logs to, so the Docker filesystem suite's check that this line precedes PostgreSQL's shutdown request stops depending on how Docker interleaves two streams.

No caller, schema, configuration or API breaks.

## Out of scope

- The TimescaleDB restore failure `schema "public" already exists`. Its cause is unknown and needs its own investigation.
- `Test_WalStream_WhenSourceIsIdle_KeepsReceiverAliveAndDoesNotRebuildSlot`, which failed once because the receiver dropped its replication connection on an idle source. Nothing points to a test-only cause, so it needs its own investigation before anything is changed.
- Negative assertions that sleep before checking that nothing happened, in scheduler, notification and disk-watch tests. They pass reliably today and may miss bugs, but they are not flaky.
- Fixed sleeps in test cleanup that wait for cascading deletes.
- The 10-second budget for `docker stop` in the Docker filesystem suite, which has never failed in CI.
- CI workflow changes, reruns and retry wrappers around flaky tests.

## Capabilities

### New Capabilities

- `database-tool-failure-output`: what a restore and the WAL receiver do with the error output of the database tool they run when that tool fails.

### Modified Capabilities

- `logical-backup-outcome`: the fail message of a logical backup includes the dump tool's error output, and a storage that fails before reading any backup data is reported as the storage error.

## Impact

- Backend restore usecases for PostgreSQL, MySQL, MariaDB and MongoDB, logical backup usecases for the same engines, and the WAL receiver of physical backups.
- The error-output pipe and its goroutine in each of those places, which give way to a buffer plus a bounded wait. The WAL receiver drops its use of the physical backup package's capture helper, which stays for `pg_basebackup`.
- Backend test containers for the SSH bastion, Samba and FTP.
- The telemetry background service test and the verification agent's idle-timeout reader tests.
- `docker/start.sh` and the `clean-shutdown` case of the Docker filesystem suite.

The change answers to the root `AGENTS.md`, `backend/AGENTS.md` and `agent/verification/AGENTS.md`.
