## Purpose

Defines the commands a developer or an agent uses on a local Fedora host to run Databasus, run the backend suite and check a commit, and what each command stops or leaves running when it is interrupted.

## ADDED Requirements

### Requirement: One command starts the local app

The repository SHALL provide `just run` at its root. It SHALL start the development database, the backend and the frontend development server, and SHALL show them in one terminal window that lists every process separately, with each process's output available on its own. It SHALL work in a fresh worktree that has a `.env` file but has never generated the backend's API documentation or installed frontend dependencies. When a backend from an earlier run still holds the backend port, the command SHALL stop that process before starting the new backend, so that requests reach the code just started.

#### Scenario: Developer starts the app

- **WHEN** a developer runs `just run` on a Fedora host with a `.env` file
- **THEN** the development database, the backend and the frontend start
- **AND** the window lists each of them as a separate process with its own output

#### Scenario: Fresh worktree

- **WHEN** a developer runs `just run` in a worktree that has a `.env` file and nothing else prepared
- **THEN** the backend compiles and starts without a separate generation command
- **AND** the frontend development server starts without a separate dependency installation command

#### Scenario: Stale backend holds the port

- **WHEN** a backend from an earlier run still listens on the backend port
- **THEN** `just run` stops that process and the new backend serves requests on that port

### Requirement: Interrupting the local app stops what it started

When a developer interrupts `just run` with Ctrl+C, the backend and the frontend development server SHALL exit and the development database container SHALL stop. The database container SHALL be stopped rather than removed, so its data survives into the next run. Containers the command did not start, including the shared backend test fixtures, SHALL keep running.

#### Scenario: Developer presses Ctrl+C

- **WHEN** a developer presses Ctrl+C in the `just run` window
- **THEN** no backend or frontend process from that run remains
- **AND** the development database container is stopped

#### Scenario: Data survives a restart

- **WHEN** a developer interrupts `just run` and runs it again
- **THEN** the development database still holds the data written during the first run

#### Scenario: Test fixtures are running

- **WHEN** the shared backend test fixtures are running and a developer interrupts `just run`
- **THEN** the test fixtures are still running afterwards

### Requirement: One command runs the full backend suite

The repository SHALL provide `just test` at its root. It SHALL assume a Fedora host and, when `libedit` is installed on that host, SHALL make the bundled database clients startable without any further manual step. When `libedit` is missing, it SHALL stop before running tests and name the package to install. It SHALL start the shared test fixtures, pull the database engine and object storage images the backend tests reference, reset and migrate every per-worker test database, and then run every backend test package. It SHALL NOT run the Docker filesystem suite. Its exit status SHALL be non-zero when any step or any test fails.

#### Scenario: Developer runs the suite on Fedora

- **WHEN** a developer runs `just test` on a Fedora host with a `.env` file
- **THEN** every backend test package runs against freshly migrated per-worker databases
- **AND** tests that start a bundled database client do not fail on a missing shared library

#### Scenario: A test fails

- **WHEN** any backend test fails during `just test`
- **THEN** the command exits with a non-zero status

#### Scenario: Filesystem suite is not part of the run

- **WHEN** a developer runs `just test`
- **THEN** no Docker filesystem case runs and no candidate image is built

### Requirement: Interrupting the backend suite leaves no test processes or containers

When a developer interrupts `just test` with Ctrl+C, every test process SHALL exit, and every container on the Docker daemon that carries the test container label SHALL be removed, including containers another worktree's run started. The shared test fixtures SHALL keep running, so that the next run and other worktrees can use them.

#### Scenario: Developer presses Ctrl+C mid-suite

- **WHEN** a developer presses Ctrl+C while `just test` runs tests
- **THEN** no test process from that run remains
- **AND** no container with the test container label remains

#### Scenario: Shared fixtures survive the interrupt

- **WHEN** a developer interrupts `just test`
- **THEN** the shared test fixtures are still running afterwards

### Requirement: Test images can be pulled ahead of a run

The repository SHALL provide `just pull-testcontainers` at its root. It SHALL pull the database engine images (PostgreSQL, MySQL, MariaDB and MongoDB) and the object storage image that the backend tests reference, SHALL retry each failed pull a bounded number of times, and SHALL exit with a non-zero status when an image still cannot be pulled. Other test images, such as the FTP, SFTP, SMB, mail and Azure storage emulators, are pulled by the tests on first use.

#### Scenario: Pulling succeeds

- **WHEN** a developer runs `just pull-testcontainers` with registry access
- **THEN** every database engine and object storage image the backend tests reference is present locally afterwards

#### Scenario: An image cannot be pulled

- **WHEN** an image still fails to pull after its retries
- **THEN** the command names that image and exits with a non-zero status

### Requirement: Root commands lint and format every module

The repository SHALL provide `just lint` and `just format` at its root, covering the backend, the verification agent and the frontend. `just lint` SHALL exit with a non-zero status when any module fails its checks. `just format` SHALL rewrite files in all three modules to their formatter's output.

#### Scenario: One module fails lint

- **WHEN** the frontend has a lint error and the backend and the verification agent are clean
- **THEN** `just lint` exits with a non-zero status and reports the frontend error

#### Scenario: Developer formats everything

- **WHEN** a developer runs `just format`
- **THEN** a later `just lint` reports no formatting difference in any of the three modules

### Requirement: Tool versions are pinned in the repository

The repository SHALL pin the exact versions of a defined tool set, so that two hosts set up from the same commit run the same versions of those tools. The pinned set SHALL cover the backend and frontend toolchains, the command runner, the commit hook runner, the process window, the migration tool, the API documentation generator and the Go linter. Docker, the shell and system libraries such as `libedit` come from the host and are not pinned.

#### Scenario: New host setup

- **WHEN** a developer installs the pinned tools on a new Fedora host and runs `just lint`
- **THEN** every tool from the pinned set that the recipe calls resolves to its pinned version

### Requirement: Commits are checked before they are created

Creating a commit SHALL format the staged files of each module the commit touches and add the formatted result to the commit. It SHALL run that module's lint checks, and for the frontend its type check as well, and SHALL refuse the commit when any of them fails. A module the commit does not touch SHALL NOT be checked.

#### Scenario: Unformatted file is staged

- **WHEN** a developer commits a staged frontend file that is not formatted
- **THEN** the commit contains the formatted version of that file

#### Scenario: Lint error in a staged module

- **WHEN** a developer commits a backend change that fails the backend lint checks
- **THEN** no commit is created and the lint output is shown

#### Scenario: Untouched module

- **WHEN** a commit touches only the backend
- **THEN** no frontend or verification agent check runs
