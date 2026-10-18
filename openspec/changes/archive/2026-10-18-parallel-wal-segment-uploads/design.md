## Context

See proposal.md for the motivation and the specs for the required behavior. Paths are under `backend/internal/features/` unless stated otherwise.

### How archiving works today

All paths in this part are under `backups/backups/`.

- `pg_receivewal` writes finalized segments into `wal-queue/<database_id>/`. The uploader loop wakes every second (`usecases/physical/postgresql/wal_watch_dir.go:30`). `scanAndUpload` (`wal_watch_dir.go:77`) walks the directory in name order and calls `ProcessSegment` for one file at a time. `sweepPendingUploads` (`wal_watch_dir.go:111`) then does the same for segments that `realignResumePath` moved into `pending-upload/` (`wal_resume.go:108`). On startup, `recoverLocalSegmentsOnStartup` (`wal_watch_dir.go:47`) takes over this database's own leftover claims in a single pass, one file at a time, before the receiver spawns (`wal_stream.go:277`).
- `ProcessSegment` (`wal_upload.go:82`) and `RecoverSegment` (`wal_upload.go:128`) insert the claim row (`wal_upload.go:109`, `wal_upload.go:155`), then `uploadAndCommit` (`wal_upload.go:229`) does the rest. It compresses and encrypts the segment while writing it to storage, writes the metadata sidecar, and commits the row and both write receipts in one transaction. After that it runs the gap probe and removes the local file. A failed step releases the claim and keeps the file for the next pass.
- `probeChainGap` (`wal_upload.go:324`) compares the committed segment with the newest committed segment before it, using `FindLatestCommittedBefore` (`core/physical/repositories/wal_segment_repository.go:298`). Repeat notifications are suppressed through `lastNotifiedGapEnd` (`wal_upload.go:64`), which only moves forward.
- The archive-staleness tracker (`wal_staleness.go:37`) treats any rise of the newest committed segment as progress. It reads that segment through `getLastCommittedWal` (`wal_staleness.go:77`), which calls `FindLatestCommittedSegment`. That query orders by timeline first, then by position (`wal_segment_repository.go:335`).
- Every 30 seconds the lag monitor logs one rollup line (`lag_monitor.go:173`). The counters it reads are swapped to zero on read (`wal_upload.go:184`).
- The compression encoder is created per segment with default options (`wal_upload.go:597`). Its stream encoder holds about 17 to 18 MB at the default level, mostly the 8 MB window and its 16 MB history buffer.
- Restore planning already stops at the first missing segment: `contiguousWalRun` (`core/physical/chain_view/restore_set.go:275`). The restorable point users see is therefore correct no matter in which order segments commit.

### How a storage write works today

- `Store.WriteFile` (`storages/files/store.go:45`) inserts a pending-deletion row (`store.go:57`), asks the locator for the storage (`store.go:62`), calls the model's `SaveFile` and then extends the row's window (`store.go:77`). The locator runs `FindByID` on every call (`storages/file_locator.go:18`), which is one query plus one per storage type (`storages/repository.go:135`). Against the internal database this costs a few milliseconds.
- No storage model keeps a client between calls. Each `SaveFile` builds its own:
  - S3: a new `minio.Client` with a new `http.Transport` (`storages/models/s3/model.go:101`, `model.go:855`), so every write starts a new TCP connection, and a new TLS session over HTTPS. Each S3 write also allocates a 16 MB buffer for the first part (`model.go:120`).
  - Azure Blob: a new client and transport (`storages/models/azure_blob/model.go:83`).
  - SFTP: a new TCP connection, SSH handshake and SFTP session (`storages/models/sftp/model.go:58`). The copy at `sftp/model.go:116` ends in the library's `ReadFrom`. For a reader of unknown size, such as our compression pipe, it writes one 32 KB block at a time and waits for each answer (`pkg/sftp@v1.13.11/client.go:2022`). One 16 MB segment takes about 512 round trips.
  - FTP: a new connection and a full login (`storages/models/ftp/model.go:58`), then a directory walk with one command per path segment, then a new data connection.
  - NAS: a new SMB session and share mount (`storages/models/nas/model.go:63`), then a directory walk with one stat per path segment.
  - rclone: a new remote through `fs.NewFs` (`storages/models/rclone/model.go:56`), under a process-wide mutex (`rclone/model.go:29`) that serializes every rclone storage in the process.
- Requests made by the SFTP and SMB libraries ignore the caller's context. A connection whose peer vanished without closing it hangs the write until the operating system gives up on the socket.
- The storage test containers (`backend/internal/util/testing/containers/`) run on the same host as the tests. A round trip takes microseconds, so none of the costs above shows up in any test.

### Constraints

- `backend/AGENTS.md`, Stored files: every write attempt gets its own object name, and a file is kept only when a committed transaction claims it. The per-attempt object name (`wal_upload.go:500`) and the commit transaction already satisfy both rules for concurrent writes. Reused connections change how bytes travel, not which files are kept.
- `backend/AGENTS.md`, Background services, Logging and Testing: one background loop per file, `job_id`/`job_name` scoping, and unit tests only where no API exists. No HTTP surface covers WAL archiving or storage writes, so tests drive the uploader, the dispatcher and `Store.WriteFile` directly.
- ADR-0004 asks for bounded memory through streaming with backpressure. Upload concurrency and pooled connections are both bounded by fixed limits, and the local queue keeps its existing high watermark (`wal_backpressure.go`).
- ADR-0011 keeps partial segments out of storage. Nothing here changes that.
- The `single-instance-runtime` spec runs one application process per installation. In-memory state of the uploader is authoritative for its database, and an in-memory connection cache is authoritative for the process.

## Goals / Non-Goals

**Goals:**

- Remove per-write connection setup from every storage type except Google Drive and local storage.
- Make SFTP write a segment in a handful of round trips instead of hundreds.
- Hide the remaining per-segment latency by overlapping up to four uploads per database.
- Keep the gap notification and the staleness alert correct when segments commit out of order.
- Leave enough timing in the logs to tell latency-bound archiving from bandwidth-bound archiving.
- Prove the write rate against every covered storage type with a test that injects real network delay.

**Non-Goals:**

- Faster compression, sidecar removal or fewer database round trips per segment.
- Any change to receiving WAL, the local queue limit or restore.
- Google Drive's write path.

## Decisions

### D1. A continuous pool of four workers fed in WAL order

The uploader loop becomes a dispatcher. On each tick, and whenever a worker finishes, it lists the watch directory and `pending-upload/`, sorts the segment files by timeline and then by WAL position, and starts uploads for files that are not in flight while fewer than four are running. Every start goes through one path: live segments, pending-upload segments, and the startup recovery of leftover claims. Recovery keeps calling `RecoverSegment` and live files keep calling `ProcessSegment`. Timeline `.history` files stay on the dispatcher goroutine, as they are rare and need a source connection.

Startup recovery stays a single pass. The dispatcher lists the two directories once, starts every segment found there through the four slots, and returns when those uploads have finished, whatever their outcome. A failed segment waits for the live loop, as it does today. Recovery never loops until the queue is empty; otherwise a storage outage at startup would keep the receiver from ever spawning.

Starting in WAL order keeps the archive progress point (D4) moving forward and drains the oldest WAL first under backpressure. Nothing depends on claims being inserted in that order. A failed segment is retried after later segments have claimed, and goroutines started in order may reach the database out of order. The gap probe (D3) works from local files and the catalog, so it does not need that order either.

Alternatives rejected:

- *A batch per tick.* Start up to four uploads from the current listing and wait for all of them before the next tick. There is no shared in-flight state, but one slow upload holds the next batch for up to `walSegmentUploadTimeout`, two minutes (`wal_upload.go:33`). The three free workers would sit idle exactly when the queue is longest.
- *Pipelining inside one worker.* Compress segment N+1 while N uploads. At most a 2x gain, and none when the storage round trip, not compression, dominates.
- *A setting for the worker count.* Kept out of scope by the proposal. A constant can become a setting later without breaking anyone, while removing a setting would break existing configurations.

Why four: the user's source needs about 1.3 segments per second. Connection reuse (D7) and pipelined SFTP writes (D8) bring a single write against a storage 50 ms away down to a few hundred milliseconds, but one TCP stream still caps how fast a single upload moves bytes over a long path. Four workers reach the required rate with a full cycle of up to about three seconds per segment. Higher numbers raise the risk of hitting FTP or SFTP per-user connection limits and multiply encoder and buffer memory across every streaming database.

### D2. An in-memory in-flight set owned by the dispatcher

The dispatcher keeps the set of file names currently uploading. Only the dispatcher reads it; the gap probe (D3) and the progress point (D4) work from the directory listings instead. A name enters the set when its upload starts and leaves it when the worker returns, whatever the outcome. The set prevents a second start of the same segment. It also prevents the "held by another in-flight claim" warning (`wal_upload.go:219`), which would otherwise fire on every tick for a slow upload.

The set is keyed by WAL file name, not path. `realignResumePath` can move a file from the watch directory into `pending-upload/` while its upload runs, and the same segment must not start again from the new path. On Linux an open file survives a rename, so the running upload finishes normally. Its `removeLocal` of the old path then finds nothing, and the next pass sees the moved copy as already committed and drops it through `handleClaimConflict`. That is today's behavior and stays as it is.

Alternative rejected: *rely on the claim row's uniqueness alone.* It is already race-safe (`Test_WalUpload_ConcurrentClaimSameSegment_OnlyWinnerInserts`), but every loser would do a probe query and log a warning on every tick. Under the single-instance runtime the database adds no safety that the in-process set lacks.

### D3. The gap probe asks whether the missing predecessor is still coming

The uploader gets the watch directory in its dependencies. When the newest committed predecessor does not end where the committed segment starts, the probe looks up the segment directly before it, the one ending at `StartLSN`, before notifying:

1. If its full-size file exists in the watch directory, or after that in `pending-upload/`, it is still coming, and no gap is reported. A file shorter than a full segment does not count. `pg_receivewal` resumes after the highest complete segment (`GetResumeSegmentNo`, `wal_resume.go:39`), `movePendingUploadsOutOfResumePath` leaves short files where they are (`wal_resume.go:87`), and the uploader only warns about them (`wal_upload.go:88`), so a short file below the resume point is never replaced. Counting it would suppress the notification for that hole forever. An uploading segment's file stays in one of these two places until `removeLocal` runs after the commit, so the in-flight set is not consulted and the uploader keeps no dependency on dispatcher state.
2. Otherwise the probe looks it up in the catalog by chain key, whether committed or claimed. If a row exists, it was committed or claimed between the two reads, and no gap is reported.
3. Otherwise the predecessor will not arrive, and a gap is reported.

The order of the checks closes both races:

- The watch directory is checked before `pending-upload/`. `realignResumePath` only moves files from the first into the second, so a file renamed between the two checks is still found in the second.
- The local check runs before the catalog check. A predecessor that commits concurrently writes its row before it removes its local file (`wal_upload.go:229` onwards), so at least one of the two checks sees it.

The reported gap ends at `StartLSN`. It starts at the larger of two positions: the end of the newest committed predecessor, and the end of the highest full-size segment file on the same timeline that still sits in the two directories below `StartLSN`. Under concurrent uploads the newest committed predecessor can lag behind a segment that is still uploading, and the first position alone would report a hole wider than the real one. The probe already lists both directories for step 1, so the second position costs nothing extra.

Each segment checks only its direct predecessor, so only the first segment after a real hole reports it. That segment commits at most once, because the commit is guarded by `file_name IS NULL`, and re-processing a committed segment goes through `handleClaimConflict`, which runs no probe. Each hole is therefore reported at most once per session without any bookkeeping. `lastNotifiedGapEnd` and the lock around it are deleted rather than replaced. Under out-of-order commits the forward-only marker would also hide a lower gap found after a higher one.

Alternatives rejected:

- *Report a gap only when the hole outlives a grace period.* It needs a timer and per-hole state, delays real gap alerts, and still needs the "is it still coming" check to avoid alerting on a long retry.
- *Keep a set of reported gaps.* It guards against a duplicate that the commit guard already rules out, and it grows for the whole session.

### D4. Staleness follows the archive progress point

The tracker's input changes from "newest committed segment" to the archive progress point defined in the spec. Two goroutines hold the two halves, so the value is assembled in the lag monitor:

- **The dispatcher owns the lowest pending segment.** After each listing of the watch directory and `pending-upload/`, it stores the timeline and start position of the lowest segment it found, or "none", in a field on the streamer guarded by a mutex. "Lowest" compares the timeline first and the position second, the same order `FindLatestCommittedSegment` uses. Files of uploading segments are still in the listing, so in-flight segments count without a separate lookup. Files shorter than a full segment count as well: restore cannot pass that position until the segment arrives, so the point must not pass it either. This is how a torn segment surfaces: through the staleness alert and the existing incomplete-segment warning, never through a gap notification (D3). Between listings the stored value can only be lower than the truth, never higher, which errs towards reporting staleness rather than hiding it.
- **The lag monitor combines the two.** `reportArchiveStalenessIfDue` (`lag_monitor.go:194`) already reads the newest committed segment through `getLastCommittedWal`. It reads the dispatcher's value and passes both to a pure function. That function returns the lowest pending start when the lowest pending segment lies below the newest committed one in (timeline, position) order, and the newest committed end otherwise. The result goes to `recordSampleAndDetectStaleness` in place of the newest committed end. `getLastCommittedWal` returns the newest committed segment's timeline alongside its end for this comparison.

This adds no extra query. The tracker logic in `wal_staleness.go:37` stays the same; only the value it receives changes, so the new behavior is tested on the pure function. `ChainRiskReport.LastArchivedWalAt` keeps reporting the newest committed segment's time, since the alert text explains that the recovery point stopped advancing.

Alternative rejected: *compute the gap-free run from the catalog with `contiguousWalRun`.* It is exact, but it reads the whole chain's WAL every 30 seconds for every streaming database. The `background-load` spec exists to keep that kind of work out of periodic loops.

### D5. Step timing collected per upload, reported in the existing rollup

Four steps are measured. The claim insert is timed in `ProcessSegment` and `RecoverSegment`, where it happens, and handed to `uploadAndCommit` with the claim. `uploadAndCommit` times the segment write (compression and encryption run inside it through the pipe), the sidecar write, and the commit including the gap probe. Each upload adds its durations to accumulators under the uploader's mutex.

The dispatcher knows how many uploads run, so it reports the count to the uploader through one method each time it starts an upload. The uploader keeps the highest count since the last report. `TakeArchivedSinceLastReport` becomes a take of one snapshot struct with the counts, the per-step totals and maximums, and the highest concurrency, and resets all of them. `reportArchivingProgress` (`lag_monitor.go:173`) formats the snapshot into the existing INFO line. Values go into the message and IDs stay as kv pairs, per `backend/AGENTS.md` Logging.

Reading the line: if the segment write takes about 16 MB divided by the link speed, archiving is bandwidth-bound and more workers will not help. If it takes much longer, the time is round trips.

Alternatives rejected:

- *One log line per segment at INFO.* A busy source finalizes several segments a second, and the existing comment at `wal_upload.go:301` already rejects that rate.
- *The dispatcher keeps its own counter and the lag monitor reads two sources.* Two snapshots taken at different moments would not describe the same interval.

### D6. Bounded encoder memory

Each worker builds its own zstd encoder (`wal_upload.go:597`) with a concurrency of one and the library's lower-memory mode. A concurrency of one encodes on the caller's goroutine instead of a second one. The lower-memory mode shrinks the history buffer from twice the window to the window plus one block, which takes the encoder from about 18 MB to about 9.5 MB. Neither option changes the compressed format or the window, so existing segments, restore and compression ratio stay the same.

Alternatives rejected:

- *A shared encoder pool.* It adds lifecycle code for a saving that matters only with dozens of streaming databases on one instance.
- *A smaller window.* It saves more memory but changes the compression ratio of every segment.

### D7. Reused connections: shared transports for HTTP storages, a session pool for the rest

The fix differs by protocol, because what each write rebuilds differs.

**S3 and Azure Blob keep building a client per call, over a shared transport.** The only per-write cost these two types have beyond the request itself is the fresh `http.Transport` (`s3/model.go:855`, `azure_blob/model.go:504`). Each package keeps its transports at package level instead: S3 one per `SkipTLSVerify` value, Azure one. `MaxIdleConnsPerHost` is raised from Go's default of two, so four concurrent writes keep their connections. The client is still built per call from freshly decrypted credentials. TCP and TLS connections are then reused, a changed endpoint lands in a different connection pool by itself, and changed credentials take effect on the next request, because both protocols sign every request rather than authenticating a connection. The existing `IdleConnTimeout` of 90 seconds closes idle connections.

A client built per call loses minio's per-client bucket-location cache. When `S3Region` is empty, which `Validate` allows, every write would pay an extra location request (`minio-go/v7@v7.2.1/bucket-cache.go:49`). The S3 package therefore keeps a small map from endpoint and bucket to the region it looked up once with `GetBucketLocation`, and builds each client with that region.

**rclone caches the remote.** `fs.NewFs` is the expensive part and runs under a process-wide mutex. The rclone package keeps a map from storage ID to a remote plus a fingerprint, a SHA-256 over the remote's configuration text. A write whose fingerprint differs from the cached one builds a new remote, so `fs.NewFs` and the mutex run only on a miss or after a change, and retires the old remote. A remote unused for five minutes is retired and dropped. A retired remote is shut down once the last call using it returns, so a full backup streaming through it is never cut off by a settings change. The connection test builds its own remote, since it often runs for settings that are not saved under a storage ID yet.

**SFTP, FTP and SMB get a session pool with exclusive checkout.** These protocols authenticate the connection itself, so a session carries the credentials it was opened with. A new package, `storages/connections`, provides `SessionPool`. Each of the three model packages keeps one pool at package level, because the locator hands out a freshly loaded model on every write. The pool key is the storage ID plus a fingerprint: a SHA-256 over the fields the session is built from, as loaded from the database, with credential fields in their encrypted form. A changed host, user, password, key or TLS flag gives a new fingerprint, so the next write opens a new session without any invalidation hook. When the pool sees a new fingerprint for a storage ID, it closes every idle session under the old one, and a session that comes back under a stale fingerprint is closed instead of kept.

A write takes an idle session or dials a new one, and gives it back when it finishes. A session given back after an error is closed. At most four idle sessions stay per key, and extra ones are closed. A session idle for more than 30 seconds is probed before reuse with one cheap request under a five-second timeout: `Getwd` for SFTP, `NoOp` for FTP, a stat of the share root for SMB. A failed probe closes it, and the write takes the next one or dials. A session idle for five minutes is closed. Each session ensures the target directory once, when it is dialed, instead of walking the path on every write.

The five-minute idle timeout frees server-side connection slots for storages nobody writes to, including deleted ones. No write can reach a deleted storage, because the locator no longer finds it.

Alternatives rejected:

- *Cache whole S3 and Azure clients behind a settings fingerprint.* It needs fingerprint methods and invalidation for clients whose only per-call cost is the transport. Sharing the transport gives the same connection reuse with none of that.
- *Cache the storage model and invalidate it on save and delete.* It needs a hook in every path that saves a storage, including token refreshes and workspace transfers, to save a few milliseconds per write. A missed hook would keep using old credentials, which is the failure the spec forbids.
- *One shared session per SFTP or SMB storage.* Both libraries allow concurrent files on one session, but a full backup streaming over the same TCP connection would make WAL uploads wait behind it, and one dead connection would fail every write in flight.
- *rclone's own `fs/cache`.* It keys on the remote's name and root, not on its settings, so a changed password would keep the old remote until its expiry.

### D8. SFTP writes through the library's concurrent path

`SaveFile` for SFTP replaces the plain copy with `ReadFromWithConcurrency` on the remote file, with the library's default limit of 64 requests in flight. It dispatches blocks without waiting for each answer, whatever the reader's size. On an error the remote file may hold holes. The write then fails, and the caller discards the attempt through the pending-deletion path, as for any failed write.

Alternatives rejected:

- *Enable `UseConcurrentWrites` and keep the copy.* For a reader of unknown size, such as the compression pipe, the library silently falls back to sequential writes (`pkg/sftp@v1.13.11/client.go:2022`).
- *Wrap the reader in an `io.LimitedReader` with a huge limit.* It reaches the same code path through a side effect of the size check, which a library update could change.

### D9. Cancelling a write closes its connection

Every checkout from the session pool registers `context.AfterFunc` on the write's context. If the context ends before the write returns, the function closes the session. The blocked library call then fails, the write returns, and the session goes back as failed and is not reused. The registration is stopped when the write returns normally. FTP data connections are dialed with the same timeout as the control connection instead of none. The FTP client closes a data connection only when its transfer ends, so an FTP session tracks every connection it opens and closing the session closes them all.

S3 and Azure need no extra handling, because their HTTP requests already honor the context and a failed request does not return its connection to the transport's idle pool.

Alternatives rejected:

- *An SSH or SMB keepalive timer.* It finds a dead connection between writes, but a write already blocked on it stays blocked.
- *A read and write deadline on the raw socket.* The libraries own the socket after the handshake, and a fixed deadline would also cut off a large full-backup upload that is progressing.

### D10. A throughput test with real network delay

Three helpers join the test container package. `AddEgressDelay` starts a one-shot sidecar inside the network namespace of a storage container, with the `NET_ADMIN` capability, and runs `tc qdisc add dev eth0 root netem delay 50ms`. The delay applies to every packet the storage sends, so every round trip from the test grows by 50 ms, including FTP passive data connections. `SetEgressLoss` replaces the root qdisc with one that drops every packet, which makes the storage go silent mid-connection. `ClearEgressShaping` replaces it with a plain queue again, and calling it twice is harmless. Each helper waits for the sidecar to exit and fails the test on a non-zero code. After adding the delay, the helper dials the storage port at the container's own address, since Docker's userland proxy completes a handshake on the mapped host port by itself, and fails the test when the TCP handshake took less than the delay, so a host where shaping silently does nothing cannot produce a passing run. The sidecar image, `nicolaka/netshoot`, carries iproute2, is pinned by tag and is added to the `pull-testcontainers` recipe. The storage helpers gain variants that return the container handle the sidecar needs.

The throughput test runs one subtest per covered storage type, in parallel. Each subtest starts its container, adds the delay (except for local storage), and registers a storage row pointing at it. It then writes 24 pairs of a 16 MB file and a 1 KB file through `Store.WriteFile`, the call WAL archiving uses, from four writers. The clock starts with the first write, so connection setup counts. The subtest fails when twelve seconds pass before all pairs are written, and the failure states how many pairs were written and how long that took. At the end it requests deletion of every written file and drains the deletion worker. Before any storage fix, the test runs once and the numbers per storage type are recorded under its task as the baseline.

Two seconds per pair is about 1.5 times the rate of the user who reported the problem. The expected rate after the fixes is several pairs per second for every type, and the current SFTP rate is far below one pair per second, so the floor has room for a loaded CI runner while still catching a regression.

Alternatives rejected:

- *toxiproxy.* FTP in passive mode tells the client to open a data connection to a second port, which goes around the proxy. Every protocol would also need its own proxy setup.
- *A delay inside the test process, wrapping each `net.Conn`.* Several libraries build their own connections and accept no custom dialer, so the test would miss exactly the paths it is meant to measure.
- *Assert the number of connections or handshakes.* It is deterministic, but it pins the mechanism rather than the rate the user needs, and it breaks under refactoring that keeps the rate.
- *A bandwidth test without delay.* On a shared CI runner it measures the runner's disk and CPU and would flake.

## Risks / Trade-offs

- [An FTP or SFTP server limits connections per user below four, so some uploads fail every pass.] → A failed upload releases its claim and retries, so throughput never drops below one upload at a time. The step-timing line and the existing upload-failure warning make the cause visible.
- [Storage providers with request-rate limits throttle four parallel writers.] → Same retry path. Four is low enough to stay under common per-user limits.
- [A server drops an idle pooled session in less than 30 seconds, and the next write fails on it without a probe.] → The write fails, the session is closed, and the segment is retried on the next pass one second later. The failure shows up as an upload warning.
- [The netem sidecar needs the `sch_netem` kernel module and permission to add `NET_ADMIN`.] → Both exist on the Fedora development host and on GitHub's Ubuntu runners with rootful Docker. Rootless Docker may refuse the capability; the helper then fails loudly instead of skipping.
- [The FTP test container binds its passive ports one to one on the host, so two FTP containers cannot run at once.] → The throughput test starts one FTP container, and no other package starts one.
- [The storage test package writes about 400 MB per storage container.] → Files are deleted at the end of each subtest. Subtests run in parallel, so the extra wall time is close to the slowest one.
- [The out-of-order window shows a transient "after a gap" mark on a WAL range row in the backup list.] → It lasts as long as one upload, since the `physical-backup-wal-ranges` spec marks gaps from committed rows. The mark clears when the predecessor commits. No notification is sent.
- [More segments pending at once make `realignResumePath` move files more often.] → The move is a rename within one directory tree. D2 keys the in-flight set by name, so a moved file is not uploaded twice.
- [Memory per streaming database on S3 grows to about 100 MB.] → It is bounded by the worker count. A shared limit across databases can be added later if many streaming databases share one instance.

## Migration Plan

No schema or API change. Deploying the new binary switches to concurrent uploads on the next streamer start and to reused storage connections on the first write. Rolling back to the previous binary restores sequential uploads and per-write connections. Claims left by either version are taken over by startup recovery as they are today.

## Open Questions

None.
