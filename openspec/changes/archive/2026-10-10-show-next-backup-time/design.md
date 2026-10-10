## Context

See proposal.md for the motivation. The facts below shape the approach.

- Both schedulers decide whether a backup is due with `Interval.ShouldTriggerBackup(now, lastBackupTime)` in `backend/internal/features/intervals/model.go:72`.
- The logical scheduler calls it on a 15-second tick (`backuping/logical/scheduler.go:19`, `:280`) and passes the newest backup of any status. `FindLastByDatabaseID` in `core/logical/repository.go:122` orders by `created_at` with no status filter. A due backup still does not start when the config has no storage (`StartBackup`, `backuping/logical/scheduler.go:114`).
- The physical scheduler ticks every second (`backuping/physical/constants.go:16`). It skips a config with no storage (`evaluateConfig`, `backuping/physical/scheduler.go:138`). It schedules nothing while a system identifier mismatch is unresolved (`:193`, `hasUnresolvedSystemIdentifierMismatch` at `:523`). It checks the full cadence against the newest full backup of any status (`:178`, `:222`). It checks the incremental cadence only when `isIncrementalEnabled` holds (`:931`), against the newer of that full and the newest incremental (`:205`, `:242`, `newestCreatedAt` at `:847`). When the incremental cadence is due and no chain can be extended, `decideReAnchoringFull` (`:874`) schedules nothing in two cases. One is a chain broken because WAL summarization was off (`:880`). The other is no completed full among the newest attempts (`:894`). In both cases the cadence stays due indefinitely.
- `Interval.NextTriggerTime` already exists (`intervals/model.go:96`), but nothing in production calls it, only `model_test.go:798` onwards. It returns nil when no backup exists. For daily, weekly and monthly schedules it ignores the last backup and returns the next slot after `now` (`nextDailyTrigger` at `:306`, `nextWeeklyTrigger` at `:324`, `nextMonthlyTrigger` at `:358`). After a missed slot it therefore names a future slot while the scheduler fires on its next tick. A nil `TimeOfDay` on a daily interval panics in `nextDailyTrigger` (`:307`), and a malformed one makes it return `now` (`:309`) while the due check returns false (`:164`).
- `ShouldTriggerBackup` catches up a missed slot for daily, hourly and cron schedules. Weekly and monthly schedules compare only against the current week's or month's target (`:187`, `:234`), so a slot missed in an earlier week or month is never caught up.
- A monthly schedule on day 29, 30 or 31 skips every month that lacks that day. `shouldTriggerMonthly` builds the target from the current month with `time.Date` (`:239`). For day 31 in February that target normalizes to March 3, which lies outside February and is never reached while the month is still February. In March the target becomes March 31. `nextMonthlyTrigger` builds the following month with `AddDate(0, 1, 0)` (`:380`) and therefore names the overflow date, such as March 3, that the scheduler never fires on. The reviewer confirmed this by running the current code: day 31, last backup 2026-01-31 03:00, now 2026-01-31 10:00 gives 2026-03-03 03:00, while the due check first fires at 2026-03-31 03:00. Fixing the scheduler is out of scope (see proposal). The display reports the skipped months as they are.
- `BackupConfigService.GetBackupConfigByDbId` inserts a default config when none exists, in both `backups/config/logical/service.go:128` and `backups/config/physical/service.go:173`. Their repositories' `FindByDatabaseID` return nil without writing.
- All schedule times are UTC. The frontend converts them to the browser's timezone with `frontend/src/shared/time/utils.ts`.
- The browser computes "Next run" for cron schedules from the current moment in `ShowLogicalBackupConfigComponent.tsx:163` and in `renderCronNextRun` inside `ShowPhysicalBackupConfigComponent.tsx:103`. The two editors carry their own copies. `DatabaseCardComponent.tsx:28` fetches the backup config for every card in an effect keyed on `[database.id, database.type]`. `DatabasesComponent.tsx:74` reloads the database list every 5 minutes and passes fresh `database` objects to the cards.
- Import direction: `backups/backups/core/logical` imports `backups/config/logical` (`core/logical/interfaces.go`), so the config package cannot import backup history. `backups/backups/services` already imports both backup config packages and both `backuping` packages, and the controllers import `services`.

## Goals / Non-Goals

**Goals:**

- One source of truth for "when is the next backup": the interval rules and the scheduler's own inputs.
- A test that fails if the next-trigger calculation and the due check ever disagree.

**Non-Goals:**

- Changing when any backup runs. `ShouldTriggerBackup` and both schedulers' decisions stay as they are. Extracting an existing condition into a shared helper is allowed only when the tick's behavior does not change.
- Showing next times in lists other than the database card and the read-only settings.

## Decisions

### 1. Compute the time in the backend, not in the browser

The backend computes the next backup time and the frontend only formats it.

Rejected alternative: compute it in the browser from the interval and the card's `lastBackupTime`. The browser lacks the scheduler's input. `lastBackupTime` advances only on success (`backuping/logical/backuper.go:227`), while the scheduler counts failed and in-progress backups too. The browser also cannot see a missing storage, an identity mismatch or a held-back incremental chain. It would also add a fourth copy of the schedule rules in TypeScript.

### 2. Replace `NextTriggerTime` with `GetNextTrigger`, which returns the due state explicitly

`NextTriggerTime` is replaced by `GetNextTrigger(now, lastBackupTime) *NextTrigger`. `NextTrigger` is a small type in the `intervals` package with `ScheduledAt time.Time` (`json:"scheduledAt"`) and `IsDue bool` (`json:"isDue"`). The rename follows the house rule that getters take `Get`.

- `IsDue` is `ShouldTriggerBackup(now, lastBackupTime)`, never inferred from the returned time. When it is true, including the no-backup case, `ScheduledAt` is `now`.
- When nothing is due, `ScheduledAt` is the first slot after `now`. Hourly uses `lastBackupTime + 1h`, and cron uses the schedule's next time after `lastBackupTime`. Daily and weekly reuse `nextDailyTrigger` and `nextWeeklyTrigger`. When nothing is due, the last backup is at or after the latest slot, so the slot after `now` is the right answer for these two.
- The monthly slot is rewritten. It steps month by month from `now`'s month and takes the first month whose `time.Date(year, month, day)` stays inside that month and lies after `now`. Day 31 after January 31 therefore gives March 31, not March 3.
- It returns nil for any interval it cannot evaluate: an unknown type, a missing or malformed `TimeOfDay` on a daily, weekly or monthly interval, a missing weekday or day of month, or a missing or unparsable cron expression. It never panics and never reports a due state the due check would deny.

The contract is the invariant "`ShouldTriggerBackup(t, last)` is false for every minute `t` in `[now, ScheduledAt)` and true at `ScheduledAt`, and `IsDue` equals `ShouldTriggerBackup(now, last)`". A table-driven test in `intervals/model_test.go` walks minute by minute across a window of several months. It covers every interval type, a missing last backup, a missed daily slot, the weekly previous-week case, monthly days 29, 30 and 31 across February and 30-day months, and cron. The invariant is only checked for intervals that pass `Validate`. Invalid intervals get separate cases that expect nil. The existing `TestInterval_NextTriggerTime_*` cases are rewritten to the new function and contract.

Rejected alternatives:

- Keep returning `*time.Time` and derive `isDue` from "returned `now`". A malformed `TimeOfDay` already returns `now` while nothing is due (Context), so the derivation reports false due states.
- Search forward minute by minute with `ShouldTriggerBackup` in production. It is correct by construction, but costs up to tens of thousands of iterations for a monthly schedule on every request.
- Replace both functions with one slot generator that the scheduler also uses. It is the cleaner end state, but it changes scheduler code that decides real backups.

### 3. Put the read-only queries on the schedulers

Each scheduler gets a read-only method that loads exactly the inputs its tick uses and applies `GetNextTrigger`. No method writes.

- Config lookups go through a new `FindBackupConfigByDbId` on each `BackupConfigService`. It wraps the repository's `FindByDatabaseID` and returns nil when no config exists, unlike `GetBackupConfigByDbId`, which inserts a default.
- `BackupsScheduler.GetNextScheduledBackup(databaseID, now) (*intervals.NextTrigger, error)` returns nil when no config exists, backups are disabled or the config has no storage. Otherwise it applies `GetNextTrigger` to the newest backup of any status, loaded through the `FindLastByDatabaseID` call the tick uses.
- `PhysicalBackupsScheduler.GetNextScheduledBackups(databaseID, now) (full, incremental *intervals.NextTrigger, err error)` returns two nils when no config exists, backups are disabled, the config has no storage or an identity mismatch is unresolved. The full time uses the newest full of any status. The incremental time is nil unless `isIncrementalEnabled` holds. It is also nil while the incremental chain is held back: no extendable chain exists, and either `isChainBrokenBySummarizerOff` holds or `hasNoRecentCompletedFull` holds for the newest attempts. Otherwise it comes from `newestCreatedAt`. These conditions do not depend on whether the cadence is due right now, so a held-back chain reports no incremental time in either case.
- The physical method reuses the repository calls and predicates the decision path uses: `FindLastFullAnyStatusByDatabase`, `FindLastByDatabase`, `FindLastExtendableChainByDatabase`, `FindNewestAnyStatusByDatabase` with `recentFullAttemptsWindow`, `hasUnresolvedSystemIdentifierMismatch`, `isChainBrokenBySummarizerOff`, `hasNoRecentCompletedFull` and `newestCreatedAt`. The held-back check may be extracted from `decideReAnchoringFull` into one predicate that both use, but only if the tick's behavior stays the same. Any change to which backup counts as "last", or to when a chain is held back, then lands in one place.

Rejected alternatives:

- Compute it in `backups/backups/services` with its own queries. That works, but it repeats the scheduler's choice of inputs in a second package, which is how the two would drift.
- Add the field to the backup config response. The config package cannot import backup history without an import cycle (see Context).
- Add a field to the `Database` model returned by the database list. It would make the `databases` feature depend on backups, and that model is persisted with GORM.

### 4. Two additive GET endpoints, authorized like the backup lists

- `GET /api/v1/backups/database/:id/next-backup` returns `GetNextScheduledBackupResponse` from `backups/backups/dto/logical`: `{ "nextBackup": { "scheduledAt": "<RFC 3339 UTC>", "isDue": false } }`. `nextBackup` is null when no automatic backup will run.
- `GET /api/v1/backups/physical/database/:id/next-backups` returns `GetNextScheduledBackupsResponse` from `backups/backups/dto/physical`: `{ "nextFullBackup": {...} | null, "nextIncrementalBackup": {...} | null }`.
- Both object fields are `*intervals.NextTrigger`. One type serves both responses and adds no duplicate DTO.
- The service methods are `LogicalBackupService.GetNextScheduledBackup(ctx, user, databaseID)` and `PhysicalBackupService.GetNextScheduledBackups(ctx, user, databaseID)`.

Authorization goes through `DatabaseService.GetDatabase(ctx, user, id)` (`databases/service.go:340`), the same check the physical backup list uses (`services/physical_service.go:80`). It rejects users without access to the database's workspace before any schedule data is read. The controllers sit behind the existing auth middleware, and each endpoint gets swagger annotations.

A request for the wrong database kind is rejected with 400 Bad Request. That covers the logical endpoint for a `POSTGRES_PHYSICAL` database and the physical endpoint for any other type. The kind check runs after the access check, so it reveals nothing to a user without access.

`isDue` comes from the server, so a client clock behind the server cannot turn "due now" into "in a few seconds". The browser still treats a `scheduledAt` that has passed as due when it renders, because the page may stay open past it.

Rejected alternatives:

- One endpoint for both database kinds. The response shapes differ, and the frontend already splits every backup request by kind (`DatabaseCardComponent.tsx:31`).
- Return null for the wrong kind. That hides a caller bug as "no schedule".
- A workspace-wide batch endpoint. It would avoid one request per card, but the card already loads its config per card, and batching is a separate optimization (see proposal, Out of scope).

### 5. Frontend: refresh with the list, one line per schedule

- Types: `ScheduledBackupTime` (`{ scheduledAt: Date; isDue: boolean }`) in its own file under `frontend/src/entity/backups/shared/model`. `NextScheduledBackupResponse` in its own file under `frontend/src/entity/backups/logical/model`. `NextScheduledBackupsResponse` in its own file under `frontend/src/entity/backups/physical/model`. Each is exported through its slice's `index.ts`.
- API methods: `logicalBackupsApi.getNextScheduledBackup(databaseId)` and `physicalBackupsApi.getNextScheduledBackups(databaseId)`.
- `DatabaseCardComponent` loads the next backup in its own effect, keyed on the `database` object. Every 5-minute list reload passes a new object, so the next backup refreshes with the list and with "Last backup". The config effect stays keyed on the id and type, so the config is not refetched. The card renders "Next backup in 3 hours" under "Last backup", or two lines for a physical database: next full and next incremental.
- Due state: a line reads as due now when `isDue` is set or `scheduledAt` is not after the current time. One pure helper, `isScheduledBackupDue`, holds this rule for all three components and is covered by a vitest test. The check runs at render, so a passed time switches to due now at the next refresh, within the same 5 minutes that already apply to "Last backup". No per-card timer is added.
- `ShowLogicalBackupConfigComponent` and `ShowPhysicalBackupConfigComponent` load the next backup when they mount and render a line for every interval type. The two components sit in separate feature slices, which FSD forbids from importing each other, so the line itself is `NextScheduledBackupComponent` in `frontend/src/entity/backups/shared/ui`. The line uses the format of the old "Next run" line: date and time in `dateTimeFormat`, then the relative time. The browser-side cron calculation and its `cron-parser` import are deleted from these two files. The editors keep theirs because they preview an unsaved expression.
- New strings go to all six dictionaries in `frontend/src/shared/i18n/locales`, with English as the source of truth. They cover the card lines for next backup, next full and next incremental, the settings line, and a due-now variant of each. Each is a whole sentence with interpolated values. The English text uses plain hyphens.

Rejected alternatives:

- Reuse the existing `backups.config.schedule.nextRun` key. Its wording says "run", and the editors still use it for the cron preview, so its meaning would split in two.
- Key the card request on `database.lastBackupTime`. That value advances only on success, so a failed backup would leave "due now" on screen until a page reload.
- A per-card timer that re-renders when `scheduledAt` passes. It adds one interval per card to fix staleness that "Last backup" already accepts.

### Module rules this design is constrained by

- `backend/AGENTS.md`: controller tests over unit tests, so the endpoints are covered by `controller_test.go` cases, including denial for a user outside the workspace and for an unauthenticated request. Source-split test files. Swagger annotations on new routes. Getter names start with `Get`.
- `frontend/AGENTS.md`: one type per file in `model/`, no literal user-facing strings, dictionaries typed from `en.ts` so a missing key fails `pnpm build`, whole-sentence keys, plain hyphens in `en.ts`.
- `AGENTS.md`: English in code and identifiers, no issue numbers in production code, no compatibility shims. `NextTriggerTime` and the old browser-side cron line are deleted, not kept alongside their replacements.

## Risks / Trade-offs

- [The displayed incremental time is when the incremental cadence comes due, but the scheduler may take a full backup at that moment when no chain can be extended and the chain is not held back (`decideReAnchoringFull`)] → Accepted and listed as out of scope. A physical backup still happens at the displayed time. Only its kind differs.
- [A failed logical backup with retries enabled is retried on the next tick, while the display names the next scheduled slot] → The gap lasts one tick, 15 seconds at most. After that the retry creates a new backup, and the next refresh recomputes the time.
- [A database with no storage, an identity mismatch or a held-back incremental chain shows no next time, which a user might read as "not scheduled"] → This is accurate: nothing will run until the cause is fixed. Explaining the cause on the card is left out of this change.
- [A monthly schedule on day 29-31 now visibly skips short months] → The display tells the truth about an existing scheduler bug. The fix belongs in a separate change (see proposal).
- [One extra request per database card, repeated every 5 minutes] → Each one is a handful of indexed lookups. Batching can follow if a workspace with many databases shows it.

## Migration Plan

No schema or data migration. The endpoints are additive and the frontend ships in the same release. Rolling back the release removes both together.
