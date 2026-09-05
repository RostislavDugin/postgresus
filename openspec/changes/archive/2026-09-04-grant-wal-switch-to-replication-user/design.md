## Context

See `proposal.md`: Why for the motivation and the field report.

Four pieces of the current implementation shape the approach.

The connection test is binary and already carries machine-readable refusals. `checkReplicationReadiness` (`backend/internal/features/databases/databases/postgresql/physical/model.go:805`) opens a replication connection, then an ordinary one for inspection queries (`:824`), and returns a `ConnectionTestError` with a code for each unmet precondition: `wal_level`, `max_wal_senders`, `max_replication_slots`, `summarize_wal`, custom tablespaces, system-identifier mismatch (`:841-865`). The controller turns that into `400 {"code": ...}` and nothing else (`controller.go:280`). There is no shape in this API for "passed, with a remark".

One of those checks is already conditional on the backup type. `summarize_wal` is demanded only when `p.BackupType.IsRequireWalSummary()` (`model.go:852`), and its sibling predicate `IsWalStreaming()` already exists (`enums.go:15`) with no caller in this file.

Provisioning is one short transaction. `CreateReplicationOnlyUser` (`model.go:723`) runs `CREATE USER ... LOGIN`, calls `grantReplication` for the detected platform (`model.go:983`), verifies the role exists and commits. `grantReplication` already carries per-platform branches and already converts a permission failure into an actionable message for Azure and GCP, so the shape for a second, best-effort grant is established there.

The frontend already owns every word of connection-test copy. `physicalConnectionErrorContent.ts` maps each code to a title, a summary, optional shell steps and a `managedNote` for RDS, Azure and GCP where the commands do not apply. The map is an exhaustive `Record<ConnectionErrorCode, ...>`, so adding an enum member makes the compiler demand the copy.

Constrained by `backend/AGENTS.md` (Swagger comments on every endpoint, controller-level tests over unit tests, comments explain *why*) and `frontend/AGENTS.md`.

## Goals / Non-Goals

**Goals:**

- A configuration that promises recovery to a point in time is refused unless the source can actually deliver it.
- Backup types that do not read archived WAL keep working with credentials that cannot force a switch.
- Provisioning improves the outcome where it can, so Databasus does not hand a user credentials it will then reject.

**Non-Goals:**

- Detecting *why* a platform refuses the grant. RDS, Azure and GCP each refuse differently; the product needs the answer, not the taxonomy.
- Guaranteeing the grant survives. An operator can revoke it later, which is why the streamer's runtime detection stays the source of truth.
- A privilege model for replication users beyond this one function.
- Reworking notifications, or re-testing saved databases in the background.

## Decisions

### Refuse in the connection test rather than report a finding beside it

A source that can neither force a switch nor close segments on a timer fails `checkReplicationReadiness` with a new `ConnectionErrorCode`, alongside the preconditions already checked there.

Alternatives rejected:

- **Pass the test and return a finding.** The endpoint returns `200 {"message": "connection successful"}` or a `400` with a code (`controller.go:280`); a third outcome means inventing a response shape and teaching every caller to read it, to describe a configuration that cannot do its job. The recent verification of minimum dump privileges for logical engines took the refusal route, not the remark route.
- **Store the answer on the database and show it as a state in the database view.** A nullable column, writes from provisioning, from testing and from the streamer, and a rule for clearing it, all to display a state the user could not have created if the test had refused it.
- **Leave it to the `WAL_ROTATION_DENIED` notification.** That is today's behavior. It arrives after the first quiet interval and only when the database has `CHAIN_BROKEN` enabled (`wal_supervisor.go:453`).

### Demand it only for `FULL_INCREMENTAL_WAL_STREAM`

The gate is `p.BackupType.IsWalStreaming()`, mirroring the `summarize_wal` line two checks above it.

`FULL` and `FULL_INCREMENTAL` never read an archived segment on restore: `pg_basebackup --wal-method=fetch` inlines the WAL a base backup needs (`pg_basebackup.go:69`) and `ResolveRestoreSetForBackup` ships no WAL and replays nothing (`restore_set.go:83`). Only `ResolveRestoreSet`, which walks the contiguous archived run from the last backup's `stop_lsn` (`restore_set.go:61`), depends on segments being finalized.

Alternatives rejected:

- **Demand it for every physical backup type.** Refuses configurations that work, and refuses them on managed platforms where nothing the operator does can produce the privilege.
- **Demand it for the incremental type too.** Incrementals are built by `pg_basebackup --incremental` against a manifest and restored by `pg_combinebackup` without replay. The archived WAL is not in that path.

### The privilege is the only accepted answer

The check passes on `has_function_privilege` alone. No source setting substitutes for it.

`archive_timeout` is the obvious candidate and `buildChainRiskAlert` names it as a remedy (`wal_supervisor.go:432`). It would work mechanically: the server switches the segment on its own timer, our `pg_receivewal` reaches the boundary and closes the file, the uploader ships it. Nothing here waits on the source's archiver. The problem is the precondition. `archive_timeout` has no effect while `archive_mode` is off, and turning `archive_mode` on commits the cluster to archiving every segment: until `archive_command` reports success, PostgreSQL keeps the segment in `pg_wal` and will not recycle it. On a source with no real archive command that is a full disk, and the failure lands on the user's production cluster rather than on a backup. A remedy Databasus prints in an error card has to be safe to follow literally.

Alternatives rejected:

- **Accept `archive_mode` on with a non-zero `archive_timeout`.** Keeps WAL streaming reachable on managed platforms, at the price of telling users to enable archiving without an archiver. The disk-fill risk is on their primary.
- **Accept it, and also tell the user to set `archive_command` to a no-op.** Databasus would be dictating an unrelated cluster-wide setting whose failure mode it cannot see, to work around a missing privilege.
- **Read `archive_timeout` alone.** Accepts a source where the setting has no effect at all.

### Probe with `has_function_privilege`, not by calling the function

Availability is read with `SELECT has_function_privilege(<role>, 'pg_switch_wal()', 'EXECUTE')` over the ordinary connection `checkReplicationReadiness` already opens.

Alternatives rejected:

- **Call `pg_switch_wal()` and catch `42501`.** A successful probe rotates a segment as a side effect, so testing a connection would archive a padded segment. Unacceptable in a path a user may run repeatedly, and worse in `CreateDatabase` and `UpdateDatabase`, which run the test on every save.
- **Infer from `rolsuper`.** Wrong in both directions: a non-superuser with an explicit grant can rotate, and a managed platform's "superuser-like" role often cannot.

### The grant is best effort inside the existing provisioning transaction

`grantReplication` gains a sibling that issues `GRANT EXECUTE ON FUNCTION pg_switch_wal() TO "<role>"` and treats a permission failure as a recorded outcome rather than an error. The result travels back with the created credentials so the caller can report it.

Failing provisioning on a refused grant is not an option: the replication user is fully functional for `FULL` and `FULL_INCREMENTAL` without it, and on managed platforms the refusal is the normal case.

The statement runs inside a savepoint. A refused statement puts the enclosing transaction in an aborted state, so without one the refusal would take down the `CREATE USER` it was meant to survive, and provisioning would fail on exactly the platforms this decision exists to serve.

Alternatives rejected:

- **A separate transaction after provisioning.** A failure would leave a user with no record of whether the grant was attempted, and the caller would have to reconcile two outcomes.
- **Skip the grant and let the connection test refuse.** Databasus would offer to create a user and then reject the user it created, on the platform where it could have simply granted the privilege.
- **Grant to `PUBLIC`.** Widens the privilege far beyond the role Databasus created.

### The refusal carries a sentence as well as a code

`ConnectionTestError` has an optional `Message` "for paths that surface only `Error()` (database create / update), so they don't show a bare code". Those are exactly the paths this change breaks: `CreateDatabase` and `UpdateDatabase` run the same test but answer with `{"error": ...}` rather than `{"code": ...}` (`controller.go`), so a user editing a saved streaming database would otherwise be shown the literal string `no_wal_switch_privilege`. The code therefore carries a sentence naming the role, the `GRANT`, and the option of a backup type without WAL replay.

Alternatives rejected:

- **Leave `Message` empty, like `wal_level_invalid` and its neighbours do.** Those codes are reached mostly from the test-connection endpoints, where the frontend renders a full card. This one is reached from the edit form by every user whose existing configuration this change refuses.
- **Teach create and update to answer with the code instead.** A wider API change than this needs, and it would alter the shape of every other failure those endpoints return.

### The backend sends a code, the frontend owns the words

The new code gets its entry in `physicalConnectionErrorContent.ts`: the `GRANT EXECUTE` line as a command step, and a `managedNote` stating that RDS, Azure and GCP do not let a customer role confer this privilege, so continuous WAL streaming is unavailable there and the backup type stays at `FULL_INCREMENTAL`. `buildChainRiskAlert` keeps its own wording unchanged, including the `archive_timeout` hint it gives an operator who is already streaming.

Alternatives rejected:

- **Extract one shared remedy sentence for both surfaces.** The alert's `Message` interpolates `database_id=%s reason=%s` around the remedies (`wal_supervisor.go:427-433`), it is log-shaped by design, and the card is a titled panel with commands and a managed-platform note. Sharing one sentence between them would fit neither.
- **Send a human-readable message from the backend.** Every other physical connection error sends a bare code and lets the frontend render it; `ConnectionErrorCode.ts` says so in its header comment.

### Names

Named here because `AGENTS.md` puts the naming check in the post-planning review, while names are still cheap to change:

The capability is called *forced WAL rotation* throughout, following the streamer's existing `runForcedWalRotation` and `DefaultForcedRotationInterval`. The exception is the wire code and the grant helper, which name the SQL function instead, because what is missing is a privilege on `pg_switch_wal()` and the remedy is a `GRANT` naming that function.

- `canForceWalRotation(ctx, q, roleName)` for the probe. A predicate, so no `Get` prefix. Unexported: no caller outside the package.
- `grantWalSwitchIfPermitted(ctx, tx, username) (isGranted bool, err error)` as `grantReplication`'s sibling. The suffix carries the best-effort contract, so a caller does not have to read the body to learn that a refusal is not an error. No platform argument: the statement is identical everywhere and only the outcome differs.
- `ConnErrNoWalSwitchPrivilege = "no_wal_switch_privilege"` for the code, matching the existing `no_replication_privilege`, and `ConnectionErrorCode.NoWalSwitchPrivilege` for its frontend mirror.
- `ReplicationOnlyUser{Username, Password, IsForcedWalRotationAvailable}` as `CreateReplicationOnlyUser`'s result, so the call sites do not carry two adjacent strings plus a bare flag.
- `CreateReplicationOnlyUserResponse` for the endpoint's own response, since the shared `CreateReadOnlyUserResponse` also serves the logical read-only user, which has no such flag.

The new probe and grant live in `forced_wal_rotation.go` with a mirrored `_test.go`, rather than growing `model.go`, which already holds the model, platform detection, readiness and provisioning.

## Risks / Trade-offs

- **Continuous WAL streaming becomes unavailable on RDS, Azure and GCP.** This is the deliberate cost of the decision above. Those users keep `FULL` and `FULL_INCREMENTAL`, and learn it at the moment they configure rather than from a recovery point that silently stops advancing. If a platform later exposes the grant, nothing in this change blocks it: the probe simply starts returning true.
- **An existing `FULL_INCREMENTAL_WAL_STREAM` database becomes uneditable** until its source is fixed, because `UpdateDatabase` runs the same test (`service.go:209`). Its backups keep running; nothing re-tests a saved database. The error card names both ways out, one of which is lowering the backup type. Called out in the proposal as the breaking part of this change and belongs in the upgrade note.
- **The probe runs as the admin, not the new role.** `has_function_privilege` takes a role argument, so the provisioning path probes the created role by name rather than `current_user`. Getting this wrong reports a false positive on every platform, so it is worth a dedicated test.
- **A managed platform may report the grant as succeeding while still refusing at call time.** The streamer's runtime detection remains authoritative and still stops the loop and notifies.
- **A privilege revoked after setup is not caught until a rotation is attempted.** Unchanged from today, and the existing notification covers it.

## Migration Plan

None. No schema change, no new column, no data to backfill. Rollback is reverting the change.

Existing databases are re-evaluated only when someone saves them. Replication users provisioned by earlier versions are not re-provisioned; their next connection test names the limitation and the remedies, and acting on it is the operator's call.
