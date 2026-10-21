## MODIFIED Requirements

### Requirement: One command starts the local app

The repository SHALL provide `just run` at its root. It SHALL start the development database, the shared backend test fixtures, the live WAL backup source with its load generator, the backend and the frontend development server, and SHALL show the database, the load generator, the backend, the frontend and the seed in one terminal window that lists every process separately, with each process's output available on its own. It SHALL work in a fresh worktree that has a `.env` file but has never generated the backend's API documentation or installed frontend dependencies. When a backend from an earlier run still holds the backend port, the command SHALL stop that process before starting the new backend, so that requests reach the code just started. It SHALL also stop a backend file watcher left over from an earlier run in the same worktree, so that no old watcher rebuilds and restarts a backend after the new run starts. A saved change to a frontend source file SHALL reach the open page without restarting any process.

#### Scenario: Developer starts the app

- **WHEN** a developer runs `just run` on a Fedora host with a `.env` file
- **THEN** the development database, the shared test fixtures, the WAL backup source, its load generator, the backend and the frontend start
- **AND** the window lists the database, the load generator, the backend, the frontend and the seed as separate processes with their own output

#### Scenario: Fresh worktree

- **WHEN** a developer runs `just run` in a worktree that has a `.env` file and nothing else prepared
- **THEN** the backend compiles and starts without a separate generation command
- **AND** the frontend development server starts without a separate dependency installation command

#### Scenario: Stale backend holds the port

- **WHEN** a backend from an earlier run still listens on the backend port
- **THEN** `just run` stops that process and the new backend serves requests on that port

#### Scenario: Watcher from a killed run survives

- **WHEN** an earlier `just run` in the same worktree was killed without cleanup and its backend file watcher is still running
- **THEN** `just run` stops that watcher before starting its own
- **AND** a later saved Go file restarts only the backend of the new run

#### Scenario: Frontend file is saved

- **WHEN** a developer saves a change to a frontend source file while `just run` is running
- **THEN** the open page shows the change without any process being restarted

### Requirement: Tool versions are pinned in the repository

The repository SHALL pin the exact versions of a defined tool set, so that two hosts set up from the same commit run the same versions of those tools. The pinned set SHALL cover the backend and frontend toolchains, the command runner, the commit hook runner, the process window, the migration tool, the API documentation generator, the Go linter, the backend file watcher and the browser automation server agents use. Docker, the shell, the Chrome browser and system libraries such as `libedit` come from the host and are not pinned.

#### Scenario: New host setup

- **WHEN** a developer installs the pinned tools on a new Fedora host and runs `just lint`
- **THEN** every tool from the pinned set that the recipe calls resolves to its pinned version

#### Scenario: Browser server starts at its pinned version

- **WHEN** an agent client starts the browser automation server registered in the repository
- **THEN** the server runs at the version pinned in the repository, without downloading a newer one

## ADDED Requirements

### Requirement: The backend restarts when its source changes

While `just run` is running, saving a change to a backend Go source file SHALL rebuild the backend and replace the running backend with the new build, without the developer touching the process window. A change to a test file alone SHALL NOT restart the backend. When the new code fails to compile, the backend that was already running SHALL keep serving requests, and the compile error SHALL appear in the backend's output in the process window. Interrupting `just run` SHALL leave no backend process from that run, including one started by a restart.

#### Scenario: Developer saves a backend change

- **WHEN** a developer changes the response of an endpoint and saves the Go file while `just run` is running
- **THEN** a request to that endpoint returns the new response once the rebuild finishes, without any manual restart

#### Scenario: The change does not compile

- **WHEN** a developer saves a Go file that does not compile
- **THEN** the backend output shows the compile error
- **AND** the previous backend still answers requests on the backend port

#### Scenario: Only a test file changes

- **WHEN** a developer saves a change to a Go test file and nothing else
- **THEN** the backend is not restarted

#### Scenario: Interrupt after restarts

- **WHEN** the backend has restarted several times and the developer presses Ctrl+C in the `just run` window
- **THEN** no backend process from that run remains

### Requirement: The local app is seeded with a dev account and databases

When the backend started by `just run` begins answering requests, the seed SHALL make sure the local app contains:

- the dev account `test@email.com` with password `Q1234567`;
- a workspace owned by that account;
- a local storage in that workspace;
- a PostgreSQL database with logical backups of the shared logical test fixture;
- a PostgreSQL database with WAL-streaming physical backups of the live WAL backup source.

The seed SHALL go through the same public API and validation as the web interface. It SHALL find each item by its name, SHALL create only the items that are missing and SHALL NOT change an item that already exists. When the seed creates a database, it SHALL switch its backups on with the local storage and SHALL start the first backup. The logical database SHALL keep its 3 newest backups, and the WAL-streaming database SHALL keep its 2 newest backup chains.

The seed command, `go run ./cmd/seed_dev` in the backend directory, SHALL also run on its own against the app already answering on the backend port, with the same behavior. The repository SHALL NOT provide a separate `just` recipe for it. When nothing answers on the backend port, the seed command SHALL exit with a non-zero status and say that the app has to be started with `just run`. When the backend answers its health check with an error status, for example because disk usage is too high, the seed SHALL exit with a non-zero status and show the reason the backend gave, and SHALL create nothing.

When the seed cannot sign in as the dev account and cannot create it, for example because the account exists with a different password or registration is closed, it SHALL exit with a non-zero status and a message that names the account and the reason the backend gave, and SHALL create nothing. When the instance requires two-factor sign-in, the seed SHALL exit with a non-zero status and a message that names the account and says that two-factor sign-in is required, and SHALL create nothing. When the backend refuses to create a seeded item after sign-in, for example because the dev account is a member that may not create workspaces, the seed SHALL exit with a non-zero status, show the name of that item and the reason the backend gave, and SHALL NOT change any item that already exists. A seed that fails SHALL NOT stop the other processes of `just run`.

#### Scenario: First run on an empty app

- **WHEN** a developer runs `just run` against a development database that has no users
- **THEN** after the backend starts, signing in as `test@email.com` with `Q1234567` succeeds
- **AND** the account's workspace lists the local storage and both databases, each with backups switched on and a first backup started
- **AND** the logical database's backup settings keep 3 backups, and the WAL-streaming database's keep 2 chains

#### Scenario: Second run

- **WHEN** a developer runs `just run` again with the seeded items already present
- **THEN** no item is created a second time
- **AND** no existing item changes, including one the developer edited after the first run

#### Scenario: One seeded database was deleted

- **WHEN** a developer deleted the WAL-streaming database and runs the seed command on its own
- **THEN** the seed creates that database again with backups switched on
- **AND** the other items stay unchanged

#### Scenario: Dev account has another password

- **WHEN** `test@email.com` exists with a password other than `Q1234567` and the seed runs
- **THEN** the seed exits with a non-zero status and a message naming the account and the failed sign-in
- **AND** no workspace, storage or database was created
- **AND** the account's password and role are unchanged
- **AND** the backend and the frontend keep running

#### Scenario: Backend refuses to create the workspace

- **WHEN** the dev account is a member, members may not create workspaces, and the seed runs
- **THEN** the seed exits with a non-zero status and a message naming the workspace and the backend's reason
- **AND** no existing item changed
- **AND** the backend and the frontend keep running

#### Scenario: Two-factor sign-in is required

- **WHEN** the instance requires two-factor sign-in and the seed runs
- **THEN** the seed exits with a non-zero status and a message naming the account and saying that two-factor sign-in is required
- **AND** no workspace, storage or database was created

#### Scenario: Backend reports itself unhealthy

- **WHEN** the backend answers its health check with an error status and the reason "disk usage is above 95%", and the seed command runs on its own
- **THEN** the seed exits with a non-zero status and shows that reason
- **AND** nothing is created

#### Scenario: App is not running

- **WHEN** a developer runs the seed command on its own while no backend answers on the backend port
- **THEN** the command exits with a non-zero status and tells the developer to start the app with `just run`

### Requirement: Seed credentials exist only in local development

The dev account, its password and the seeded databases SHALL be created only by `just run` and by the seed command run on its own in a development checkout. The production image and the production backend SHALL neither contain the seed nor run it, whatever the environment variables, and SHALL create no account with known credentials. The seed SHALL NOT change the password, the role or the settings of any existing account, including the dev account.

#### Scenario: Production image starts

- **WHEN** the production image starts with an empty data directory
- **THEN** no account exists until someone signs up, and `test@email.com` cannot sign in

#### Scenario: Another account exists

- **WHEN** the development database holds an account other than the dev account and the seed runs
- **THEN** that account's password, role and settings are unchanged afterwards
