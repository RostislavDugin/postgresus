## Why

On Namespace runners a backend test group starts in under a minute, so the remaining CI time is bounded by single slow packages. The `other` group took about 400 s of test time, and 359 s of it was one package, `backups/usecases/physical/postgresql`. The `physical` group took about 300 s, and that was its `pg17` package. A few tests dominate both: one of 150 s and one of 93 s in the WAL usecases package, and one of 109 s in `pg17`. Spreading packages across more groups cannot get below such a package, so the groups have to split the tests inside it. The Docker filesystem suite had the same shape. One job built the candidate image and then ran 23 cases one after another.

## Governing docs

This change answers to [`AGENTS.md`](../../../AGENTS.md), including its Security rules for workflows, and to [`backend/AGENTS.md`](../../../backend/AGENTS.md) for the backend test recipes. It changes `.github/workflows/ci-release.yml`, `backend/Justfile`, the new `backend/test-shard-weights.txt` and `e2e/docker-storage/run.sh`. It changes no Go code, migrations or tests.

## What Changes

- The backend suite runs as eight checks instead of three. `logical` and `other` stay whole. `physical` runs as three shards, and the WAL usecases package `internal/features/backups/backups/usecases/physical/postgresql` leaves `other` for its own group, `wal-usecases`, which also runs as three shards.
- A shard group is named `<group>-<shard>-of-<count>`. A new `test-group-tests` recipe in `backend/Justfile` prints the `-run` pattern of one shard. It reads the top-level test names from the test sources, then deals the slowest first, each to the least-loaded shard by the weights in `backend/test-shard-weights.txt`. Every test lands in exactly one shard. A test missing from the weights file counts as one second. A stale weight makes a shard slower but never skips a test.
- `test-group-packages` accepts the shard names and the new `wal-usecases` group, and `other` drops the WAL usecases package.
- **BREAKING** for anything that matches check names: the backend checks are now `test-backend (other)`, `test-backend (logical)`, `test-backend (physical-1-of-3)` through `(physical-3-of-3)` and `test-backend (wal-usecases-1-of-3)` through `(wal-usecases-3-of-3)`. No branch protection or active ruleset requires the old names.
- The Docker filesystem suite runs as four parallel checks, one per case group: `permissions`, `network`, `upgrades` and `lifecycle`. A separate `build-storage-image` job builds the amd64 candidate once and hands it to the four checks as a workflow artifact, so no check builds its own. `run.sh` accepts `CASE_GROUP` to run one group, and `CASE=<case>` still runs a single case.

## Capabilities

### New Capabilities

None.

### Modified Capabilities

- `ci-pipeline`: the backend suite is split into whole groups and sharded groups instead of exactly three groups, and the check names, coverage rule and release gate follow.
- `docker-storage-compatibility`: CI runs the filesystem suite as parallel case groups from one candidate image that is built once and transferred between jobs as an artifact, instead of one job that builds its own candidate and never transfers it.

## Impact

- `.github/workflows/ci-release.yml`: `test-backend` becomes a matrix of eight legs. `other` runs on an 8 vCPU runner, and the seven legs that run one to four packages run on 4 vCPU runners. The test step passes the shard's `-run` pattern to `go test`. `test-filesystems` becomes a matrix of four case groups behind the new `build-storage-image` job.
- `backend/Justfile`: `test-group-packages` changes, and `test-group-tests` is new.
- `backend/test-shard-weights.txt`: a new file of test durations from one local run.
- `e2e/docker-storage/run.sh`: case groups and `CASE_GROUP`.
- Each CI run uses five more backend runners and four filesystem runners instead of one.

## Out of scope

- Making the slow tests themselves faster, and `t.Parallel()` inside the sharded packages.
- Refreshing the weights file automatically from CI timings.
- Sharding `logical` or `other`.
- The choice of runners and caches on Namespace, which the specs do not describe.
