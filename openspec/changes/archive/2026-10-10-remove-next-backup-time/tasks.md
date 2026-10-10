## 1. Revert

- [x] 1.1 Apply `git revert --no-commit 4bb7ffa8`, delete the conflicting `NextScheduledBackupComponent.tsx`, keep `openspec/specs/backup-schedule-visibility/` and the archived `2026-10-10-show-next-backup-time` change at `HEAD`, and verify that `git grep -n "NextScheduled\|GetNextTrigger\|NextTrigger\b" -- backend/internal frontend/src` and `git grep -n "nextBackup\|nextFullBackup\|nextIncrementalBackup\|isScheduledBackupDue" -- frontend/src` find nothing

## 2. Remove what the revert restores

- [x] 2.1 Delete `Interval.NextTriggerTime`, the `next*Trigger` helpers and their tests from `backend/internal/features/intervals`, and verify with `go build ./...` and `just test-package ./internal/features/intervals/...` in `backend/`
- [x] 2.2 Delete the cron "Next run" line from `ShowLogicalBackupConfigComponent.tsx` and `ShowPhysicalBackupConfigComponent.tsx` with the imports it alone used, keep the editor previews, and verify with `pnpm typecheck` in `frontend/` and with `git grep -n "CronExpressionParser\|schedule.nextRun" -- frontend/src/features/backups`, which must list only `EditLogicalBackupConfigComponent.tsx` and `PhysicalIntervalEditor.tsx`

## 3. Verification

- [x] 3.1 Run the touched packages with `just test-package ./internal/features/intervals/... ./internal/features/backups/...` in `backend/`, which covers the intervals, scheduler, service, controller and config packages
- [x] 3.2 Run `just lint` at the root, and `pnpm test` and `pnpm build` in `frontend/`
- [x] 3.3 Regenerate the untracked swagger with `swag init -d . -g cmd/main.go -o swagger` in `backend/` and verify `grep -n "next-backup" swagger/swagger.yaml` finds nothing
- [x] 3.4 Run `openspec validate remove-next-backup-time --strict`
