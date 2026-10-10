## Why

After setting up a backup schedule, users cannot see when the next automatic backup will run. They have to work it out from the interval or the cron expression, and a timezone or schedule mistake stays hidden until a backup fails to appear. The interface shows a "Next run" line only for cron schedules. That line is computed in the browser from the current moment, so it can disagree with what the scheduler will actually do. The disagreement is largest right after a failed or missed backup, which is exactly when users check the schedule.

## What Changes

- The backend reports the next scheduled backup time for a database and computes it from the same interval rules the scheduler uses to decide whether a backup is due. When a backup is already due, the response says so instead of naming a future slot.
- For a logical database, the backend reports one next backup time.
- For a physical database, the backend reports the next full backup time and, when incremental backups are enabled, the next incremental backup time. WAL streaming is not considered.
- The database card shows "Next backup" under "Last backup".
- The read-only backup settings show a next-backup line for every interval type, both logical and physical. It replaces the cron-only "Next run" line the browser computes there today.
- The interval's next-trigger calculation is replaced and pinned by a test against the scheduler's due check. The current calculation ignores a missed daily, weekly or monthly slot and names the following one, while the scheduler fires on its next tick. For a monthly day the next month lacks, it names an overflow date the scheduler never fires on. Nothing calls it in production today.
- New interface strings are added to all six interface dictionaries.

No existing API, schema or config breaks. The new endpoints are additive. The browser-side cron calculation in the two read-only settings views is deleted, not kept beside the new line.

## Capabilities

### New Capabilities

- `backup-schedule-visibility`: what the user is told about upcoming automatic backups for logical and physical databases, including due, disabled and suppressed states, plus who may read it.

### Modified Capabilities

None. Existing scheduling behavior does not change. This change only reports it.

## Impact

- Backend: `backend/internal/features/intervals` (next-trigger calculation and its tests), the logical and physical backup schedulers (a read-only query that reuses their inputs), `backend/internal/features/backups/backups/services`, the logical and physical backup controllers (two new GET endpoints), swagger docs regenerated with `make swagger`.
- Frontend: backup API clients in `frontend/src/entity/backups`, `DatabaseCardComponent`, `ShowLogicalBackupConfigComponent`, `ShowPhysicalBackupConfigComponent`, and the six dictionaries in `frontend/src/shared/i18n/locales`.
- The database card already loads the backup config for each database. It now makes one more request per card.
- Rules this change answers to: `AGENTS.md`, `backend/AGENTS.md`, `frontend/AGENTS.md`.

## Out of scope

- Changing when backups actually run. That includes the weekly and monthly schedules that skip a missed slot from a previous week or month. It also includes a scheduler bug found while planning: a monthly schedule on day 29, 30 or 31 skips every month without that day, so day 31 runs only seven times a year. The display reports these behaviors as they are. The monthly bug deserves its own change.
- WAL streaming for physical databases.
- Predicting automatic retries after a failed backup, or manual "back up now" requests. The scheduler acts on both within one tick, so the displayed time is stale for seconds at most.
- Telling the user that an incremental slot will produce a full backup because no chain is left to extend. The physical next-incremental time reports the cadence, not the backup kind the scheduler picks at that moment.
- Explaining on the card why no next backup is shown, for example a missing storage, an identity change or a held-back incremental chain.
- Verification schedules, including their cron-only "Next run" line.
- The cron preview in the backup settings editors, which describes an unsaved expression and stays browser-side.
- A batch endpoint for all databases in a workspace.
- A server-configured display timezone. Times are shown in the browser's local timezone, like every other time in the interface.
