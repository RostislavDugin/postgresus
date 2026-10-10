## 1. Planning review

- [x] 1.1 Pass the planning review gate. The review hooks run it automatically when they are active. When they are disabled, run the reviewer subagent on proposal.md, the backup-schedule-visibility spec, design.md and this task list. Resolve every CHANGES REQUIRED finding before writing code

## 2. Next-trigger calculation

- [x] 2.1 Add the invariant test from design decision 2 to `backend/internal/features/intervals/model_test.go`, written against `GetNextTrigger` and `NextTrigger`. It covers hourly, daily, weekly, monthly and cron schedules, a missing last backup, a missed daily slot, a weekly slot missed in an earlier week, monthly days 29, 30 and 31 across February and 30-day months, and a cron catch-up. Add separate cases that expect nil for invalid intervals: unknown type, missing or malformed time of day, missing weekday or day of month, missing or unparsable cron expression. Verify that the test fails to compile or fails before 2.2, with `go test ./internal/features/intervals/... -count=1`
- [x] 2.2 Replace `Interval.NextTriggerTime` with `GetNextTrigger` and add the `NextTrigger` type. `IsDue` comes from `ShouldTriggerBackup`. When it is due, `ScheduledAt` is `now`, otherwise the per-type slot. Rewrite the monthly slot to skip months without the configured day. Return nil for every interval the slot functions cannot evaluate. Delete `NextTriggerTime` and rewrite the `TestInterval_NextTriggerTime_*` cases for the new function. Verify with `go test ./internal/features/intervals/... -count=1`

## 3. Logical backend

- [x] 3.1 Add `FindBackupConfigByDbId` to the logical `BackupConfigService`. It returns nil when no config exists and never inserts one
- [x] 3.2 Add `GetNextScheduledBackupResponse` to `backend/internal/features/backups/backups/dto/logical`, with `NextBackup *intervals.NextTrigger`
- [x] 3.3 Add `BackupsScheduler.GetNextScheduledBackup(databaseID, now)`. It returns nil when no config exists, backups are disabled or storage is missing. Otherwise it applies `GetNextTrigger` to the newest backup of any status through `FindLastByDatabaseID`. Verify through the controller tests in 3.5
- [x] 3.4 Add `LogicalBackupService.GetNextScheduledBackup(ctx, user, databaseID)`. It authorizes through `DatabaseService.GetDatabase` and then rejects a physical database as a bad request. Add `GET /backups/database/:id/next-backup` with swagger annotations to the logical backups controller
- [x] 3.5 Add controller tests: daily schedule with a future slot, missed slot reported as due, no backups reported as due, disabled backups return a null next backup, missing storage returns a null next backup, a database without saved backup settings returns a null next backup and gains no config row, a physical database is rejected with 400, a user outside the workspace is rejected, and an unauthenticated request is rejected. Verify with `go test ./internal/features/backups/backups/controllers/logical/... -count=1`

## 4. Physical backend

- [x] 4.1 Add `FindBackupConfigByDbId` to the physical `BackupConfigService`. It returns nil when no config exists and never inserts one
- [x] 4.2 Add `GetNextScheduledBackupsResponse` to `backend/internal/features/backups/backups/dto/physical`, with `NextFullBackup` and `NextIncrementalBackup` as `*intervals.NextTrigger`
- [x] 4.3 Add `PhysicalBackupsScheduler.GetNextScheduledBackups(databaseID, now)` as design decision 3 describes. It reports nothing when no config exists, backups are disabled, storage is missing or an identity mismatch is unresolved. The full time comes from the newest full of any status. The incremental time is reported only when incremental backups are enabled and the chain is not held back, and comes from `newestCreatedAt`. If the held-back condition is extracted from `decideReAnchoringFull` into a shared predicate, verify that the tick is unchanged with `go test ./internal/features/backups/backups/backuping/physical/... -count=1`
- [x] 4.4 Add `PhysicalBackupService.GetNextScheduledBackups(ctx, user, databaseID)`. It authorizes through `DatabaseService.GetDatabase` and then rejects a non-physical database as a bad request. Add `GET /backups/physical/database/:id/next-backups` with swagger annotations to the physical backups controller
- [x] 4.5 Add controller tests: full and incremental both reported, full-only configuration reports no incremental, full plus incremental plus WAL stream reports both, disabled backups report neither, missing storage reports neither, an unresolved identity mismatch reports neither, a chain broken by WAL summarization being off reports full only, no completed full among the newest attempts reports full only, a logical database is rejected with 400, a user outside the workspace is rejected, and an unauthenticated request is rejected. Verify with `go test ./internal/features/backups/backups/controllers/physical/... -count=1`
- [x] 4.6 Regenerate swagger with `make swagger` and confirm both new routes appear in the generated docs

## 5. Frontend

- [x] 5.1 Add `ScheduledBackupTime` under `frontend/src/entity/backups/shared/model`, `NextScheduledBackupResponse` under `logical/model` and `NextScheduledBackupsResponse` under `physical/model`, one type per file and exported through each slice's `index.ts`. Add `logicalBackupsApi.getNextScheduledBackup` and `physicalBackupsApi.getNextScheduledBackups`
- [x] 5.2 Add the new strings to `frontend/src/shared/i18n/locales/en.ts` and translate them in `ru.ts`, `es.ts`, `pt.ts`, `zh.ts` and `fr.ts`: card next backup, next full and next incremental lines, the settings line, and a due-now variant of each. Verify that `pnpm build` passes type checking of every dictionary
- [x] 5.3 Add a pure helper `isScheduledBackupDue(scheduledBackupTime, now)` next to `ScheduledBackupTime`, true when `isDue` is set or `scheduledAt` is not after `now`, with a vitest test covering both flags and the boundary. Verify with `pnpm test`
- [x] 5.4 Render the next backup under "Last backup" in `DatabaseCardComponent`. Load it in its own effect keyed on the `database` object, so it refreshes with every list reload. Render one line for logical databases and two for physical ones. Show the due-now text when `isScheduledBackupDue` holds, and no line when the response has none. Verify with `pnpm lint` and `pnpm build`
- [x] 5.5 Replace the browser-side cron "Next run" block in `ShowLogicalBackupConfigComponent` with the API-backed next backup line for every interval type, using `isScheduledBackupDue`, and delete its `cron-parser` usage. Verify with `pnpm lint` and `pnpm build`
- [x] 5.6 Replace `renderCronNextRun` in `ShowPhysicalBackupConfigComponent` with API-backed lines under the full and incremental schedules, using `isScheduledBackupDue`, and delete its `cron-parser` usage. Verify with `pnpm lint` and `pnpm build`

## 6. Verification and review

- [x] 6.1 Run `make lint` in `backend/`, and `pnpm lint`, `pnpm test` and `pnpm build` in `frontend/`. Fix every failure
- [x] 6.2 From `backend/`, rerun the packages this change touches and fix every failure: `go test ./internal/features/intervals/... ./internal/features/backups/config/... ./internal/features/backups/backups/backuping/... ./internal/features/backups/backups/controllers/... -count=1`
- [x] 6.3 Pass the final review gate. The review hooks run it automatically when they are active. When they are disabled, run the reviewer subagent on the working-tree diff. Resolve every CHANGES REQUIRED finding and list refactor suggestions for the diff
