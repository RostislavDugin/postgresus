## Why

The next scheduled backup time, added by the `2026-10-10-show-next-backup-time` change, does not earn its place in the interface. On the database card it adds one to two lines of text to every card, and the product owner decided the feature is not needed. Removing it now, before a release ships it, avoids carrying two endpoints, a scheduler query and a spec for a display nobody asked to keep.

## What Changes

- **BREAKING** The `GET /backups/database/{id}/next-backup` and `GET /backups/physical/database/{id}/next-backups` endpoints are deleted. Only the Databasus frontend calls them, and they have not shipped in a release.
- The database card no longer shows the next backup line, and the read-only logical and physical backup settings no longer show a next backup line under each interval.
- The browser-side "Next run" line under a cron interval in the read-only backup settings is deleted as well. It predates the removed change and is computed from the current moment rather than from the last backup, so after a missed backup it names a time the scheduler will not act on. Keeping it would leave a partial and less accurate version of the removed feature.
- The backend code that only served the feature is deleted: the scheduler queries `GetNextScheduledBackup` and `GetNextScheduledBackups`, `Interval.GetNextTrigger` with its `NextTrigger` type, and the config services' `FindBackupConfigByDbId`.
- The interval code goes back to its form before the removed change, minus `Interval.NextTriggerTime` and its `next*Trigger` helpers. `NextTriggerTime` had no production caller before the removed change either, so it is deleted with its tests rather than restored.
- The interface strings for the next backup line are deleted from all six dictionaries.
- The `backup-schedule-visibility` capability is removed.

What stays:

- The cron preview in the logical and physical backup settings editors. It tells the user when an expression being typed would fire, which helps write the expression, and the removed change never touched it.
- When backups actually run. The scheduler's due checks keep their current behavior.
- The archived `2026-10-10-show-next-backup-time` change, as the record of why the feature existed.

Out of scope:

- The monthly scheduling bug on days 29, 30 and 31 that the removed change recorded. It belongs to its own change.
- The cron "Next run" line in the verification schedule settings.
- Any other indicator of upcoming backups.

## Capabilities

### New Capabilities

None.

### Modified Capabilities

- `backup-schedule-visibility`: every requirement is removed, and the capability goes with them.

## Impact

- Backend: `backend/internal/features/intervals`, the logical and physical schedulers under `backend/internal/features/backups/backups/backuping`, the backup services and controllers under `backend/internal/features/backups/backups`, and the logical and physical config services. Swagger is regenerated locally; it is not tracked.
- Frontend: the backup API clients and models in `frontend/src/entity/backups`, `DatabaseCardComponent`, `ShowLogicalBackupConfigComponent`, `ShowPhysicalBackupConfigComponent`, and the six dictionaries.
- The database card makes one request fewer per database.
- Rules this change answers to: `AGENTS.md`, `backend/AGENTS.md`, `frontend/AGENTS.md`.
