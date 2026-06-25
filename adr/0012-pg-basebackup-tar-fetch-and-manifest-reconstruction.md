# ADR-0012: pg_basebackup streams a single tar to stdout; backup_manifest is reconstructed on Databasus

- **Status:** Accepted
- **Date:** 2026-05-27
- **Tags:** backups, postgresql, physical, pg_basebackup, compression, manifest

## Context

[ADR-0008](./0008-why-pg17-native-backups-with-mandatory-wal-summary.md)
commits Databasus to driving PG's native physical-backup binaries;
[ADR-0009](./0009-why-remote-physical-backups-instead-of-agents.md)
commits to running them remotely from the Databasus host. Within those
constraints the FULL/INCR executor must satisfy three things at once:

1. **No local staging.** The host serves many sources; staging a 3 TB
   database per concurrent backup would need tens of TB of disk that
   sits cold between backups.
2. **One self-contained artifact** per backup — base data plus the WAL
   needed to make it consistent — ready for one-shot restore.
3. **A `backup_manifest` alongside every artifact.**
   `pg_basebackup --incremental=<parent_manifest>` needs the parent
   manifest to find changed blocks; no manifest means no incrementals,
   which ADR-0008 forbids.

`pg_basebackup` exposes its features through CLI flags, and the wrong
combination silently breaks one of these. This ADR records the chosen
flag set — and the one conflict (server-side compression vs. embedded
manifest) that forces us to rebuild the manifest ourselves.

## Decision

Invoke `pg_basebackup` with
`--pgdata=- --format=tar --wal-method=fetch --compress=server-zstd:5 --no-manifest`
(INCR adds `--incremental=<parent_manifest>`). The compressed stdout is
teed on Databasus into two independent branches: one uploads the bytes
to storage as-is, the other decompresses and walks the tar to
reconstruct `backup_manifest`. Each flag earns its place:

**`--format=tar --pgdata=-`** streams one tar to stdout, piped straight
into the storage `SaveFile(io.Reader)`. `--format=plain` would need a
real disk path (breaks no-staging) or ~1 TB of uncompressed staging.
ADR-0010 already forbids custom tablespaces, so there is exactly one
tar — no concatenation on our side.

**`--wal-method=fetch`** collects WAL through the same replication
connection at end-of-backup and inlines it as `pg_wal/...` tar entries,
keeping the single-stream invariant. `--wal-method=stream` needs a
second output channel that stdout-tar mode refuses. Fetch's one
weakness — the source must retain WAL until end-of-backup — is covered
by the per-backup replication slot (`backup_slot.go`), which pins WAL
for the backup's duration.

**`--compress=server-zstd:5`** — on TB-scale databases the
PG→Databasus link dominates, and compressing on the source sends ~1/3
of the bytes:

| Link           | client-zstd | server-zstd |
|----------------|-------------|-------------|
| 1 Gbit/s LAN   | ~7 hours    | ~2.3 hours  |
| 100 Mbit/s WAN | ~3 days     | ~22 hours   |

More PG-side CPU for far less wire time is the right trade on
multi-hour windows, so server-side compression is mandatory here.
Because it depends on the source build, the codec degrades
`server-zstd:5 → server-gzip:6 → none`.

**`--no-manifest` + reconstruction.** Server compression plus
tar-to-stdout makes pg_basebackup refuse to embed the manifest
(`cannot inject manifest into a compressed tar file`): the stream is
already compressed when bytes leave PG, so it cannot re-open it to
append the trailing manifest entry, and there is no side-channel flag
under `--pgdata=-`. `--no-manifest` removes the conflict but takes
manifests away, which breaks incrementals — so we rebuild it. Every
manifest field is derivable on Databasus from the tar stream plus a
couple of values already on hand (the cluster's system identifier and
the backup's LSN range), targeting PG's version-2 format.
(`--manifest-checksums` is itself incompatible with `--no-manifest`, so
it is omitted; our serializer fixes the per-file checksum at SHA-256.)

We don't aim for byte-identity with PG's own manifest — only a valid,
self-consistent one that truthfully describes the artifact.
`pg_verifybackup` validates that directly: pointed at the produced
compressed-tar artifact plus its reconstructed sidecar (`-m`) it
recomputes every checksum and size, the file set and the manifest's
own SHA-256, so it passes only if the reconstruction is correct — and
catches PG format-drift in CI before production. Reading a compressed
tar needs **PG 18**'s `pg_verifybackup` (tar-format support landed in
PG 18), a test-only tool dependency even though the executor drives
PG 17.

The reconstructed manifest is a separate sidecar object alongside the
artifact, encrypted independently with its own key material. INCR
downloads the sidecar, materialises it as a temp file and passes it to
`--incremental=`.

## Alternatives considered

- **Stay on `--compress=client-zstd:5`.** Zero new code. Rejected:
  7+ hours per FULL on gigabit LAN is not viable as fleets grow into
  TB-scale databases. WAN is unusable.

- **Local staging (`--pgdata=<dir> --compress=server-zstd:5`).** PG
  emits its own manifest as a side file `backup_manifest`; no
  reconstruction needed. Rejected because it requires ~1 TB of
  staging disk per concurrent backup — exactly the constraint the
  streaming pipeline was designed to avoid. A backup host serving 10
  sources would need tens of TB of staging-only disk sitting cold
  between backups.

- **Speak the replication protocol directly (pgMoneta-style).** Send
  `BASE_BACKUP COMPRESSION 'server-zstd:5' MANIFEST 'yes'` over a
  libpq replication connection; manifest arrives as a separate
  CopyData stream byte-exact from PG. Rejected because it requires
  re-implementing pg_basebackup's full surface — multi-tablespace
  handling, WAL fetch trailer, error recovery, incremental protocol —
  and maintaining that against every PG release. The single benefit
  (no reconstruction) costs 2-3 KLOC of replication-protocol code
  carried for our use only. The spirit of ADR-0008 — use PG's native
  binaries, don't fork them — applies; if we ever need features
  pg_basebackup cannot provide (mid-stream heartbeats for WAN,
  structured progress events) this is the path we revisit.

## Consequences

### Positive

- PG→Databasus traffic compressed ~3x. 1-3 TB backups complete in
  hours, not days; WAN deployments become viable.
- Self-contained tar artifact preserved (base + WAL + reconstructed
  manifest sidecar).
- Incremental chain unblocked — the manifest sidecar is consumed by
  `--incremental=` exactly as PG's own would have been.
- No new deployment requirement. pg_basebackup remains the workhorse
  and keeps owning retry, error mapping and multi-tablespace handling
  for free.

### Negative

- Manifest-reconstruction code (codec reader + tar walker + PG-format
  serialiser) to maintain.
- Reconstruction-correctness risk against PG's manifest format,
  mitigated by the `pg_verifybackup` gate (PG 18): a drifted
  reconstruction fails in CI, not production.
- Net CPU on Databasus to decompress and SHA-256 the full backup
  contents. Acceptable on a dedicated backup host; the levers are the
  server-side zstd level and the manifest-path worker count.
