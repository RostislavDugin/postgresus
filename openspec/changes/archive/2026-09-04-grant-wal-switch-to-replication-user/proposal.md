## Why

Databasus offers to provision a replication-only PostgreSQL user, then relies on a privilege it never grants that user. `CreateReplicationOnlyUser` (`backend/internal/features/databases/databases/postgresql/physical/model.go:723`) issues `CREATE USER ... LOGIN` plus a replication grant and stops there. Meanwhile `runForcedWalRotation` (`backend/internal/features/backups/backups/usecases/physical/postgresql/wal_rotation.go:70`) calls `pg_switch_wal()` every five minutes, and PostgreSQL restricts that function to superusers by default. The provisioned user gets `ERROR: permission denied for function pg_switch_wal (SQLSTATE 42501)`, the rotation loop logs one warning and returns for the lifetime of the streamer.

The degradation is real and silent, and it falls on exactly one feature. A FULL backup does not need the privilege: `pg_basebackup` runs with `--wal-method=fetch` (`pg_basebackup.go:69`), so the WAL between `start_lsn` and `stop_lsn` travels inside the same tar, and a restore to a specific backup ships no archived WAL at all (`chain_view/restore_set.go:83`). Incrementals inherit that. What breaks is recovery to a point in time. `ResolveRestoreSet` replays the contiguous run of archived segments starting at the last backup's `stop_lsn` (`restore_set.go:61`) and refuses any target beyond it with `WalGapBeforeTargetError`. A segment reaches storage only once finalized, so on a source that writes a few bytes an hour the newest WAL sits in `pg_wal` until the 16 MB segment fills. The user can restore the FULL and nothing after it, for hours or days. Bounding exactly that is why `DefaultForcedRotationInterval` exists (`wal_rotation.go:22`).

So the product currently accepts a configuration it knows cannot deliver what that configuration promises. A user who picks `FULL_INCREMENTAL_WAL_STREAM` and accepts Databasus's own offer to create a user ends up with a worse recovery point than a user who pastes in superuser credentials, and nothing in the interface says so. The only signal arrives later, as a notification, and only when `CHAIN_BROKEN` is among the database's enabled notification kinds (`wal_supervisor.go:453`), so a user who declined that notification learns nothing at all.

The streaming side itself behaves as designed and stays as it is. `runForcedWalRotation` recognizes SQLSTATE `42501`, stops the loop and reports through `buildChainRiskAlert` (`wal_supervisor.go:425`), which already names both remedies. Two gaps remain: provisioning never issues the grant it could issue, and connection testing accepts credentials that cannot support the backup type the user chose.

## What Changes

- Connection testing refuses credentials that cannot force a WAL segment switch, for the WAL-streaming backup type only. The test fails with a new machine-readable code, the way `wal_level`, `max_wal_senders` and `summarize_wal` already fail. The privilege is the only accepted answer: `archive_timeout` would also close segments on a timer, but it does nothing unless `archive_mode` is on, and an enabled `archive_mode` with no working `archive_command` makes PostgreSQL keep every segment in `pg_wal` until archiving succeeds. Databasus is not going to hand a user a remedy that fills their disk.
- `FULL` and `FULL_INCREMENTAL` are unaffected. Those backup types do not read archived WAL on restore, so demanding the privilege for them would refuse working configurations. The gate reuses the shape of the existing `summarize_wal` check, which is already conditional on the backup type (`model.go:852`).
- User provisioning grants `EXECUTE` on `pg_switch_wal()` to the role it creates, on platforms where the connected admin can. The grant is best effort: on managed PostgreSQL the admin frequently cannot grant execute on a restricted system function, and that must not fail the provisioning that otherwise succeeded. This matters more now that the test refuses: without the grant, Databasus's own offer to create a user would produce credentials Databasus then rejects.
- Provisioning reports whether forced WAL rotation will be available with the credentials it just created, so the answer is visible at the moment the user makes the choice rather than buried in a background log line.
- The frontend gets one more entry in its connection-error content map (`frontend/src/entity/databases/model/postgresql/physical/physicalConnectionErrorContent.ts`), with the `GRANT` command and a managed-platform note saying that where the platform will not confer the privilege, continuous WAL streaming is not available and the backup type has to stay at `FULL_INCREMENTAL`.

BREAKING for one existing configuration. A database already saved as `FULL_INCREMENTAL_WAL_STREAM` whose source refuses the switch will fail its connection test, which both `CreateDatabase` and `UpdateDatabase` run (`service.go:117`, `:209`). Backups of that database keep running: nothing re-tests a saved database on its own. But editing it fails until the operator grants the privilege or drops the backup type down to `FULL_INCREMENTAL`. That is the intended consequence of refusing a configuration that cannot do what it claims, and the error names both ways out.

## Capabilities

### New Capabilities

- `physical-backups/replication-credentials`: what a Databasus-provisioned replication user can do on the source cluster, and which source capabilities the product requires before it accepts a configuration that promises recovery to a point in time.

### Modified Capabilities

None. `openspec/specs/` holds no physical-backup capability yet, so this behavior is stated for the first time here.

## Impact

Code:

- `backend/internal/features/databases/databases/postgresql/shared/connection_error.go`: one new `ConnectionErrorCode`.
- `backend/internal/features/databases/databases/postgresql/physical/model.go`: the readiness check gains the privilege probe; the provisioning transaction gains the best-effort grant.
- `backend/internal/features/databases/service.go`, `controller.go` and `dto.go`: the provisioning endpoint gets its own `CreateReplicationOnlyUserResponse` carrying the rotation-capability result, instead of borrowing the logical `CreateReadOnlyUserResponse`, plus the Swagger annotations the endpoints already require.
- `frontend/src/entity/databases/model/postgresql/physical/`: the new code in `ConnectionErrorCode.ts` and its content in `physicalConnectionErrorContent.ts`. That map is an exhaustive `Record` over the enum, so the compiler demands the copy.
- `frontend/src/features/databases/ui/edit/CreateReadOnlyComponent.tsx`: the provisioning wizard step reports a refused grant, with the response type in `entity/databases/model/`.

External behavior: on self-managed PostgreSQL the grant normally succeeds during provisioning and WAL streaming works as intended. On RDS, Azure and GCP the grant is expected to fail, because the function's owner there is the platform's own superuser. Continuous WAL streaming becomes unavailable on those platforms until the platform allows the grant, and the error card says so instead of leaving the user to discover it from a recovery point that never advances. Backups themselves stay available there at `FULL` and `FULL_INCREMENTAL`.

No schema change, no migration, no new persisted state. Rotation availability is a property of the credentials and the source settings, read when they are tested rather than stored.

Dependency: none blocking. `physicalConnectionErrorContent.ts` holds English literals today, which the root `AGENTS.md` language rule allows; if the unmerged `translate-frontend-ui` change has converted that file to dictionary lookups by the time this lands, the new entry follows whatever shape the file then has.

Rules this change answers to: `AGENTS.md` (root), `backend/AGENTS.md`, `frontend/AGENTS.md`.

## Out of scope

- Persisting rotation availability on the database record and showing it as a state in the database view. Refusing the configuration at the door removes the state that surface existed to show. The one case it does not cover, a privilege revoked after setup, is already covered by the `WAL_ROTATION_DENIED` notification.
- Accepting a source-side archive timer in place of the privilege, or teaching Databasus to configure `archive_mode` and `archive_command` on the source.
- Changing anything about `FULL` or `FULL_INCREMENTAL` readiness.
- The WAL orphan sweep deleting a FULL backup's boundary segment. Separate change: `fix-boundary-wal-orphan-deletion`. The two were reported together and are unrelated in cause.
- Changing the rotation interval, the rotation trigger, or the decision to stop the loop permanently after a refusal.
- Re-testing saved databases in the background so an existing bad configuration surfaces without an edit.
- Retrofitting the grant onto replication users provisioned by earlier versions. The connection test names the remedy; the product does not silently re-provision.
- Granting anything beyond `pg_switch_wal` execute. Widening what a replication-only user can do needs its own justification.
