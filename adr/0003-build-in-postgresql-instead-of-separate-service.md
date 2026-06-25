# ADR-0003: Embed PostgreSQL in the image instead of a separate service

- **Status:** Accepted
- **Date:** 2025-07-21
- **Tags:** storage, database, packaging, ops

## Context

[ADR-0002](0002-build-in-postgresql-instead-of-sqlite.md) chose PostgreSQL as
the internal datastore and shipped it as a second `docker-compose` service
next to the app. In practice the two-service layout turned out to be heavier
to operate than expected: users have to manage a compose file, two images,
two volumes and the dependency between them, and the project can no longer
be installed with a plain `docker run`. Moreover, users do not upgrade
their PostgreSQL with Postgresus upgrade.

We need to decide whether to keep the sibling service or fold PostgreSQL into
the app image.

## Decision

We embed PostgreSQL directly into the Postgresus image and run it as a
process inside the same container as the app. The install flow goes back to
a single `docker run` (or a one-service compose file).

## Alternatives considered

- **Keep PostgreSQL as a sibling compose service** (previous design).
- **Embed PostgreSQL in the app image** (chosen).

## Consequences

### Positive

- **Single-container install.** `docker run` is enough — no compose
  file required, no service ordering, no inter-service networking to
  explain.
- **Easier to manage.** One image, one volume, one lifecycle. Upgrades,
  backups of the install itself and rollbacks all act on a single unit.
- **More predictable environment.** The exact PostgreSQL version, config and
  data directory layout are pinned by the image — users can't accidentally
  point the app at an incompatible external instance or skew versions
  between the two services.

### Negative

- **No ability to connect to a remote PostgreSQL.** The internal datastore
  is now tied to the image; pointing Postgresus at a managed or external
  PostgreSQL for its own state is no longer supported. Backups still target
  arbitrary external PostgreSQL instances — this only affects Postgresus's
  own state store.
