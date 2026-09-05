## Context

See the Why section in `proposal.md` for the motivation and the field report.

The orphan sweep first finds candidates with an anti-join. Before deleting a candidate, `DeleteOrphanWalSegmentsInSpan` acquires the database advisory lock and rechecks active FULL claims and completed anchors in the deletion transaction. Cascade deletion acquires the same lock before calculating its WAL span. The scheduler acquires that lock before inserting a FULL claim, so no segment can be deleted across the transition from an unanchored database to an active FULL.

Two facts about LSN alignment decide the whole design:

- A segment's `start_lsn` and `end_lsn` come from its filename, not from the stream. `segmentBounds` (`backend/internal/features/backups/backups/usecases/physical/postgresql/wal_upload.go:428`) derives them, so both are always exact multiples of the segment size.
- A FULL's `start_lsn` is a real recovery position inside a segment. In the field report it is `0/AB000028` in every reproduction, offset `0x28` past the segment start, which is where the first record lands after a long page header.

So `f.start_lsn <= w.start_lsn` is false for the boundary segment in every realistic case, and the segment holding both `start_lsn` and (for a short backup) `stop_lsn` is classified as an orphan.

The correct predicate already exists a few lines above in the same file. `FindByChainSpan` (same file, line 60) selects on `end_lsn > startLSN AND start_lsn < endLSN`, and its comment states the invariant outright: a FULL's `start_lsn` is usually mid-segment, so excluding the segment whose file-boundary start sits below it "would drop the very segment that bridges the FULL's stop_lsn into the next segment, manufacturing a false WAL gap". `FindOrphans` never got the same treatment. The fix is to make the two agree.

Constrained by `backend/AGENTS.md`: comments carry *why* only, tests follow `Test_What_WhenCondition_Expected`, and coverage is preferred at the level real callers see rather than at the repository.

## Goals / Non-Goals

**Goals:**

- One predicate governs whether a segment is expendable, and it agrees with `FindByChainSpan` on what a chain covers.
- Regression coverage fails against today's code, at both the sweep level and the full backup-to-restore level.
- Restore visibility is independent of the non-overlapping ownership spans used by retention.
- Deleting a FULL never removes WAL that a surviving predecessor needs after it becomes the selected restore base.
- Every cascade path that removes WAL participates in the active-FULL claim handoff.

**Non-Goals:**

- Introducing a retention margin (extra segments kept "just in case"). The boundary segment is genuinely referenced; keeping it is correctness, not slack.
- Schema or index changes.

## Decisions

### Compare against the segment's end, not its start

The predicate becomes: a segment is referenced when some COMPLETED FULL on the same timeline has `f.start_lsn < w.end_lsn`. Read plainly: keep the segment if any retained backup starts before the segment runs out, which is exactly "some retained backup needs bytes from this segment or from a later one".

Alternatives rejected:

- **Round the FULL's `start_lsn` down to its segment boundary** before comparing (`f.start_lsn - (f.start_lsn % segment_size) <= w.start_lsn`). Arithmetically equivalent for aligned segments, but it puts the 16 MB segment size into SQL. The size is runtime-configurable (`walmath.SetWalSize`), and the value would have to be threaded into the query or hardcoded. Comparing against the stored `end_lsn` needs no size at all.
- **Keep the predicate and special-case the boundary segment in the cleaner** by skipping any segment whose range contains a FULL's `start_lsn`. Two places would then encode chain coverage, and the query would still return rows that the caller must remember to filter. The next caller of `FindOrphans` inherits the bug.
- **Retain the newest N segments unconditionally** as a safety margin. Papers over the wrong predicate, keeps deleting referenced WAL once the margin scrolls past, and makes the retained set depend on arrival order rather than on what a restore needs.

### Keep the sweep anchored on FULLs only, not on incrementals

An incremental's span sits inside its chain, and the chain is anchored by its root FULL, whose `start_lsn` is lower. `GetChainSpan` already treats the root FULL's `start_lsn` as the chain's lower bound.

Alternatives rejected:

- **Add COMPLETED incrementals to the anti-join** so the sweep anchors on any backup rather than on FULLs. It would not retain a single extra segment, because every incremental's chain starts at a FULL whose `start_lsn` is lower, while it doubles the anti-join's cost on a query the comment at `wal_segment_repository.go:80` already flags as a sequential scan.

### Separate WAL ownership from restore visibility

`GetChainSpan` keeps the non-overlapping retention range `[full.start_lsn, successor.start_lsn)`. `ResolveRestoreSet` uses `CompletedAt` to select a safe base backup, then queries WAL from that chain through `LSNMax` and ships the full contiguous run up to the first LSN gap. PostgreSQL applies `recovery_target_time`; `ReceivedAt` is only the existing catalog estimate for rejecting targets beyond available WAL.

Moving the ownership boundary to `successor.stop_lsn` is rejected because it gives two chains overlapping deletion ranges. Trimming the restore set at the first segment received after the target is also rejected: receipt time does not prove that a segment contains a transaction timestamp at or after the requested target.

### Preserve shared WAL when deleting a successor

The WAL attached to a newer FULL is also the continuation of every older restore base on the same timeline. If an older completed FULL survives, cascade deletion removes no WAL with the newer FULL. If no predecessor survives, deletion uses the target FULL's ownership span and removes only fully contained segments.

`GetDependentsSummary` and cascade deletion share this selection rule, including the active FULL claim guard. The preview takes the same advisory lock and database snapshot as cascade deletion, so it cannot report WAL or timeline history that an observed active claim makes non-deletable. The preview cannot use the overlap query because the destructive predicate deliberately excludes a segment crossing the upper ownership boundary.

`DeleteChainDependentsKeepFull` remains different by design. The `FULL_BACKUPS` policy keeps a standalone FULL while explicitly shedding its incrementals and owned WAL, so predecessor preservation applies only when the FULL row is removed.

### Serialize first-FULL scheduling with orphan deletion

Scheduling a FULL claim and deleting orphan WAL acquire the same transaction-scoped advisory lock keyed by database ID. The deletion transaction checks for an active FULL claim first, then checks whether a completed FULL covers the candidate. An `IN_PROGRESS` row without a live claim does not protect WAL because the claim is the source of truth for active work.

Cascade deletion acquires this lock before locking its target FULL. After the lock is acquired, both full removal and `DeleteChainDependentsKeepFull` recheck the active FULL claim. If a claim exists, they preserve WAL. Full removal also preserves timeline history until the active FULL has published its completed anchor. This lock ordering prevents a cascade from calculating an unbounded ownership span while a replacement FULL claim is being committed.

### Coverage at two levels

`chain_view/service_test.go:343` seeds a FULL with a segment-aligned `start_lsn` (`4*segmentBytes`), and `cleaner_test.go:249` does the same through `seedChainFull`. That is the one alignment where the buggy predicate is accidentally correct. Both files get a case with a mid-segment `start_lsn`, so the offset that matters lives in the fixture rather than being assumed away. (`cleaner_test.go:236` seeds no FULL at all, so it covers the "no COMPLETED FULL on this timeline" branch and never the boundary case under either predicate.)

That alone would not have caught the field failure, because it does not exercise the upload-then-sweep sequence against a real cluster. The `features/tests/physical/postgresql` suite already has the pieces, all unexported inside its `shared` package: `seedChainAndStreamPastTarget` (`shared/helpers.go:786`), `waitForReplayableThroughLSN` (`:760`) and `requestRestoreTokenViaAPI` (`:606`), plus a working PITR assertion in `RunFullTwoIncrementalsPlusWalRecoversToTarget`. The new scenario therefore lands in `shared` as an exported `Run...` function and is wired into both `pg17/backup_restore_test.go` and `pg18/backup_restore_test.go`, matching every existing case: the per-version files are one-line dispatchers.

Alternative rejected: asserting on log output (`deleted orphan wal segment`). It tests the message rather than the outcome, and it would pass on any refactor that deletes the segment silently.

### The cleaner needs a single-pass test entry point

`shared` cannot drive the sweep today. `cleanOrphans` and `cleanOrphanWalForDatabase` are unexported (`cleaner.go:492` and `:508`), and the only exported entry is `Run` (`cleaner.go:40`), an infinite ticker that panics when called twice. `shared/setup.go` already follows a house pattern for this with `StartPhysicalSchedulerForTest` and `StartPhysicalWalStreamSupervisorForTest`; the cleaner gets its sibling, running one pass rather than starting a loop.

Alternatives rejected:

- **Call `Run` and wait out a tick.** The 3 second `cleanerTickInterval` becomes a sleep in an already slow suite, the panic guard makes a second pass impossible in the same process, and the test would depend on timing rather than on sequence.
- **Move the e2e case into the `backuping/physical` package** to reach the unexported methods. That package has no real cluster fixture, and `features/tests/` exists precisely for backup-to-restore cycles per `backend/AGENTS.md`.

## Risks / Trade-offs

- **The new predicate keeps a segment that is genuinely expendable.** The window is one segment per database and timeline, since the anti-join matches on `timeline_id`: a segment ending after the earliest retained FULL's start but beginning before it. That segment holds the bytes leading into the backup, so keeping 16 MB is the intended cost of a restorable chain. No mitigation needed.
- **Installations already damaged stay damaged.** WAL deleted before the fix is gone from storage, and PostgreSQL will have recycled it from `pg_wal` once the slot advanced. Users must take a fresh FULL after upgrading to get a restorable chain. This belongs in the release note, not in code.
- **The e2e case adds container time to an already slow suite.** Extend an existing scenario rather than standing up a fresh cluster if the runtime grows past what `make test` tolerates.
- **The unit fixtures could drift back to aligned LSNs** in a later refactor, quietly disarming the regression. The test name states the condition (`WhenFullStartsMidSegment`), so a fixture change that flattens it is visible in review.
- **Deleting a newer FULL retains more WAL than its preview used to report.** This is required while an older FULL survives. Later deletion of the earliest surviving FULL or the orphan sweep reclaims the WAL when no restore base needs it.

## Migration Plan

No schema change and no data migration. Deployment is the ordinary upgrade; the sweep simply stops deleting the boundary segment on its next run. Rollback is reverting the predicate, which restores the old, incorrect deletions and needs no data step.

Operationally, an installation that already lost boundary segments needs a new FULL backup before PITR works again. Nothing in the code can recover the deleted WAL.
