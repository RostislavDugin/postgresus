# ADR-0011: No partial WAL uploads to cloud storage

- **Status:** Accepted
- **Date:** 2026-05-27
- **Tags:** backups, postgresql, physical, wal, pitr

## Context

`pg_receivewal` streams WAL from the source cluster to Databasus in real time, writing into a `.partial` file that grows as the source writes. When the segment fills (default 16 MB) PostgreSQL rotates it and `pg_receivewal` renames the file to its final name. The plan's invariant — `.partial never uploaded` — means we only ship fully-rotated segments to cloud storage.

That bounds cloud-side RPO by `wal_segment_size`. On an active OLTP cluster a segment fills in seconds and the gap is invisible. On an idle cluster the same 16 MB might accumulate over hours or days, and a source crash in that window can lose everything since the last completed rotation.

PostgreSQL has a native knob for this exact problem: `archive_timeout`. Setting it on the source forces a WAL switch every N seconds regardless of fill, capping the size of any unrotated tail. Every mainstream physical backup tool (pgBackRest, WAL-G, Barman) takes the same shape we do — upload completed segments only, document `archive_timeout` as the RPO control.

## Decision

We upload completed WAL segments and nothing else. `.partial` stays a transient file on the Databasus host and never reaches storage.

Customers who need a tighter RPO than `wal_segment_size` set `archive_timeout` on their source cluster (or the parameter-group equivalent on managed PG). The UI surfaces this as a one-line hint next to the physical-backup config: the SQL command for self-managed, the parameter-group key for RDS / Cloud SQL / Azure.

Databasus does not call `pg_switch_wal()` on the source. RPO control lives where the data lives — on the customer's cluster.

## Alternatives considered

- **Partial-tip channel.** Snapshot the current `.partial` every N seconds, compress, encrypt, upload under a tip key that gets overwritten on each snapshot. Delete the tip when the underlying segment rotates and uploads normally. Rejected because it doubles the WAL pipeline: a new `physical_wal_partial_tips` table, a snapshotter goroutine alongside the existing streamer, lifecycle handover between tip-deleter and snapshotter, restore-side merge logic, plus ongoing storage PUT cost (~$8/day per 100 DBs at a 5-second interval on S3). None of pgBackRest, WAL-G or Barman ship this — we'd be inventing a non-standard mechanism for an effect `archive_timeout` already delivers.
- **Databasus-driven `pg_switch_wal()` on a timer.** Force rotation from Databasus every N seconds without touching the customer's GUC. Rejected because it mutates source-cluster state beyond what `REPLICATION` privilege normally implies, masks the real WAL behaviour from the operator (they see "RPO 5s" without understanding why, then get a surprise after failover or when Databasus is paused) and creates two competing rotation cadences (ours plus whatever the operator may set later).

## Consequences

### Positive

- The WAL pipeline stays a one-direction conveyor: `pg_receivewal` produces completed segments, we ship them. No mutable artifacts in storage, no tip lifecycle, no merge step at restore time. Every row in `physical_wal_segments` is immutable once `file_name` is non-NULL.
- RPO control sits with the customer, where the storage cost of tighter RPO (more segments per day, each mostly zero-padded) is also paid. We don't make budget decisions for them.

### Negative

- Cloud-side RPO on an idle cluster with default settings can be arbitrarily large. Users who don't read the help text and run an idle DB will be unpleasantly surprised after a source-side disaster.
- Managed PG users who can't set `archive_timeout` through their provider's parameter group have no escape hatch. In practice every major managed PG (RDS, Cloud SQL, Azure DB) exposes the parameter, but we accept being blocked on the rare provider that doesn't.
