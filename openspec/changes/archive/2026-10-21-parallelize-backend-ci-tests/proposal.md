## Why

A pull request into `main` waits about 18 minutes for its backend test result, and nearly all of that time sits on one sequential path. The backend tests start only after the backend lint finishes, which adds about 4 minutes, and then all 138 backend packages run on a single 4-vCPU runner for about 9 minutes. Lint almost never fails in CI, because the pre-commit hooks already run it locally. The suite is CPU-bound on one runner, while the repository is public and extra hosted runners cost nothing.

## Governing docs

This change answers to [`AGENTS.md`](../../../AGENTS.md), including its Security rules for workflows, and to [`backend/AGENTS.md`](../../../backend/AGENTS.md) for the backend test recipe it adds. It changes `.github/workflows/ci-release.yml` and `backend/Justfile`. It changes no Go code, migrations or tests.

## What Changes

- The backend test job no longer waits for the backend lint job. Both start together.
- The backend test job runs as three parallel groups on separate runners. The `physical` group holds the end-to-end physical backup tests, and the `logical` group holds the end-to-end logical backup tests. The `other` group holds every other backend package, including any new package that does not belong to the first two groups.
- One `backend/Justfile` recipe prints the package list of a group. CI uses it, and a local check uses it to prove that the three groups cover every package exactly once.
- A failure in one group no longer cancels the other two, so the developer can rerun only the failed group.
- Image publishing and the release still require a green backend lint and all three test groups. Today the verification image reaches the lint only through the backend tests, so the release gate gains the lint as a direct dependency.
- The backend test check appears in GitHub as three checks named `test-backend (physical)`, `test-backend (logical)` and `test-backend (other)` instead of one `test-backend`. No branch protection or active ruleset requires the old name.

## Capabilities

### New Capabilities

- `ci-pipeline`: the order of the backend checks in CI, how the backend suite is split across runners, and which checks gate image publishing and the release.

### Modified Capabilities

None.

## Impact

- `.github/workflows/ci-release.yml`: `test-backend` drops `needs: [lint-backend]` and becomes a matrix job with `fail-fast: false`. Its test step runs the package list of its group. `determine-version` keeps `needs: test-backend`, which now waits for every matrix leg, and adds `lint-backend` to its `needs`.
- `backend/Justfile`: a new recipe that prints the packages of one test group.
- Each pull request into `main` and each push to `main` uses two more hosted runners. Every group repeats the full setup of about 4.5 minutes and pulls the full set of test images.
- The real wall-clock gain can only be measured by a CI run after merge.

## Out of scope

- Flaky backend tests. They are being fixed separately.
- Skipping lint and tests on pushes to `develop`. That is deliberate and stays.
- Pulling a smaller image set per group and dropping or shortening the "Free disk space" step.
- Speeding up the lint job itself, for example by caching `golangci-lint`.
- Rebalancing the groups beyond the three agreed paths, for example by moving large packages from `other` into `physical`.
- Adding `actionlint` or any other tool to `mise.toml` or to CI.
- Changing `scripts/test.sh` or `just test`. The local full suite still runs every package in one process.
