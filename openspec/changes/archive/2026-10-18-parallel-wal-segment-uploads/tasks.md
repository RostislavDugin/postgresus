All commands run in `backend/` unless stated otherwise. Short names used below:

- `<containers>` stands for `./internal/util/testing/containers/...`
- `<storages>` stands for `./internal/features/storages/...`
- `<wal>` stands for `./internal/features/backups/backups/usecases/physical/postgresql/...`

The storage tasks (1 to 5) and the WAL tasks (6 to 10) touch different packages and can be done in either order. Within each group the order below is the dependency order.

## 1. Network delay rig for storage tests

- [x] 1.1 Give each storage container helper a variant that returns the container handle: `StartMinioContainer`, `StartAzuriteContainer`, `StartSambaContainer`, `StartFtpContainer` and `StartSftpContainer` in `internal/util/testing/containers/`, each returning `ContainerHandle` through `startContainer`. Make the existing `Start*` functions call the new variant and return its `Endpoint`, so no current caller changes. Verify with `just test-package <storages>`: `Test_Storage_BasicOperations` and the model tests still pass.
- [x] 1.2 Add `internal/util/testing/containers/netem.go` with three helpers, each taking `t` and a `ContainerHandle` (design D10):
  - `AddEgressDelay(t, handle, delay)` runs `tc qdisc add dev eth0 root netem delay <delay>`;
  - `SetEgressLoss(t, handle)` runs `tc qdisc replace dev eth0 root netem loss 100%`;
  - `ClearEgressShaping(t, handle)` runs `tc qdisc replace dev eth0 root pfifo`, so a cleanup may call it again after the test already cleared the shaping.

  Each helper starts a one-shot sidecar through `testcontainers.Run` with `WithHostConfigModifier` setting `NetworkMode` to `container:<id>` and `CapAdd` to `NET_ADMIN`, with no exposed ports and `wait.ForExit()`. It fails the test when the exit code is not zero and includes the sidecar's log in the message. After adding the delay, `AddEgressDelay` dials the container's own address and fails the test when the TCP handshake took less than the delay. The mapped host port does not work for this check, because Docker's userland proxy completes that handshake itself. Pin the sidecar image by tag in a constant next to the helpers and extend the `pull-testcontainers` regex in `backend/Justfile` so the image is pulled with the others. Verify with `netem_test.go`, run through `just test-package <containers>`:
  - `Test_AddEgressDelay_WhenAppliedToMinio_TcpHandshakeTakesAtLeastTheDelay`: the handshake is under 50 ms before the call and at least 50 ms after it;
  - `Test_SetEgressLoss_WhenApplied_DialTimesOut`: a dial with a two-second timeout fails after the call;
  - `Test_ClearEgressShaping_AfterLoss_DialSucceedsAgain`: the dial succeeds after shaping is cleared.
  - `Test_ClearEgressShaping_WhenCalledTwice_Succeeds`: a second clear succeeds and the dial still works.

## 2. Baseline throughput test

- [x] 2.1 Extract the per-type storage setup that `Test_Storage_BasicOperations` builds inline in `internal/features/storages/model_test.go` (the MinIO bucket, the Azurite container, the SFTP, FTP and Samba credentials, the rclone remote pointing at MinIO, the local temp directory) into one test helper. It takes a storage type and a container handle and returns a storage row saved through `storageRepository.Save` with encrypted credentials, removed through `RemoveTestStorage` in `t.Cleanup`. Make `Test_Storage_BasicOperations` use the helper instead of its inline setup. Verify with `just test-package <storages>`: `Test_Storage_BasicOperations` still passes for every type it covered.
- [x] 2.2 Add `Test_WriteFile_WhenStorageAnswers50msLate_SustainsTwoSegmentsPerSecond` to `model_test.go`, with one parallel subtest per type, labelled `S3`, `AzureBlob`, `SFTP`, `FTP`, `NAS`, `Rclone` and `Local`. Each subtest:
  - starts its container (the rclone subtest starts MinIO) and calls `AddEgressDelay` with 50 ms, except `Local`;
  - registers the storage with the helper from 2.1;
  - generates one random 16 MiB buffer and one 1 KiB buffer;
  - feeds 24 jobs to four goroutines, and each job writes the 16 MiB file and then the 1 KiB file under unique names through `GetStorageFileStore().WriteFile`;
  - starts its clock at the first write and fails when 12 seconds pass before all 24 pairs are written, stating how many pairs were written and the elapsed time;
  - requests deletion of every written file in one transaction through `RequestFileDeletions` and calls `DrainStorageFileDeletions` in `t.Cleanup`.

  Verify by running `just test-package <storages>` once before any fix from section 4. Record under this task, as an indented note, each subtest's result with pairs written and elapsed time. `Local` is expected to pass and `SFTP` to fail far below the floor; the other results are recorded as observed and are the baseline for section 4.

  Baseline recorded on 2026-10-18 on the Fedora development host:
  - `Local`: passed, 24 pairs in 1.1 s.
  - `S3`: passed, 24 pairs in 5.3 s.
  - `AzureBlob`: passed, 24 pairs in 6.3 s.
  - `Rclone`: passed, 24 pairs in 6.7 s.
  - `FTP`: failed, 4 pairs within 12 s.
  - `SFTP`: failed, 0 pairs within 12 s.
  - `NAS`: failed after 4 pairs. Concurrent writes race on creating the target directory, and the losers fail with "mkdir test-files: file already exists".

## 3. Session pool for SFTP, FTP and SMB

These tasks touch only the new package, so their verification runs only that package. The throughput subtests from 2.2 stay red until section 4.

- [x] 3.1 Create the package `internal/features/storages/connections` (package `storage_connections`) with:
  - `Key`, holding the storage ID and a fingerprint;
  - `Fingerprint(parts ...any) string`, a hex SHA-256 over the JSON encoding of its arguments;
  - `SessionPool[T]`, built with a close function and an idle cap (4 in production), whose `Checkout(ctx, key, dial)` returns a lease that hands out the session and has `Release(err error)`. The dial function comes with each checkout, because only the caller holds the decrypted settings the key was fingerprinted from.

  An idle session for the key is reused, otherwise a new one is dialed. `Release(nil)` returns the session to the idle list unless the idle cap is reached, in which case the session is closed. `Release` with an error closes the session. Verify with `session_pool_test.go` against a fake session type, run through `just test-package ./internal/features/storages/connections/...`:
  - `Fingerprint` gives the same value for equal arguments and a different one when any argument changes;
  - sequential checkouts reuse one session;
  - two concurrent checkouts get two different sessions;
  - a release with an error closes the session and the next checkout dials;
  - a fifth idle session is closed on release.
- [x] 3.2 Extend `SessionPool` with a probe function, a probe threshold (30 seconds), a probe timeout (5 seconds) and an idle timeout (5 minutes), all passed at construction so tests can use short values (design D7, D9):
  - a session idle longer than the probe threshold is probed under the probe timeout before reuse, and a failed probe closes it and moves on to the next idle session or a dial;
  - `context.AfterFunc` on the checkout's context closes the session if the context ends before `Release`, and `Release` stops it;
  - a checkout with a new fingerprint for a known storage ID closes every idle session of the old one, and `Release` of a lease under a stale fingerprint closes the session instead of keeping it;
  - an idle session is closed after the idle timeout through a timer, with no background loop.

  Verify with `session_pool_test.go`, run through `just test-package ./internal/features/storages/connections/...`:
  - a session idle past the probe threshold is probed, and when the probe fails the checkout dials a new one;
  - cancelling the context of a lease whose holder is blocked closes its session, and the blocked call returns;
  - a new fingerprint closes the old fingerprint's idle sessions, and a lease of the old fingerprint released afterwards is closed rather than kept;
  - an idle session is closed after the idle timeout, which the test waits for as a condition with a limit.

## 4. Reused connections per storage type

Each task here makes its own throughput subtest pass. Until the last task of this section is done, subtests of types whose task is still open may keep failing, and each verification below says which subtests must pass.

- [x] 4.1 S3: replace the per-call `http.Transport` built in `getClientParams` with package-level transports, one per `SkipTLSVerify` value, created once with the existing timeouts and `MaxIdleConnsPerHost` set to 16. Keep building `minio.Client` and `minio.Core` per call from freshly decrypted credentials over those transports. When `S3Region` is empty, look the region up once with `GetBucketLocation` and keep it in a package-level map keyed by endpoint and bucket, so later clients get it without a request (design D7). Verify with `just test-package <storages>`: the `S3` and `Local` throughput subtests, `Test_Storage_BasicOperations` and the S3 model tests (`internal/features/storages/models/s3`) pass. Throughput subtests of types whose task is still open may fail.
- [x] 4.2 Azure Blob: replace the per-call transport built in `buildClientOptions` with one package-level transport with the existing timeouts and `MaxIdleConnsPerHost` set to 16, and keep building the client per call. Verify with `just test-package <storages>`: the `AzureBlob` throughput subtest, `Test_Storage_BasicOperations` and the Azure model tests pass. Throughput subtests of types whose task is still open may fail.
- [x] 4.3 rclone: keep a package-level map from storage ID to the built remote and a fingerprint of the remote's configuration text (design D7). A write with a matching fingerprint reuses the remote. A write with a different fingerprint builds a new remote and retires the old one. A remote unused for the idle timeout is retired and dropped through a timer that starts when its last use ends. A retired remote is shut down once no call uses it, so a long full backup through it is never cut off. `TestConnection` builds its own remote and shuts it down afterwards, because it often runs for settings that are not saved yet. The idle timeout is five minutes in production and injectable for tests. `fs.NewFs` and `rcloneConfigMu` then run only on a miss. Route every remote-building method through the map. Verify with tests in `internal/features/storages/models/rclone/model_test.go` against the existing local-type remote, with a short idle timeout:
  - two writes with the same configuration build the remote once;
  - a write with changed configuration text builds a new remote and retires the old one;
  - a remote that is not used is retired and dropped after the idle timeout, which the test waits for as a condition with a limit.

  They must pass with `just test-package <storages>`, together with the `Rclone` throughput subtest, `Test_Storage_BasicOperations` and the existing rclone model tests. Throughput subtests of types whose task is still open may fail.
- [x] 4.4 SFTP: add a fingerprint method over the host, port, user, password, private key and path. Keep a package-level `SessionPool` whose session is the SSH client plus the SFTP client. The dial creates the target directory once, the probe calls `Getwd`, and the close function closes both clients. Make `SaveFile` check out a session under its context and write with `ReadFromWithConcurrency(reader, 0)` instead of `io.Copy` (design D8). Make each read attempt of `GetFile` hold its own lease until that attempt's reader is closed, and make the other methods check out and release around their calls. Make the dial honor its context through the SSH handshake, and make the directory creation succeed when a concurrent session created the same directory first. Verify with `just test-package <storages>`: the `SFTP` throughput subtest, `Test_Storage_BasicOperations` and the SFTP model tests pass.
- [x] 4.5 FTP: add a fingerprint method over the host, port, user, password, TLS mode and path. Keep a package-level `SessionPool` whose session is a logged-in `ServerConn`. The dial creates the target directory once, the probe calls `NoOp`, and the close function closes every connection the session opened. A custom dial function tracks the control and data connections and dials both with the existing 30-second timeout, so closing a session also unblocks a data transfer stuck on a silent server. Make the directory creation succeed when a concurrent session created the same directory first. Route every method through the pool, with `GetFile` holding its lease until its reader is closed. Verify with `just test-package <storages>`: the `FTP` throughput subtest and `Test_Storage_BasicOperations` pass.
- [x] 4.6 NAS: add a fingerprint method over the host, port, share, user, password, domain, TLS flag and path. Keep a package-level `SessionPool` whose session is the TCP connection, the SMB session and the mounted share. The dial creates the target directory once, the probe stats the share root, and the close function closes the connection, which drops the tree and the session with it. Make every call use the share through `WithContext` with the caller's context, and route every method through the pool. Make the directory creation succeed when a concurrent session created the same directory first, which the baseline in 2.2 showed failing. Verify with `just test-package <storages>`: the `NAS` throughput subtest, `Test_Storage_BasicOperations` and the NAS model tests pass.
- [x] 4.7 Local: check that the `Local` throughput subtest still passes with every other change in place. If it does not, find the cause before changing the floor. Verify with `just test-package <storages>`: every throughput subtest and every other test in the package pass.

## 5. Credential changes and silent storages

- [x] 5.1 Add two tests to `model_test.go` for the spec requirement "Storage connections opened with old settings are never used again", each with subtests for `S3`, `AzureBlob`, `SFTP`, `FTP`, `NAS` and `Rclone`:
  - `Test_WriteFile_WhenCredentialsChangeToWrongOnes_NextWriteFailsUntilCorrected` writes one file successfully through `GetStorageFileStore().WriteFile`, then saves the storage with a wrong password or secret through `storageRepository.Save` with the value encrypted, and requires the next write to fail. It then saves the correct value back and requires the next write to succeed.
  - `Test_WriteFile_AfterStorageIsDeleted_Fails` writes one file successfully, removes the storage with `RemoveTestStorage`, and requires the next write for that storage ID to fail.

  The five-minute close of idle connections is covered by the pool test in 3.2 for SFTP, FTP and SMB, by the remote-cache test in 4.3 for rclone, and by the transports' existing 90-second idle timeout for S3 and Azure. Verify with `just test-package <storages>`.
- [x] 5.2 Add `Test_WriteFile_WhenStorageGoesSilentMidConnection_FailsByDeadlineAndRedials` to `model_test.go`, with subtests for `S3`, `AzureBlob`, `SFTP`, `FTP` and `NAS` (spec: "A storage that stops answering does not hold an upload"). Each subtest:
  - writes one file successfully, so a pooled connection exists;
  - calls `SetEgressLoss`;
  - writes again with a five-second context and requires the call to return an error within eight seconds, the deadline plus the three seconds the spec allows, waiting on a channel with that limit;
  - calls `ClearEgressShaping` and requires the next write to succeed, which shows that the dead connection was not reused.

  Verify with `just test-package <storages>`.

## 6. Step timing in the archiving rollup

- [x] 6.1 Time the claim insert in `ProcessSegment` and `RecoverSegment` and pass the duration to `uploadAndCommit`. Time the segment write, the sidecar write and the commit with the gap probe inside `uploadAndCommit`. Add `RecordConcurrentUploads(count int)`, which keeps the highest count since the last take. Replace `TakeArchivedSinceLastReport` with a method returning one snapshot struct that carries the archived count and bytes, the per-step totals and maximums and the highest concurrency, and resets all of them. Verify with a test in `wal_upload_test.go` that archives two segments through the mock storage, records a concurrency of 3, and asserts the snapshot's counts, the non-zero step maximums, the concurrency, and that a second take is empty. It must pass with `just test-package <wal>`.
- [x] 6.2 Format the snapshot into the existing INFO line in `reportArchivingProgress`: segments and size, average and longest time per step, highest concurrency, and bytes behind the source. Keep the DEBUG line for a quiet interval with only the bytes behind. Verify with a test in `lag_monitor_test.go` that captures the log for a busy interval and a quiet interval, and passes with `just test-package <wal>`.

## 7. Gap probe for out-of-order commits

- [x] 7.1 Add `WatchDir` to `WalUploadDeps`, set it in `NewWalStreamSupervisor`, and make the test uploader helper in `wal_upload_test.go` use the directory its tests write segment files into. Make `probeChainGap` follow design D3:
  - check the segment that ends at the committed segment's start: its file in the watch directory, then in `pending-upload/`, then its row in the catalog by chain key, and report only when all three miss;
  - count only full-size files in both directories, so a short file never suppresses a gap;
  - start the reported range at the larger of the newest committed predecessor's end and the end of the highest full-size same-timeline segment file below the committed segment in the two directories;
  - delete `lastNotifiedGapEnd`, its lock use in the probe and the comments that describe it.

  Verify with tests in `wal_upload_test.go`, run through `just test-package <wal>`, for these spec scenarios:
  - "Later segment finishes first";
  - "Earlier segment failed and waits for a retry";
  - "Segment that was never received";
  - "Segment before the hole is still uploading": segment 17 committed, the file of 18 in the watch directory, 20 committed, and the reported range starts at the end of 18;
  - "Hole spanning several segments";
  - "Two gaps finish out of order";
  - one case where the predecessor's file sits only in `pending-upload/`;
  - "Segment before the gap left only a short file": the file of segment 20 is shorter than a full segment, 19 is committed, 21 commits, and one gap from the end of 19 to the start of 21 is reported.

  The existing `Test_WalUpload_PostUpdateGapProbe_EmitsSingleNotificationPerOutage` and `Test_WalUpload_ContiguousSegments_NoGapNotification` must still pass.

## 8. Concurrent dispatcher

- [x] 8.1 Replace the sequential `scanAndUpload` and `sweepPendingUploads` walk with a dispatcher. It lists the watch directory and `pending-upload/`, sorts segments by timeline and WAL position, keeps an in-flight set keyed by WAL file name, starts uploads while fewer than four run, reports each new count through `RecordConcurrentUploads`, and wakes on its tick and on every worker completion. Timeline history files stay on the dispatcher. Verify with tests in `wal_watch_dir_test.go`, run through `just test-package <wal>`, against a mock storage whose writes block until released:
  - eight queued segments reach four concurrent writes and never five;
  - the fifth segment starts as soon as one write is released, which the test waits for as a condition with a limit;
  - the spec scenario "Start order follows WAL position": with 10 to 13 and 15 in the watch directory and 14 in `pending-upload/`, 14 starts before 15 when the first slot frees up;
  - a segment whose write is still blocked is not started again on the next tick, and the captured log shows no "held by another in-flight claim" warning while the write stays blocked.
- [x] 8.2 Route the startup recovery of leftover segments through the same dispatcher with `RecoverSegment`, as one pass over the files listed at startup that returns when those uploads have finished (design D1). `Run` keeps calling it before the receiver spawns. Verify with tests in `wal_watch_dir_test.go`, run through `just test-package <wal>`:
  - several finalized segments with their own NULL claims all commit, and recovery used more than one concurrent write;
  - with a mock storage whose writes always fail, recovery returns after one attempt per segment and the segment files stay in place.
- [x] 8.3 Make the existing pending-upload tests pass on the dispatcher: `Test_SweepPendingUploads_WhenUploadSucceeds_RemovesSegmentFromStagingDir` and `Test_SweepPendingUploads_WhenUploadFails_RetainsSegmentForNextTick`, renamed if the function they name no longer exists. Add one test where a segment is renamed into `pending-upload/` while its upload is blocked, and assert it is written to storage once. Verify with `just test-package <wal>`.

## 9. Staleness on the archive progress point

- [x] 9.1 Make `getLastCommittedWal` also return the newest committed segment's timeline. Add the pure function that combines the lowest pending segment (timeline and start, or none) with the newest committed segment (timeline and end) into the archive progress point (design D4). Verify with tests in `wal_staleness_test.go`, run through `just test-package <wal>`:
  - "One segment keeps failing while others succeed": the point stays at segment 50's start while the newest committed end moves through 51 to 90, and the tracker reports staleness once the threshold passes;
  - "Out-of-order finishing within the threshold": the point moves to the newest committed end once nothing is pending below it;
  - "Idle source": the point equals the newest committed end when nothing is pending;
  - a pending segment on an older timeline holds the point even when its position is higher than the newest committed end on the newer timeline.
- [x] 9.2 Have the dispatcher store the lowest listed segment, short files included, under a mutex after each listing. Have `reportArchiveStalenessIfDue` pass the combined progress point to the tracker. Verify with tests in `wal_watch_dir_test.go`, run through `just test-package <wal>`:
  - with segment 50's write blocked while 51 to 53 commit, the stored lowest segment is 50;
  - with a short file for segment 60 in the watch directory, the stored lowest segment is 60.

  The existing staleness tests must still pass.

## 10. Encoder memory

- [x] 10.1 Create the WAL segment zstd encoder with `zstd.WithEncoderConcurrency(1)` and `zstd.WithLowerEncoderMem(true)` (design D6). Verify with `just test-package <wal>`: `Test_BuildWalSegmentArtifactReader_WhenLargeSegment_StreamsAndCountsArtifact` and the restore-side decoding tests in the package pass.

## 11. Wrap-up

- [x] 11.1 Run the neighbouring packages that exercise WAL streaming end to end. Verify with `just test-package ./internal/features/backups/backups/backuping/physical/... ./internal/features/backups/backups/core/physical/...` passing.
- [x] 11.2 Search `docs/`, the READMEs, `adr/` and code comments for statements that segments upload one at a time, that each storage write opens its own connection, or that describe `lastNotifiedGapEnd`, and bring each in line with the new behavior. Then run the formatter and the linters. Verify with `just format` and `just lint` at the repository root finishing clean.

## Change verification

```sh
just test-package ./internal/util/testing/containers/...
just test-package ./internal/features/storages/...
just test-package ./internal/features/backups/backups/usecases/physical/postgresql/...
just test-package ./internal/features/backups/backups/backuping/physical/... ./internal/features/backups/backups/core/physical/...
```
