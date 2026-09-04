## 1. Pin the bug with a failing test

- [x] 1.1 Add `Test_CleanOrphanWalForDatabase_WhenFullStartsMidSegment_KeepsBoundarySegment` to `backend/internal/features/backups/backups/backuping/physical/cleaner_test.go`: seed a COMPLETED FULL whose `start_lsn` and `stop_lsn` both fall inside segment 1 (offset `0x28` past the boundary, matching the field report), seed the archived segment 1 with file-aligned bounds, run `cleanOrphanWalForDatabase`, assert the segment still exists. Verify with `make test` in `backend/` that the test fails against the current predicate and names the deleted segment in its failure message.
- [x] 1.2 Add `Test_FindWalOrphansByDatabase_WhenFullStartsMidSegment_BoundarySegmentIsNotOrphan` to `backend/internal/features/backups/backups/core/physical/chain_view/service_test.go`, next to the aligned-LSN case at line 343 so the two alignments sit side by side. Verify it fails for the same reason via `make test`.

## 2. Fix the predicate

- [x] 2.1 Change the anti-join in `PhysicalWalSegmentRepository.FindOrphans` from `f.start_lsn <= w.start_lsn` (`backend/internal/features/backups/backups/core/physical/repositories/wal_segment_repository.go:100`) to `f.start_lsn < w.end_lsn`. Verify both tests from group 1 pass and the existing `Test_CleanOrphanWalForDatabase_WhenWalOutsideAllChains_DeletesOrphan`, `Test_CleanOrphanWalForDatabase_WhenWalCoveredByChain_KeepsIt` and `Test_FindWalOrphansByDatabase_WhenWalOutsideAllChainSpans_ReturnsOrphan` still pass.
- [x] 2.2 Add a second why line to the comment above `FindOrphans` (`wal_segment_repository.go:80`) stating that the comparison anchors on `end_lsn` because a FULL's `start_lsn` sits mid-segment while a segment's bounds are file-aligned. Keep the existing sequential-scan note, which is unrelated and still accurate. Verify with `make lint` in `backend/` and by the group 1 tests still passing.

- [x] 2.3 Add `Test_DeleteFull_WhenSuccessorStartsMidSegment_KeepsSuccessorBoundarySegment` to `backend/internal/features/backups/backups/core/physical/service/service_test.go`: prune a chain whose successor FULL starts mid-segment and assert the successor's boundary segment and its stored object survive. Not in the original plan, which assumed `FindOrphans` was the only place the alignment mistake lives. It covers the delta spec's "The anchoring backup is pruned" scenario, which the predicate fix alone does not satisfy.
- [x] 2.4 Bound `deleteWalInSpanBudgeted` on `end_lsn <= span.End` instead of `start_lsn < span.End` (`backend/internal/features/backups/backups/core/physical/service/service.go:604`), so a chain's deletion span stops at segment boundaries rather than cutting into the successor's boundary segment. Reached from retention pruning (`cascadeDelete` via `GetChainSpan`) and from the user-initiated `DeleteWalSegment`, both of which end their span on a mid-segment anchor LSN.

## 3. Make the sweep drivable from the end-to-end suite

- [x] 3.1 Add a single-pass cleaner entry point for tests to `backend/internal/features/backups/backups/backuping/physical/`, alongside `StartPhysicalSchedulerForTest` and `StartPhysicalWalStreamSupervisorForTest` in `backend/internal/features/tests/physical/postgresql/shared/setup.go`. `Run` (`cleaner.go:40`) cannot serve: it is an infinite ticker that panics on a second call. Verify with `make test` that a test can run one sweep and observe its effect without waiting out `cleanerTickInterval`.

## 4. Cover the sequence end to end

- [x] 4.1 Add an exported `Run...` scenario to `backend/internal/features/tests/physical/postgresql/shared/` that runs the sweep between the backup and the restore: take a FULL, stream WAL past a target time via `seedChainAndStreamPastTarget` (`shared/helpers.go:786`), run one cleaner pass from 3.1, then request a restore token for that target and assert it is issued rather than refused with 422. Wire it into both `pg17/backup_restore_test.go` and `pg18/backup_restore_test.go` as one-line dispatchers, matching every existing case. Verify with `make test` (or `make test-fedora` on a Fedora host) that both versions pass and that they fail when the predicate from 2.1 is reverted.
- [x] 4.2 Add a case asserting the sweep still reclaims WAL that sits entirely below the earliest COMPLETED FULL's `start_lsn`. The existing `Test_CleanOrphanWalForDatabase_WhenWalOutsideAllChains_DeletesOrphan` (`cleaner_test.go:236`) seeds no FULL at all, so it covers the "no COMPLETED FULL on this timeline" branch and never this one. Verify with `make test`.

## 5. Close out

- [x] 5.1 Run `make lint` and `make test` in `backend/`; fix every finding before finishing.
- [x] 5.2 Run the `reviewer` subagent over the working-tree diff and resolve every CHANGES REQUIRED finding.
- [ ] 5.3 Put the upgrade note in the pull request description, since the repository keeps no changelog file: installations affected before the upgrade must take a fresh FULL backup, because the deleted WAL cannot be recovered, and the symptom is a `latest restorable point` equal to the FULL's `stop_lsn`. Verify the PR description contains it before requesting review.

## 6. Cover FULL execution windows

- [x] 6.1 Make `ResolveRestoreSet` query WAL through `LSNMax` after selecting the completed base backup, instead of limiting PITR to the selected chain's retention span.
- [x] 6.2 Serialize FULL claim creation and orphan deletion with the per-database advisory lock, then recheck the live FULL claim and completed anchor in the deletion transaction.
- [x] 6.3 Add regressions for PITR during a newer FULL and WAL archived while the first FULL has an active claim.

## 7. Preserve shared WAL during cascade deletion

- [x] 7.1 Add a regression that deletes a successor FULL while its predecessor remains, asserts that no WAL is removed, and resolves a target from the predecessor afterward.
- [x] 7.2 Make cascade deletion skip WAL when an older COMPLETED FULL remains on the same database and timeline. Keep the existing ownership span for a FULL with no surviving predecessor.
- [x] 7.3 Make `GetDependentsSummary` use the same WAL selection rule and fully-contained predicate as cascade deletion, including count and size.

## 8. Complete regression coverage

- [x] 8.1 Strengthen the restore-set regression so the target lies after the successor's stop position but before its completion time, and assert that all available contiguous WAL is returned.
- [x] 8.2 Add a restore-token controller regression for a target during a newer FULL.
- [x] 8.3 Add a cleaner regression proving that an `IN_PROGRESS` FULL row without a live claim does not retain orphan WAL.

## 9. Validate the completed change

- [x] 9.1 Run focused chain-view, physical-service, cleaner, scheduler, and controller tests. Confirm the new deletion and preview regressions fail when their production change is reverted.
- [x] 9.2 Run `make lint` and the full backend test suite. On Fedora, record the expected host-native `make test` loader failure for `libedit.so.2`, then run the supported `make test-fedora` shim to completion.
- [x] 9.3 Run the mandatory implementation reviewer and resolve every `CHANGES REQUIRED` finding.

## 10. Serialize cascade deletion with active FULL creation

- [x] 10.1 Acquire the database backup-and-cleanup advisory lock before cascade deletion locks its target FULL, then recheck the active FULL claim inside the transaction.
- [x] 10.2 Preserve WAL in both full-removal and `DeleteChainDependentsKeepFull` paths while a FULL claim is active. Preserve timeline history in the full-removal path until the active FULL publishes a completed anchor.
- [x] 10.3 Add a concurrency regression for both cascade modes. Hold the advisory lock, start deletion, commit a FULL row and claim, then prove deletion waited and retained the WAL row and object.
- [x] 10.4 Make `GetDependentsSummary` use the shared active-claim decision under the advisory lock, and cover a preview/delete pair that both report zero WAL and history while a FULL claim is active.
