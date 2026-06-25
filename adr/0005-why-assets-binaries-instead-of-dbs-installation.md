# ADR-0005: Vendor PostgreSQL client binaries under `assets/` instead of installing them in the Dockerfile

- **Status:** Accepted
- **Date:** 2026-05-24
- **Tags:** packaging, ci, supply-chain, dockerfile

## Context

Postgresus needs the per-version client tools for every database engine it
supports — `pg_dump` / `pg_restore` / `pg_basebackup` for PostgreSQL,
`mysqldump` / `mysql` for MySQL and MariaDB, `mongodump` / `mongorestore` for
MongoDB. The original approach installed all of them at image-build time
straight from upstream apt repositories (PGDG, the MySQL/MariaDB apt repos,
MongoDB's apt repo), one `apt-get install` invocation per supported major.

This worked when only a couple of PostgreSQL versions were supported. Then
MySQL, MariaDB and MongoDB landed on top of PostgreSQL, and the supported
matrix grew to **18 engine/major combinations**:

- PostgreSQL: 12, 13, 14, 15, 16, 17, 18 (7 majors)
- MySQL: 5.7, 8, 9 (3 majors)
- MariaDB: 10, 11, 12 (3 majors)
- MongoDB: 4.2+, 5, 6, 7, 8 (5 majors)

At that size the build-time install model fell over:

- **CI time crossed 30 minutes** and kept climbing. Most of that time was
  spent re-downloading the same `.deb` packages on every build because the
  apt layers invalidated easily.
- **CI failed often** for reasons that had nothing to do with our code —
  PGDG / MySQL / MariaDB / MongoDB mirror timeouts, transient apt errors,
  DNS hiccups, GPG key churn. Each engine added its own apt repo, its own
  signing key and its own way to flake. Every external dependency in the
  build path is another way the build can fail.
- **The Dockerfile carried a lot of accidental complexity** — four sets of
  repo setup, four sets of key imports, version pinning per engine,
  multi-major installs and apt-cache cleanup.

We need to decide whether to keep installing clients from upstream apt
repositories at build time or vendor the binaries we actually use.

## Decision

We download the client binaries we need once across all supported engines,
commit them under `assets/` keyed by engine, major version and architecture
and `COPY` them into the image. The Dockerfile no longer talks to PGDG, the
MySQL / MariaDB repos, the MongoDB repo or any other external package
repository for database client tools.

## Consequences

### Positive

- **CI is dramatically faster and more stable.** The `COPY` of vendored
  binaries is a cached layer that costs near-zero on rebuilds. The build no
  longer downloads hundreds of megabytes of `.deb` packages on every run.
- **No external dependency in the build path.** PGDG outages, mirror
  flakiness and key-rotation events stop breaking our builds. Builds are
  reproducible from a clean checkout without network access to third-party
  repositories.
- **Smaller supply-chain attack surface.** The build no longer fetches and
  executes arbitrary versions of third-party packages resolved at build
  time. The exact bytes that ship in the image are the exact bytes
  committed to the repo and reviewable in a PR.
- **Smaller, simpler Dockerfile.** Repo setup, GPG key handling, pinning,
  multi-version installs and apt-cache cleanup all go away. What's left is
  a handful of `COPY` lines that say plainly what is in the image.
- **Version pinning is explicit and auditable.** Upgrading a client is a
  visible commit that swaps the binary under `assets/`, not an opaque
  `apt-get install` that resolves to whatever PGDG happens to serve that
  day.

## References

- [ADR-0001](./0001-use-docker-as-base.md) — Docker base image (the
  single-container environment whose Dockerfile this decision simplifies).
