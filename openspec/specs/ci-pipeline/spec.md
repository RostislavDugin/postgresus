# CI Pipeline Specification

## Purpose

Defines when the backend checks run in CI, how the backend test suite is split across runners, and which checks must pass before an image is published or a release is made.

## Requirements

### Requirement: Backend tests start without waiting for the backend lint

When CI runs the backend lint and the backend tests for a commit, the backend tests SHALL start without waiting for the lint result. A failed lint SHALL NOT cancel or skip the backend tests.

#### Scenario: Lint and tests start together

- **WHEN** CI starts the backend checks for a pull request
- **THEN** the backend lint check and every backend test check start without one waiting for the other

#### Scenario: Lint fails

- **WHEN** the backend lint check fails while the backend tests run
- **THEN** the backend test checks run to completion and report their own results

### Requirement: The backend suite runs as whole and sharded groups that cover every test once

CI SHALL run the backend test suite as separate checks on separate runners, and the checks SHALL run at the same time. The `physical` group SHALL run the end-to-end physical backup tests, and the `logical` group SHALL run the end-to-end logical backup tests. The `wal-usecases` group SHALL run the package of the physical backup and WAL streaming use cases. The `other` group SHALL run every remaining backend package. Each backend package SHALL belong to exactly one group. A backend package added outside these named groups SHALL run in the `other` group without any change to the CI configuration.

A group MAY run as several shards. Each shard SHALL run the tests of its group's packages that fall into that shard, and every top-level test of those packages SHALL run in exactly one shard. A test added to a sharded group SHALL run in one of its shards without any change to the CI configuration. How tests are assigned to shards MAY use recorded test durations, and a missing or outdated duration SHALL only change which shard runs a test, never whether it runs. `physical` and `wal-usecases` SHALL run as three shards each, and `logical` and `other` SHALL each run whole.

#### Scenario: The groups cover the suite

- **WHEN** the package lists of the groups are combined
- **THEN** they contain every backend package exactly once

#### Scenario: The shards cover their group

- **WHEN** the tests of all shards of one group are combined
- **THEN** they contain every top-level test of that group's packages exactly once

#### Scenario: A new package is added

- **WHEN** a developer adds a backend package outside the named groups
- **THEN** CI runs its tests in the `other` group

#### Scenario: A new test is added to a sharded group

- **WHEN** a developer adds a top-level test to a package of a sharded group and records no duration for it
- **THEN** CI runs that test in exactly one shard of the group

#### Scenario: Checks are reported per group

- **WHEN** CI runs the backend tests
- **THEN** each whole group and each shard reports its own check, named `test-backend (other)`, `test-backend (logical)`, `test-backend (physical-1-of-3)` through `test-backend (physical-3-of-3)` and `test-backend (wal-usecases-1-of-3)` through `test-backend (wal-usecases-3-of-3)`

### Requirement: A failed group neither cancels nor hides the others

When one backend test group fails, the other groups SHALL run to completion and report their own results. The developer SHALL be able to rerun only the failed group.

#### Scenario: One group fails

- **WHEN** a test in the `other` group fails while the `physical` and `logical` groups still run
- **THEN** the `physical` and `logical` groups keep running and report their own results

#### Scenario: Rerun of the failed group

- **WHEN** the developer reruns the failed jobs of that workflow run
- **THEN** only the failed group runs again, and the groups that passed keep their results

### Requirement: Publishing and release require the lint and every test group

CI SHALL publish an image or create a release only when the backend lint and every backend test check, each whole group and each shard, have passed for that commit. Running the lint and the tests in parallel SHALL NOT let a commit with a failed lint, a failed group or a failed shard reach publishing or release.

#### Scenario: Lint fails on main

- **WHEN** the backend lint fails on a push to `main` while every backend test check passes
- **THEN** CI publishes no image and creates no release for that commit

#### Scenario: One group fails on main

- **WHEN** the lint and every other backend test check pass on a push to `main` but one shard fails
- **THEN** CI publishes no image and creates no release for that commit
