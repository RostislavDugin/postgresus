# ADR-0013: Reuse one in-memory test database per version instead of one container per test

- **Status:** Accepted
- **Date:** 2026-06-08
- **Deciders:** Backend team
- **Tags:** backend, testing, ci

## Context

Our database tests run the same checks against many versions of each engine (PostgreSQL, MySQL,
MariaDB, MongoDB). We tried two setups before this one and both broke.

**Before — one big docker-compose.** We started every version of every engine up front and kept them
all running. That is ~50 containers alive at the same time and a 16 GB CI machine ran out of memory:

```
docker-compose up   (everything, all the time)
  postgres:12 .. postgres:18
  mysql:5.7 .. mysql:8.4
  mariadb:10.6 .. mariadb:12.0
  mongo:4.0 .. mongo:8.0
  ──────────────────────────────
  ~50 containers alive at once  →  out of RAM
```

**Naive fix — a fresh container per test.** Every single test booted its own database and killed it
afterwards. A database cold-starts in 20–60s and one package has 40+ tests, so the suite hit the
15-minute timeout:

```
test 1 → boot DB → run → kill DB
test 2 → boot DB → run → kill DB
...        (40+ times per package)        →  too slow
```

We want both: fast tests and bounded memory, without tests stepping on each other.

## Decision

For each database version we boot one container, run all the test functions against it, then shut it
down before the next version:

```
boot postgres:16   (once)
  ├─ run test 1
  ├─ run test 2
  └─ run test 3
shut down postgres:16
→ boot postgres:17 ...
```

Two more things make this fast and safe: the database files live in **RAM** (so each boot and restore
is quick) and the test packages run **in parallel** (so we get the speed back) while each one stays
isolated (so peak memory stays bounded).

- **One container per version, reused across the version's tests.** Each matrix package declares its
  version list once and loops it with an outer `t.Run`. Inside that subtest it boots the server with
  `StartPostgres` / `StartMysql` / … (`containers/{postgres,mysql,mariadb,mongodb}.go`), then runs the
  former test functions as inner subtests. `StartXxx` registers `t.Cleanup`, which Go runs when the
  version subtest returns — so the container is torn down before the next version boots and **only one
  matrix container is alive per package at a time**. Each `go test` package is its own process, so its
  containers belong only to it. The one rule for test authors: when you create a fixed-name object (a
  table or user without a random suffix), start with `DROP ... IF EXISTS` — within a version the server
  is reused and the subtests run one after another.

- **Database files in RAM.** Each engine's data dir is mounted on tmpfs `rw,size=512m` (size is pinned —
  otherwise Docker reserves half the host RAM per container). Crash-safety is off because we throw these
  servers away: Postgres `fsync=off / full_page_writes=off / synchronous_commit=off`; MySQL & MariaDB
  `innodb-flush-log-at-trx-commit=0 / innodb-doublewrite=0 / sync-binlog=0 / skip-log-bin`. Files:
  `containers/{postgres,mysql,mariadb,mongodb}.go`.

- **Run packages in parallel.** `go test -p=8` (`TEST_PARALLEL_WORKERS` in `backend/Makefile`). One
  matrix container is alive per package, so peak memory stays around 10–11 GB instead of the old
  all-at-once that ran out of RAM.

### How parallel workers stay isolated

Eight packages run at the same time and share the same Postgres, Valkey and backup infrastructure, so
each one grabs its own **worker slot** using a Postgres advisory lock
(`config.go` → `applyTestWorkerSlot` / `claimTestWorkerSlot`). The slot gives it a private copy of
everything shared:

- its own **metadata database**, named `…_w{slot}`;
- its own **Valkey/Redis database** (the slot number);
- its own **cache prefix** `"w{slot}:"`, which also tags the **backup-node registry**
  (`backups/backups/backuping/nodes/registry.go`) — every Redis key and pub/sub channel.

So two packages running side by side never touch each other's databases, cache entries, backup nodes or
pub/sub channels.

## Alternatives considered

- **docker-compose, all versions up at once.** Rejected: ~50 containers exhaust RAM on 16 GB CI and the
  packages share one set of databases with no isolation.
- **A fresh container per test.** Rejected: 40+ boots per package × 20–60s cold start → 15-minute
  timeouts.
- **Database files on disk instead of RAM.** Rejected: the slow part is fsync on cold start and on the
  write-heavy restore; RAM removes it for servers we throw away anyway.

## Consequences

### Positive

- The two packages that used to time out at 900s now finish in ~30s; the whole suite went from timing
  out to ~1.5–4 minutes.
- Peak memory is bounded and predictable — one matrix container per package — and parallel packages are
  fully isolated.

### Negative

- Because a server is reused across a version's subtests, authors must `DROP ... IF EXISTS` for
  fixed-name objects and must not run those subtests with `t.Parallel`.
- Data in RAM is volatile — fine for throwaway test servers, but there is no crash safety.
- If a CI runner is short on memory, lower `TEST_PARALLEL_WORKERS` (e.g. to 6).

### Neutral / follow-ups

- On a hard `-timeout` or `os.Exit`, `t.Cleanup` is skipped; the Makefile/CI label-sweep cleans up any
  leftover containers.

## References

- `backend/internal/util/testing/containers/{postgres,mysql,mariadb,mongodb}.go` (the `StartXxx` helpers)
- `backend/internal/features/tests/logical/postgresql/backup_restore_test.go` — the per-version
  orchestrator pattern
- `backend/internal/features/tests/physical/postgresql/{pg17,pg18}` — the physical backup→restore
  E2E split one package per PostgreSQL major, so the two versions run as isolated, parallel test
  binaries (each with its own control plane and throwaway source + restore-target containers)
- `backend/internal/config/config.go` (`applyTestWorkerSlot` / `claimTestWorkerSlot`)
- `backend/internal/features/backups/backups/backuping/nodes/registry.go`
- `backend/Makefile`
