# ADR-0002: Built-in PostgreSQL instead of SQLite

- **Status:** Accepted
- **Date:** ~2024-06-01
- **Tags:** storage, database, ops

## Context

Postgresus needs an internal datastore for its own state (schedules, backup
metadata, users). We need to decide what that datastore is and how we ship
it.

## Decision

We use PostgreSQL as the internal datastore and ship it as a second service
in the `docker-compose.yml` alongside the Postgresus app.

## Alternatives considered

- **SQLite** as an embedded datastore — no separate service, single file on
  disk.
- **Full PostgreSQL as a sibling compose service** (chosen).
- **Letting the user choose between SQLite and PostgreSQL.** Rejected: a
  single supported storage engine keeps the test matrix and support burden
  small.

## Consequences

### Positive

- **Remote access if needed in the future.** PostgreSQL can be exposed over
  the network without changing the storage engine.
- **Heavy writing if needed in the future.** PostgreSQL handles concurrent
  writers without the single-writer limitation of SQLite.
- **More complex toolchain if needed in the future.** Extensions, replication
  and richer query features are available when we need them.
- **We can connect the system to an external PostgreSQL.** Same engine means
  swapping the bundled instance for a managed one is a config change, not a
  rewrite.
- **Subjective experience.** SQLite tends to cause problems as a project
  grows. Personally I (Rostislav) have had many issues with remote access,
  backups and multiple writers. SQLite is fine for the current scope, but
  choosing the heavier option up front avoids a forced migration later.

### Negative

- **Separate service to run and operate.** SQLite would have been a single
  file inside the app container; PostgreSQL is a second compose service with
  its own image, volume and lifecycle. Acceptable cost for the upside above.
- **More complicated to back up right now.** Backing up a live PostgreSQL
  instance is more involved than copying a single SQLite file.
- **Extra CPU and RAM usage.** Running a PostgreSQL process adds overhead,
  but it is small enough to be affordable on the target hosts.
