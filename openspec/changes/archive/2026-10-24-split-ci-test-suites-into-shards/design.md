## Context

See proposal.md, Why. The backend groups come from `test-group-packages` in `backend/Justfile:23`, and the CI test step passes its output to `go test -p=8` (`.github/workflows/ci-release.yml:423`). Sharding adds `test-group-tests` (`backend/Justfile:42`), which prints a `-run` pattern for the same step. The filesystem cases are grouped in `e2e/docker-storage/run.sh:1289`, and CI builds the candidate in `build-storage-image` (`.github/workflows/ci-release.yml:429`) for the `test-filesystems` matrix (`.github/workflows/ci-release.yml:491`).

The approach follows the root `AGENTS.md`, including its Security rules for workflows: every action stays pinned to a full SHA, and no new secret or permission is added. It also follows `backend/AGENTS.md` for the test recipes. The tests themselves do not change, so the shared test database rules of ADR-0013 keep applying to each shard as they did to each group.

## Goals / Non-Goals

**Goals:**

- No backend check is bounded by a whole slow package any more.
- Coverage stays provable locally: every package in one group, and every top-level test of a sharded group in one shard.

**Non-Goals:**

- Balancing shards exactly. A shard can be uneven when the weights are stale.

## Decisions

### Shard by top-level test name inside the group's packages

A shard runs its group's packages with `-run` limited to its share of the top-level test names. Names that are not in a package match nothing there, so one pattern serves several packages, for example `pg17` and `pg18`, which share most names.

Rejected alternatives:
- More whole-package groups. A group cannot be faster than its slowest package, and that package took most of the group's time.
- `t.Parallel()` inside the slow packages. Their tests share physical PostgreSQL instances and replication slots, so each test would need its isolation checked one by one.
- `go test -shuffle` or test sharding by a test binary flag. Go has no built-in sharding flag, and shuffling changes order, not the set of tests per runner.

### Read test names from the sources, not from `go test -list`

`go test -list` runs the test binary, and these packages' `TestMain` starts their database containers before listing anything. A grep for top-level `func Test`, `func Example` and `func Fuzz` in `_test.go` files needs no build and no Docker. `TestMain` is excluded by name.

Rejected alternative: `go test -list` with a dummy environment. It still compiles each package and depends on `TestMain` failing harmlessly.

### Deal the slowest tests first by recorded weights

Tests are sorted by their weight in `backend/test-shard-weights.txt`, heaviest first, then by name, and each goes to the least-loaded shard. A test missing from the file weighs one second. Measured locally, round-robin by sorted name put the two slowest WAL tests in the same shard, which gave shard loads of 58, 79 and 298 s. The weighted deal gave 164, 137 and 134 s. For `physical` it gave 109, 105 and 105 s instead of 89, 170 and 61 s.

Rejected alternatives:
- Round-robin or hashing by name. Five tests carry most of the time, so the result depends on where their names happen to sort.
- Weights pulled from CI timings at run time. That needs a store for the timings and adds a moving part to every run. A hand-refreshed file is enough while a handful of tests dominate.

### Run the small groups on 4 vCPU runners

Only `other` runs many packages at once. `logical` runs four packages, and every shard runs one or two. On 8 vCPU runners the eight legs and the lint jobs together exceeded the Namespace concurrency limit and queued. Measured Go test times in the earlier benchmark were about the same on 4 and 8 vCPU for single-package work.

Rejected alternative: every leg on 8 vCPU. It holds twice the share of the concurrency limit for no measured gain.

### Build the filesystem candidate once and pass it as an artifact

`build-storage-image` builds the amd64 candidate on the Remote Builder and uploads it as a compressed `docker save` archive with one-day retention. Each case group downloads it and loads it locally. Uploads and downloads retry up to three times with a per-attempt timeout.

Rejected alternatives:
- Each group builds its own candidate. Four builds instead of one, and the groups would test four builds rather than one.
- Pushing the candidate to a registry. It needs registry credentials in pull request jobs and publishes an untested image.

## Risks / Trade-offs

- [Weights go stale as tests change] → A stale weight only unbalances shards. Refresh the file from a local `go test -json` run when a shard becomes the slowest check.
- [The grep misses a top-level test] → That test would run in no shard. Go runs only functions named `Test…`, `Example…` and `Fuzz…`, and the formatter the backend lint enforces puts every top-level `func` at the start of a line, which is the form the grep matches.
- [More checks per run] → Each run uses five more backend runners and three more filesystem runners, and each repeats its setup. On Namespace that setup is under a minute.
