# ADR-0010: No support for custom tablespaces in physical backups

- **Status:** Accepted
- **Date:** 2026-05-26
- **Tags:** backups, postgresql, physical, tablespaces

## Context

Physical backups stream `pg_basebackup --pgdata=- --format=tar` directly into storage with zero local disk (see [ADR-0004](./0004-focus-on-streaming-chunk-by-chunk-with-backpressure.md)). `pg_basebackup` writes one tar to stdout for the main data directory plus one tar per custom tablespace. Since stdout is a single stream `pg_basebackup` refuses `-D -` whenever any tablespace other than `pg_default` and `pg_global` exists.

Two paths could preserve physical backups for those clusters: a bounded local staging buffer that catches the multi-file tar output and multipart-uploads each file in flight; or a custom implementation of the PG BASE_BACKUP replication protocol that bypasses `pg_basebackup` entirely (the path pgBackRest and WAL-G take). Both require significant new architecture.

Custom tablespaces are a minority configuration even among production PG operators we target. Managed PG (RDS, Aurora, Cloud SQL, Azure DB) restricts or forbids them. Kubernetes operators (CloudNativePG, Zalando, Crunchy) deploy one PVC per instance. Modern self-hosted PG runs on a single large NVMe where ZFS or LVM has replaced the I/O-tiering motivation tablespaces historically served.

The remaining slice is either legacy installations carrying a tablespace decision made years ago, or specialty workloads like Greenplum (which ships its own `gpbackup`) and TimescaleDB chunk-tiering (which sits on pgBackRest / WAL-G already and is not part of our target audience).

## Decision

Physical backups refuse clusters with any tablespace outside `pg_default` and `pg_global`. The refusal is a pre-flight check (`CheckNoCustomTablespaces` in `timeline.go`) that runs before any backup row is created. The scheduler logs `physical_backup_rejected_unsupported_tablespaces` with the offending spcnames and sends one notification per database per 24 hours. The user keeps the logical backup path which handles tablespaces transparently through `pg_dump`'s SQL emission.

This is a permanent decision not a roadmap item. We will not revisit it without a concrete user request carrying real numbers — cluster size, tablespace count and willingness to provision staging disk.

## Alternatives considered

- **Bounded local staging.** Reserve a local volume on the Databasus host, let `pg_basebackup -D <dir>` write multiple tars there, watch with inotify, multipart-upload to storage as files grow and delete after upload. Rejected because it doubles pipeline complexity (loses the kernel-pipe back pressure that comes free with stdout streaming), introduces an ENOSPC failure mode that depends on relative speed of write and upload and serves an audience that has not asked for it.
- **Own PG replication protocol (pgBackRest / WAL-G shape).** Parse `BASE_BACKUP` replication messages directly through `pglogrepl`, emit our own multi-tablespace archive format. Rejected as the explicit anti-direction in [ADR-0008](./0008-why-pg17-native-backups-with-mandatory-wal-summary.md) — we delegate the backup format to PG precisely so we do not own a parallel implementation of PG-internal structure that has to track every major release.

## Consequences

### Positive

- Streaming with kernel-pipe back pressure stays the universal physical-backup pipeline. No staging-disk mode to test, document or fail at runtime.
- The metadata sidecar drops `Tablespaces` and `TablespaceLocation`. Restore-time tablespace mapping code is never written. A class of restore failures ("the original mount path no longer exists on the target host") is removed by construction.

### Negative

- Users with custom tablespaces who want physical / PITR get a clean refusal and must either drop the tablespaces or stay on logical backups. We estimate this affects at most 5–15% production setups.
