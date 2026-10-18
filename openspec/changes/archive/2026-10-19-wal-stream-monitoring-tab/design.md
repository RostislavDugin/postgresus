## Context

See proposal.md for the motivation. Backend paths below are under `backend/internal/features/`, frontend paths under `frontend/src/`, and `<pg>` stands for `backups/backups/usecases/physical/postgresql`.

The change has two consumers in mind. The Monitoring tab is the first. The second is anyone reading the authenticated API, so the statistics endpoint is a contract in its own right, not the tab's private feed.

What exists today:

- Every 30 seconds the WAL streamer's lag monitor (`<pg>/lag_monitor.go:121`) probes the slot through `GetSlotStateIfReachable` (`<pg>/slot_lsn_watcher.go:74`), which calls `InspectSlot` (`<pg>/slot.go:29`). The query returns `active`, `wal_status`, `restart_lsn` and the lag of `restart_lsn` behind `pg_current_wal_lsn()`, but neither `safe_wal_size` nor the current position itself. When the probe fails, the tick does nothing else (`lag_monitor.go:135-138`). The probe's connection has no timeout: the backend sets no `connect_timeout` anywhere (`backuping/physical/constants.go:64-67`), so a silent source holds the tick.
- The source's current position is derived as `restart_lsn + lag` (`lag_monitor.go:219`). For a `lost` slot `InspectSlot` substitutes `0/0` and `0` (`slot.go:38-42`), so that sum becomes 0. The archive progress point is 0 while nothing has been archived (`<pg>/wal_staleness.go:86-90`).
- The probe returns `(nil, false)` both for an unreachable source and for a source without the slot (`slot_lsn_watcher.go:93-98`). Its other callers, the slot-LSN watcher (`slot_lsn_watcher.go:125`) and the resume path (`<pg>/wal_resume.go:108`), treat both the same way.
- The same tick logs the archiving rollup from one upload snapshot (`lag_monitor.go:175-180`). The snapshot holds the archived segment count, their compressed size, the per-step upload timings and the highest concurrency (`<pg>/wal_upload_stats.go:35-45`). Taking it resets the counters (`wal_upload_stats.go:71-78`, through `<pg>/wal_upload.go:190`).
- The tick feeds two trackers that decide notifications. The slot classifier remembers since when the slot's status has been `extended` or `unreserved` and alerts after `warningSlotStatusHoldPeriod`, 5 minutes (`lag_monitor.go:18`, `:60-114`). The staleness tracker remembers since when the archive progress point has stood still while the source writes, keeps that clock through quiet ticks, and alerts after `DefaultArchiveStalenessThreshold`, 30 minutes (`wal_staleness.go:13`, `:29-82`). For a `lost` slot, or a slot lag past `WalLagThresholdBytes` while the receiver runs and the status is neither `extended` nor `unreserved`, the classifier asks for a rebuild, which drops and recreates the slot in the same tick (`lag_monitor.go:88-114`, `:160-167`) and requests a full backup (`backuping/physical/wal_supervisor.go:370-394`). Session start also recreates a missing slot silently (`databases/databases/postgresql/physical/model.go:347`, called from `<pg>/wal_stream.go:273`). Both leave a WAL gap in the chain; the slot looks healthy right after.
- The forced WAL rotation loop stops for the rest of the session when the source refuses `pg_switch_wal()` and sends a `WAL_ROTATION_DENIED` alert (`<pg>/wal_rotation.go:100-113`).
- The local queue size is computed on demand by `backlogBytes` (`<pg>/wal_backpressure.go:71`). The receiver is paused at `highWatermarkBytes`, at least 512 MiB and at least eight segments (`<pg>/wal_stream.go:245`). Back pressure and a stalled slot both stop the receiver through `receiverInternalRestart` (`<pg>/receivewal_process.go:122`); before respawning, the supervision loop waits for the queue to drain (`wal_stream.go:340`). Nothing counts restarts or pauses.
- A receiver that keeps exiting quickly escalates after five rapid failures and ends the session with an error (`wal_stream.go:401-412`). With the source down, that takes under a minute, and the next session fails in `VerifyWalSlot` before its lag monitor starts.
- Streamer ownership lives in `physical_wal_streamers` (`backups/backups/core/physical/models/wal_streamer.go`): status `RUNNING` or `FAILED`, `started_at` and a heartbeat. The supervisor ticks every 15 seconds and reclaims a `FAILED` row or one whose heartbeat is older than 90 seconds, resetting `started_at` (`backuping/physical/constants.go:48`, `:54`; `backups/backups/core/physical/repositories/wal_streamer_repository.go:42-76`). Session errors and failed starts mark the row `FAILED` (`wal_supervisor.go:307-321`, `:588-592`). A `FAILED` row therefore lasts at most one tick, and a `RUNNING` row can belong to a process that is gone.
- A WAL streaming config always has a rebuild lag: `WalLagThresholdBytes` defaults to 256 MiB (`backups/config/physical/constants.go:6`, `model.go:150-153`). Whether streaming should run is `isWalStreamCandidate`, applied to configs with backups enabled (`wal_supervisor.go:142`, `:175-182`).
- WAL gaps in a chain are computed by `ChainViewService.FindWalGapsInChain` from the committed segments within the chain's span (`backups/backups/core/physical/chain_view/service.go:83-123`). The newest completed full backup comes from `FindCompletedNewestFirstByDatabase` (`chain_view/service.go:25-45`).
- The retention cleaner ticks every minute (`backuping/physical/cleaner.go:40-51`, `constants.go:30`).
- The only periodic-sample table today is `healthcheck_attempts`, with an index on `(database_id, created_at)` (`backend/migrations/20250704095930_add_healthckeck.sql:40`).
- The physical backup endpoints authorize through `databaseService.GetDatabase(ctx, user, databaseID)` (`backups/backups/services/physical_service.go:85`) and answer a refusal with 400 and the error text (`backups/backups/controllers/physical/controller.go:91-94`). `PhysicalBackupService` holds the backup config service (`physical_service.go:58`).
- Packages under `<pg>` and `backuping/physical` import `backups/backups/core/physical`; nothing in `core/physical` imports them.
- The database page builds its tabs by hand (`features/databases/ui/DatabaseComponent.tsx:28`, headers at `:65-86`); the Verifications tab is conditional on the database type (`:79`). `recharts` 3 is in `package.json` but nothing imports it since monitoring was removed in `fcc894d1`. The healthcheck component polls only its "today" view (`features/healthcheck/ui/HealthckeckAttemptsComponent.tsx:105-110`). Translation keys are typed against `en.ts`, and every other dictionary is typed `typeof en`, so a key exists for `pnpm typecheck` only once it is in all six dictionaries.

Constraints:

- `backend/AGENTS.md`:
  - Migrations: TIMESTAMPTZ columns, with the table, constraints and indexes as separate statements.
  - Controllers and services: one controller per feature, Swagger annotations on every endpoint, no injecting another feature's repository.
  - Testing: controller tests preferred; unit tests only for complex model logic with no API surface.
  - Time: `time.Now().UTC()`.
  - Logging: lowercase messages, IDs as key-value pairs.
- `frontend/AGENTS.md`:
  - AntD, `@ant-design/icons` and Tailwind only.
  - Component member order.
  - No literal user-facing strings; every key in all six dictionaries; counts phrased so one string fits every number.
  - Enum labels through `*_LABEL_KEYS` tables; code outside components holds keys, not text.
  - `useLocale()` for `formatNumber` and `formatRelativeTime`.
  - API errors through `translateApiError`.
  - FSD imports only downward; one type per file in `model/`.
- ADR-0008: physical backups run on PostgreSQL 17 and 18, so `safe_wal_size` (added in 13) is always available.
- The `single-instance-runtime` spec runs one application process. Samples still have to outlive a restart, because the hour before a restart is what an operator wants to see.

## Goals / Non-Goals

**Goals:**

- One verdict a junior operator understands without knowing what a replication slot is.
- When the verdict is not healthy, the reason, its size, its age and what to do.
- One statistics API other consumers can rely on, with units and codes instead of display text.
- The verdict agrees with the existing notifications about when a condition began.
- The verdict stays red while the backup chain is actually broken, not only while the slot looks bad.
- No extra queries against the source beyond widening the existing slot query.

**Non-Goals:**

- New notifications or alerting on sample values.
- Sub-30-second resolution.
- Configurable thresholds.

## Decisions

### D1. Samples persisted in a new table

New table `physical_wal_stream_samples`, one row per streaming database every 30 seconds:

| Column | Type | Null means |
|---|---|---|
| `id` | UUID, `gen_random_uuid()` | |
| `database_id` | UUID, FK to `databases(id)` ON DELETE CASCADE | |
| `sampled_at` | TIMESTAMPTZ | |
| `session_started_at` | TIMESTAMPTZ | |
| `source_state` | TEXT: `REACHABLE`, `UNREACHABLE`, `SLOT_MISSING` | |
| `source_unreachable_since` | TIMESTAMPTZ | source reachable |
| `source_current_lsn` | PG_LSN | no source-side values |
| `archive_progress_lsn` | PG_LSN | nothing archived yet |
| `source_written_bytes` | BIGINT | see D4 |
| `archived_segments` | INTEGER | |
| `archived_wal_bytes` | BIGINT | |
| `archived_stored_bytes` | BIGINT | |
| `wal_segment_size_bytes` | BIGINT | |
| `highest_concurrent_uploads` | INTEGER | |
| `segment_write_avg_millis` | INTEGER | nothing archived since the previous sample |
| `segment_write_max_millis` | INTEGER | nothing archived since the previous sample |
| `slot_wal_status` | TEXT | no source-side values |
| `slot_extra_wal_since` | TIMESTAMPTZ | the classifier holds no start time |
| `slot_lag_bytes` | BIGINT | no source-side values |
| `slot_safe_wal_size_bytes` | BIGINT | no limit, or no source-side values |
| `unarchived_wal_bytes` | BIGINT | no source-side values, or nothing archived yet |
| `archive_stalled_since` | TIMESTAMPTZ | the staleness tracker holds no start time |
| `local_queue_bytes` | BIGINT | |
| `local_queue_pause_bytes` | BIGINT | |
| `receiver_restarts` | INTEGER | |
| `receiver_pauses` | INTEGER | |
| `is_receiver_running` | BOOLEAN | |
| `is_forced_rotation_denied` | BOOLEAN | |

"No source-side values" means `source_state` is not `REACHABLE`. A NULL `slot_safe_wal_size_bytes` with a reachable source therefore means no limit. The LSN columns use `PG_LSN`, like `physical_wal_segments`.

There are two indexes: `(database_id, sampled_at)` for reads, and `(sampled_at)` for the global expiry delete in D7.

`archived_wal_bytes` is the archived segment count times `wal_segment_size_bytes`, so it compares directly with `source_written_bytes`. `archived_stored_bytes` is the compressed and encrypted size the snapshot already counts, so the ratio of the two shows compression. `wal_segment_size_bytes` and `local_queue_pause_bytes` depend on the source's segment size, which only the streamer knows. Storing them per sample costs 16 bytes and keeps the read path free of streamer internals.

A row is about 250 bytes with its index entries. At 2,880 rows a day and 7 days that is about 5 MB per streaming database.

Alternatives rejected:

- *An in-memory ring buffer in the streamer.* No migration, but a restart or a crash loop erases the history that explains it.
- *Deriving throughput from `physical_wal_segments`.* It has no source-side values, its timestamps are set before the upload, and retention deletes the rows the history needs.
- *A boolean `is_source_reachable`.* It cannot tell a missing slot from an unreachable source. A missing slot means the chain is broken, while an unreachable source may be a network blip.

### D2. Samples are taken on the lag monitor's tick

The lag monitor already wakes every 30 seconds, probes the slot and takes the upload snapshot. The tick's body becomes a method that takes `now`, probes, classifies, records the sample and then acts on the classification. Tests call that method directly with a real test source instead of waiting 30 seconds per sample. The sample itself is assembled by a pure function from the tick's inputs: the probe outcome, the snapshot, the previous position, the tracker times and `now`. Edge cases a test source cannot produce on demand, such as a `lost` slot or a timeline switch, are tested through that function. The sample goes through a repository passed in `WalStreamSpec`, the same way `WalSegmentRepo` and `HistoryRepo` are passed today (`wal_supervisor.go:283-284`).

Order within a tick:

1. Probe the source with a 10-second timeout on the probe's context.
2. Feed the classifier and, when reachable, the staleness tracker, as today.
3. Record the sample. Its slot values describe the slot as probed, before any rebuild in step 4.
4. Act on the classification: alert or rebuild, as today.

When the probe fails, the tick still records a sample with the source-side columns NULL instead of skipping. The trackers keep their last state on such ticks, so the "since" columns carry over. The verdict ignores reasons read from them while the source is unreachable (D8).

The probe changes to return one of three outcomes: reachable with a slot state, reachable without the slot, unreachable. The slot-LSN watcher and the resume path map "without the slot" to their current behavior; only the lag monitor uses the difference.

`source_unreachable_since` is kept by the lag monitor. A new session starts it from the newest stored sample of the database when that sample was unreachable, so the duration survives the crash loop a down source causes.

The upload snapshot is taken once per tick and used by both the log line and the sample, because it resets on read and two readers would each see half the data.

Alternatives rejected:

- *A separate sampler goroutine.* It would probe the slot a second time every 30 seconds and need its own coordination for the reset-on-read counters.
- *Making the poll interval a `WalStreamSpec` field for tests.* Tests would still wait real time, only less of it; calling the tick directly makes them deterministic.

### D3. The slot query reads the WAL allowance and the current position

`InspectSlot` adds `s.safe_wal_size` and `pg_current_wal_lsn()` to its select list. `safe_wal_size` is NULL when `max_slot_wal_keep_size` is `-1` and when the slot is `lost`, so `SlotState` carries it as a nullable value; together with `wal_status`, that tells "no limit" from "already lost". The current position replaces `restart_lsn + lag` in the staleness tracker, in D4 and in `unarchived_wal_bytes`, which then stay correct for a lost slot.

`unarchived_wal_bytes` is the current position minus the archive progress point, NULL while the progress point is 0 because nothing has been archived yet.

Alternative rejected: *reading `current_setting('max_slot_wal_keep_size')` as well.* `safe_wal_size` already says how much room is left and whether a limit exists. The setting adds a value nobody shows and GUC unit parsing.

### D4. Source write volume from consecutive positions

The lag monitor keeps the previous tick's current position in memory. `source_written_bytes` is the difference to the current position. It is NULL on the first tick of a session, after a tick without source-side values, and when the position moved backwards, which happens on a timeline switch to a promoted replica.

Alternative rejected: *storing only the raw position and differencing at read time.* It moves the same logic to the read path and makes bucketing harder, without being more accurate. The raw position is stored anyway for the chain gap rule in D8.

### D5. Restarts, pauses and repeated failures

An atomic counter on the streamer grows each time the supervision loop respawns `pg_receivewal`. A respawn that followed a wait for the local queue to drain counts as a pause in `receiver_pauses`; any other counts as a restart in `receiver_restarts`. Back pressure is Databasus working as designed, so it must not look like a crash. The lag monitor takes and resets both counters with each sample.

A whole-streamer restart starts a new session. Each sample carries `session_started_at`, the time the current `WalStreamSupervisor` started running, so the read path counts session starts by `session_started_at` within a window, not by `sampled_at`.

A session that fails before its first tick leaves no sample. To see that, `physical_wal_streamers` gains a nullable `failing_since` TIMESTAMPTZ. `MarkFailed` sets it with `COALESCE(failing_since, NOW())`, so a crash loop keeps the first failure. The lag monitor clears it when it inserts the first sample of a session. `MarkStaleRunningFailed` at startup sets it the same way; a healthy restart clears it within one tick.

The rotation loop sets an atomic flag when the source refuses `pg_switch_wal()`. Each sample copies it to `is_forced_rotation_denied`.

Alternatives rejected:

- *Counting session starts in `physical_wal_streamers`.* That row is overwritten on each claim and has no history.
- *Storing the failure's error text.* Connection errors carry host names and user names, and the spec forbids connection details in the statistics.

### D6. Condition start times come from the notification trackers

The classifier's `warningStatusSince` and the staleness tracker's `fallingBehindSince` are written into `slot_extra_wal_since` and `archive_stalled_since` on every tick. The thresholds that turn these times into reasons live in one place, see D8.

The verdict then agrees with the notifications about start times by construction. A user who gets an "archiving fell behind" message and opens the tab sees the same reason at the broken level, with the same start time. The levels differ in one case by design: a steadily `extended` slot is a notice in the tab (D8), while the existing notification still fires for it.

Alternative rejected: *recomputing the conditions from sample history at read time.* It duplicates the tracker logic, and the two copies would drift.

### D7. Retention by one delete per cleaner tick

The retention cleaner, which ticks every minute, deletes all samples older than 7 days in one statement on the `(sampled_at)` index. This also expires the samples of a database that stopped streaming, which a delete on insert would never reach. Database removal is covered by ON DELETE CASCADE.

Alternatives rejected:

- *A delete after each insert, as `healthcheck_attempts` does.* It never runs for a database that stopped streaming, so its samples would outlive the 7 days.
- *Raw samples for one day plus 5-minute rollups for a week.* It saves about 4 MB per database, but adds a second table, a rollup job and two read paths for the week view.

### D8. The verdict is computed in the backend

A new package `backups/backups/core/physical/wal_health` holds the verdict as a pure function of its inputs: the samples of the last hour with the newest one, the streamer row, the rebuild lag, whether streaming is expected, whether a full backup has completed, whether the newest chain has a permanent gap, whether a full backup is pending, and `now`. A full backup is pending when the config's `ForceFullRequestedAt` is set or a full backup is in flight. It returns the level, the reasons and the notices.

The package also holds every threshold as a named constant. `warningSlotStatusHoldPeriod`, `DefaultArchiveStalenessThreshold` and the 90-second takeover age (`streamerHeartbeatStaleness`) move here, and `<pg>` and `backuping/physical` import them, so the notification trackers, the supervisor and the verdict share one value. The import direction already allows that. `isWalStreamCandidate` becomes `PhysicalBackupConfig.IsWalStreamingExpected()`, which also checks `IsBackupsEnabled`, so the supervisor and the read path share it.

Levels, first match wins: `NOT_STREAMING`, `COLLECTING_DATA`, `BROKEN`, `NEEDS_ATTENTION`, `HEALTHY`. The spec defines them. The streamer state is `RUNNING` only when the row says `RUNNING` and the heartbeat is younger than the takeover age, `NOT_RUNNING` otherwise, `NOT_STARTED` without a row.

Reasons. "2 samples" marks the rule that the condition must hold in the two newest samples. "Slot" marks reasons dropped while the newest sample has no source-side values. Every reason except `WAL_CHAIN_BROKEN`, `STREAMING_STOPPED` and `SOURCE_UNREACHABLE` is dropped when the newest sample is older than 2 minutes.

| Code | `NEEDS_ATTENTION` | `BROKEN` | Carries |
|---|---|---|---|
| `STREAMING_STOPPED` | | `failing_since` 1 min or older, or newest sample older than 2 min; not when the newest sample is `UNREACHABLE` | `since` = newest `sampled_at` or `failing_since` |
| `SOURCE_UNREACHABLE` | newest sample `UNREACHABLE`, `source_unreachable_since` 1 to 5 min ago | the same, 5 min or longer | `since` |
| `WAL_CHAIN_BROKEN` | | the newest completed full backup's chain has a gap ending at or below the newest `archive_progress_lsn` | |
| `SLOT_MISSING` | | newest `source_state` is `SLOT_MISSING` | |
| `SLOT_LOST` | | newest `slot_wal_status` is `lost` (slot) | |
| `ARCHIVE_NOT_ADVANCING` | `archive_stalled_since` 10 min or older (slot) | 30 min or older | `since` |
| `ARCHIVE_SLOWER_THAN_SOURCE` | growth over 10 min at least the backlog threshold (slot, 2 samples) | | `valueBytes`, `thresholdBytes` |
| `SLOT_KEEPS_EXTRA_WAL` | `slot_extra_wal_since` older than 5 min, and status `unreserved` or slot lag growing (slot) | | `since` |
| `SLOT_LIMIT_APPROACHING` | `slot_safe_wal_size_bytes` over the slot lag's growth rate under 1 h (slot, 2 samples) | | `valueBytes`, `estimatedSeconds` |
| `NEAR_SLOT_REBUILD` | `slot_lag_bytes` at least 80% of `WalLagThresholdBytes`, status neither `extended` nor `unreserved` (slot, 2 samples) | | `valueBytes`, `thresholdBytes` |
| `LOCAL_QUEUE_FILLING` | `local_queue_bytes` at least half of `local_queue_pause_bytes` (2 samples) | | `valueBytes`, `thresholdBytes` |
| `FREQUENT_RESTARTS` | restarts plus session starts in the last hour, 3 or more | | `count` |
| `FORCED_ROTATION_REFUSED` | newest `is_forced_rotation_denied` | | |

Notices, level `INFO`, never raise the level: `NO_SLOT_WAL_LIMIT`, `SLOT_KEEPS_EXTRA_WAL_STEADILY` and `NO_FULL_BACKUP_YET`, with the conditions in the spec.

The backlog threshold is `max(64 MiB, 4 × wal_segment_size_bytes)`. Growth is the lowest unarchived value in the window's last 2.5 minutes minus the lowest in its first 2.5 minutes. The lowest values strip the sawtooth of in-flight uploads, which reaches four segments plus the open one in normal operation.

`WAL_CHAIN_BROKEN` reads durable state, not the slot, because a rebuild makes the slot look healthy within one tick. A gap counts only when it ends at or below the archive progress point: that point never passes a pending segment, so a hole left by out-of-order parallel uploads is still above it. The reason clears when a newer full backup completes and starts a new chain. Databasus requests that backup itself only after a rebuild in the lag monitor (`wal_supervisor.go:370-394`); the silent recreate at session start requests none. The reason's "what to do" sentence therefore depends on whether a full backup is pending. A new `ChainViewService.GetNewestCompletedFullWithGaps(databaseID)` returns the newest completed full backup, whatever its extendability, with the `FindWalGapsInChain` result for its chain, or nil without one. `FindLastExtendableChainByDatabase` is not used, because it returns nil when an incremental broke the chain, which would hide both the gap and the existing full backup.

`NEAR_SLOT_REBUILD` stays listed while the receiver is paused. The classifier rebuilds on the first tick the receiver runs again with the lag above the threshold, so a paused receiver above it is exactly the risky case.

The thresholds sit just ahead of what already acts. Half the pause size warns before back pressure stops the receiver. 80% of the rebuild lag warns before Databasus sacrifices the chain. Three restarts an hour tolerate a deploy and one blip.

Alternatives rejected:

- *Computing the verdict in the browser.* Most thresholds live in the backend, the API would have no verdict for other consumers, and two copies of the rules would drift.
- *Making the missing slot limit or a steadily `extended` slot orange.* PostgreSQL's default is no limit, and `extended` is routine on a busy cluster (`lag_monitor.go:64-67`). Either would show "needs attention" permanently and teach the reader to ignore orange.
- *Recording slot rebuilds in samples and clearing on the next full backup.* It misses the silent recreate at session start and any other gap; the chain itself is the authority.

### D9. Keeping-up summary

The same package computes the keeping-up summary from the samples of the last 10 minutes:

- `sourceWriteBytesPerSecond` and `archiveBytesPerSecond`: the sums of `source_written_bytes` and `archived_wal_bytes` over the samples that hold a write value, both divided by the seconds those samples cover, so a hole in the window lowers neither rate alone;
- `unarchivedWalBytes` and `unarchivedWalChangeBytesPerSecond`, from the lowest values at both ends of the window as in D8;
- `slotLagBytes` and `slotLagChangeBytesPerSecond`, the same way;
- `slotRebuildLagBytes`, the config's `WalLagThresholdBytes`;
- `estimatedCatchUpSeconds`, `estimatedSlotRebuildSeconds` and `estimatedSlotLimitSeconds`, under the conditions in the spec;
- `state`: `UNKNOWN`, `FALLING_BEHIND`, `CATCHING_UP`, `IDLE` or `KEEPING_UP`, first match wins.

The window is 10 minutes because a 30-second rate swings with every upload batch, while an hour hides a backlog that started five minutes ago.

Alternative rejected: *letting the frontend derive rates from the series.* The series' bucket length differs by period, so the "now" numbers would change with the selected period.

### D10. One statistics endpoint with server-side bucketing

`GET /backups/physical/database/:id/wal-stream/statistics?period=HOUR|DAY|WEEK` on the existing `PhysicalBackupController`. `PhysicalBackupService.GetWalStreamStatistics` authorizes through `databaseService.GetDatabase` like `GetBackups`, refuses a database that is not PostgreSQL physical, reads the backup config through its config service, asks the chain view for the newest completed full backup and its gaps, reads whether a full backup is pending, and calls a new read method on the physical core service. That service owns the samples repository and the streamer repository and calls `wal_health`. Refusals return 400 with the error text, as `GetBackups` does. An unknown ID and a database in another workspace therefore get different messages. The spec accepts that, because every other physical backup endpoint behaves the same way and changing it for one endpoint would protect nothing.

Response:

- `health`: `level`, `reasons`, `notices` and `isFullBackupPending`. Each entry has `code`, `level` and, when known, `since`, `valueBytes`, `thresholdBytes`, `count` and `estimatedSeconds`.
- `streamer`: `state`, `sessionStartedAt`, `failingSince`, `restartsLastHour`.
- `current`: the newest sample's values under the column names in camelCase, plus `sampledAt` and `lastArchivedWalAt`, the `received_at` of the newest committed segment, which the "archiving fell behind" notification also reports.
- `keepingUp`: D9.
- `series`: `period`, `bucketSeconds`, `from`, `to` and `points`. Each point has `startedAt`, `sourceWriteBytesPerSecond`, `archiveBytesPerSecond`, `archivedStoredBytesPerSecond`, `maxSlotLagBytes`, `maxUnarchivedWalBytes`, `maxLocalQueueBytes`, `localQueuePauseBytes`, `restarts`, `pauses`, `sessionStarts`, `highestConcurrentUploads`, `segmentWriteAvgMillis` and `segmentWriteMaxMillis`.

Enums are Go string types with Swagger enum annotations, so clients get the code lists from the API docs.

Buckets are 1 minute for an hour (60 points), 4 minutes for a day (360 points) and 28 minutes for a week (360 points). Buckets align to `date_bin(stride, ts, '2000-01-01')`. The series starts one stride after `date_bin` of `now - period`, so it holds exactly period ÷ stride buckets and the last one contains `now`. A `generate_series` of bucket starts is left-joined to the binned samples, so empty buckets come back with NULL values. Rates divide by the seconds the bucket's samples cover, so the partial last bucket is not understated. The hour view uses 1-minute buckets because 30-second buckets would hold zero or two samples depending on the sampling phase, which draws false gaps and doubled rates.

Alternatives rejected:

- *Bucketing in the browser.* A week is about 20,000 rows, about 2 MB of JSON per refresh, into a chart that cannot draw more points than it has pixels.
- *A separate status endpoint.* The tab needs both parts on every refresh, and one request keeps them from the same moment.
- *Naming the endpoint after the tab (`/wal-stream-monitoring`).* The endpoint serves any consumer.

### D11. The tab's layout answers the question first

`DatabaseComponent` gains a `'monitoring'` tab when `database.type` is `POSTGRES_PHYSICAL` and the physical backup type is `FULL_INCREMENTAL_WAL_STREAM`. The tab is a new slice, `features/backups/physical/monitoring`, exported through `features/backups/physical`, which `DatabaseComponent` already imports. It reads through `physicalWalStreamStatisticsApi` in `entity/backups/physical/api`. Top to bottom:

1. **Verdict banner.** Full width, background and border from AntD tokens through `theme.useToken()`: `colorSuccessBg`/`colorSuccessBorder`, `colorWarningBg`/`colorWarningBorder`, `colorErrorBg`/`colorErrorBorder`, `colorFillTertiary` for gray. The tokens follow the dark theme by themselves. A large icon per level (`CheckCircleFilled`, `ExclamationCircleFilled`, `CloseCircleFilled`, `ClockCircleOutlined`, `PauseCircleFilled`, `QuestionCircleFilled` for "cannot confirm") and the headline in an `aria-live="polite"` region. Under the headline, the keeping-up sentence. On the right, the live indicator: a dot that pulses while the newest sample is younger than 2 minutes, and "measured 12 s ago" from `current.sampledAt`. Below `md`, the indicator moves under the headline.
2. **Reasons.** A list under the banner, broken first, each with its level icon, the "what happened" sentence and the "what to do" sentence from the table below. Notices follow in a blue `Alert` group. The `NO_SLOT_WAL_LIMIT` notice is collapsible and names `max_slot_wal_keep_size` through `InlineCodeComponent`.
3. **Path of changes.** Three cards, Source, Databasus and Storage, joined by arrows labeled with the write rate and the archive rate. Source shows "not received by Databasus yet" with an AntD `Progress` against the rebuild lag, labeled "at 256 MB Databasus starts a new chain from a full backup", plus the remaining allowance or "no limit". Databasus shows the local queue against its pause size and "receiving paused" while the receiver waits. Storage shows "last change saved 40 s ago". An amount over its limit shows the bar full in the error color with "over the limit". A card's border takes the color of the most severe reason mapped to its stage. Each amount has a `Tooltip` on a focusable element with its plain meaning and its technical name. Below `md` the cards stack and the arrows point down.
4. **Charts.** A `Segmented` control for hour, day and week, then three `recharts` charts of fixed height, about 220 px, sharing one `syncId`. Titles are questions: "Is Databasus keeping up?", "How much is not saved yet?" with the subtitle "Not received by Databasus yet is part of it", and "Databasus buffer". The Y axes format through `formatBytes`, with "/s" for rates. The X axis shows `HH:mm` for hour and day and weekday plus `HH:mm` for week, in local time. Tooltips show formatted values. Legends sit on top. Restarts and pauses are markers on the buffer chart without a second axis, and the pause size is a labeled dashed reference line. When the series has no point with values, the charts are replaced by "Charts appear after a few minutes of streaming". When data starts inside the period, a caption says "Data since HH:mm".
5. **Details.** A `Collapse`, closed by default, with the raw values under their technical names: slot status, remaining allowance (`safe_wal_size`), average and longest segment write, upload concurrency, WAL-to-stored ratio, WAL segment size, session start.

Reason and notice texts, stage and action. The exact wording is in the dictionaries; the table fixes the meaning.

| Code | Stage | What happened | What to do |
|---|---|---|---|
| `STREAMING_STOPPED` | Databasus | Databasus stopped receiving changes N min ago | Databasus retries by itself; if this lasts, check the Databasus logs |
| `SOURCE_UNREACHABLE` | Source | Databasus cannot reach the database for N min | Check that the database server is running and reachable from Databasus |
| `WAL_CHAIN_BROKEN` | Storage | Some changes were lost between backups, so recent changes cannot be restored yet | With a full backup pending: a new full backup is on its way, watch it on the Backups tab. Without: start a full backup on the Backups tab |
| `SLOT_MISSING` | Source | The database no longer keeps changes for Databasus | Databasus recreates it when streaming restarts, but changes in between are lost: start a full backup on the Backups tab unless one is pending, and check that nobody removes it |
| `SLOT_LOST` | Source | The database deleted changes Databasus had not received | Databasus starts over with a new full backup by itself; to avoid this, see the speed advice below |
| `ARCHIVE_NOT_ADVANCING` | Storage | No new changes reached storage for N min while the database keeps writing | Check that the storage is reachable and has free space |
| `ARCHIVE_SLOWER_THAN_SOURCE` | Storage | The database writes faster than Databasus saves; the backlog grew by X in 10 min | Check the network speed to the storage, or move the storage closer |
| `SLOT_KEEPS_EXTRA_WAL` | Source | The database keeps more of its change log than usual for Databasus, and it grows | Make sure the database server has free disk space |
| `SLOT_LIMIT_APPROACHING` | Source | At this pace the database will delete unsaved changes in about N min | Speed up saving, or raise `max_slot_wal_keep_size` |
| `NEAR_SLOT_REBUILD` | Source | Unreceived changes reached X of the Y at which Databasus starts over | Speed up saving, or raise the limit in the backup settings |
| `LOCAL_QUEUE_FILLING` | Databasus | Databasus's buffer is X of Y full | Databasus pauses receiving by itself when it is full; check the storage speed |
| `FREQUENT_RESTARTS` | Databasus | Receiving restarted N times in the last hour | Check the Databasus logs for the cause |
| `FORCED_ROTATION_REFUSED` | Source | The database refuses to close its change file on time, so quiet periods reach storage later | Grant the backup user permission to run `pg_switch_wal()` |
| `NO_SLOT_WAL_LIMIT` | Source | The database keeps changes for Databasus without a limit | A limit protects the server's disk if Databasus stays down long; the cost is a new full backup when it is hit |
| `SLOT_KEEPS_EXTRA_WAL_STEADILY` | Source | The database keeps more of its change log than usual for Databasus | Normal under heavy load; nothing to do while it does not grow |
| `NO_FULL_BACKUP_YET` | Storage | The first full backup has not finished, so saved changes cannot be restored yet | Wait for it, or start it on the Backups tab |

`NOT_STREAMING` says "Continuous saving of changes is off because backups are disabled or no storage is selected" with a link that switches to the Config tab. `COLLECTING_DATA` says "Streaming started; the first numbers appear within a minute".

The keeping-up sentence for falling behind reads like "The database writes changes faster than Databasus saves them: 21 MB/s against 12 MB/s. 3.4 GB is not saved yet, and that grows by about 32 GB an hour." With an estimate under a day it adds the consequence: "At this pace Databasus will start a new backup chain in about 40 minutes." Idle reads like "The database writes almost nothing; new changes reach storage within minutes."

Chart lines differ in line style as well as hue, so they survive color blindness and grayscale. Light and dark variants: source writes blue `#1677ff` / `#4096ff` solid, archiving orange `#fa8c16` / `#ffa940` dashed, not saved yet purple `#722ed1` / `#9254de` solid, not received yet cyan `#13c2c2` / `#36cfc9` dashed, local queue purple solid with the pause size as a gray dashed line, restart markers red `#f5222d` / `#ff4d4f`, pause markers gray. Axis, grid and tooltip colors come from the AntD tokens, because recharts' default grid gray nearly vanishes on the dark background. Gaps stay gaps (`connectNulls={false}`).

Alternatives rejected:

- *A row of AntD `Statistic` items as the top of the tab.* It shows numbers without saying whether they are good, which is the question the tab exists to answer.
- *AntD charts or another library.* AntD 5 ships no chart components, `recharts` is already a dependency, and `frontend/AGENTS.md` forbids other UI kits.

### D12. Live updates without flicker

The tab fetches on open, on period change, every 30 seconds in every period, and at once when the browser tab becomes visible again. It does not fetch while `document.visibilityState` is `hidden`. The healthcheck component's "poll only today" pattern is not followed, because the banner must stay current whatever period the charts show. A day refresh reads about 2,900 rows into 360 buckets on one index, which is cheap.

A refresh keeps the previous response rendered until the next one arrives, so nothing collapses to a spinner. The first load renders a skeleton at the final heights, so the layout does not jump; a failed first load shows the error through `translateApiError` and a Retry button. A later failed refresh keeps the last response, shows a small inline warning, and the next scheduled refresh retries. When the last successful refresh is older than 2 minutes, the banner switches to the gray "cannot confirm the current state" with the last known level and its age.

Charts animate on first render and on period change only; refreshes pass `isAnimationActive={false}`, otherwise every 30 seconds each line would redraw from zero. Numbers in the banner and the path cards ease to their new value over about 400 ms, and the banner's colors transition over 300 ms. Under `prefers-reduced-motion: reduce` there are no transitions, no pulse and no chart animation. The "measured N s ago" label ticks every second.

Alternative rejected: *server-sent events or a WebSocket.* Samples arrive every 30 seconds, so polling at the same rate shows the same data without a new transport.

### D13. Display rules as pure functions

The rules that decide what the tab shows live in pure functions in the slice's `lib/`, so vitest covers them without rendering:

- whether to show the Monitoring tab;
- the banner's tone and icon per level, the "cannot confirm" override, and the order of reasons;
- the "what happened" and "what to do" keys and parameters per reason and notice code, as `LocalizedText`, with the pending-full variant for `WAL_CHAIN_BROKEN` and `SLOT_MISSING`;
- the keeping-up sentence per state, with the consequence sentence when an estimate is under a day;
- the stage per reason and the color of each stage;
- the mapper from series points to chart rows, keeping NULL as a gap;
- rounding of durations and estimates for reading;
- the refresh schedule, as a small controller with the timer and the page visibility injected.

A new `shared/lib/formatBytes.ts` formats byte amounts with base 1024 and the labels B, KB, MB, GB and TB through `formatNumber`, matching how the backup lists already label megabytes. The existing local copies on the backup lists take megabytes and stay as they are, because this change does not touch those components.

## Risks / Trade-offs

- [Samples add about 2,900 inserts per streaming database per day, and one expiry delete a minute, to the internal PostgreSQL.] → Single-row inserts on two indexes, and a delete that touches about one row per streaming database per tick in steady state.
- [Thresholds are guesses until real installations report.] → They are named constants in one package, tested by value, and sit ahead of the existing notifications rather than replacing them.
- [A streamer that crash-loops with a session that takes longer than 2 minutes to fail shows "collecting data" until its first failure.] → `failing_since` turns it red one minute after that failure, and the Backups tab's healthcheck shows the source state meanwhile.
- [The first sample after a restart has no source write rate.] → The chart shows one missing point. Inventing a rate from the slot lag would mislead.
- [A timeline switch makes the source position jump backwards.] → D4 treats that tick as unknown instead of recording a negative write volume.
- [Changing the probe's return touches two other callers.] → They map the new outcome to their current behavior, and their existing tests must keep passing.
- [Moving four thresholds and one predicate touches the streamer, the supervisor and the config.] → Pure moves with no value change; the existing tests of those packages pin the behavior.

## Migration Plan

One goose migration creates the samples table, its foreign key and its two indexes, and adds the nullable `failing_since` column to `physical_wal_streamers`. Its `Down` section drops the column, the indexes, the constraint and the table, as recent migrations do (`backend/migrations/20260920215925_add_two_factor_auth.sql`). The migration is additive, so the previous binary runs fine with it applied, and a rollback of the binary alone loses nothing but the samples it stops writing. Samples start appearing on the first lag-monitor tick after deploy, so the tab shows "collecting data" for up to 30 seconds per database.
