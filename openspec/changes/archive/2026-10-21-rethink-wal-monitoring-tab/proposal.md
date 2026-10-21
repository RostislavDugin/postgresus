## Why

The Monitoring tab of a WAL-streaming database answers an engineer's question, "where are the bytes right now", while the people who open it ask "can I restore my database, to which moment, and what do I do if I can't". Juniors, self-hosters and freelancers make up most of these users. The answer is on the screen, but they have to assemble it from six blocks, and some of those blocks contradict each other.

A broken chain shows the problem on the tab today:

- The headline says "Something is broken" without naming what broke. Under it sits "The database writes almost nothing; new changes reach storage within minutes", which reads as good news.
- The storage card turns red while it says "Last change saved a few seconds ago".
- The notice about a missing limit, which appears on almost every installation because PostgreSQL sets no limit by default, takes a box as large as the failure itself.
- Three charts fill the hour axis even when only two minutes of data exist. Their axes auto-scale to 0-100 B, so noise looks like a signal.
- Nothing on the tab says to which moment the backups can restore, although that is the reason anyone turns on WAL streaming.

## What Changes

- The statistics API gains the latest moment the backups can restore to right now. It is the same moment a restore to the latest available point reaches, so the tab and the restore can never disagree. It is empty while no full backup has completed.
- The statistics API also gains the stretches of time the newest chain can restore to, so the tab can draw a WAL gap that an incremental backup later bridged.
- The broken chain reason is listed only for a gap the latest restore cannot cross. Today a gap that a completed incremental backup bridged keeps the verdict broken: a live database showed "Changes were lost between backups" in red next to a correct "You can restore … up to 3:17 PM".
- The tab is rebuilt from the simple answer down to the detail:
  - **Verdict banner.** The headline names the most severe problem, or says that all changes are being saved. The banner always says to which moment the database can be restored. When the fix is a full backup, the banner offers a button that starts one, to users who may start backups. Other problems appear as short lines inside the same banner, so there is no separate reasons list that repeats it.
  - **Four current figures** in cards: when the last change was saved, how much is waiting to be saved, how fast the database writes and how fast Databasus saves. The keeping-up sentence moves out of the banner and becomes the caption of these figures, so it no longer appears under an unrelated failure.
  - **Restore timeline**: one bar from the start of the newest chain to now that marks which stretches can be restored and which cannot, with a sentence for each stretch that cannot.
  - **One chart**: database writes against saved to storage, for the last hour, day or week. It appears only once there is enough data to read, its time axis starts where the data starts, and its amount axis has a minimum scale and round ticks, so a quiet database does not look busy. It keeps a fixed height, keeps equal lines distinguishable, and shades and names stretches without measurements.
  - **Technical details**: one open section under the chart with the path of changes from the database through Databasus to storage, the tips as quiet notes, the "not saved yet" and buffer charts, and the raw values. The no full backup yet notice is no longer a tip, because the banner's restore sentence says it.
- The tab stops growing at about 1150 px, so a 2K screen shows it at laptop proportions.
- The path of changes stops coloring a stage for a broken chain. A broken chain is a problem of the backup history, not of any stage. Each arrow between stages shows one direction instead of both.
- The tab reads the running streamer's current values every second through a new endpoint, so the buffer, the unsaved amount and the last save move in about real time. Values from the source database follow the streamer's existing probe, at least every 10 seconds. The verdict, the rates and the charts keep refreshing every 30 seconds.
- The path of changes in the technical details becomes a pipeline: neutral stages with a large main number, "X of the limit" next to a thin meter, a problem shown by icon and word, and connectors that carry the rates.
- **BREAKING** for the tab layout only. The standalone reasons list and the large notice box are deleted, not kept as an alternative view. The API change is additive: two new fields in the statistics response, `latestRestorableAt` and `restorableRanges`, and a new endpoint, `GET /backups/physical/database/:id/wal-stream/live`, for the current values; no field is removed or renamed.

## Capabilities

### New Capabilities

None.

### Modified Capabilities

- `physical-backups/wal-stream-monitoring`: the API response gains the latest restorable moment and the restorable stretches; the broken chain reason ignores gaps a completed incremental bridged; the requirements for the Monitoring tab's verdict, keeping-up explanation, path of changes and charts change to the layout above; the tab gains the full backup action, and the tips move into the technical details.

## Impact

- Backend:
  - a read of the restore coverage in the chain view, built from the same code that resolves a restore to the latest point: the latest restorable moment, the restorable stretches and the gaps the latest restore cannot cross;
  - the verdict receives only those gaps;
  - two new fields in the statistics response, their Swagger annotation and regenerated Swagger docs;
  - the WAL stream supervisor remembers its newest source probe and answers `GetLiveState`, and `PhysicalWalStreamSupervisor` keeps each running supervisor reachable by database;
  - the endpoint `GET /backups/physical/database/:id/wal-stream/live` with its Swagger annotation;
  - chain view, live state and controller tests for the coverage, the verdict and the current values.
- Frontend:
  - the monitoring slice under `features/backups/physical/monitoring`: the banner, the figure cards, the restore timeline, one main chart, the tips and technical details sections, and the existing path of changes moved into technical details;
  - the database page passes whether the user may manage databases to the tab, which needs it for the full backup button;
  - the API model type gains the new fields, and new models and an API method cover the current values;
  - a controller reads the current values every second while the browser tab is visible, and the tab lays them over the statistics;
  - new and changed strings in all six interface dictionaries, with the strings the deleted blocks used removed.
- Database load: unchanged in reads of WAL segments. The restore coverage reads the newest chain's segments once, and the verdict's gaps come from the same rows instead of a read of their own.

## Out of scope

- Changes to the verdict's other reasons, their thresholds, the keeping-up summary or the sampling. The broken chain reason changes only in which gaps it counts.
- The write and save rates jumping between a value and zero on a database that writes in bursts. The 10-minute window averages the samples it has; smoothing it is a separate change.
- The Backups tab's backup list being requested every few seconds while the Monitoring tab is open.
- New notifications.
- Restoring from the Monitoring tab. The restore stays on the Backups tab.
- A monitoring summary in the database list or in the sidebar card.
- The disagreement between the sidebar's "Unavailable" healthcheck badge and the streaming source's reachable state. It is a separate problem of the healthcheck.
- Charts for periods other than the last hour, day and week, or zoom on a chart.
- Starting an incremental backup from the tab.
- Other physical backup modes and logical databases, which have no Monitoring tab.
