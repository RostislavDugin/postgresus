## 1. Timeline classification

- [x] 1.1 Add `TIMELINE_SWITCH_DETECTED` and `FAILOVER_DURING_BACKUP` error reasons, and verify enum serialization with `cd backend && TEST_PARALLEL_WORKERS=1 go test -p=1 -count=1 ./internal/features/backups/backups/core/physical/enums`
- [x] 1.2 Split FULL and INCR timeline decisions so INCR uses its root FULL timeline, and verify equal, newer, older, and different-cluster cases with `cd backend && TEST_PARALLEL_WORKERS=1 go test -p=1 -count=1 ./internal/features/backups/backups/usecases/physical/postgresql`
- [x] 1.3 Recheck cluster identity after non-completed FULL and INCR streams while execution remains active, preserving cancellation and failed-probe results, and verify the executor tests with `cd backend && TEST_PARALLEL_WORKERS=1 go test -p=1 -count=1 ./internal/features/backups/backups/usecases/physical/postgresql`

## 2. Automatic re-anchoring

- [x] 2.1 Add scheduler decisions for one immediate FULL after `TIMELINE_SWITCH_DETECTED` or `FAILOVER_DURING_BACKUP`, carry the triggering row into transaction-time revalidation, suppress automatic scheduling while no completed FULL is newer than the latest `SYSTEM_IDENTIFIER_MISMATCH`, and verify priority, two-scheduler stale decisions, generic replacement failure, second promotion, mismatch failure, generic failure, cancellation and successful forced FULL attempts after mismatch, plus `SUMMARIZER_OFF`, with `cd backend && TEST_PARALLEL_WORKERS=1 go test -p=1 -count=1 ./internal/features/backups/backups/backuping/physical`

## 3. WAL and restore safety

- [x] 3.1 Serialize FULL claim creation and orphan WAL deletion with the shared per-database transaction advisory lock, and verify both lock orderings plus `Test_CleanOrphanWalForDatabase_WhenFirstFullInProgress_KeepsWalArchivedDuringBackup` with `cd backend && TEST_PARALLEL_WORKERS=1 go test -p=1 -count=1 ./internal/features/backups/backups/backuping/physical`
- [x] 3.2 Let PITR resolution read unbounded contiguous WAL on the selected root FULL's timeline without changing retention spans, and verify `Test_ResolveRestoreSet_WhenTargetDuringNewerFull_UsesOlderChainWalThroughTarget` plus existing gap tests with `cd backend && TEST_PARALLEL_WORKERS=1 go test -p=1 -count=1 ./internal/features/backups/backups/core/physical/chain_view`

## 4. Promotion coverage and final validation

- [x] 4.1 Add PostgreSQL 17 and 18 promotion coverage showing that retained timeline history does not let a stale INCR cross its root FULL timeline, followed by automatic FULL on the promoted timeline, cleanup, and restore from the new chain, and verify with `cd backend && make test`
- [x] 4.2 Run the complete backend checks with `cd backend && make test && make lint`
