## Context

See `proposal.md` for motivation and `specs/postgresql-physical-backup-failover/spec.md` for observable behavior.

`CheckTimelineCompatibility` currently compares the live timeline with the maximum timeline found in completed FULL and `.history` rows (`backend/internal/features/backups/backups/usecases/physical/postgresql/timeline.go:55-110,293-322`). Once the WAL streamer catalogs a new `.history` row, this maximum can match the promoted server even though an INCR still points to a root FULL on the previous timeline.

The scheduler checks incremental cadence before it reacts to a missing extendable chain (`backend/internal/features/backups/backups/backuping/physical/scheduler.go:161-242`). A newly broken chain can therefore wait for a full incremental interval. Existing terminal backup rows, the single in-flight claim, and the three-attempt FULL brake already provide enough durable coordination to avoid another state table.

Orphan WAL cleanup discovers and deletes segments without serializing against FULL creation (`backend/internal/features/backups/backups/backuping/physical/cleaner.go:508-546`). Restore selection also uses the retention-owned chain span directly, so an older chain cannot see same-timeline WAL beyond the start of a newer FULL (`backend/internal/features/backups/backups/core/physical/chain_view/restore_set.go:41-82`).

The implementation is constrained by `AGENTS.md` and `backend/AGENTS.md`.

## Goals / Non-Goals

**Goals:**

- Use the exact root FULL as the timeline authority for every INCR.
- Detect promotion both before and during `pg_basebackup`.
- Start one prompt replacement FULL without creating a scheduler retry loop.
- Preserve WAL and same-timeline PITR while that replacement is running.

**Non-Goals:**

- Restore through more than one PostgreSQL timeline.
- Persist the current timeline on the database model.
- Add a reanchor table, new backup configuration, or a new retry policy.

## Decisions

### Use root FULL timeline for INCR compatibility

Create separate FULL and INCR identity decisions. The INCR decision receives the root FULL timeline and compares it directly with the live value. The FULL decision keeps the current comparison with catalog history because a FULL may establish a new timeline and must still reject a regression or another cluster.

The INCR executor will load its root FULL through the existing repository before opening the backup slot. Missing or incomplete root metadata remains a broken-parent condition.

Rejected alternative: store a current timeline on the database row. That duplicates data already attached to backup artifacts and still does not identify the timeline that a particular INCR must extend.

Rejected alternative: add a reanchor state table. The broken INCR or failed FULL row already survives restarts and records why a replacement is needed.

### Recheck identity only after unexpected stream failure

The preflight decision returns the live timeline observed immediately before streaming. When a stream returns a non-completed result and its execution context is still active, the executor opens a fresh inspection connection and evaluates identity again.

For INCR, a newer timeline replaces a transient stream result with `CHAIN_BROKEN/TIMELINE_SWITCH_DETECTED`. For FULL, a timeline newer than its preflight value becomes `ERROR/FAILOVER_DURING_BACKUP`. A lower timeline or changed system identifier uses the existing safety refusal. A failed recheck leaves the original result unchanged and logs the failure.

Rejected alternative: classify `signal: interrupt` text. The same text can result from cancellation or an internal stream watchdog, so stderr alone cannot prove promotion.

Rejected alternative: recheck every successful backup. Successful stream metadata already supplies the completed artifact timeline, and the existing history validation handles that path.

### Derive one immediate replacement from terminal rows

Scheduler precedence will be:

1. Explicit forced FULL.
2. Timeline-related terminal backup that has not yet received a replacement attempt.
3. Existing forced INCR and cadence decisions.

A timeline-broken INCR is pending replacement only while no FULL row has a `CreatedAt` later than the INCR's `CompletedAt`. A latest FULL with `FAILOVER_DURING_BACKUP` is itself the pending signal. Inserting the replacement row changes the latest catalog state before its goroutine starts, and the existing in-flight claim prevents concurrent inserts.

An immediate decision carries the triggering backup kind and ID. After acquiring the same per-database advisory lock used for claim creation, `claimAndInsert` reloads the latest FULL and INCR rows and repeats the trigger predicate. It inserts the replacement only if that exact terminal row is still current and no later FULL attempt exists. This closes the stale-decision window where another scheduler could finish the first replacement and release its claim before a peer acts on an older decision.

If the replacement fails for another reason or is canceled, its terminal row no longer matches either immediate condition. Existing cadence and `recentFullAttemptsWindow` logic then controls later attempts. If another promotion interrupts it, the new `FAILOVER_DURING_BACKUP` result creates one new immediate attempt.

`SYSTEM_IDENTIFIER_MISMATCH` is not repairable by a FULL against the wrong cluster. The scheduler finds the newest mismatch completion time across FULL and INCR rows, then compares it with the newest completed FULL completion time. Automatic cadence and reanchor decisions remain suppressed while no completed FULL is newer than that mismatch. An explicit forced FULL remains higher priority so an operator can retry after correcting the connection, but normal preflight still refuses a different cluster. A forced FULL that fails generically or is canceled does not prove cluster identity and cannot clear suppression. Only a later completed forced FULL supersedes the mismatch, creates the new extendable chain, and restores normal scheduling.

Rejected alternative: set `ForceFullRequestedAt` from the executor. Persisting the terminal result and setting the request would span two writes with a crash window, while scheduler derivation needs no second state transition.

Rejected alternative: move every broken chain ahead of cadence. That would change `SUMMARIZER_OFF` behavior and could run repeated FULL backups for a source-side setting that a FULL cannot repair.

Rejected alternative: rely only on the in-flight claim to deduplicate stale decisions. The claim disappears when the first replacement finishes, so a peer can otherwise insert a second replacement from a decision made before that first attempt.

### Serialize FULL creation and orphan deletion

Add one shared repository-level operation that takes a transaction-scoped PostgreSQL advisory lock using `hashtextextended('physical-backup:' || database_id::text, 0)`. Both paths acquire it before any row lock:

- Scheduler transaction: advisory lock, in-flight claim, typed backup row insert.
- Orphan deletion transaction: advisory lock, read in-flight claim, skip if it is a FULL, otherwise delete the selected storage objects and catalog rows before commit.

Rejected alternative: check the in-flight claim without a shared lock. A FULL can claim the database immediately after the cleaner's check and lose WAL during its run.

Rejected alternative: lock a backup configuration row. The physical catalog service would then depend directly on another feature's private persistence model.

### Separate restore reachability from retention ownership

Retention keeps the existing chain span ending at the next completed FULL. `ResolveRestoreSet` will instead query committed WAL from the selected root FULL start through the unbounded end of the same timeline, then apply the existing contiguous-run and target-time checks. This preserves the current behavior of shipping the full contiguous replay window while allowing an older FULL to reach a target during a newer FULL.

Rejected alternative: widen every chain span. That would give the same WAL to multiple retention chains and change deletion accounting.

Rejected alternative: follow `.history` ancestry into a new timeline. The current manifest and restore path describe one WAL range, so that requires a separate cross-timeline design.

## Risks / Trade-offs

- [Identity recheck cannot reach the promoted server immediately] -> Preserve the original failure; the next normal attempt will run preflight again.
- [A corrected connection remains suppressed until the operator acts] -> Permit an explicit forced FULL; only its successful completion after the newest mismatch restores normal scheduling.
- [Advisory lock collision] -> Prefix the UUID before hashing so this lock namespace does not overlap other application advisory locks; the remaining 64-bit hash collision risk is negligible.
- [Object storage deletion holds the advisory lock] -> Keep the existing WAL byte and row budgets so the transaction remains bounded.
- [No cross-timeline PITR before replacement FULL completes] -> Return the existing unreachable-target error and document this as an explicit capability boundary.

## Migration Plan

No database migration is required. Deploy backend code and run the PostgreSQL 17 and 18 promotion tests before release. Rollback consists of reverting the backend change; existing rows and API payloads remain readable because the added error reasons are text values.
