This change builds on the applied `parallel-wal-segment-uploads`: it reads that change's archive progress point and upload snapshot. Backend commands run in `backend/` unless a task says otherwise, frontend commands in `frontend/`. `<pg>` stands for `./internal/features/backups/backups/usecases/physical/postgresql/...`, `<core>` for `./internal/features/backups/backups/core/physical/...` and `<backuping>` for `./internal/features/backups/backups/backuping/physical/...`.

Every frontend task that introduces a translation key adds it to all six dictionaries in the same task, written in each language rather than word for word, because the keys are typed against `en.ts` and every other dictionary is typed `typeof en`.

## 1. Storage

- [x] 1.1 Create the migration from the repository root with `goose -dir backend/migrations create add_physical_wal_stream_samples sql` and fill it per design D1 and D5: the samples table, its foreign key to `databases` with ON DELETE CASCADE, the `(database_id, sampled_at)` and `(sampled_at)` indexes, the nullable `failing_since` TIMESTAMPTZ column on `physical_wal_streamers`, and a `Down` section that drops them. From the repository root, with `.env` exported by `set -a; source .env; set +a`:
  - run `goose -dir backend/migrations up`, then `down`, then `up` against the dev database;
  - apply it to every test slot database, because `just test-package` does not migrate them: `for ((i = 0; i < ${TEST_PARALLEL_WORKERS:-8}; i++)); do GOOSE_DRIVER=postgres GOOSE_DBSTRING="$(sed -E "s#/([^/?]+)\?#/\1_w$i?#" <<<"$GOOSE_TEST_DBSTRING")" goose -dir backend/migrations up; done`, the loop from `scripts/test.sh:49-54`.

  Verify that every goose command exits with status 0.
- [x] 1.2 Add the `PhysicalWalStreamSample` model in `core/physical/models` and its repository in `core/physical/repositories`: insert, delete older than a time across all databases, newest for a database, samples since a time, session starts since a time (by `session_started_at`), and the bucketed series (D10). Add `failing_since` to the streamer model, make `MarkFailed` and `MarkStaleRunningFailed` set it with `COALESCE`, and add a method that clears it. The controller tests in section 5 pin the series query. Verify with `just lint` in `backend/` and with `just test-package <core>` passing, the existing streamer repository tests included.

## 2. Shared health rules

- [x] 2.1 Create the `core/physical/wal_health` package (D8). Move `warningSlotStatusHoldPeriod`, `DefaultArchiveStalenessThreshold` and `streamerHeartbeatStaleness` into it as exported constants, and have `<pg>` and `<backuping>` use them. Replace `isWalStreamCandidate` with `PhysicalBackupConfig.IsWalStreamingExpected()`, which also checks `IsBackupsEnabled`, and use it in the supervisor. Verify with `just test-package <pg> <backuping> ./internal/features/backups/config/physical/...` passing, the existing classifier, staleness and supervisor tests included.
- [x] 2.2 Add the streamer state rule and the verdict function (D8) as pure functions. The verdict is complex rule logic, and testing each threshold through HTTP would need a seeded database per case, so it gets a table-driven unit test in `wal_health`. Verify with `just test-package <core>` passing, covering:
  - every reason at each of its levels, with values just below and just above each threshold;
  - the two-sample rule: "Single unusual sample";
  - the spec scenarios "Everything in order", "Short network blip", "Source unreachable for a few minutes", "Source down for a while", "Archive stuck", "Streamer stopped", "Restart after a deploy", "Crash loop on a new database", "Slot removed on the source", "Just enabled", "Backups disabled", "Several problems at once", "Busy cluster with steady extra WAL", "Paused receiver above the rebuild lag", "No limit on the source" and "Slot already lost";
  - the gap input set for "Slot rebuilt" gives `WAL_CHAIN_BROKEN`, and the same input without a gap gives none;
  - `NO_FULL_BACKUP_YET` listed without a completed full backup and the verdict unchanged by it;
  - "Stale heartbeat" and "Streamer never started" for the streamer state;
  - a growth of four segments plus one in the unarchived WAL's sawtooth stays below the backlog threshold.
- [x] 2.3 Add the keeping-up summary (D9) to the same package. Verify with a table-driven test in `wal_health`, run through `just test-package <core>`, covering the spec scenarios "Archiving keeps up", "Archiving falls behind", "Catching up after a burst", "Running out of allowance", "Quiet database" and "Quiet database with a stuck backlog", the unknown state for a window without write values, the first-match order for a shrinking backlog on a quiet source, both rates sharing one denominator when the window has a hole, and no rebuild estimate while the slot's status is `extended`.

## 3. Sampling inputs in the streamer

- [x] 3.1 Add `safe_wal_size` and `pg_current_wal_lsn()` to `InspectSlot` (D3). Use the real current position in the staleness tracker. Verify with tests against the test PostgreSQL container, run through `just test-package <pg>`: the allowance is NULL with `max_slot_wal_keep_size = -1` and positive with a limit set, and the returned position is not below the position read just before the call. The existing staleness tests must still pass.
- [x] 3.2 Make the slot probe return reachable with a slot state, reachable without the slot, or unreachable, with a 10-second timeout on its context (D2). Map "without the slot" to the current behavior in the slot-LSN watcher and the resume path. Verify with tests in `slot_lsn_watcher_test.go`, run through `just test-package <pg>`:
  - "Slot removed on the source": dropping the slot on a reachable source gives the "without the slot" outcome;
  - "Silent source": a probe against a local TCP listener that accepts and never answers returns unreachable within 11 seconds;
  - the existing watcher and resume tests still pass.
- [x] 3.3 Count receiver restarts and pauses with atomic counters that the sampler takes and resets (D5). Verify with a test in `wal_stream_test.go`, run through `just test-package <pg>`: one back-pressure pause and one receiver exit of its own give one pause and one restart, and the next take gives zero of each ("Pause for a full local queue").
- [x] 3.4 Set an atomic flag when the rotation loop gets the insufficient-privilege refusal (D5). Verify with a test in `wal_rotation_test.go`, run through `just test-package <pg>`, against a test source whose backup user lacks `EXECUTE` on `pg_switch_wal()`: the flag is set after the refusal.

## 4. Sampling in the streamer

- [x] 4.1 Add the pure sample builder (D2, D3, D4). Verify with unit tests in `lag_monitor_test.go`, run through `just test-package <pg>`, covering:
  - "Slot lost on the source": a `lost` state keeps the real current position and gives no negative unarchived value;
  - "Source position moves backwards": no write value and no negative one;
  - "Source write volume after a restart" and "Source reachable again";
  - no unarchived value while the archive progress point is 0;
  - source-side columns NULL for the unreachable and the without-slot outcomes, other columns filled.
- [x] 4.2 Turn the lag monitor's tick body into a method that takes `now`, records a sample built by 4.1 after classifying and before acting (D2), carries the trackers' times, keeps `source_unreachable_since` and seeds it from the newest stored sample at session start, and clears `failing_since` on a session's first sample. Pass the repository through `WalStreamSpec`. Verify with tests in `lag_monitor_test.go` that call the tick method against the test source, run through `just test-package <pg>`, covering:
  - "Healthy streaming": four ticks give four reachable samples with the listed values;
  - "Source unreachable": ticks against a stopped source give unreachable samples sharing one unreachable-since time;
  - "Unreachable across a session restart";
  - a slot held in `extended` gets `slot_extra_wal_since` equal to the time the classifier started its hold;
  - a rebuild in a tick leaves that tick's sample with the slot values probed before the rebuild;
  - the first sample of a session clears `failing_since`.
- [x] 4.3 Wire the repository in `backuping/physical/wal_supervisor.go`. Verify with tests in `wal_supervisor_test.go`, run through `just test-package <backuping>`, covering:
  - "Database not streaming": reconcile a full + incremental database alongside a WAL streaming control database, wait until the control database has its first sample, and assert that the full + incremental database has none;
  - "Crash loop before the first sample": sessions that fail at start keep the first `failing_since`.
- [x] 4.4 Delete samples older than 7 days on each retention cleaner tick (D7). Verify with tests in `cleaner_test.go`, run through `just test-package <backuping>`, covering "Expired samples" and "Streaming switched off".

## 5. Statistics API

- [x] 5.1 Add the read method on the physical core service and `GetWalStreamStatistics` on `PhysicalBackupService` with the current values and the series, authorizing through `databaseService.GetDatabase` and refusing a database that is not PostgreSQL physical (D10). Add the `GET /backups/physical/database/:id/wal-stream/statistics` route with Swagger annotations. Seed samples through a helper in `core/physical/testing`. Verify with controller tests in `controllers/physical/controller_test.go`, run through `just test-package ./internal/features/backups/backups/controllers/physical/...`, covering:
  - "Member of the database's workspace" gets current values and series;
  - "User from another workspace", "Unauthenticated request", "Nonexistent database" and "Logical database" are refused, the nonexistent one with no statistics in the body;
  - "Unknown period" returns 400;
  - "Week view": at most 360 points, each archive rate equal to the seeded WAL bytes of its interval divided by the seconds its samples cover;
  - the hour view has 60 one-minute points and the last point contains the current moment;
  - "Streamer was down for an hour": the points covering the hole carry NULL values;
  - "Database removed": deleting the database leaves no samples;
  - `lastArchivedWalAt` equals the newest committed segment's `received_at`.
- [x] 5.2 Add `ChainViewService.GetNewestCompletedFullWithGaps` (D8). Add `health`, `streamer` and `keepingUp` to the response, with the full backup and chain gap inputs from that method, `isFullBackupPending` from the config's `ForceFullRequestedAt` and the in-flight backup, and the rebuild lag from the backup config. Add Swagger enum annotations for every code. Verify with controller tests in the same file, run through the same command, covering:
  - "Codes, not text": a needs-attention database returns a level, reasons and notices whose codes are all in the documented lists, and no string field outside the codes, times and IDs;
  - "Streamer stopped": a `FAILED` row and a newest sample ten minutes old give `BROKEN` with `STREAMING_STOPPED` and that sample's time;
  - "Slot rebuilt": seeded segments with a gap below the progress point after the newest full give `WAL_CHAIN_BROKEN`, and seeding a newer completed full clears it;
  - the same gap with an incremental of the newest full marked chain-broken still gives `WAL_CHAIN_BROKEN` and no `NO_FULL_BACKUP_YET`;
  - `isFullBackupPending` is true with `ForceFullRequestedAt` set and false without it or an in-flight full;
  - "Backups disabled" and "Switched out of WAL streaming": `NOT_STREAMING` with the seeded series still returned;
  - "Stale heartbeat": a `RUNNING` row with an old heartbeat gives the not running state.
- [x] 5.3 Regenerate the API docs with `swag init -d . -g cmd/main.go -o swagger` in `backend/`. Verify that `grep -c 'wal-stream/statistics' swagger/swagger.json` prints at least 1, and that `grep -c 'WAL_CHAIN_BROKEN' swagger/swagger.json` prints at least 1.

## 6. Frontend

- [x] 6.1 Add `shared/lib/formatBytes.ts` (D13). Verify with a vitest test next to it covering the B, KB, MB, GB and TB boundaries with base 1024, then `pnpm test` and `pnpm typecheck` passing.
- [x] 6.2 Add the response types, one per file under `entity/backups/physical/model`, the label key tables for every code enum with their keys, and `physicalWalStreamStatisticsApi` under `entity/backups/physical/api`, exported through the slice's `index.ts`. Verify with `pnpm typecheck` and `pnpm test` passing; `dictionaries.test.ts` checks the placeholders.
- [x] 6.3 Add the pure display rules from design D13, except the refresh schedule, to the monitoring slice's `lib/`, with the reason, notice, keeping-up and banner sentence keys from the D11 table. Verify with vitest tests next to them, run through `pnpm test`, covering:
  - the tab rules for a logical database, a physical database without WAL streaming and a WAL streaming database;
  - "Healthy database" and "Problems listed": the banner tone per level, broken reasons first;
  - "Every reason has an action": every reason and notice code maps to both a "what happened" and a "what to do" key;
  - "Broken chain without a pending full backup": `WAL_CHAIN_BROKEN` and `SLOT_MISSING` map to the "start a full backup" action without a pending full and to the "on its way" action with one;
  - "Plain names": no default label or sentence in the English dictionary for this slice contains "WAL", "slot", "LSN" or "lag", except "what to do" sentences that name a setting;
  - "Falling behind explained": parameters of about 9 MB/s, 3.4 GB and about 32 GB an hour;
  - "Consequence only when near": the consequence sentence present for an estimate under a day and absent for three days;
  - "Backlog in the local queue": the Databasus stage in the needs-attention color and the source stage in the healthy color;
  - an amount above its limit marked as over the limit;
  - "Archiving falls behind on the chart": rows mapped from 21 and 12 MB/s points put the archive series below the source series at every point and keep the not-saved-yet series rising;
  - NULL points staying gaps, and "No history yet" when no point has values;
  - "Refresh keeps failing": a last successful refresh older than 2 minutes gives the "cannot confirm" banner with the last known level;
  - durations rounded to minutes under an hour, hours and minutes under a day, days and hours beyond.
- [x] 6.4 Add the refresh schedule (D12) as a small controller in the slice's `lib/` with the timer and the page visibility injected. Verify with vitest tests using fake timers, run through `pnpm test`, covering: a refresh every 30 seconds in every period ("Status changes while open"); none while hidden and one at once when visible again ("Browser tab hidden"); one at once on a period change; a failed refresh keeps the previous response and the next one still runs.
- [x] 6.5 Build the monitoring slice's UI per design D11 and D12: the verdict banner with its live indicator and `aria-live` headline, the reasons and notices, the path of changes, the period selector, the three synchronized charts with formatted axes and the empty state, the collapsed details, the first-load skeleton and error with Retry, the token-based colors and the reduced-motion rules. Take every display decision from 6.3 and the schedule from 6.4. Verify with `pnpm lint`, `pnpm typecheck` and `pnpm test` passing.
- [x] 6.6 Add the `'monitoring'` tab to `DatabaseComponent` behind the predicate from 6.3, and let the not streaming banner's link switch to the Config tab. Verify with `pnpm typecheck` and `pnpm test` passing.

## 7. Wrap-up

- [x] 7.1 Run the formatters and linters for both modules. Verify with `just format` and `just lint` at the repository root finishing clean.

## Change verification

```sh
just test-package ./internal/features/backups/backups/core/physical/...
just test-package ./internal/features/backups/backups/usecases/physical/postgresql/...
just test-package ./internal/features/backups/backups/backuping/physical/...
just test-package ./internal/features/backups/config/physical/...
just test-package ./internal/features/backups/backups/controllers/physical/...
pnpm --dir frontend test
```
