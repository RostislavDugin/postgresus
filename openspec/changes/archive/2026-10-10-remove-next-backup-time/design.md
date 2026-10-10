## Context

The feature came in as one commit, `4bb7ffa8`. Besides the new surface, that commit reworked code the feature touched:

- In `backend/internal/features/intervals/model.go` it replaced `NextTriggerTime` with `GetNextTrigger` (`model.go:97` at `HEAD`) and rewrote the `shouldTrigger*` helpers around shared slot helpers. The rewrite kept the due checks' behavior; its tests pinned `GetNextTrigger` against `ShouldTriggerBackup`.
- In the physical scheduler it extracted `getChainlessIncrementalHoldBackReason` from `decideReAnchoringFull`, so the new query and the scheduler share one hold-back check.
- In the read-only settings views it deleted the browser-computed cron "Next run" line and put the server-reported next backup line in its place.

Before that commit, `Interval.NextTriggerTime` existed with no production caller (`model.go:96` after a revert), and the read-only settings computed a cron "Next run" from the current moment (`ShowPhysicalBackupConfigComponent.tsx:101`, `ShowLogicalBackupConfigComponent.tsx:163` after a revert).

Later commits edited some of the feature's files: `2b0e5c64` reformatted `NextScheduledBackupComponent.tsx` and reordered imports in `logicalBackupsApi.ts` and `physicalBackupsApi.ts`, and `c05a682a` changed the `walRange` keys in all six dictionaries. Only the first overlaps the reverted hunks, and that file is deleted here; the other edits sit in unrelated lines, so the revert applies to them cleanly.

## Goals / Non-Goals

**Goals:**

- No next backup time anywhere in the read-only interface, the API or the backend.
- Scheduling behavior identical to the code that ran before the feature.

**Non-Goals:**

- Keeping the interval or scheduler refactors from `4bb7ffa8` for their own sake.
- Changing the cron preview in the settings editors.

## Decisions

### Revert the commit, then delete what the revert brings back

`git revert 4bb7ffa8` removes the feature together with its refactors and returns the scheduler and interval code to what ran in production before. Two pieces the revert restores are then deleted: `Interval.NextTriggerTime` with `nextDailyTrigger`, `nextWeeklyTrigger`, `nextMonthlyTrigger`, `nextCronTrigger` and their tests, and the cron "Next run" line in both read-only settings views.

Rejected: deleting the feature surface by hand and keeping the refactors. The result would be smaller, but it would be new code that never ran in a release, reviewed without the feature that justified it. `getChainlessIncrementalHoldBackReason` would be left with one caller, and the interval rewrite's tests would lose their subject, since they compare `GetNextTrigger` to the due check. A revert is checkable line by line against a known commit.

Rejected: a plain revert with nothing deleted afterwards. It restores a dead function with known wrong answers for missed slots and short months, and a "Next run" line that can disagree with the scheduler. Both contradict the decision to remove the feature.

### The editor cron preview stays

`EditLogicalBackupConfigComponent.tsx:420` and `PhysicalIntervalEditor.tsx:165` show when an expression being typed would fire. They describe an unsaved value, so there is no last backup to compute from, and their purpose is writing a correct expression. The `backups.config.schedule.nextRun` key stays for them.

Rejected: deleting the editor preview too. It is input help, not a schedule display, and the removed feature left it alone for the same reason.

### Module rules

`backend/AGENTS.md` and `frontend/AGENTS.md` apply as usual. The frontend dictionaries are typed `typeof en`, so the keys leave all six in one step, and `pnpm build` catches a leftover. The root `AGENTS.md` rule against unrequested backward compatibility means the endpoints are deleted, not kept as stubs.

## Risks / Trade-offs

- [A script outside the frontend calls the removed endpoints] -> The endpoints were added on 2026-10-10 and have not shipped in a release. Any caller gets a 404.
- [The revert conflicts with later edits] -> The only conflict is `NextScheduledBackupComponent.tsx`, which is deleted anyway. The later edits to the two API clients and the dictionaries do not overlap the reverted lines.
- [The reverted interval code differs from what the backend tests expect] -> The interval and scheduler test packages run against the reverted code before the change is done.

## Migration Plan

No data or config migration. Rolling back means reverting this change's commit, which restores the feature.
