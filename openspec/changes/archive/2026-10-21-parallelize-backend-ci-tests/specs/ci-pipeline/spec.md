## Purpose

Defines when the backend checks run in CI, how the backend test suite is split across runners, and which checks must pass before an image is published or a release is made.

## ADDED Requirements

### Requirement: Backend tests start without waiting for the backend lint

When CI runs the backend lint and the backend tests for a commit, the backend tests SHALL start without waiting for the lint result. A failed lint SHALL NOT cancel or skip the backend tests.

#### Scenario: Lint and tests start together

- **WHEN** CI starts the backend checks for a pull request
- **THEN** the backend lint check and every backend test check start without one waiting for the other

#### Scenario: Lint fails

- **WHEN** the backend lint check fails while the backend tests run
- **THEN** the backend test checks run to completion and report their own results

### Requirement: The backend suite runs as three groups that cover every package once

CI SHALL run the backend test suite as three checks on separate runners, and the checks SHALL run at the same time. The `physical` group SHALL run the end-to-end physical backup tests, and the `logical` group SHALL run the end-to-end logical backup tests. The `other` group SHALL run every remaining backend package. Each backend package SHALL run in exactly one group. A backend package added outside the end-to-end physical and logical backup tests SHALL run in the `other` group without any change to the CI configuration.

#### Scenario: The groups cover the suite

- **WHEN** the package lists of the three groups are combined
- **THEN** they contain every backend package exactly once

#### Scenario: A new package is added

- **WHEN** a developer adds a backend package outside the end-to-end physical and logical backup tests
- **THEN** CI runs its tests in the `other` group

#### Scenario: Checks are reported per group

- **WHEN** CI runs the backend tests
- **THEN** each group reports its own check, named `test-backend (physical)`, `test-backend (logical)` and `test-backend (other)`

### Requirement: A failed group neither cancels nor hides the others

When one backend test group fails, the other groups SHALL run to completion and report their own results. The developer SHALL be able to rerun only the failed group.

#### Scenario: One group fails

- **WHEN** a test in the `other` group fails while the `physical` and `logical` groups still run
- **THEN** the `physical` and `logical` groups keep running and report their own results

#### Scenario: Rerun of the failed group

- **WHEN** the developer reruns the failed jobs of that workflow run
- **THEN** only the failed group runs again, and the groups that passed keep their results

### Requirement: Publishing and release require the lint and every test group

CI SHALL publish an image or create a release only when the backend lint and all three backend test groups have passed for that commit. Running the lint and the tests in parallel SHALL NOT let a commit with a failed lint or a failed group reach publishing or release.

#### Scenario: Lint fails on main

- **WHEN** the backend lint fails on a push to `main` while all three test groups pass
- **THEN** CI publishes no image and creates no release for that commit

#### Scenario: One group fails on main

- **WHEN** the lint and two test groups pass on a push to `main` but the third group fails
- **THEN** CI publishes no image and creates no release for that commit
