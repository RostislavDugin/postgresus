## Why

The WAL orphan sweep deletes the very WAL segment that carries a FULL backup's `start_lsn` and `stop_lsn`, so point-in-time restore breaks a few seconds after a FULL backup reports success. `FindOrphans` in `backend/internal/features/backups/backups/core/physical/repositories/wal_segment_repository.go:83` keeps a segment only when some COMPLETED FULL satisfies `f.start_lsn <= w.start_lsn`. A segment's `start_lsn` is derived from its filename and is therefore file-aligned (`0/AB000000`), while a FULL's `start_lsn` sits just past the segment's long page header (`0/AB000028`). The comparison fails on the boundary segment every time.

This is not a rare race. A user report against `v3.55.3` reproduces it five times in a row, always with the same `0x28` offset (`0/8F000028`, `0/92000028`, `0/96000028`, `0/98000028`, `0/AB000028`), and the cleanup log follows two seconds behind the upload log. After the deletion `ResolveRestoreSet` cannot advance past the FULL's `stop_lsn` and reports `wal gap before target; latest restorable point is <stop_lsn>`, while the UI still shows the FULL and the WAL archive as successful. A user believes PITR works when the recovery chain is already broken.

The blast radius grows over time. The predicate is satisfied by *any* older COMPLETED FULL, so with several FULLs on a database only the oldest chain loses its boundary segment. When retention prunes that oldest chain, the next chain's boundary segment becomes an orphan and is deleted in turn, so a chain that restored correctly yesterday stops restoring today.

Two longer windows expose the same failure through different paths. While a new FULL runs, point-in-time restore must still use the preceding completed FULL, but restore-set assembly used that chain's retention boundary and hid WAL archived after the new FULL started. When every backup has been deleted, WAL archived during the replacement FULL had no completed anchor yet, so the orphan sweep could delete it before the FULL published its recovery positions.

## What Changes

- The orphan predicate anchors on the segment's end rather than its start: a WAL segment is retained when a COMPLETED FULL on the same timeline starts anywhere before the segment's `end_lsn`. The boundary segment that straddles a FULL's `start_lsn` is no longer classified as an orphan.
- Regression coverage pins the real-world shape the current tests miss. `chain_view/service_test.go:343` places the FULL's `start_lsn` exactly on a segment boundary, which is the one alignment where the buggy predicate happens to be correct; new coverage uses a mid-segment `start_lsn`.
- End-to-end coverage asserts that a FULL followed by WAL streaming leaves a restorable chain after the retention cleanup job has run, so the upload-then-delete sequence cannot regress silently.
- Restore-set assembly selects the base backup by completion time, then reads the full contiguous WAL run independently of the chain's retention boundary. Targets during a newer FULL continue to restore from the preceding completed FULL.
- FULL scheduling, orphan deletion, and cascade deletion use the same database advisory lock. Each destructive transaction rechecks the active FULL claim before removing WAL, so a newly claimed FULL cannot lose WAL before it publishes a completed anchor.
- Deleting a newer FULL preserves WAL while an older completed FULL remains on the same timeline, because that older FULL becomes the restore base after deletion. The deletion preview uses the same predicate as the destructive path.
- Not BREAKING. No schema, API, config or DTO changes. Existing rows are unaffected; the sweep simply stops deleting a segment it should never have deleted. WAL already destroyed on affected installations cannot be recovered by this change, and the proposal does not claim otherwise.

## Capabilities

### New Capabilities

- `physical-backups/wal-retention`: which archived WAL segments a database keeps and which the retention sweep may delete, expressed in terms of the recovery chains a user can still restore from.

### Modified Capabilities

None. `openspec/specs/` holds no physical-backup capability yet, so the retention behavior is stated for the first time here.

## Impact

Code:

- `backend/internal/features/backups/backups/core/physical/repositories/wal_segment_repository.go`: the `FindOrphans` predicate.
- `backend/internal/features/backups/backups/core/physical/chain_view/service_test.go`: orphan coverage with a mid-segment FULL `start_lsn`.
- `backend/internal/features/backups/backups/backuping/physical/cleaner_test.go`: the sweep leaves the boundary segment in place.
- `backend/internal/features/backups/backups/backuping/physical/`: a single-pass cleaner entry point for tests, since the only exported entry today is an infinite ticker.
- `backend/internal/features/tests/physical/postgresql/shared/` plus its `pg17` and `pg18` dispatchers: the backup-to-restore case that runs the sweep in between.

Behavior downstream: `ChainViewService.ResolveRestoreSet` reads WAL beyond the selected chain's retention boundary. `FindWalGapsInChain` continues to describe the chain-owned span used by retention.

Operations: one extra 16 MB segment is retained per database and timeline whose oldest COMPLETED FULL has no earlier FULL to anchor it. The anti-join matches on `timeline_id`, so the bound is per timeline, not per database. That is the correct cost of a restorable chain.

Rules this change answers to: `AGENTS.md` (root) and `backend/AGENTS.md`.

## Out of scope

- The missing `pg_switch_wal` grant on the replication-only user Databasus provisions. Separate change: `grant-wal-switch-to-replication-user`.
- Surfacing chain health in the UI, so that a broken chain stops reading as a successful backup. Worth doing and deliberately not done here.
- Any repair or re-fetch of WAL already deleted on a running installation.
- The `physical_backup_retention_cleanup` job's other duties: chain retention policies, GFS tiers, the grace period, and abandoned-claim reaping.
- Adding a switch to disable orphan cleanup.
- WAL archived on a new timeline after a promotion when no FULL claim or completed FULL anchors that timeline. Automatic re-anchoring after failover belongs to its own change.
