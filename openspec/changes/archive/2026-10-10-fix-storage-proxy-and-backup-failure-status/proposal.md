## Why

GitHub issue #816 reports two faults that together left a user's backups broken for hours without anyone being told.

The first fault is about outbound proxies. The user runs Databasus in a network where outbound traffic may only leave through a proxy, configured with the standard `HTTPS_PROXY` variable. Most of Databasus honours that variable, but S3 uploads ignore it and connect directly. The firewall drops those connections, and every backup fails with a timeout. Azure Blob and Google Drive storages build their HTTP connections the same way and fail the same way. So do Mattermost notifications when the notifier skips TLS verification.

The second fault hides the first. When a storage upload fails with a network timeout, the logical backup is recorded as cancelled instead of failed. A cancelled backup sends no failure notification and does not mark the database as failing, because cancellation is normally something the user did on purpose. This happens with any storage timeout: an unreachable S3 endpoint or a dropped network connection is enough, no proxy required. A storage error that is not a timeout, such as a refused access to the bucket, is already reported as a failure.

## What Changes

- S3, Azure Blob and Google Drive storages send their requests through the proxy named by `HTTP_PROXY` / `HTTPS_PROXY` and skip it for hosts listed in `NO_PROXY`, the same way the rest of Databasus already does. When none of these variables is set, connections stay direct, as today.
- A Mattermost notifier with TLS verification turned off honours the same variables. With verification on, it already does.
- A logical backup is recorded as cancelled only when a user asked to cancel that backup. Every other interruption is a failure: the backup gets the failed status and a fail message, the database is marked as failing, and the failure notification goes out. This covers storage timeouts, storage errors that wrap a cancellation, and the backup time limit.
- The fail message of a backup interrupted by a storage error carries the storage error itself instead of the bare text "backup cancelled".
- The four logical engines (PostgreSQL, MySQL, MariaDB, MongoDB) share one implementation of the backup's time limit and shutdown watch, and one function that decides how an interrupted backup is reported, instead of four identical copies of each.
- A backup that runs into the backup time limit is now reported as a failed backup with a failure notification. Today it is silently recorded as cancelled. This changes observable behavior; it is not an API, schema or config break.
- The fail messages for the time limit and for shutdown stop saying "cancelled", since both backups are recorded as failed. They become `backup exceeded the time limit of 23 hours` and `backup interrupted by application shutdown`. Backups already stored keep their old messages.

Out of scope:

- Physical PostgreSQL backups and WAL archiving. Their stream already reports a storage failure as a failure before considering cancellation.
- Restores. A restore does not cancel its own context when a storage read fails; it tracks storage read failures separately, so the reported fault does not reach it. Its error-text matching stays as it is.
- Rclone storages. Rclone's own HTTP layer already reads the proxy variables.
- SFTP, FTP and NAS storages. They do not speak HTTP, and HTTP proxy variables do not apply to them.
- A proxy setting in the storage form. Operators configure the proxy for the whole container through the environment, as the issue does.
- Notifications for backups the user cancelled. They stay silent.
- Backups the previous run left in progress. The scheduler keeps recording them as failed with a restart message and a notification at startup.
- Changing the backup time limit, the S3 timeouts or the shutdown handling, which keeps reporting a backup interrupted by shutdown as failed.
- Other code the four engines duplicate, such as the copy loop and the writer cleanup. This change merges only the part that decides how an interruption is reported.

## Capabilities

### New Capabilities

- `outbound-http-proxy`: storages and notifiers that talk HTTP reach their endpoints through the proxy configured in the process environment, and respect its exclusion list.
- `logical-backup-outcome`: how a logical backup that did not complete is reported, as failed or cancelled, and what each outcome triggers.

### Modified Capabilities

None.

## Impact

- Backend storages: `backend/internal/features/storages/models/s3`, `azure_blob` and `google_drive`.
- Backend notifiers: `backend/internal/features/notifiers/models/mattermost`.
- Logical backups: the four engine use cases under `backend/internal/features/backups/backups/usecases/logical`, the shared logical backup core in `backend/internal/features/backups/backups/core/logical`, and the backuper in `backend/internal/features/backups/backups/backuping/logical`.
- Tests: unit tests next to each changed HTTP client, a shared test helper with a local tunnelling proxy in `backend/internal/util/testing`, a classification test in the logical backup core, backuper tests, and one regression test per logical engine that runs a real dump against the test fixtures. The existing backuper test that simulates a cancellation by returning an error text is rewritten to cancel the backup for real.
- No API, schema, migration, config or frontend change.
- Rules this change answers to: `AGENTS.md`, `backend/AGENTS.md`.
