## Why

PostgreSQL promotion changes the live timeline and invalidates an incremental chain rooted on the previous timeline. Databasus can currently miss that mismatch after a `.history` file is cataloged, and a failed backup does not cause a prompt replacement FULL, so operators must trigger one manually.

## What Changes

- Validate each INCR against the timeline of its own root FULL instead of the newest timeline found across FULL and `.history` rows.
- Recheck cluster identity after an unexpected stream failure so promotion during `pg_basebackup` is classified correctly.
- Schedule one immediate replacement FULL after a timeline-related INCR or FULL failure, then retain the existing cadence and failed-FULL brake for later failures.
- Prevent orphan cleanup from deleting WAL while a FULL is active.
- Allow same-timeline PITR from an older FULL to use WAL received while a newer FULL was still running.
- Add distinct `TIMELINE_SWITCH_DETECTED` and `FAILOVER_DURING_BACKUP` error reasons. This extends the existing API enum without removing or renaming existing values.

Out of scope:

- PITR across multiple PostgreSQL timelines.
- Timeline ancestry resolution and manifests with multiple `WAL-Range` entries.
- Changing retention policy or backup cadence configuration.

This change answers to `AGENTS.md` and `backend/AGENTS.md`.

## Capabilities

### New Capabilities

- `postgresql-physical-backup-failover`: Detect timeline changes around physical backups, re-anchor automatically, and preserve the WAL needed during re-anchoring.

### Modified Capabilities

None.

## Impact

The change affects PostgreSQL physical backup timeline validation, backup result classification, scheduler decisions, WAL cleanup coordination, and same-timeline restore selection. It adds no table, configuration field, endpoint, or external dependency.
