# ADR-0004: Stream backup and restore data chunk-by-chunk with backpressure

- **Status:** Accepted
- **Date:** 2025-12-05
- **Tags:** backups, storage, performance, reliability

## Context

The original backup pipeline ran `pg_dump` and pushed its output to the
configured storage backend (S3 or local) as a single in-process flow,
accumulating the unsent tail of the dump in memory. With a single backup
running at a time this was fine.

Once real users started backing up 5–10 databases in parallel, the host ran
out of memory. The cause was the same on every install: the producer
(`pg_dump`) was consistently faster than the consumer (upload to storage),
so the difference accumulated in RAM. A slow storage backend, a transient
network error or an S3 throughput cap made the situation strictly worse —
the pipeline kept reading from `pg_dump` with nowhere to send the bytes,
and a single stalled backup could OOM-kill the whole container along with
every other backup running next to it.

The same shape applies to restore: reading a backup faster than the target
PostgreSQL can ingest it would let the buffered tail grow without bound.

## Decision

All backup and restore data moves through the pipeline as a stream of
fixed-size chunks (illustrative ~8 MB; the exact size depends on the
operation and the storage backend). The producer (e.g. `pg_dump` stdout,
`pg_basebackup` stream or the storage downloader during restore) and the
consumer (the storage uploader during backup, `pg_restore` or PostgreSQL
during restore) are coupled by **backpressure**: the producer cannot
enqueue chunk *N+2* until chunk *N* has been accepted by the consumer. A
small bounded in-flight window (illustrative ~16 MB per pipeline) caps
per-process memory regardless of any upstream/downstream speed mismatch.

**Backup and restore data does not land on the Postgresus host's disk.**
Bytes stream directly between PostgreSQL and the storage backend in both
directions. The Postgresus container's local disk is not used as a staging
area for backup payloads.

**Exception — operations that genuinely require a materialised file.**
`pg_restore -j >1` cannot consume a pipe; it needs a real file on disk to
parallelise across workers. In that case Postgresus explicitly pre-checks
that the local disk has enough free space to hold the full backup and
refuses the operation with a clear error if it does not. The streaming
model remains the default; materialisation is an opt-in fallback gated by
a precondition check.

## Consequences

### Positive

- **Bounded per-pipeline memory.** Memory usage is capped by the in-flight
  window, independent of the size of the dump or the speed of the storage
  backend. A 500 GB database costs the same RAM as a 500 MB one.
- **Predictable parallelism.** N parallel backups consume N × window-size
  of memory, not N × dump-size. Capacity planning becomes a straight-line
  function of how many concurrent jobs the operator allows.
- **Slow storage degrades gracefully.** A slow, throttled or stalled
  storage backend slows the producer instead of crashing the host. The
  worst case is a long-running backup, not an OOM.
- **No local disk requirement for normal backup/restore.** Operators do
  not need to provision Postgresus-host disk equal to their largest
  backup, and S3-as-primary-storage deployments work without local scratch
  space.
- **Backups larger than the host's free disk are supported.** Database
  size is bounded by the storage backend's capacity, not by the
  Postgresus container's filesystem.

## References

- [ADR-0001](./0001-use-docker-as-base.md) — Docker base image (defines
  the single-container environment whose memory limits this decision is
  written against).
