## Context

See proposal.md for the motivation and specs/physical-backups/wal-stream-monitoring/spec.md for the requirements. Frontend paths below are under `frontend/src/`, and `<mon>` stands for `features/backups/physical/monitoring`. Backend paths are under `backend/internal/features/backups/backups/`.

What exists today:

- The tab renders five blocks in a fixed order (`<mon>/ui/WalStreamMonitoringComponent.tsx:188-225`): the verdict banner, the reasons list, the path of changes, the charts and the details.
- The banner's headline is the level label (`<mon>/lib/verdictBanner.ts:57-61`), so a broken verdict reads "Something is broken" (`shared/i18n/locales/en.ts`, `monitoring.levels`). For every verdict about streaming, the banner also shows the keeping-up sentence (`WalStreamMonitoringComponent.tsx:190-194`). That is why "The database writes almost nothing" appears under a broken chain.
- The reasons and the notices render in a separate component below the banner (`<mon>/ui/MonitoringReasonsComponent.tsx`). The notices share one blue `Alert`, and the no slot WAL limit notice sits there as a `Collapse` (`MonitoringReasonsComponent.tsx:56-80`).
- `REASON_STAGES` maps `WAL_CHAIN_BROKEN` to the storage stage (`<mon>/lib/reasonTexts.ts:20`), and `getStageTones` colors that stage red (`<mon>/lib/pathStages.ts:33-43`). The storage card then turns red while it says that the last change was saved seconds ago.
- Each arrow between stages renders `ArrowRightOutlined` with `hidden md:block` and `ArrowDownOutlined` with `md:hidden` (`<mon>/ui/ChangesPathComponent.tsx:130-131`). At 1440 px both are visible. The likely cause is that AntD's `.anticon { display: inline-flex }` rule wins over Tailwind's `hidden` on the icon element itself.
- The charts' time axis always spans the whole period (`<mon>/ui/WalStreamChartsComponent.tsx:115`), and the amount axis scales to the data (`:120`). The empty state appears only when no point has values (`<mon>/lib/chartRows.ts:25-26`).
- The statistics response holds `health`, `streamer`, `current`, `keepingUp` and `series` (`dto/physical/wal_stream_statistics.go:42-48`). Nothing in it says to which moment the backups can restore.
- A restore to the latest available point is resolved by `ChainViewService.ResolveRestoreSet(databaseID, nil)` (`core/physical/chain_view/restore_set.go:41-102`). It takes the newest chain and its completed incrementals; `includedIncrementals` moves the starting point to the newest completed incremental's stop LSN (`restore_set.go:243-273`). `contiguousWalRun` then skips WAL below that point, stops at the first gap above it and returns the reachable time: the `ReceivedAt` of the last contiguous segment, or the last backup's completion time without WAL (`restore_set.go:275-302`). A gap below the newest incremental therefore does not limit the latest restore, and a lossy chain stays extendable, so incrementals keep completing after a gap (`chain_view/service.go:248-251`). The existing test `Test_ResolveRestoreSet_WhenLatestWithGap_ResolvesToPreGapCeilingWithoutError` pins the gap case without a later incremental (`restore_set_test.go:275`).
- The statistics service already reads the newest full backup's gaps on every request (`services/physical_wal_stream_statistics.go:55`). `FindWalGapsInChain` lists every gap between consecutive segments of the chain (`chain_view/service.go:106-146`), and `getWalChainBrokenReason` reports the chain broken for any of them below the archive progress point (`core/physical/wal_health/verdict.go:222-235`). Neither looks at incrementals, so a gap that a completed incremental bridges keeps the verdict broken while the restore already crosses it. A live database showed exactly that: a red "Changes were lost between backups" next to a correct "You can restore … up to 3:17 PM (12 min ago)".
- A full backup is requested through `POST /backups/physical/database/:id/trigger` with type `full` (`controllers/physical/controller.go:155-186`). `RequestFullBackup` authorizes through `authorizeManage` and sets `ForceFullRequestedAt` before it answers (`services/physical_service.go:165-183`, `backend/internal/features/backups/config/physical/service.go:225`), and the statistics read exactly that field as a pending full backup (`services/physical_wal_stream_statistics.go:93-96`). `WalStreamRefreshController.retry()` refreshes at once (`<mon>/lib/walStreamRefreshController.ts:74`). The frontend calls it through `physicalBackupsApi.triggerPhysicalBackup(databaseId, 'full')` (`entity/backups/physical/api/physicalBackupsApi.ts:34`), and the Backups tab does so from a dropdown without a confirmation dialog (`features/backups/physical/ui/PhysicalBackupsComponent.tsx:167-179`, `:398-408`).
- `DatabaseComponent` knows `isCanManageDBs` and passes it to the Backups tab, but not to the Monitoring tab (`features/databases/ui/DatabaseComponent.tsx:143-148`).
- Frontend tests run in vitest with the `node` environment (`frontend/vitest.config.ts:12`). UI tests render static markup (`<mon>/ui/WalStreamMonitoringSections.test.tsx`), so click handling cannot be exercised there.

Constraints:

- `frontend/AGENTS.md`: AntD, `@ant-design/icons` and Tailwind only; no literal user-facing strings; every key in all six dictionaries; counts phrased so one string fits every number; code outside components holds keys, not text; `useLocale()` for `formatNumber` and `formatRelativeTime`; API errors through `translateApiError`; FSD imports only downward; one type per file in `model/`.
- `backend/AGENTS.md`: controller tests preferred; Swagger annotations on every endpoint, regenerated docs; no injecting another feature's repository; `time.Now().UTC()`.
- `AGENTS.md`: no backward-compatibility shims; refactor the code the change touches.

## Goals / Non-Goals

**Goals:**

- The tab and the restore name the same restorable moment, by construction.
- The rules that decide what the banner, the figures and the charts show stay pure functions in `<mon>/lib`, so vitest covers them without a DOM.
- Removed blocks leave no unused component, helper or dictionary key behind.

**Non-Goals:**

- A DOM test environment for the frontend. The click flow of the full backup button is covered through its pure state function.
- Changing the verdict rules in `core/physical/wal_health`. Only the meaning of the gaps it receives changes (D1), and the input field is renamed to say so.

## Decisions

### D1. The restorable moment comes from the restore's own resolution

`ResolveRestoreSet` is split. A new private method resolves the latest reach of a database: it selects the newest chain, its completed incrementals and the contiguous WAL run after the newest of them, and returns them with the reachable LSN and time. A gap below the newest completed backup does not limit the reach, exactly as in a restore today. `ResolveRestoreSet` calls it and adds the history files and the target checks it has today; `GetRestoreCoverage` below calls it too. Both paths share `orderedRestoreCandidates`, `selectRestoreChain`, `includedIncrementals` and `contiguousWalRun`, so a later change to restore resolution changes the tab with it.

The public read is `ChainViewService.GetRestoreCoverage(databaseID) (*RestoreCoverage, error)`, which replaces the first version's `GetLatestRestorableAt`:

```go
type TimeRange struct{ From, To time.Time }

type RestoreCoverage struct {
	LatestRestorableAt *time.Time  // nil without a restorable chain
	RestorableRanges   []TimeRange // newest chain, oldest first, merged
	BlockingGaps       []LSNRange  // gaps the latest restore cannot cross
}
```

- `LatestRestorableAt` is the latest reach's time. `ErrNoChainForRestore` gives an empty coverage, not an error.
- `BlockingGaps` are `FindWalGapsInChain(chain.RootFull.ID)` filtered to `gap.End > reach.lastStopLSN`, the stop LSN of the newest completed backup of the chain. That is exactly where `contiguousWalRun` stops; a gap with `End <= lastStopLSN` is bridged. `FindWalGapsInChain` itself stays as it is, because the Backups tab and `WaitForWalGap` read every gap.
- `RestorableRanges`: every anchor, the full backup and each completed incremental chosen by the same filter as `includedIncrementals`, opens `[anchor.CompletedAt, time the contiguous WAL run from anchor.StopLSN reaches]`, and overlapping or touching ranges merge. Calling `contiguousWalRun` once per anchor costs anchors × segments, so one backward pass over the sorted segments stores the end of each segment's contiguous run, and a binary search finds the segment that holds each anchor's stop LSN. The last range ends at `LatestRestorableAt` by construction.

The statistics service calls `GetRestoreCoverage` once and passes `BlockingGaps` to the verdict, which renames `NewestChainGaps` to `RestoreBlockingGaps` in `wal_health/dto.go` and `core/physical/service/dto.go`. `getWalChainBrokenReason` keeps its archive progress point check, because a segment still being uploaded also looks like a gap. `HasCompletedFullBackup` is read from the coverage having a latest restorable moment, so `GetNewestCompletedFullWithGaps` and its `FullBackupWithGaps` type lose their last caller and go. The gaps come from the segments the latest reach already read, through the helper `FindWalGapsInChain` now shares, so the request makes no extra query for them. The response gains a top-level `latestRestorableAt *time.Time` and `restorableRanges []{from, to}` in UTC, never null, with a Swagger description. Neither is part of `health`, because neither is a verdict.

Alternatives rejected:

- *Computing the moment in `wal_health` from the samples.* Samples know the archive progress point but not the backups or the gaps. The verdict would grow a second notion of "restorable" that could drift from the restore.
- *Calling `ResolveRestoreSet(databaseID, nil)` as it is.* It also reads every history file of the database on each refresh, and it returns the whole restore set only to take one time out of it.
- *Deriving the moment in the frontend from the Backups tab's list.* The list is paged, and its WAL ranges do not say which chain a restore would pick.
- *Using `current.lastArchivedWalAt`.* It is the newest archived segment, including segments after a gap, so it overstates what can be restored exactly in the case the tab must get right.
- *Filtering bridged gaps inside `FindWalGapsInChain`.* The Backups tab marks every gap of the history, bridged or not, and the test helper `WaitForWalGap` waits for any gap.
- *Showing a bridged gap as a notice.* The user asked for it as a picture rather than as one more line of text; D10 draws it.

### D2. One pure function composes the banner

`getVerdictBanner` takes the statistics, the time of the last successful refresh and `now`, and returns the whole banner:

- the tone and icon, unchanged;
- the headline: a new healthy key for the healthy verdict, the title of the most severe reason for needs attention and broken, and today's keys for collecting data, not streaming and the unconfirmed state;
- the headline reason's two sentences, from `getReasonTexts`;
- the other reasons, broken first, each as title plus two sentences;
- the restore sentence as `LocalizedText`, chosen by the four cases of the spec, for every verdict including collecting data.

The full backup button is not part of the banner; D3 decides it in a function of its own.

The "only up to" sentence follows the `WAL_CHAIN_BROKEN` reason alone. Since D1 the reason is listed only for a gap the latest restore cannot cross, so the moment is then always below the newest archived segment. The first version also compared the moment with `current.lastArchivedWalAt`, because the reason stayed listed after an incremental bridged the gap; that comparison and its comment go.

When the age of the moment cannot be computed, the sentence uses a variant without the age instead of showing "NaN d NaN h" (D12).

While the full backup button is visible, the banner component leaves out the headline reason's "what to do" sentence, because the button is that action. A user who cannot manage the database still reads it. The broken chain's "what happened" sentence is rewritten so it no longer repeats its title. The "Measured … ago" indicator moves into the headline row, and the text column keeps `max-w-3xl` (D11).

The reasons' order uses the existing `sortReasonsBrokenFirst`, which keeps the API order within one level. The restore moment formats as a local time for today and a local date and time otherwise, through the user's clock format (`shared/time`), followed by its age from `getRoundedDuration`.

`MonitoringReasonsComponent` is deleted. Its reason rendering moves into the banner as a compact list, and its notice rendering moves into the tips section (D5).

Alternatives rejected:

- *Keeping the reasons list under the banner and only changing the headline.* The headline would repeat the first list item, which is the duplication the change removes.
- *A headline per verdict level with reworded labels.* "Something is broken" stays vague whatever the wording; only the reason knows what broke.

### D3. The full backup button reuses the existing trigger, without a dialog

`DatabaseComponent` passes `isCanManageDBs` to `WalStreamMonitoringComponent`. The tab holds one flag, `isRequestingFullBackup`. A pure function returns the action from the level, whether the state is confirmed, the reasons, `latestRestorableAt`, `health.isFullBackupPending`, `isCanManageDBs` and that flag:

- hidden when the user cannot manage the database, when the verdict is not streaming or collecting data, when the state is unconfirmed, or when no broken chain, no missing slot and no lack of a full backup calls for it;
- hidden while the statistics report a full backup requested or running, because the reason's or the restore sentence's text then says that one is on its way;
- requesting while the flag is set;
- available otherwise.

`FullBackupAction` therefore holds three values: hidden, requesting and available. The tab passes the result to the banner component next to the banner.

The request flow lives in `FullBackupRequestController` in `<mon>/lib`, a class without React state like `WalStreamRefreshController`, so vitest covers it with an injected request, refresh and reporters. The no full backup case is read from `latestRestorableAt` being null, so it needs no notice. The button calls `physicalBackupsApi.triggerPhysicalBackup(database.id, 'full')`. The backend stores the request before it answers, so on success the tab calls `WalStreamRefreshController.retry()`, and the next response reports the full backup pending. The flag clears when the refresh after a success lands, and a scheduled refresh that lands while the request is still being sent leaves it set. On failure the flag clears at once, `translateApiError` shows through `message.error`, and the action falls back to available.

Alternatives rejected:

- *A link to the Backups tab.* It adds a step and a context switch for the one action the banner exists to offer.
- *A confirmation dialog.* A full backup destroys nothing, and the Backups tab starts one without a dialog. Two different flows for the same request would surprise.
- *Hiding the button behind a "fix it" menu.* There is exactly one fix.

### D4. Four figure cards replace the stage cards in the main area

A pure function `getMonitoringFigures(statistics)` in `<mon>/lib` returns null for the collecting data and not streaming verdicts, and otherwise the three figures as values a component formats, each value null when unknown so the component renders "not measured". The figures row shows the last save as `formatRelativeTime(current.lastArchivedWalAt)`, the amount waiting to be saved from `keepingUp.unarchivedWalBytes`, and the write and save rates from `keepingUp.sourceWriteBytesPerSecond` and `keepingUp.archiveBytesPerSecond`. The amounts are taken from `keepingUp` rather than `current` so the figures and the keeping-up sentence underneath agree. Numbers ease with `useEasedTransition`, as the stage cards do today. The figures sit in four cards, `grid grid-cols-2 lg:grid-cols-4`, each with a muted `text-sm` label and a `text-xl font-semibold tabular-nums` value: last change saved, waiting to be saved, database writes and saved to storage. The first version put both rates in one slot as a sentence, "27.3 KB/s written, 27.3 KB/s saved", which read as text rather than a number and repeated the caption. The two rate cards reuse the chart's `charts.databaseWrites` and `charts.savedToStorage` labels and show a dot in the chart line's color; `figures.speed` and `figures.speedValue` go. Apart from those dots the figures use the AntD text colors only. The keeping-up sentence and its consequence sentences move from the banner to the caption under the figures. The unknown state renders no sentence.

Alternatives rejected:

- *Coloring a figure by the reasons about it.* It would add a second severity signal next to the banner, which is how the red storage card ended up contradicting its own text.
- *Keeping the three stage cards in the main area.* Their progress bars against the rebuild lag and the pause size are engineering limits a junior cannot act on.
- *AntD `Statistic`.* Its title and value styling fits a KPI dashboard and ignores the eased value. A small grid of Tailwind cards is less code.

### D5. Technical details are an open section, with the tips inside

The first version hid tips and technical details in two AntD `Collapse` panels closed by default. Readers had to click to see the numbers that explain the verdict, and "Tips: 1" said nothing about what the tip was. The technical details are now a plain `section` under the main chart, separated by a border and headed "Technical details": the path of changes, then the tips, then the two secondary charts and the raw values table. Tips list the notices except `NO_FULL_BACKUP_YET` as quiet notes, each an info icon with the notice's title and two sentences in neutral text; the "Tips: {{count}}" title goes. The charts now render in a visible container from the start, so the recharts warning about a zero-sized container also goes.

Alternatives rejected:

- *A separate "Advanced" tab.* A fourth tab for detail of the third splits one subject across two places.
- *Tooltips instead of a details section.* Raw values need to be read and copied, not hovered.

### D6. Chart rules as pure functions

`<mon>/lib/chartRows.ts` gains:

- `hasEnoughChartPoints(series)`: at least four points with values, replacing `hasAnyChartValues`;
- `getChartTimeDomain(series)`: from the first point with values, or from `series.from` when the data covers the whole period, to `series.to`;
- `getBinaryAxisTicks(dataMaximum, minimumTop)`: round ticks for the amount axis, replacing the first version's `getAmountAxisMaximum`. It picks the unit `1024^k` of the larger of the two values, then a step of 1, 2, 2.5 or 5 × 10^n in that unit for about four intervals, and ends on a multiple of the step. 1 MiB gives 0, 256 KB, 512 KB, 768 KB and 1 MB. Recharts' own ticks are decimal while `formatBytes` is binary, which is how "293 KB/s, 585.9 KB/s, 878.9 KB/s" ended up on the axis. The Y axis gets these `ticks` and `domain={[0, last tick]}`;
- `getMissingDataRanges(series)`: runs of two or more points without values after the first point with values. The chart shades each with a `ReferenceArea`, and a caption says "No measurements from … to …" for up to three of them.

The chart header holds the title and an HTML legend on the left and, for the main chart, the period `Segmented` on the right, replacing recharts' centered `Legend`. The plots have fixed heights, 260 px for the main chart and 200 px for the technical charts: the tab has a maximum width (D11), so a height that grows with the window would only make the charts taller than the tab is wide in proportion. The saved-to-storage line becomes a wide translucent band (`strokeWidth` 6, `strokeOpacity` 0.35) under the thin database writes line, so the two stay distinguishable when they are equal, which is the healthy case.

The minimum tops are 64 KiB/s for the rate chart and `current.walSegmentSizeBytes` for the two amount charts, or 16 MiB when no sample exists yet. `getDataSinceMs` keeps driving the "Data since" caption. The empty state's text changes to "Not enough data for this period yet", with the hint that a shorter period may already have a chart.

Alternatives rejected:

- *A time threshold such as 10 minutes of data.* The week view's 28-minute buckets would need more than two hours for four points, and a message promising 10 minutes would be false.
- *Keeping the full-period axis.* Two minutes of data then fill 3% of the width.
- *A logarithmic amount axis.* It reads poorly for this audience, and it distorts the gap between writes and saves, which is the chart's point.
- *recharts `ResponsiveContainer aspect`.* It has no maximum height in every version we ship, so a 2K screen would get a very tall chart. A fixed height class renders into static markup a test can check.

### D7. Stage colors and arrows in the path of changes

`REASON_STAGES` becomes `Record<WalStreamReasonCode, PathStage | null>` with `WAL_CHAIN_BROKEN` mapped to null, and `getStageTones` skips null stages. Each arrow puts the responsive classes on a wrapper `span` and leaves the icon unstyled, so AntD's icon rule cannot override them.

Alternatives rejected:

- *Mapping the broken chain to the source stage.* The source is fine; the gap is in the history Databasus holds.
- *One rotating arrow icon.* It works too, but a transform is harder to read in the markup than two wrappers.

### D8. Strings

New keys cover the healthy headline, the four restore sentences, the full backup button and its success message, the three figure labels and "not measured", the tips title, the technical details title and the new empty chart text. `monitoring.levels` keeps its texts, because the unconfirmed state names the last known level with them. The keys only the deleted blocks used are removed: the keeping-up unknown sentence and the old empty chart text. The follow-up adds the restore timeline's title, segment names and lost-stretch sentence, the restore sentence without an age, and the missing measurements caption; it rewrites the broken chain's "what happened" sentence and removes `figures.speed` and `figures.speedValue`. The width follow-up adds the restore sentence from the start of the newest restorable stretch, with and without an age, and removes the tips title. Every key lands in all six dictionaries, written in each language rather than translated word for word.

### D9. Tests

- Backend: controller tests in `controllers/physical/controller_test.go` for the restorable moment without gaps, with a gap in the newest chain, with an incremental backup completed after the gap, with a full backup and no WAL, and without a full backup. The split of `ResolveRestoreSet` stays covered by the existing `chain_view` tests.
- Backend, restore coverage: `chain_view` tests for `GetRestoreCoverage` with no full backup, contiguous WAL, a gap without an incremental, a gap bridged by a completed incremental, an incremental before the gap, a failed, chain-broken or running incremental after the gap, two gaps with only the first bridged, an incremental whose stop LSN falls into a missing segment, a segment still uploading, and a newer full backup. Every case also checks that the last range ends at the latest restorable moment. Controller tests run the verdict with a streamer and samples: a bridged gap is healthy with two ranges (written first and run against the old code, where it fails with the broken chain), a gap after the newest incremental is broken, a failed incremental after the gap is broken, and no full backup gives empty ranges. The response format check also covers `restorableRanges`.
- Width follow-up: a backend test that a progress point past the probed position gives zero unsaved, written first and failing on the old code; vitest for the restore sentence naming the newest stretch's start, the plain sentence for one stretch, the overlay moving the restore point with the live save and keeping it where a gap stopped it; static markup for the open technical details with their heading and the tips shown as notes.
- Frontend, follow-up: vitest for `getBinaryAxisTicks`, `getMissingDataRanges`, `getRestoreTimeline` and non-finite durations, and static-markup tests for the four figure cards, the timeline's lost-stretch sentence, and the "what to do" sentence hidden next to the button and shown without it.
- Frontend: vitest for the banner composition, the restore sentence cases, the full backup action states, the figures, the chart rules and the stage tones. Static-markup tests in `WalStreamMonitoringSections.test.tsx` for the healthy and broken banners, the absent button for a read-only user, the technical details body, the chart captions, and each arrow icon sitting in a wrapper that carries its breakpoint class while the icon carries none. Static markup cannot apply CSS breakpoints, so the test pins the markup that makes them work.

### D10. The restore timeline

`<mon>/lib/restoreTimeline.ts` exports `getRestoreTimeline(ranges, now)`. It returns the segments of one bar from the first range's start to `now`: restorable for each range, lost for each stretch between two ranges, and not saved yet for the tail from the last range's end to `now`. Each segment carries its start and end and its position in percent, and a lost segment keeps a minimum width, so a gap of a few hours stays visible on a chain months long. `RestoreTimelineComponent` draws the bar from plain divs between the figures and the main chart, with green for restorable, hatched red for lost and gray for not saved yet, the two edge dates under the bar, an AntD `Tooltip` on each segment, and one sentence per lost stretch, up to three. It renders nothing without ranges.

Alternatives rejected:

- *A recharts chart.* One bar with three kinds of segment needs no axes, and divs keep the meaning in text a static-markup test can read.
- *Every chain of the database.* The ranges describe the newest chain, which is what a restore to the latest point uses; older chains stay on the Backups tab.

### D11. Wide screens

The first version kept the page width, like the Config and Backups tabs, and let the plots grow with the window. At 2560 px everything read as too large, so the tab's card now stops at `max-w-6xl` (1152 px) and stays left-aligned under its tab button. Below that width nothing changes; above it the tab keeps its laptop proportions. The banner's text column keeps its own maximum width, the figures sit in cards, the values use `text-xl`, and the plots keep fixed heights (D4, D6).

The path of changes decides its layout by its own width with a Tailwind container query (`@container` on a wrapper, `@4xl:` on the grid and the arrow wrappers) rather than by the window: the sidebar and the tab's maximum width decide how much room it gets, so at 1240 px the stages stack with downward arrows instead of squeezing three stages and two connectors into about 800 px.

Alternatives rejected:

- *Centering the capped card.* The tab buttons sit on the left, and a centered card would detach from them.
- *Viewport breakpoints for the pipeline.* The section is 400 px narrower than the window, so a viewport rule either squeezes the stages at 1240 px or stacks them at 1440 px.

### D12. Invalid durations

`getSecondsBetween` in `<mon>/lib/roundedDuration.ts` returns null when either time cannot be read, and its callers leave the age out: the restore sentence switches to a variant without it, and the measured label falls back to "not measured yet". The duration helpers stay as they are, because every number they receive comes from a time difference or from the API's estimates. "NaN d NaN h ago" appeared on a live screen while a frontend that already expected `latestRestorableAt` ran against a backend started before the field existed: `dayjs(undefined)` printed the current time and `new Date(undefined)` gave NaN. One image ships both halves, so production cannot meet that mismatch; the guard only keeps an unreadable time from reaching the screen.

### D13. Current values every second, from the running streamer

The tab asked for numbers that move in about real time, while samples are taken every 30 seconds. Taking them every second would multiply the stored rows and the source connections by 30 and change the meaning of the rate formulas and the thresholds that count samples. Instead the stored sample stays as it is, and a new endpoint reads the running streamer:

- `WalStreamSupervisor` in `usecases/physical/postgresql` remembers its newest slot probe with its time, from the 10-second liveness watcher and the 30-second sampler alike, and exposes `GetLiveState(now)`. That reads the local queue from the watch directory, the pause size and the receiver flag from their atomics, and the newest committed segment from the catalog, then derives the archive progress point and the unsaved amount exactly as the sample does. It never consumes the upload snapshot or the restart counters, which reset on read.
- `PhysicalWalStreamSupervisor` keeps each running supervisor on its `runningStreamer` and answers `GetLiveState(databaseID)`; the deployment runs in one process, so the HTTP handler and the streamer share memory.
- `GET /backups/physical/database/:id/wal-stream/live` authorizes like the statistics and answers `isStreamerRunning` with the values, or without them.
- The frontend's `WalStreamLiveController`, a class like `WalStreamRefreshController`, reads it every second while the browser tab is visible. A failed reading keeps the statistics' values. The tab overlays the newer live values onto `current` and the unsaved amount before the figures, the restore timeline's end and the path of changes read them; the verdict, the rates and the charts keep the 30-second statistics.
- While the statistics' `latestRestorableAt` equals their `current.lastArchivedWalAt`, the restore reached the newest save and no gap lies ahead, so the overlay moves `latestRestorableAt` and the end of the newest restorable range to the live `lastArchivedWalAt`. Without that, the banner named a moment up to 30 seconds older than the "last change saved" figure next to it. A restore stopped short of the newest save stays where the statistics put it; a new gap appearing between two refreshes shows within one statistics refresh.
- The source probe can be up to 10 seconds old while segments keep landing, so the archive progress point may pass the probed position. The live unsaved amount is then zero, not unknown: everything the source had written by the probe is saved. Reporting it as unknown made the figure flicker to "not measured" after every saved segment.

Alternatives rejected:

- *Sampling every second.* 30 times the rows and the source connections, and the rate formulas assume one interval for all rows.
- *A one-second source probe on a held connection.* Fresher source values, but a second long-lived connection per database; the user chose the 10-second probe.
- *Server-sent events.* The backend has no streaming endpoint, and a one-second poll is as fresh for one reader.

### D14. The path of changes as a flow strip

The first version drew three cards with thick borders in the healthy color, a progress bar that looked broken at 0 B, and small arrows with the rate floating between them. The second drew three neutral cards with a large "0 B" over an empty meter in two of them and the rates above loose arrows; readers asked why everything was 0 B while the speed was 54 KB/s. Both values were right: the rate is the 10-minute average of the keeping-up window, the amount is the current queue, and a database that writes in bursts empties its queue within seconds. The strip now says so and stops drawing an empty queue as a number:

- One frame holds a five-column grid, stage, link, stage, link, stage, whose cells span its two rows through `grid-rows-subgrid`: the icons, titles and lines share the first row, the details and link names the second. A container query (`@2xl:`) on the frame's content decides between the row and the stacked layout.
- Each stage has a round icon, neutral or in the tone color with a problem, its name, the level icon and word with a problem, and a line running from the title to the next link, so the strip reads as one pipe. Each link carries its rate as a pill on that line, ending in an arrowhead, and its chart name and color under it.
- `PathAmountComponent` shows "nothing waiting" with a success icon for an empty queue, and a number with "X of Y" over a thin meter otherwise. The rebuild and pause limit sentences appear only for a queue above zero; the allowance sentence only when the source sets a limit, because the no slot WAL limit tip already says when it does not.
- The storage stage shows the last save and the unsaved amount under it.
- A caption under the strip says that rates are averages over the last 10 minutes and amounts are right now, and the two rate cards above say "avg. 10 min" next to their labels.

## Risks / Trade-offs

- [The latest reach reads the newest chain's WAL segments on every statistics request, every 30 seconds per open tab.] → The gap check used to make that read on its own; it now reuses the reach's segments, so the request makes the same number of segment reads as before this change.
- [Every open tab reads the current values every second.] → The reading touches memory, the watch directory and one indexed catalog query, opens no source connection, and stops while the browser tab is hidden.
- [The restorable ranges walk every WAL segment of the newest chain on each request.] → One backward pass and a binary search per anchor keep it at O((segments + anchors) · log segments), over rows the latest reach reads anyway.
- [The restorable moment is the time Databasus received a segment, a little after the last change inside it.] → It is the moment the restore accepts as a target, so the screen never promises more than the restore delivers.
- [The week view shows no chart for about the first two hours of streaming.] → The empty state says that a shorter period may already have one.
- [Removing the reasons list changes the layout existing users know.] → The tab was added by the `wal-stream-monitoring-tab` change, archived on 2026-10-19, so few users know it yet.

## Migration Plan

No migration. The API field is additive, and the backend and frontend ship in one image. Rolling back the image restores the previous tab and drops the field.
