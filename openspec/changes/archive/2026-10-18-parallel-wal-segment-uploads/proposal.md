## Why

Databasus archives a database's WAL one segment at a time. Each 16 MB segment waits for a claim row, compression, two storage writes and a commit before the next one starts. A source that writes about 1.25 GB of WAL per minute produces roughly 80 segments a minute, so every segment has about 750 ms for that whole cycle.

Each storage write also costs far more round trips than it needs. Every write opens a new connection to the storage: a new TCP and TLS session for S3 and Azure, a new SSH handshake for SFTP, a new login for FTP and a new SMB session for a NAS. SFTP then sends the data in 32 KB blocks and waits for an answer to each one, so one 16 MB segment takes about 500 round trips. With a storage 50 ms away, a single SFTP segment needs about 26 seconds, and the other storage types spend most of the 750 ms budget on handshakes.

When archiving falls behind, the local queue reaches its limit and Databasus stops the WAL receiver. The replication slot then keeps WAL on the source, and the source reports the slot as `extended`. Operators see a repeating "WAL chain at risk" notification and a source disk that keeps filling. A user running a busy production database hit this the day they enabled WAL streaming. Nothing can be tuned to fix it, because both the one-at-a-time loop and the per-write connections are fixed in code.

## What Changes

- Each streaming database uploads up to four finalized WAL segments at the same time. The limit is fixed in code and has no setting.
- Uploads start in WAL order, and the same segment is never uploaded twice at once.
- Storage connections are reused between writes for S3-compatible storage, Azure Blob, SFTP, FTP, SMB and rclone. A connection opened with settings that have since changed is never used again.
- SFTP sends a file's blocks without waiting for each block's answer.
- A storage that stops answering in the middle of a write no longer holds the upload past its timeout.
- An automated test adds 50 ms of network delay to each storage type except Google Drive and requires at least two segments per second through the same write path WAL archiving uses.
- The WAL gap notification fires only for a segment that will never arrive, and its range starts at the nearest segment that is archived or still queued. A later segment that finishes before an earlier one no longer counts as a gap.
- The "archiving fell behind" alert follows the point up to which WAL is archived without holes. A single stuck segment trips the alert even while later segments keep finishing.
- The periodic archiving log line also reports how long each upload step took and how many uploads ran at once. Operators can then tell slow storage round trips from a saturated network link.

## Capabilities

### New Capabilities

- `physical-backups/wal-archiving`: how finalized WAL segments of a streaming database reach storage. It covers how many segments upload at once and in what order, the write rate archiving sustains against a distant storage, how storage connections are reused and retired, when a WAL gap is reported, when archiving counts as fallen behind, and what the periodic archiving log reports.

### Modified Capabilities

None. Restore planning already stops at the first missing segment, so the recovery point users restore to does not change.

## Impact

- Module rules: `AGENTS.md` and `backend/AGENTS.md`.
- Code:
  - the WAL uploader, the watch-directory upload loop, the gap probe and the archive-staleness tracker under `backend/internal/features/backups/backups/usecases/physical/postgresql/`;
  - the S3, Azure Blob, SFTP, FTP, NAS and rclone storage models under `backend/internal/features/storages/models/`, plus a new session pool package next to them;
  - the test container helpers under `backend/internal/util/testing/containers/`, which gain a network-delay sidecar.
- Every storage write uses the reused connections, not only WAL archiving. Full backups, metadata files and deletions of the six storage types above share them.
- Storage servers: each streaming database keeps up to four connections open to its storage instead of opening one per write. Idle connections close after five minutes. An FTP or SFTP server with a low limit on connections per user can refuse some of them. A refused upload is retried on a later pass, as it is today.
- Memory per streaming database on S3 grows from about 34 MB to about 100 MB: four workers, each with a compression encoder of about 9.5 MB and the S3 write buffer of 16 MB. Today one worker holds an encoder of about 18 MB and the same buffer.
- Tests: the storage test package starts six more storage containers, each with a short-lived sidecar, and writes about 400 MB to each.
- No database migration. No API change. No frontend change.

## Out of scope

- A setting for upload concurrency, per database or global.
- Google Drive. No emulator exists to test it against, and its write path stays as it is.
- Caching storage settings read from the database. Each write still reads them, so a change takes effect on the next write.
- The two security defects found while planning this change: SFTP never verifies the server's host key, and the Google Drive token refresh prints token prefixes to standard output. Each gets its own fix.
- Section names that collide in the in-memory rclone configuration.
- The monitoring tab and stored metric history. The `wal-stream-monitoring-tab` change covers them.
- Changing how `pg_receivewal` receives WAL, its `--synchronous` flag or the local queue limit.
- Uploading partial segments. ADR-0011 keeps them out.
- Dropping or merging the per-segment metadata sidecar.
- Notification wording, including adding the workspace name.
