## Context

See proposal.md for the report and the measured load. Paths below are relative to `backend/internal/features/backups/backups/`.

Two background loops drive the load:

- The physical scheduler ticks every second (`backuping/physical/constants.go:16`). The comment above it (`constants.go:11`) assumes each tick is "a handful of cheap indexed queries per enabled DB". `decideBackupKind` calls `FindLastExtendableChainByDatabase` at `backuping/physical/scheduler.go:291`, before it checks the incremental cadence at `scheduler.go:300`, so the chain is built on every tick and discarded on almost all of them.
- The retention cleaner ticks every three seconds (`constants.go:29`). Each retention policy calls `nonExtendableChainsNewestEndFirst` (`backuping/physical/cleaner.go:119`), which builds every closed chain through `FindNonExtendableChainsByDatabase` and then calls `GetChainEndTimestamp` per chain (`cleaner.go:130`). `isChainWithinGrace` (`cleaner.go:225`) calls `GetChainEndTimestamp` a second time for every candidate it weighs.

A chain is built by the chain service in `core/physical/chain_view/service.go`:

- `getChainState` (`service.go:259`) loads the full backup by ID, then `findNewerCompletedFulls` (`service.go:294`) reloads every completed full backup of the database to look for a newer one, then loads the chain's incremental backups. It runs once per full backup, so building all chains reads the full backup list N times.
- `buildChainView` (`service.go:316`) computes the span, loads the incremental backups, loads the chain's WAL with `FindByChainSpan`, and calls `FindWalGapsInChain` (`service.go:128`), which recomputes the span and loads the same WAL again.
- `GetChainEndTimestamp` (`service.go:195`) reloads the full backup, the incremental backups and the span, and loads the chain's WAL a third time to take the newest `received_at`.

No production code reads the WAL that `buildChainView` loads. `ChainView.WalSegments`, `Gaps` and `MaxReplayableLSN` (`core/physical/chain_view/view.go:20-34`) are read only by tests. The scheduler and the backup service use the chain's root full backup; the cleaner uses the root full backup and the end time; restore uses the root full backup, the span start and the incremental backups, and loads its own WAL (`core/physical/chain_view/restore_set.go:61`).

The WAL lookup `FindByChainSpan` (`core/physical/repositories/wal_segment_repository.go:60`) filters on `end_lsn > start AND start_lsn < end`. The only index that matches is the unique key on `(database_id, timeline_id, start_lsn)` (`backend/migrations/20260523120000_add_physical_pg_17_backups.sql`), which bounds the scan from above only. Every lookup reads the database's WAL from its first segment up to the chain's end, and for the newest chain the end is `LSNMax`. Forced WAL rotation every five minutes produces at least 288 segments per database per day, so each lookup reads thousands of rows after a few weeks of retention.

Constraints from `backend/AGENTS.md`: tests live next to their source and follow `Test_WhatWeDo_WhichConditions_WhatWeExpect`; background services keep the `hasRun` guard; migrations, if any, use separate statements per index. Constraints from `AGENTS.md`: no backward-compatibility shims for internal callers; names state intent, with getters taking `Get`; comments carry only the reason the code cannot.

## Goals / Non-Goals

**Goals:**

- Scheduler and cleaner passes whose catalog reads do not depend on how many WAL segments a database retains.
- Each chain built with a constant number of indexed reads, and the full backup list read once per database per pass.
- No change to any scheduling, retention or restore outcome.

**Non-Goals:**

- Changing what the restore path, the backup list or the WAL range rows read, beyond the shared WAL lookup bound.
- Adding instrumentation or a query budget to the test suite.

## Decisions

### Decision 1: Ask the narrow question instead of materializing the answer

The background loops need two facts: which chain is extendable, and how old each closed chain is. The change makes those two questions cheap to answer from the catalog on every pass. It does not store chain state or chain end time anywhere.

Alternatives rejected:

- **Columns or triggers holding chain state and end time.** Every event that changes them would need to update them: a full backup completing, an incremental backup breaking the chain, a WAL segment arriving, retention deleting rows, a timeline switch. A missed update shows up as retention deleting a chain it should keep or keeping one it should delete. After this change the direct answer costs a few indexed reads, so the stored copy would buy nothing worth that risk.
- **An in-memory cache invalidated on writes.** It has the same invalidation problem, and it diverges as soon as a second process writes to the catalog.
- **Only lowering the tick frequency.** The per-pass cost would still grow with retention. Doubling the retained days would double the idle load again.

### Decision 2: The chain model drops WAL; chains are built from one list of full backups

`ChainView` keeps `RootFull`, `Incrementals` and `Span`, and loses `WalSegments`, `Gaps` and `MaxReplayableLSN`. `buildChainView` stops reading WAL. The view's doc diagram shrinks to the span. `FindWalGapsInChain` stays as a standalone method, because a test helper polls it (`usecases/physical/postgresql/testing.go:968`), and it keeps loading the chain's WAL once on its own.

`FindLastExtendableChainByDatabase` and `FindNonExtendableChainsByDatabase` read the completed full backups once, newest first. `getChainState` closes a chain whenever any newer completed full backup exists, so only the first entry of that list can be extendable. It is extendable when none of its incremental backups has status `CHAIN_BROKEN`, which the built view already holds. Every later entry is closed. `getChainState`, `findNewerCompletedFulls` and the `ChainState` enum are deleted. The rule that a WAL gap never changes chain state, today written on the enum (`core/physical/chain_view/enums.go:3-6`), moves to the check that decides extendability.

`GetChainSpan(rootFullBackupID)` today reloads the root full backup with `FindByID` before its successor query (`service.go:77-80`). Its body moves into a private `getChainSpanForFull(full)`, which `GetChainSpan` wraps after the lookup and `buildChainView` calls with the full backup it already holds. Building a chain then costs the successor query plus one read of its incremental backups.

`FindWalSegmentsInSpan` (`service.go:120`) has no production caller and is deleted. The two span tests that call it read through the repository instead.

Alternatives rejected:

- **Keep the WAL fields and load them lazily.** Lazy loading in a plain struct means a method with hidden I/O, and no caller asks for the data.
- **Compute every span in memory from the loaded full backup list.** It saves one indexed read per chain but duplicates the successor rule, including its timeline filter and its treatment of a missing `start_lsn`. Keeping one implementation is worth more than that read.
- **Batch every chain's incremental backups into one query.** It saves N small indexed reads per database per minute and adds a grouping step. The chain count per database is small, and the cleaner runs once a minute after this change.

A tie on `created_at` between two completed full backups is the one input where the new rule can differ: today both count as not closed by a newer backup. Two full backups of one database cannot complete in the same instant, because the scheduler serializes backups per database through the in-flight claim, so this case is not handled.

### Decision 3: A chain's end time is computed from the built chain, with one WAL aggregate

`GetChainEndTimestamp` takes the `*ChainView` the caller already holds. It takes the latest of the root full backup's `CompletedAt` and the incremental backups' `CompletedAt` from the view, and asks the WAL repository for one value: the newest `received_at` among the segments overlapping the chain's span. A new repository method, `GetLatestReceivedAtInChainSpan`, returns that value with `MAX(received_at)` under the same span condition as `FindByChainSpan`, and returns nil when the span holds no segment.

The cleaner computes each candidate's end time once in `nonExtendableChainsNewestEndFirst` and passes it to `isChainWithinGrace`, which takes the end time instead of a full backup ID. The second computation per candidate disappears. Today `isChainWithinGrace` treats a failure to read the end time as "within grace" (`cleaner.go:225`), and `nonExtendableChainsNewestEndFirst` already skips a chain whose end time fails to load (`cleaner.go:130`). A skipped chain is never deleted, so dropping the second read keeps the safety property.

Alternatives rejected:

- **Use the `received_at` of the last segment by position.** It is one index probe, but it equals the maximum only while segments arrive in position order. A segment re-received after its abandoned claim was reaped breaks that order, and the result would make a chain look older than it is to the grace check.
- **One query per database that joins every chain's span to the WAL.** A range join between spans and segments has no hash or merge strategy, so PostgreSQL would run it as a nested loop. It ends up doing the same work as one aggregate per chain, with a harder query to read.

### Decision 4: Bound the WAL span lookup from below through the segment that holds the span start

The span condition gains a lower bound on the indexed column. Segments of one database and timeline never overlap, which the orphan sweep already relies on (`cleaner.go:526`). Any segment that ends after the span start therefore starts at or after the start of the segment holding the span start, meaning the greatest `start_lsn` at or below it. The condition becomes:

```sql
database_id = ? AND timeline_id = ?
AND start_lsn >= COALESCE(
    (SELECT start_lsn FROM physical_wal_segments
     WHERE database_id = ? AND timeline_id = ? AND start_lsn <= ?::pg_lsn
     ORDER BY start_lsn DESC LIMIT 1),
    '0/0'::pg_lsn)
AND start_lsn < ?::pg_lsn
AND end_lsn > ?::pg_lsn
```

PostgreSQL evaluates the uncorrelated subquery once as an init plan and uses its result as the lower index bound. The scan covers the chain's segments and nothing older. The `end_lsn` filter stays, so the result set is identical to today's. `FindByChainSpan` and `GetLatestReceivedAtInChainSpan` share one private helper that applies this condition, so the two cannot drift.

Alternatives rejected:

- **An index on `(database_id, timeline_id, end_lsn)`.** It bounds the scan from below only, so a lookup for an old chain would read every newer segment instead of every older one. It also adds a write to the most frequently inserted catalog table and a migration.
- **A lower bound of `start - wal_segment_size`.** The segment size is a per-cluster setting from 1 MB to 1 GB, and the catalog does not store it per database.
- **A range type with a GiST index.** It answers overlap queries directly but needs a schema change and a new column kept in sync for a lookup that the init plan already bounds.

### Decision 5: The scheduler checks the cadence before it looks up the chain

In `decideBackupKind`, the `IncrementalBackupInterval.ShouldTriggerBackup` check moves above the `FindLastExtendableChainByDatabase` call. Nothing between them reads the chain: the cadence check uses only the newest full and incremental backups, which are already loaded. The forced-incremental branch above it keeps its own lookup, because a request must be honored regardless of cadence.

The scheduler tick stays at one second. With the cadence check first, an idle pass reads the newest full backup, the system identifier mismatch marker and the newest incremental backup, all through existing indexes. The comment at `constants.go:11` becomes true again and stays.

Alternative rejected: **a longer scheduler tick.** A user pressing "back up now" would wait up to the new interval, and the idle cost is already flat after this change.

### Decision 6: The retention cleaner runs once a minute

`cleanerTickInterval` becomes one minute and its comment states the reason: retention grace periods are at least an hour (`recentBackupGracePeriod`), abandoned WAL claims are reaped after an hour (`walClaimGracePeriod`), so a minute of extra delay changes no outcome, while each pass still reads every closed chain of every database and runs the orphan anti-join. The per-pass WAL deletion budget stays as it is.

Alternatives rejected:

- **Keep three seconds.** After Decisions 2 to 4 a pass is cheap, but it still scales with the number of chains, and nothing gains from running it twenty times a minute.
- **Five seconds.** It cuts load less and protects nothing that one minute does not.
- **Event-driven retention.** It needs a signal from every write path that can change retention eligibility, including the clock crossing a grace boundary. Polling at a minute is simpler and robust to restarts.

## Risks / Trade-offs

- [Two segments of one database and timeline overlap, for example after a database record is pointed at a different cluster with another segment size] → The lower bound could skip the older, larger segment. The orphan sweep already assumes no overlap, and automatic scheduling stops after a system identifier mismatch, so such a catalog is already outside supported behavior. A repository test pins the boundary segment case that does occur: a full backup starting partway into a segment.
- [Large WAL backlog deletes more slowly] → A chain whose WAL exceeds one pass's budget now takes several minutes to delete instead of several seconds. The budget is one full backup's size per pass, which outpaces WAL production by a wide margin, so storage still shrinks.
- [Tests relied on the removed fields] → `Test_FindLastExtendableChainByDatabase_WhenChainHasGap_MaxReplayableLSNStopsAtFirstGap` tests a field that no longer exists and is deleted. Its gap behavior stays covered by the `FindWalGapsInChain` tests and by the restore set's own `MaxReplayableLSN` tests. The assertions on `WalSegments` and `Gaps` in the healthy-chain test are removed.
- [No automated guard against the load returning] → No test measures load or query counts; the tests pin outcomes only. The guard is structural: the chain model has no WAL field, so no caller can load WAL through it by accident, and the end time path exposes only an aggregate. A query-count test was rejected because it needs new GORM callback plumbing in shared test setup and breaks on unrelated query additions. A plan-shape test on the WAL lookup was considered and left out by decision: this change pins correctness, not performance.
- [Three spec scenarios have no test] → They are confirmed by code structure and review instead:
  - A scheduling pass with nothing due reads no chain history. This holds because the cadence check sits above the chain lookup in `decideBackupKind` (Decision 5); observing the absent read would need the query counting rejected above.
  - Retention runs once a minute. This is the value of `cleanerTickInterval`; a test of a constant checks nothing.
  - Each closed chain's end time is computed once per pass. This holds because `isChainWithinGrace` takes the end time as an argument and has no way to compute it (Decision 3).
- [The bounded WAL lookup decides which WAL a restore ships] → Repository tests pin the boundary cases: a full backup starting partway into a segment, a span starting exactly on a segment boundary, a span starting before the first archived segment, and segments of other timelines and databases at nearby LSNs. The backup and restore cycle packages for PostgreSQL 17 and 18 run as the final check.

## Migration Plan

No schema or data migration. The change ships with the next release; rollback is a revert of the code.
