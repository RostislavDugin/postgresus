## 1. Proxy from the environment

- [x] 1.1 Add `Proxy: http.ProxyFromEnvironment` to the S3 transport in `getClientParams` and add a test in `storages/models/s3/model_test.go` asserting that the returned transport's `Proxy` is `http.ProxyFromEnvironment` (compared through `reflect.ValueOf(...).Pointer()`) and that `TLSClientConfig.InsecureSkipVerify` still follows `SkipTLSVerify`; verify the test fails before the change and passes after it with `just test-package ./internal/features/storages/models/s3/...` in `backend/`
- [x] 1.2 Do the same for Azure Blob in `buildClientOptions`, reaching the transport through the `*http.Client` in the returned options, and verify with `just test-package ./internal/features/storages/models/azure_blob/...`
- [x] 1.3 Do the same for Google Drive in `buildHTTPClient`, reaching the transport through the `oauth2.Transport` base, and verify with `just test-package ./internal/features/storages/models/google_drive/...`
- [x] 1.4 Do the same for the Mattermost transport built when TLS verification is off in `newHTTPClient`, assert that `InsecureSkipVerify` and `MinVersion` are unchanged, and verify with `just test-package ./internal/features/notifiers/models/mattermost/...`
- [x] 1.5 Add a test helper to `backend/internal/util/testing` that starts a local `CONNECT` proxy recording every request it receives and an `httptest` TLS server recording the requests that reach it, and verify it with its own test that a plain `http.Transport{Proxy: http.ProxyURL(...)}` with `InsecureSkipVerify` reaches the server through one `CONNECT`, using `just test-package ./internal/util/testing/...`
- [x] 1.6 For each of the four builders from tasks 1.1-1.4, add a test that replaces only the built transport's `Proxy` with `http.ProxyURL` to the helper's proxy and asserts: the proxy receives exactly one `CONNECT` naming the server's host and port and no request line, headers or credentials; with TLS verification on the request fails with a certificate error; for S3 and Mattermost with TLS verification off the request reaches the server with its `Authorization` header; verify with the package commands from tasks 1.1-1.4
- [x] 1.7 Verify with `git grep -n "&http.Transport{" -- backend/internal` that every remaining transport literal either sets `Proxy` or is outside the scope listed in proposal.md, and that the existing Telegram test `Test_BuildHTTPClient_WhenProxyEnabled_UsesConfiguredProxy` still passes with `just test-package ./internal/features/notifiers/models/telegram/...`

## 2. Shared backup context and interruption classification

- [x] 2.1 Add `errors.go` to `backups_core_logical` with `ErrBackupShutdown` (`backup interrupted by application shutdown`) and `ErrBackupTimeout` (`backup exceeded the time limit of 23 hours`, the hours taken from the time limit constant), and verify the package builds with `go build ./internal/features/backups/backups/core/logical/...` in `backend/`
- [x] 2.2 Move the backup time limit, the shutdown check interval and the context setup from the engines into `backups_core_logical` as `NewBackupContext(callerCtx)`, matching today's PostgreSQL `createBackupContext`, and verify with `go build ./internal/features/backups/backups/core/logical/...`
- [x] 2.3 Add `DescribeBackupInterruption(callerCtx, backupCtx)` with the order from design.md decision 3: shutdown cause, time limit cause, caller cancelled, otherwise `save to storage: %w` with the cause; verify with `go build ./internal/features/backups/backups/core/logical/...` and with the table test in task 2.5
- [x] 2.4 Add a helper to `backend/internal/util/testing` that returns a real network timeout error, produced by an `http.Client` with a short `ResponseHeaderTimeout` against an `httptest` server that never answers, and verify with its own test that the error matches `context.DeadlineExceeded`, using `just test-package ./internal/util/testing/...`
- [x] 2.5 Add a table test for `DescribeBackupInterruption` covering shutdown, time limit, caller cancelled, a storage timeout from the helper in task 2.4, a storage error wrapping `context.Canceled`, and a caller cancelled while the cause is a storage error; assert the storage cases wrap the storage error and are not `backup cancelled`, and verify with `just test-package ./internal/features/backups/backups/core/logical/...`

## 3. Engines use the shared code

- [x] 3.1 In the PostgreSQL use case, replace `createBackupContext`, `checkCancellation`, `classifyCancellation`, the two sentinels and the moved constants with the shared code, passing the caller's context to `DescribeBackupInterruption`; delete the private copies and verify with `go build ./...` in `backend/`
- [x] 3.2 Do the same in the MySQL use case and verify with `go build ./...`
- [x] 3.3 Do the same in the MariaDB use case and verify with `go build ./...`
- [x] 3.4 Do the same in the MongoDB use case and verify with `go build ./...`
- [x] 3.5 Verify with `git grep -n "classifyCancellation\|createBackupContext\|errBackupShutdown\|errBackupTimeout" -- backend/internal/features/backups/backups/usecases/logical` that no private copy is left

## 4. Backuper decides cancellation from its own context

- [x] 4.1 Write backuper tests through `MakeBackup` with use case doubles in `backuping/logical/backuper_test.go`, none of which cancels through the registry: a storage timeout error ends as failed, with the storage error in the fail message, the database marked as failing and the failure notification sent; an error wrapping `context.Canceled` ends as failed with a failure notification; `ErrBackupTimeout` ends as failed with the fail message `backup exceeded the time limit of 23 hours`, the database marked as failing and a failure notification; `ErrBackupShutdown` ends as failed, not cancelled, with the fail message `backup interrupted by application shutdown`; name the storage-timeout test after issue #816; verify the storage-timeout, `context.Canceled` and time-limit tests fail against the current backuper
- [x] 4.2 Rewrite `Test_MakeBackup_WhenCancelledAfterStoringArtifact_ArtifactRemoved` so its use case stores the artifact, signals the test, and blocks until its context ends, while the test cancels the backup through the cancellation registry; assert the backup is cancelled, no failure notification is sent, the database is not marked as failing and the artifact is removed
- [x] 4.3 Add a registry-cancel variant whose use case, after its context ends, returns a storage timeout error instead of a cancellation error; assert the backup is cancelled, no failure notification is sent and the database is not marked as failing
- [x] 4.4 In `backuper.go`, record the backup as cancelled exactly when `executionCtx.Err() != nil` after the use case returns, delete the `strings.Contains` and `errors.Is(err, context.Canceled)` checks with the `isShutdown` flag, and verify all backuper and scheduler tests pass with `just test-package ./internal/features/backups/backups/backuping/logical/...`, including `Test_FailBackupsInProgress_WhenSchedulerStarts_CancelsBackupsAndUpdatesStatus`, which covers the restart scenario's status and fail message

## 5. Engine regression tests with a real dump

- [x] 5.1 Add a `BackupFileStore` test double to the logical backup test helpers with two modes: read the whole stream and then fail with a real timeout error, or fail before reading anything; take the timeout error from the helper in task 2.4
- [x] 5.2 Add a PostgreSQL test to `usecases/logical/postgresql/create_backup_uc_test.go` that runs the real use case against the always-on logical fixture (`databases.GetTestPostgresConfig`) with both modes of the double, and asserts each error wraps the storage error and is not `backup cancelled`; verify the "after the dump finished" case fails when task 3.1 is reverted locally, then passes with `just test-package ./internal/features/backups/backups/usecases/logical/postgresql/...`
- [x] 5.3 Add the same test for MySQL in `usecases/logical/mysql/create_backup_uc_test.go`, starting one container with `containers.StartMysql`, and verify with `just test-package ./internal/features/backups/backups/usecases/logical/mysql/...`
- [x] 5.4 Add the same test for MariaDB with `containers.StartMariadb` and verify with `just test-package ./internal/features/backups/backups/usecases/logical/mariadb/...`
- [x] 5.5 Add the same test for MongoDB in a new `usecases/logical/mongodb/create_backup_uc_test.go` with `containers.StartMongodb` and verify with `just test-package ./internal/features/backups/backups/usecases/logical/mongodb/...`

## 6. Verification

- [x] 6.1 Run the logical backup packages and the end-to-end hang regressions with `just test-package ./internal/features/backups/... ./internal/features/tests/logical/...` in `backend/`, and confirm the issue #582 hang tests still see backups to a missing bucket end as failed
- [x] 6.2 Run the storage and notifier packages with `just test-package ./internal/features/storages/... ./internal/features/notifiers/...` in `backend/`
- [x] 6.3 Run `just lint` and `just format` in `backend/`
- [x] 6.4 Run `openspec validate fix-storage-proxy-and-backup-failure-status --strict`
