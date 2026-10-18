## Why

An operator whose WAL streaming falls behind has nothing in Databasus to look at. A "WAL chain at risk" notification says the slot is behind by some number of bytes. Nothing shows whether the gap is growing or shrinking, how fast the source writes WAL compared with how fast Databasus archives it, or whether the source has a limit on how much WAL the slot may hold. The only evidence today is one log line every 30 seconds, and most operators running Databasus in a container never read it. A user with a busy production database asked for exactly these numbers on the day they enabled WAL streaming.

Raw numbers alone would not help most users either. "Slot lag: 3.2 GB" means nothing to someone who does not know what a replication slot is. The first thing the screen has to say is whether everything is fine, and if not, what is wrong, how urgent it is and what to do.

## What Changes

- Databasus records a sample of each WAL-streaming database's archiving health every 30 seconds and keeps the samples for 7 days. Besides the slot and archive values, a sample records upload timing and concurrency, receiver restarts and pauses, and the source's state: reachable, unreachable, or reachable without the slot. It also records when each tracked problem began. Those times come from the same tracking that sends the existing notifications.
- A new authenticated API endpoint returns WAL stream statistics for one database. It is a documented contract with units in field names and codes instead of display text:
  - one health verdict, healthy, needs attention, broken, collecting data or not streaming, with the reasons behind it and notices that never change it;
  - a broken verdict while the backup chain has a gap, until a newer full backup completes, even when the slot already looks healthy again;
  - a keeping-up summary for the last 10 minutes: write rate against archive rate, how much is not archived yet and whether that grows, and estimates of when the backlog clears or when the source or Databasus will act on it;
  - the streamer's state, which accounts for its heartbeat and for repeated failures before the first sample;
  - a chart series for the last hour, day or week.
- The database page gets a **Monitoring** tab, shown only for PostgreSQL physical databases in the full + incremental + WAL streaming mode. From top to bottom:
  - a colored verdict banner in plain words, with the keeping-up state as a sentence a junior understands, for example "the database writes changes faster than Databasus saves them, by 9 MB/s";
  - the reasons, each with what happened and what to do;
  - the path of changes from source through Databasus to storage, showing where unsaved changes are;
  - charts for the selected period;
  - a collapsed details section with the raw values.
- The tab refreshes every 30 seconds in every period without flicker, and shows how fresh its data is.
- A source without a limit on WAL kept for the slot gets a notice that names `max_slot_wal_keep_size`. It does not change the verdict.

## Capabilities

### New Capabilities

- `physical-backups/wal-stream-monitoring`: what Databasus records about a WAL-streaming database's archiving health, for how long, who can read it, how the verdict is decided, what the statistics API returns and what the Monitoring tab shows.

### Modified Capabilities

None. The existing notifications keep their behavior. Their thresholds move to a shared place so the verdict uses the same values, and the staleness tracker reads the source's real current position instead of deriving it from the slot.

## Impact

- Module rules: `AGENTS.md`, `backend/AGENTS.md` and `frontend/AGENTS.md`.
- Builds on `parallel-wal-segment-uploads`, which is applied: the unarchived WAL is measured from its archive progress point, and the archive counters come from its upload snapshot.
- Backend:
  - a migration for the samples table and a `failing_since` column on the streamer table;
  - sampling in the WAL streamer's lag monitor, which already probes the slot every 30 seconds;
  - the slot probe tells a missing slot from an unreachable source and gives up after 10 seconds;
  - the slot query also reads `safe_wal_size` and the source's current position, which keeps a lost slot from reporting position 0;
  - receiver restart and pause counters, and a flag for a refused forced WAL switch;
  - expiry of old samples on the retention cleaner's tick;
  - a new package with the verdict, the keeping-up summary and the thresholds. Three thresholds move there from the streamer and the supervisor, and the "should streaming run" test moves onto the backup config, so both sides share one value;
  - a new endpoint on the physical backup controller and regenerated Swagger docs.
- Frontend:
  - a new tab in the database page and a monitoring slice under the physical backups feature;
  - API and model types under the physical backups entity;
  - charts drawn with `recharts`, already a dependency;
  - a shared byte formatter;
  - new strings in all six interface dictionaries.
- Database load: one row per streaming database every 30 seconds, about 20,000 rows and 5 MB per database at the 7-day limit, plus one expiry delete a minute.

## Out of scope

- General database monitoring such as CPU, memory or IO of the source. It was removed in `fcc894d1` and stays removed.
- A Prometheus or OpenTelemetry metrics endpoint. The samples make one easy to add later.
- New notifications or alerts based on the verdict. The existing notifications stay as they are.
- Settings for the sampling interval, the retention period or the verdict thresholds.
- Metrics for full and incremental backups, logical backups, or physical databases outside WAL streaming mode.
- Free disk space on the Databasus host or on the source.
- A verdict summary in the database list.
- Adding the workspace name to notifications.
