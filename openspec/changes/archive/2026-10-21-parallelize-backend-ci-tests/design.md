## Context

See `proposal.md` for the motivation and `specs/ci-pipeline/spec.md` for the required behavior.

`test-backend` waits for `lint-backend` (`.github/workflows/ci-release.yml:248`) and runs the whole backend suite in one step: `go test -p=8 -count=1 -failfast -timeout 40m ./internal/...` (`.github/workflows/ci-release.yml:352`). The comment above that command (`.github/workflows/ci-release.yml:350-351`) notes that 8 workers oversubscribe the runner's 4 vCPU. Before the tests, the job frees disk space, starts the compose fixtures, provisions and migrates 8 slot databases and pulls every test image (`.github/workflows/ci-release.yml:250-339`).

In the last green run of the full suite (run `38079240506`), `lint-backend` took 4:09 and `test-backend` took 13:45. Of those 13:45, setup took about 4:25 and `go test` took 9:16. The 88 packages with tests reported a combined 2635 seconds:

| Group | Packages with tests | Combined time | Longest package |
|---|---|---|---|
| `physical` (`internal/features/tests/physical/...`) | 2 | 514 s | `pg17`, 318 s |
| `logical` (`internal/features/tests/logical/...`) | 4 | 511 s | `postgresql`, 181 s |
| `other` | 82 | 1610 s | `backups/backups/usecases/physical/postgresql`, 366 s |

These times were measured under the CPU contention described above, so each package should run faster on a less loaded runner. That speed-up is an expectation, not a measurement.

The publishing jobs gate on `determine-version` (`.github/workflows/ci-release.yml:780`), which needs every test job but not `lint-backend`. The main image reaches the lint through `build-image` (`.github/workflows/ci-release.yml:620`). The verification image reaches it only through `test-backend`: `push-verification-image` needs `determine-version` and `build-verification-image` (`.github/workflows/ci-release.yml:982`), and `build-verification-image` needs only the agent lint and the Dockerfile scan (`.github/workflows/ci-release.yml:754`).

Constraints: the root `AGENTS.md` requires every new action to be pinned to a full commit SHA and keeps `permissions: contents: read` at the top level. This change adds no action and no permission. `backend/AGENTS.md` governs the new `backend/Justfile` recipe. ADR-0013 boots one shared container per database version inside a test process. Splitting the suite across runners keeps that model, and each runner holds fewer containers at once.

## Goals / Non-Goals

**Goals:**

- Remove the lint wait from the path to the backend test result.
- Spread the backend suite over three runners without a hand-maintained list of packages.
- Keep every publishing path gated on the backend lint and on all three groups.

**Non-Goals:**

- An even balance between the groups. The `other` group stays the longest; see Risks.
- Faster setup inside a group.

## Decisions

### 1. Gate the release on the lint directly instead of through the tests

`test-backend` drops `needs: [lint-backend]`, and `determine-version` adds `lint-backend` to its `needs`. Every publishing job already needs `determine-version`, and its `if:` conditions use the implicit `success()`, so a failed lint skips all of them. The lint always finishes before the tests, so the extra dependency adds no wall-clock time.

Alternatives rejected:

- Drop the dependency and change nothing else. The verification image would then be published for a commit whose backend lint failed.
- Add `lint-backend` to `push-verification-image` only. It closes today's gap, but the next publishing job that hangs off `determine-version` would reopen it.

### 2. Split by package path, with `other` as the catch-all

The `physical` group is `./internal/features/tests/physical/...`, and the `logical` group is `./internal/features/tests/logical/...`. The `other` group is `go list ./internal/...` without packages whose import path matches `/internal/features/tests/(physical|logical)(/|$)`. The filter excludes only these two subtrees, not all of `internal/features/tests/`. A new sibling directory under `tests/` therefore lands in `other` instead of being skipped.

Alternatives rejected:

- Split by database engine, with each group pulling only its own images. It saves pull time and disk space, but every new package would need a deliberate assignment, and the proposal leaves per-group images out of scope.
- Shard automatically, by hash or round-robin over `go list`. Nothing would need maintenance, but the balance would be accidental and would shift with every new package.
- Exclude all of `internal/features/tests/` from `other`. A new directory there would run in no group.

### 3. One recipe prints a group's packages

A recipe `test-group-packages <group>` in `backend/Justfile` prints the import paths of `physical`, `logical` or `other` and fails for any other name. The CI step passes its output to `go test`. A local check combines the three outputs and compares them with `go list ./internal/...`, which proves that the groups cover every package exactly once.

Alternative rejected: inline the `go list` and `grep` calls in the workflow step. The partition could then be checked only by a CI run, and a typo in a pattern would silently drop packages from the suite.

### 4. A matrix job with `fail-fast: false`

`test-backend` gets `strategy.matrix.group: [physical, logical, other]` and `strategy.fail-fast: false`. The group reaches the shell through an `env` entry rather than a `${{ }}` expression inside the script. Every leg keeps the current setup steps unchanged: 8 slot databases, `-p=8`, `-failfast` and the 40-minute `go test` timeout. GitHub names the checks `test-backend (physical)`, `test-backend (logical)` and `test-backend (other)`, and `needs: test-backend` waits for every leg.

Alternatives rejected:

- Three copies of the job. Each copy would repeat about 100 lines of setup that must then be kept in sync by hand.
- Keep the default `fail-fast: true`. A failing leg would cancel the others, so a rerun of the failed jobs would repeat the cancelled legs too.
- A reusable workflow or a composite action for the setup. The matrix already removes the duplication, and both options add a file and an indirection for a single caller.

## Risks / Trade-offs

- [The `other` group stays the long pole, with 1610 of the 2635 seconds and the longest single package] → The gain on the test step is smaller than an even split would give. By rough estimate, the step drops from 9:16 to about 6–7 minutes. The path to a test result drops from about 18 minutes to about 11–12, with most of the gain coming from the removed lint wait. A follow-up can move the heaviest packages of `other` into `physical` once real timings are in.
- [Each leg repeats the setup of about 4.5 minutes, including the "Free disk space" step, which took 5:40 in one run] → The legs run in parallel, so the duplicated setup costs runner minutes but no wall-clock time. Public repositories do not pay for hosted runners.
- [Three legs pull about 22 GB of images each, three times the registry traffic of today] → The pull already retries three times per image. A per-group image set is a follow-up.
- [A lint failure no longer saves the test runners] → Publishing still stops at the lint, through decision 1.
- [The check name changes from `test-backend` to three names] → No branch protection and no active ruleset requires the old name, so nothing has to be updated.

## Migration Plan

The change lands in one commit. To roll back, revert that commit.
