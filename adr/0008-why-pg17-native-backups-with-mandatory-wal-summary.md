# ADR-0008: Use PostgreSQL 17 native physical backups with mandatory WAL summary

- **Status:** Accepted
- **Date:** 2026-05-26
- **Tags:** backups, postgresql, physical, pitr

## Context

Databasus needs physical backups with incremental support and WAL streaming
to deliver PITR. Three implementation paths exist for that pipeline today:

- **Custom block-level incremental** — read `$PGDATA` directly, hash blocks,
  store block-level diffs in a proprietary format. This is what pgBackRest
  (`--repo-block` since 2.46) and WAL-G do. The cost is a full backup format
  with bundle / pagefile / manifest code, block-checksum tracking on the
  client side and per-engine retention machinery.
- **Pre-PG-17 incrementals via WAL replay** — take periodic FULLs and treat
  `FULL + raw WAL replay since stop_lsn` as a substitute for incremental.
  The "incrementals" don't exist as artifacts; the user pays for them with
  replay time on every restore.
- **PG 17 native incremental** — `pg_basebackup --incremental=` against the
  previous backup's manifest, server-side WAL summarizer tracking changed
  blocks, `pg_combinebackup` to materialise a chain into a runnable PGDATA.
  Block-level granularity is computed by PostgreSQL itself from WAL summary
  files in `$PGDATA/pg_wal/summaries/`.

## Decision

We use **PostgreSQL 17's native physical backup stack** for every physical
flow Databasus runs: `pg_basebackup` for FULLs, `pg_basebackup --incremental=`
for incrementals against the parent backup's manifest, `pg_receivewal` for
WAL streaming and `pg_combinebackup` on restore. No custom block-level
format, no pgBackRest-style block hashing, no WAL-G-style bundle code.

The decision has two hard rules attached:

**Physical backups refuse PG < 17.** The required binaries
(`pg_basebackup --incremental`, `pg_combinebackup`) and the required GUC
(`summarize_wal`) do not exist before PG 17. Connection setup refuses
non-17+ clusters with a clear error pointing at the logical path.

**`summarize_wal = on` is mandatory whenever incrementals are selected.**
For backup types `FULL_AND_INCREMENTAL` and `FULL_INCREMENTAL_AND_WAL_STREAM`,
the database-setup flow refuses to save the config until the upstream reports
`summarize_wal = on`. The UI surfaces the fix-it command
(`ALTER SYSTEM SET summarize_wal = on; SELECT pg_reload_conf();`) and a
"try again" button. `FULL_ONLY` skips the check — no INCRs ever spawn so
no summaries are needed.

We do not support `FULL + raw WAL replay` as a substitute for incrementals.
If the user picked incrementals the system delivers incrementals — refusing
to start without summaries is the gate that keeps the runtime in a single
reliable shape.

## Alternatives considered

- **Implement our own block-level incremental** (pgBackRest / WAL-G shape).
  Rejected because PostgreSQL 17 ships the same capability natively. Writing
  our own would mean owning bundle code, pagefile code, manifest code and a
  separate backup format — large surface area to maintain and a separate
  class of bugs (delta math, block-checksum drift, manifest format
  versioning) — for no functional gain over what `pg_basebackup --incremental`
  already delivers.
- **Accept incrementals without `summarize_wal = on` by falling back to
  full WAL replay between FULLs.** Rejected because the fallback is
  unreliable (no per-cycle manifest-level integrity check) and catastrophic
  for RTO (replay of every WAL segment since the last FULL on every
  restore). The user asked for incrementals — silently delivering "FULL
  every cycle with slow restore" is worse than refusing the config.
- **Support physical backups against PG ≤ 16 via the legacy
  `pg_basebackup` + WAL-replay path.** Rejected for the same RTO and
  reliability reasons above, plus the per-version branching cost across the
  whole physical pipeline. PG ≤ 16 users stay on logical backups, which
  cover the same data with a different operational profile.

## Consequences

### Positive

- **Smaller code surface.** No bundle / pagefile / block-checksum
  machinery to maintain. The physical pipeline orchestrates PG-native
  binaries (`pg_basebackup`, `pg_receivewal`, `pg_combinebackup`) and
  records artifacts in storage; the format itself is PG's responsibility.
- **Battle-tested incremental math.** Block-level change tracking lives in
  PostgreSQL's WAL summarizer — a server-side feature exercised by
  upstream's test suite and the broader PG ecosystem. We inherit that
  testing without paying for it.
- **Forward compatibility.** Improvements to `pg_basebackup` /
  `pg_combinebackup` / summarizer in PG 18+ accrue to us automatically as
  the bundled binaries roll forward.
- **Single runtime shape.** Mandatory `summarize_wal = on` collapses
  "what happens when summaries are missing on the source" into one
  failure mode (`SUMMARIZER_OFF` ⇒ refuse) instead of a fallback
  matrix.

### Negative

- **PG 17+ requirement for physical.** Operators on PG 12–16 cannot use
  physical backups in Databasus and stay on `pg_dump`. This is a real
  exclusion — PG 16 was a current major as of late 2024 — but the trade-off
  is deliberate: we built the physical pipeline on a feature set that did
  not exist before PG 17, and back-porting it is the work we explicitly
  chose not to do.
- **Managed PG that forbids `ALTER SYSTEM`.** On a managed PG where
  `summarize_wal` cannot be flipped via SQL (RDS / Cloud SQL / Azure DB
  parameter-group flows), the user must change the GUC through the
  provider's UI before Databasus will save an incremental config. We
  surface the command for self-managed clusters and the parameter name for
  managed ones; we do not silently downgrade to FULL-only.