# ADR-0001: Ship as a Docker image

- **Status:** Accepted
- **Date:** ~2024-06-01
- **Tags:** deployment, packaging, ops

## Context

Postgresus is a system for backup and restore operations.

It require for work:
- a Go backend (Gin + GORM) with an embedded scheduler;
- a React frontend, served as static assets;
- PostgreSQL tools (`pg_dump`, `pg_restore`, `pg_receivewal`) required for backup and restore operations.

We need to decide what the deploy unit is.

## Decision

We ship as a published `docker-compose.yml` with two services: the Postgresus
app image (backend + frontend + every supported `pg_dump`/`pg_restore`
version) and a PostgreSQL service for internal state. The full install
is always:

```bash
docker compose up -d
```

against our compose file. We don't publish standalone binaries, OS packages,
a single-image `docker run` flow or hosted SaaS.

## Alternatives considered

- **A single static binary** (Go binary with the frontend embedded via
  `embed.FS`) plus PostgreSQL available on the host.

## Consequences

### Positive

- **Backend and frontend in one unit.** The frontend ships as static files
  served by the Go process. No reverse proxy needed, no version skew between
  API and UI.
- **One deploy unit, one upgrade path.** Migrations, config validation and
  the embedded scheduler all live in the app container; internal state lives
  in the sibling PostgreSQL service. `docker compose pull && docker compose
  up -d` is the entire upgrade procedure; state lives in named volumes so
  rollback is a tag change.
- **Trivial install.** `docker compose up -d` works on any host with Docker.
  No language toolchains, no `apt install postgresql-client-16`, no PATH
  games.
- **Cross-OS.** Same image works on Linux, macOS and Windows.
- **Predictable runtime.** The exact `pg_dump` used for a backup is the one
  installed during image build when we install all PostgreSQL versions.
- **Subjectively, it's just nice to work with.** This isn't a serious argument
  on its own, but it matters in a self-hosted product where the install
  experience *is* the first-impression UX.

### Negative

- **Hard dependency on Docker.** Users who can't or won't run Docker (some
  shared-hosting environments, philosophical preferences) aren't served. We
  accept this — the target user already has Docker.
- **Big image, slow first pull.** Carrying multiple PostgreSQL versions makes
  the image meaningfully larger than a Go binary would be. Multi-stage builds
  and careful layer ordering help, but the first pull is still slower than
  downloading a binary. Subsequent upgrades benefit from layer reuse.
- **Slow CI builds.** Multi-arch builds with several engines and several
  major versions per engine take longer than `go build`. Acceptable for the
  team; visible on release day.
- **"I just want a binary" friction.** Some users prefer a systemd unit or a
  native package. We're explicitly not catering to that — the maintenance
  cost of a second distribution channel outweighs the UX win.