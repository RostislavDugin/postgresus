# ADR-0009: Run physical backups remotely instead of via an agent on the database host

- **Status:** Accepted
- **Date:** 2026-05-26
- **Tags:** backups, postgresql, physical, architecture, deployment

## Context

Physical backups built on PG 17 native (per
[ADR-0008](./0008-why-pg17-native-backups-with-mandatory-wal-summary.md))
can be driven in two shapes:

- **Agent-mode.** Install a Databasus-shipped binary on the same host as the
  user's PostgreSQL. The agent reads `$PGDATA` / `pg_wal` locally, runs
  `pg_basebackup` against `localhost`, ships artifacts to storage on the
  user's behalf and coordinates with the central Databasus over an outbound
  channel. This is the shape pgBackRest / Barman use, and the shape WAL_V1
  used inside Databasus before it was deleted.
- **Remote-mode.** The Databasus host itself runs `pg_basebackup` and
  `pg_receivewal` against the user's PG over the wire, using the standard
  replication protocol. The user's host runs nothing of ours. Setup on the
  user's side is one role with `REPLICATION` plus a `replication`-line in
  `pg_hba.conf`.

We tried agent-mode in WAL_V1 and removed it.

## Decision

All Databasus physical backup flows run **remotely** from the Databasus
host against the user's PostgreSQL via the replication protocol. Databasus
never ships a per-host agent. For users whose PG is not reachable from the
Databasus host directly, the supported escape hatch is an **SSH tunnel**
from Databasus to a jump host inside the user's perimeter — same backup
code path, transport wrapped.

Three reasons drive the choice:

**1. Simpler setup for the user.** No binary install on the database host,
no init script, no agent upgrade cycle, no host-level capacity reservation.
The end-to-end setup is: create a role with `REPLICATION` privilege, add
one `replication` line to `pg_hba.conf` (or grant `rds_replication` on
managed PG), reload, paste creds into Databasus. Most managed PG providers
forbid running anything on the DB host anyway, so an agent shape would
either exclude managed PG entirely or fork into a remote-mode branch for
that case — which is the branch we now use universally.

**2. Less code and a cleaner test surface.** A single execution path —
"Databasus host runs `pg_basebackup` / `pg_receivewal` against a remote
endpoint" — keeps the physical pipeline identical to how logical backups
already work. No second binary to build, version, sign and ship; no agent
self-update protocol; no E2E matrix that stands up a real agent next to
a real PG and exercises the cross-process protocol in CI. The CI matrix for
physical backups is the same matrix we already run for logical: spawn a PG
container, point Databasus at it, run the flow.

**3. Privacy and closed-network deployments are already covered.**
Databasus is normally deployed inside the user's own perimeter — the
remote connection from Databasus to PG crosses a private network the user
controls, not the public internet. For the minority of deployments where
PG sits behind a stricter inner perimeter than Databasus itself, an SSH
tunnel from Databasus to a jump host inside the PG perimeter delivers the
same security profile as an agent would (no PG port exposed beyond the
inner perimeter, traffic encrypted, authentication via the user's own SSH
infrastructure) while keeping the backup code path unchanged. This covers
the long tail without forking the pipeline.

## Alternatives considered

- **Agent on the user's database host.** Rejected for the reasons above:
  doubles the binary / release / CI surface, excludes managed PG by
  construction (or forks into remote-mode anyway), and offers no
  functional capability that remote-mode + SSH tunnel does not already
  deliver. The historical WAL_V1 implementation is the concrete evidence —
  it carried agent-token plumbing, a separate `agent/backup/` package, a
  scheduler bail-out branch and per-method `if BackupType == WAL_V1
  { not supported }` shims through the codebase, all of which dissolved
  cleanly when remote-mode replaced it.

## Consequences

### Positive

- **One code path for physical backups.** The remote-execution shape
  matches logical backups and the verification agent's restore flow, so
  shared infrastructure (node assignment, dead-node detection, bandwidth
  fairness, encryption, storage abstraction) carries over without
  per-shape branching.
- **No agent release pipeline.** No second artifact to version, sign,
  publish, document or auto-update across the installed base. The
  Databasus binary is the only thing the user runs.
- **Managed PG works out of the box.** RDS / Cloud SQL / Azure DB ship
  with the replication protocol exposed and host-side installs forbidden.
  Remote-mode fits that shape natively; agent-mode would not.
- **Single test surface.** Physical CI exercises the same
  "spawn PG container, point Databasus at it" matrix as logical, plus an
  SSH-tunnel variant. No cross-process agent protocol to fuzz.