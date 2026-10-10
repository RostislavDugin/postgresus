## Why

A Databasus instance with physical PostgreSQL backups keeps one CPU core about 60% busy while no backup is running. A user with four databases measured about 440 committed transactions per second on the internal database, nearly all of them reads of full backups, incremental backups and WAL segments. The load does not depend on WAL streaming being on, comes back within minutes of a restart, and grows with every day of retained WAL.

The cause is the shape of the reads, not their frequency alone. Every second, the physical backup scheduler asks whether each database has a chain it can extend. Every three seconds, the retention cleaner asks how old each closed chain is. Both questions are answered by building the complete chain model: every chain's full backup list is reloaded once per chain, and every WAL segment row of every chain is loaded twice, only to be thrown away. Neither loop uses those WAL rows. The lookup that loads them is bounded on one side only, so each call reads the WAL from the start of the database's history up to the chain's end.

## What Changes

- The chain model carries a chain's full backup, its incremental backups and its LSN span. It no longer loads WAL segments, WAL gaps or the furthest replayable LSN. Production code never read those three fields; the restore path loads its own WAL.
- Chain state comes from one list of completed full backups. Only the newest full backup can head an extendable chain, so state is decided without reloading the list for every chain.
- A chain's end time is computed from the chain the cleaner already holds. The WAL part is a single aggregate over the chain's span, not a load of every row in it, and the cleaner computes it once per chain per pass instead of twice.
- The WAL span lookup is bounded on both sides, so it reads the segments of the requested chain and nothing older. No new index or migration is needed.
- The scheduler checks the incremental cadence before it looks up the extendable chain, so a pass with nothing due reads no chain data.
- The retention cleaner runs once a minute instead of every three seconds.
- **BREAKING (internal only):** the chain model loses its WAL segment, gap and replayable-LSN fields; the unused span-segment lookup on the chain service, the chain state enum and the per-chain state helper are deleted; the chain end time method takes a built chain instead of a full backup ID. No HTTP API, database schema or configuration changes.

## Capabilities

### New Capabilities

- `physical-backups/background-load`: how much catalog work the physical backup scheduler and retention cleaner do while idle, how often retention runs, and that a chain's WAL lookup reads only that chain's WAL.

### Modified Capabilities

None. Retention decisions, the grace period, chain end time, restore selection and the next-backup times reported to the interface keep their current behavior.

## Out of scope

- The scheduler tick interval. It stays at one second so out-of-cadence backup requests and crash recovery stay near-immediate; after this change a pass with nothing due is a few indexed reads per database.
- The orphan WAL sweep query, which already expects a sequential scan of full backups. It runs on the slower cleaner cadence and is not otherwise changed.
- Materializing chain state or chain end time in columns, triggers or an in-memory cache.
- Autovacuum settings of the bundled PostgreSQL and the shutdown grace period of the container, both mentioned in the report.
- The logical backup cleaner, which has its own three-second cadence and different costs.
- Other unused chain service methods that this change does not touch, such as the history file presence check.
- The backup list, WAL range rows and restore download, apart from the WAL lookup bound they share.

## Impact

- Backend only: `backend/internal/features/backups/backups/core/physical/chain_view` (chain model, chain service and its tests), `backend/internal/features/backups/backups/core/physical/repositories` (WAL segment span lookup and a new span aggregate), and `backend/internal/features/backups/backups/backuping/physical` (scheduler, cleaner and their cadence constants).
- Answers to `AGENTS.md` and `backend/AGENTS.md`.
- No security impact: no authorization path, credential or stored data changes.
- Trade-off: a chain that becomes eligible for deletion starts being deleted up to a minute later than today, and a chain whose WAL exceeds one pass's deletion budget finishes over several minutes instead of several seconds. Retention grace periods are at least an hour, so no restore point is affected.
