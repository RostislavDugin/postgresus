## Context

See proposal.md for why this change exists. This section covers only the current code the decisions rest on.

**Proxy.** Go's `http.DefaultTransport` reads the proxy variables through `http.ProxyFromEnvironment`. A transport built as `&http.Transport{...}` has no `Proxy` field and always connects directly. Four places build their own transport this way:

- S3: `backend/internal/features/storages/models/s3/model.go:855`, in `getClientParams`, used by every S3 operation and the connection test.
- Azure Blob: `backend/internal/features/storages/models/azure_blob/model.go:504`, in `buildClientOptions`.
- Google Drive: `backend/internal/features/storages/models/google_drive/model.go:618`, in `buildHTTPClient`, as the base under the OAuth transport.
- Mattermost: `backend/internal/features/notifiers/models/mattermost/model.go:299`, only when TLS verification is off.

Telegram clones `http.DefaultTransport` and replaces its proxy with the one from the notifier settings (`backend/internal/features/notifiers/models/telegram/proxies.go:43`). The other HTTP notifiers and telemetry use the default transport and already honour the variables.

**How a storage failure becomes "cancelled".** A logical backup streams the dump through a pipe into `storage_files.StartBackgroundWrite` (`backend/internal/features/storages/files/background_write.go:32`). When the upload fails, that goroutine cancels the backup's context with the storage error as its cause (line 36), so the dump process is killed. The engine use case then explains why the context ended. In PostgreSQL that is `classifyCancellation` (`backend/internal/features/backups/backups/usecases/logical/postgresql/create_backup_uc.go:578`), with identical copies in MySQL (`mysql/create_backup_uc.go:583`), MariaDB (`mariadb/create_backup_uc.go:586`) and MongoDB (`mongodb/create_backup_uc.go:431`). It returns the bare `"backup cancelled"` for any cause that matches `context.Canceled` or `context.DeadlineExceeded`, and `save to storage: <cause>` for anything else.

A network timeout matches `context.DeadlineExceeded`. Both a dial timeout (`dial tcp ...: i/o timeout`) and a response header timeout (`net/http: timeout awaiting response headers`) satisfy `errors.Is(err, context.DeadlineExceeded)` on Go 1.26.3, checked while preparing this change. So a storage timeout comes back as `"backup cancelled"`.

The use case reaches that classification on two paths:

- The dump finished before the upload failed. S3 buffers a 16 MiB part before sending anything (`defaultInnerPartSize`, `s3/model.go:45`, ADR-0004), so a small database dumps completely while S3 is still connecting. The use case then waits for the upload result, and its `saveErr` branch calls `checkCancellation` first (`postgresql/create_backup_uc.go:269`). The context has been cancelled by the upload failure, so the storage error is classified as a cancellation. This is the path in issue #816.
- Only PostgreSQL re-checks the context in that final branch. MySQL, MariaDB and MongoDB return `save to storage: ...` there, so this path misreports only PostgreSQL backups. Their regression tests therefore pass against the old classification too; they pin the outcome for every engine rather than reproduce a failure.
- The upload failed while the dump was still running. The early `earlySaveErr` check (`postgresql/create_backup_uc.go:226`) catches it only if the error is already in the channel; otherwise the `ctx.Done()` branch (line 236) classifies it the same wrong way.

The backuper then decides the status from the error text (`backend/internal/features/backups/backups/backuping/logical/backuper.go:133`). It treats any message containing `"backup cancelled"` or `"context canceled"`, or any error matching `context.Canceled`, as a cancellation unless the message also contains `"shutdown"`. A cancelled backup skips `SetBackupError` and the failure notification (lines 139-153). The time-limit message `"backup cancelled due to timeout"` contains `"backup cancelled"`, so a backup that hits the 23-hour limit is silently recorded as cancelled today.

The only thing that cancels a logical backup on purpose is the cancellation registry. `MakeBackup` creates `executionCtx` from `context.Background()` and registers its cancel function under the backup ID (`backuper.go:71-74`). The registry fires it for the user's cancel endpoint (`backend/internal/features/backups/backups/services/logical_service.go:265`) and for backups the scheduler finds left in progress after a restart (`backuping/logical/scheduler.go:316`). Shutdown and the time limit do not cancel `executionCtx`. The use cases detect them on their own context, through `createBackupContext` (`postgresql/create_backup_uc.go:403`), which is identical in PostgreSQL, MySQL and MariaDB. The MongoDB copy (`mongodb/create_backup_uc.go:271`) lacks the branch that copies the caller's cause, which `context.WithCancelCause` already propagates, so it behaves the same.

Physical backups (`backend/internal/features/backups/backups/usecases/physical/postgresql/stream.go`) already check their own sentinel causes and then `saveErr` before anything is treated as a cancellation. Restores never cancel their context on a storage error; only `executionCtx` in `backend/internal/features/restores/restoring/restorer.go:97` does.

## Goals / Non-Goals

**Goals:**

- One rule decides "cancelled": a cancellation was requested for this backup. No error text takes part in that decision.
- The storage error that interrupted a backup reaches the fail message unchanged.
- The four engines cannot drift apart again on how they report an interruption.
- Tests reproduce the reported path with a real dump, not only with mocks.

**Non-Goals:**

- Retrying a failed upload, or changing any S3, Azure or Google Drive timeout.
- A per-storage proxy setting.
- Reworking the physical backup or restore outcome logic.

## Decisions

### 1. Set `Proxy: http.ProxyFromEnvironment` on the four transports

Each of the four transports gets the `Proxy` field and keeps its own timeouts and TLS settings.

Alternatives rejected:

- **Clone `http.DefaultTransport` and override the timeouts, as Telegram does.** The clone also brings `ForceAttemptHTTP2`, connection pool limits and `ExpectContinueTimeout`. HTTP/2 would then turn on for S3, Azure and Google Drive. That is a behaviour change on S3-compatible providers that this fix does not need and that nothing here tests.
- **A shared constructor for storage transports.** The four places use different timeouts, and Google Drive wraps the transport in an OAuth layer, so the constructor would take most of its fields as parameters to save one line per place. The tests in decision 6 guard against a fifth transport forgetting the field more cheaply.

Proxied `https` requests use the standard library's `CONNECT` tunnel. TLS is negotiated with the endpoint through the tunnel, and each transport's own `TLSClientConfig` keeps applying to the endpoint certificate. The end-to-end TLS requirement in `specs/outbound-http-proxy/spec.md` needs no code of ours beyond leaving those TLS settings in place, and decision 6 adds a test that proves it for each of the four transports. The standard library never proxies localhost or loopback addresses, which the spec states as well.

### 2. The backuper decides "cancelled" from `executionCtx`, not from the error

`MakeBackup` records the backup as cancelled exactly when `executionCtx.Err() != nil` after the use case returns. Every other error is a failure. The `strings.Contains` checks and the `errors.Is(err, context.Canceled)` check in `backuper.go:133-137` are deleted.

This makes three cases correct together. A storage error that wraps a cancellation becomes a failure. A backup stopped by the time limit becomes a failure with a notification. A user cancel on MongoDB that surfaces as a `mongodump` exit error becomes a cancellation; the MongoDB final `switch` (`mongodb/create_backup_uc.go:256`) never re-checks the context, so today that case depends on the dump's error text. Shutdown keeps its current outcome, a failure, because shutdown does not cancel `executionCtx`. If a user requests cancellation and the use case then returns a storage error, the backup is cancelled, which matches what the user asked for.

In practice only the user's cancel request reaches a running backup's `executionCtx`. The scheduler's `failBackupsInProgress` (`backuping/logical/scheduler.go:303`) also requests cancellation, but it runs at startup for backups the previous process left behind, and it records them as failed with `Backup failed due to application restart` itself. This change leaves that path alone; the spec pins its status and message so the new rule is not read as covering it.

Alternatives rejected:

- **Keep the text matching and only fix the use-case messages.** The status would still depend on wording. Any storage provider that wraps `context.Canceled`, or any message that happens to contain "backup cancelled", would hide a failure again.
- **A typed `ErrBackupCancelled` returned by the use cases.** It is better than text, but it still asks four use cases to classify correctly. The backuper already holds the one fact that defines a cancellation, so it needs no help from them.

### 3. One shared implementation of the backup context and its classification

The backup context setup and the classification move out of the four engines into `backups_core_logical`, which every engine already imports. The sentinels go to a new `errors.go` there: `ErrBackupShutdown` with the message `backup interrupted by application shutdown`, and `ErrBackupTimeout` with `backup exceeded the time limit of 23 hours`, the hours taken from the same constant as the limit. Today's texts, `backup cancelled due to shutdown` and `backup cancelled due to timeout`, both say "cancelled" for backups that are recorded as failed. The second also uses the word "timeout", which a storage network timeout produces too, so a user reading the fail message could not tell the two apart. The two functions go to a new file, with working names:

- `NewBackupContext(callerCtx)` returns the backup context and its cancel function. It is the current `createBackupContext`, with the 23-hour limit and the one-second shutdown check moved along with it.
- `DescribeBackupInterruption(callerCtx, backupCtx)` returns the error the use case reports once `backupCtx` has ended. It returns `ErrBackupShutdown` or `ErrBackupTimeout` when the cause matches them, `errors.New("backup cancelled")` when `callerCtx` itself was cancelled, and `fmt.Errorf("save to storage: %w", cause)` otherwise.

The `otherwise` branch is the storage failure, because `StartBackgroundWrite` is the only other code that cancels the backup context with a cause. The check on `callerCtx` replaces today's guess from the shape of the cause. A user cancel ends `callerCtx`; a storage timeout does not. Each engine's `checkCancellation` and `classifyCancellation` become calls to this function with the caller's context, and the four private copies of the sentinels, constants and both functions are deleted.

The message `"backup cancelled"` is kept for a real cancellation. It is no longer what the backuper reads (decision 2), but it still explains the use case's error in logs.

Alternatives rejected:

- **Fix the four copies in place.** The bug lives in four identical copies, and nothing keeps them identical. Fixing it four times keeps the duplication that let this rule go wrong in all of them at once.
- **Mark the cause in `StartBackgroundWrite`** with a sentinel such as `ErrStorageWrite`, and test for it before `context.DeadlineExceeded`. It works, but it changes a helper the physical stream shares, it depends on the order of the `switch` cases, and it still classifies by the shape of an error rather than by who cancelled.
- **Put the shared code in a new package under `usecases/logical`.** The dispatcher in `usecases/logical/create_backup_uc.go` imports the engine packages, so a sibling package would be needed to avoid an import cycle. `backups_core_logical` already holds the contracts between the backuper and the engines, such as `BackupFileStore` and `BackupArtifacts`.

### 4. No change to `StartBackgroundWrite` or the use cases' early-failure check

With decision 3, a storage failure produces the right error on every path the use case can take: the early `earlySaveErr` check, the `ctx.Done()` branch and the final `saveErr` branch. The helper keeps cancelling the backup context with the storage error as the cause, which is still what kills the dump quickly.

### 5. Regression tests per engine run a real dump into a failing file store

Each engine gets a test that runs its real use case against a real database and passes a `BackupFileStore` test double. Two cases cover the two paths in the context section:

- The store reads the whole stream, so the dump completes, and then returns a timeout error.
- The store fails before reading anything, while the dump is still producing data.

Both cases assert that the returned error wraps the storage error and is not a cancellation. The timeout error is a real one: an `http.Client` with a short `ResponseHeaderTimeout` against an `httptest` server that never answers, which produces the same error type the user saw without a 30-second wait. A helper in `internal/util/testing` builds it, so the engine tests and the classification test share one source. The tests live in each engine's `create_backup_uc_test.go`, with MongoDB getting the file. PostgreSQL uses the always-on logical fixture (`databases.GetTestPostgresConfig`); MySQL, MariaDB and MongoDB start one container each through `containers.StartMysql`, `StartMariadb` and `StartMongodb`, per ADR-0013.

Alternatives rejected:

- **An end-to-end test through the API with a real S3 storage that times out.** The S3 timeouts are 30 seconds and minio retries timed-out requests, so one test would take minutes per engine. The API path also cannot swap in a file store.
- **Mock-only tests.** The defect sits in how the real use case waits on its pipes and channels, which a mock usecase skips entirely.

### 6. Unit tests for the rest

- **Proxy wiring.** For each of the four builders, a test asserts that the resulting transport's `Proxy` is `http.ProxyFromEnvironment`, compared through `reflect.ValueOf(...).Pointer()`. Go reads the proxy variables once per process and caches them, and it never proxies loopback addresses. A test that sets `HTTPS_PROXY` and sends a request through a local proxy would therefore depend on test order and on which test in the binary first touched the proxy settings. The existing `Test_BuildHTTPClient_WhenProxyEnabled_UsesConfiguredProxy` in `notifiers/models/telegram/proxies_test.go` already pins the other side: a notifier proxy wins over the environment.
- **TLS through a proxy.** A test helper in `internal/util/testing` starts a local `CONNECT` proxy that records every request it receives and an `httptest` TLS server that records the requests reaching it. For each of the four builders, a test takes the transport the builder produces and replaces only its `Proxy` with `http.ProxyURL` pointing at that local proxy, which avoids the cached environment entirely. It then asserts three things. The proxy receives exactly one `CONNECT` naming the server's host and port, and no request line, headers or credentials. A request with TLS verification on fails with a certificate error, because the test server's certificate is not trusted. For S3 and Mattermost, the same request with TLS verification turned off reaches the server with its `Authorization` header intact.
- **Classification.** A table test of `DescribeBackupInterruption` covers shutdown, time limit, caller cancelled, storage timeout, storage error wrapping `context.Canceled`, and caller cancelled while the cause is a storage error.
- **Backuper.** Tests through `MakeBackup` with use case doubles. A storage timeout error ends as failed, with the storage error in the fail message, the database marked as failing and the failure notification sent. An error that wraps `context.Canceled` without a cancel request ends as failed. `ErrBackupTimeout` ends as failed with the fail message `backup exceeded the time limit of 23 hours`, the database marked as failing and a notification. `ErrBackupShutdown` ends as failed, not cancelled, with the fail message `backup interrupted by application shutdown`. Two tests cancel through the registry: in one the use case returns once its context ends, in the other it returns a storage error after the cancel; both end as cancelled, with no failure notification and the database not marked as failing. The existing `Test_MakeBackup_WhenCancelledAfterStoringArtifact_ArtifactRemoved` fakes a cancellation by returning an error text (`backuper_test.go:284-303`). It becomes the first of these two registry tests, since that text no longer means anything to the backuper.
- **Restart.** The existing `Test_FailBackupsInProgress_WhenSchedulerStarts_CancelsBackupsAndUpdatesStatus` in `backuping/logical/scheduler_test.go` asserts the failed status and the restart fail message, which is all the restart scenario in the spec states. The notification the scheduler sends on that path is unchanged by this change and stays out of the spec.

## Risks / Trade-offs

- [An operator sets `HTTPS_PROXY` for some other tool in the container and does not list the storage host in `NO_PROXY`] → Storage traffic starts going through that proxy after the upgrade. This is the documented meaning of the variable and what the rest of Databasus already does. The pull request description states it so it reaches the release notes.
- [A proxy that only allows some hosts makes storage requests fail with a proxy error instead of a timeout] → The failure is now reported as failed with a notification, and the proxy's error is in the fail message, so the operator sees it at once.
- [Backups that hit the 23-hour limit start sending failure notifications] → Intended, and listed in proposal.md. Such a backup did not finish, and the user should know.
- [A use case path returns an unrelated error after a storage failure, for example `start pg_dump: context canceled` when the upload fails before the dump starts] → The status is still failed, because no cancellation was requested, but the message is less precise. S3 buffers a whole part before it connects, so the upload cannot fail before the dump starts producing data. Physical backups handle this case explicitly; logical backups do not need to yet.
- [The reflection-based proxy test checks a function identity rather than reading the environment] → It is the only order-independent check of the environment wiring in a shared test binary. The tunnel test covers the behaviour of each transport once a proxy is chosen.
- [Fail messages for shutdown and the time limit change wording] → Nothing parses them after decision 2. Old backups keep their stored messages.

## Migration Plan

No data migration. Backups recorded as cancelled before the upgrade keep that status. Rolling back restores the old proxy and status behaviour; nothing persisted depends on the new code.
