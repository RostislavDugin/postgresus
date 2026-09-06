## Why

The current Docker image assigns separate numeric identities to Databasus and its embedded services. This breaks writable bind mounts whose existing ownership matches an earlier image, and the filesystem suite does not exercise the same temporary-to-backup publication path that fails in production.

## What Changes

- Run Databasus, embedded PostgreSQL, and Valkey under one non-root operating-system account named `databasus`.
- Support optional `PUID` and `PGID` overrides, with automatic ID selection from existing mounted data when they are absent.
- **BREAKING**: delete `DATABASUS_PUID`, `DATABASUS_PGID`, `POSTGRES_PUID`, and `POSTGRES_PGID` without aliases or fallback handling.
- Attempt ownership and mode normalization during startup, then decide compatibility through real read, write, publish, and delete operations if metadata changes are unavailable.
- Add a `--test-storage` command-line flag that saves and deletes a unique file through the public local-storage methods.
- Keep image construction in the `Dockerfile` and move runtime startup orchestration into `docker/start.sh`, where named functions make the sequence readable.
- Preserve existing data across upgrades and preserve the deprecated Postgresus directory and `WAL_V1` startup guards.
- Expand the Docker filesystem suite with real bind, named-volume, split-filesystem, CIFS, NFS, permission-failure, identity, upgrade, and embedded-service scenarios.
- Update the six Advanced Config pages and Helm documentation for the single-account model and the `PUID`/`PGID` interface.

## Capabilities

### New Capabilities

None.

### Modified Capabilities

- `docker-storage-compatibility`: Define the single runtime account, automatic identity selection, capability-based startup checks, actionable failures, and upgrade coverage.
- `internal-postgresql-protection`: Use bootstrap-only peer mapping followed by socket rejection and SCRAM-authenticated loopback access.

## Impact

The change affects the root `Dockerfile`, `docker/start.sh`, backend command code, Docker filesystem tests and CI coverage, OpenSpec contracts, all six website Advanced Config translations, and Helm chart documentation. It answers to the root `AGENTS.md`, `backend/AGENTS.md`, and `website/AGENTS.md`.

Out of scope: publishing the embedded PostgreSQL port, changing the PostgreSQL database role name, changing backup formats, or preserving the four deleted identity variables through compatibility shims.
