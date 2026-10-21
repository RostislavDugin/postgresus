## ADDED Requirements

### Requirement: The agent can see and drive the local app

The repository SHALL register one browser automation server for both Claude Code and Codex. With it, an agent opens pages of the local app and reads what the page shows, including a screenshot, the page structure, console messages and network requests. It also clicks, types and navigates. Both clients SHALL start the server from the same repository configuration. The server SHALL drive the Chrome browser installed on the host. Each agent session SHALL get its own temporary browser profile, which is discarded when the session's browser closes, so that two sessions running at once do not block each other and no sign-in carries over between sessions.

The repository SHALL hold a skill that tells the agent how to check the app in the browser:

- When the backend and frontend ports already answer from a `just run` started in the agent's own worktree, the agent uses that app.
- When they answer from another worktree, the agent leaves that app running, does not run the browser check, and tells the user the check was skipped because the running app serves another worktree's code.
- When only one of the two ports answers, the agent does not start `just run` and does not run the browser check. It tells the user which port is held and that the check was skipped.
- When neither port answers, the agent starts `just run` without its interactive window and stops it before it finishes the task, leaving neither port held.
- The agent signs in with the seeded dev account.
- When the app of the agent's own worktree lacks the dev account or a seeded database, the agent reruns the seed command against it instead of creating the items by hand.

A browser check SHALL add to the tests a task names and SHALL NOT replace them.

#### Scenario: Agent checks a page with no app running

- **WHEN** an agent needs to see a page and nothing answers on the backend and frontend ports
- **THEN** the agent starts `just run` without its interactive window, signs in as the dev account, opens the page and reads what it shows
- **AND** no process of that run remains when the agent hands the task over

#### Scenario: The developer's app runs in the same worktree

- **WHEN** an agent needs to see a page and the running app was started in the agent's own worktree
- **THEN** the agent uses that app and leaves it running when it hands the task over

#### Scenario: The running app belongs to another worktree

- **WHEN** an agent needs to see a page and the running app was started in another worktree
- **THEN** the agent does not stop that app and does not open the page
- **AND** the agent tells the user that the browser check was skipped because the app serves another worktree's code

#### Scenario: Only one port answers

- **WHEN** an agent needs to see a page, the frontend port answers and the backend port does not
- **THEN** the agent does not start `just run` and does not open the page
- **AND** the agent tells the user that the frontend port is held and the browser check was skipped

#### Scenario: A seeded database is missing

- **WHEN** an agent uses the app of its own worktree and the WAL-streaming database is missing from it
- **THEN** the agent reruns the seed command, and the database appears with backups switched on

#### Scenario: Two agents use the browser at once

- **WHEN** two agent sessions open the browser at the same time
- **THEN** both browsers start, each with its own empty profile

#### Scenario: Codex opens a page

- **WHEN** Codex is started in the repository
- **THEN** it lists the same browser server Claude Code uses, at the same pinned version

### Requirement: The developer chooses a visible or headless browser

The browser the agent drives SHALL run headless unless the developer asks otherwise. The `BROWSER_HEADLESS` setting in the developer's `.env` file SHALL choose headless when `true` and a visible window when `false`. A value set in the environment the agent client starts from SHALL take precedence over the file. When neither sets it, the browser SHALL run headless. Any value other than `true` or `false` SHALL keep the browser server from starting, with an error that names `BROWSER_HEADLESS`. The example environment file SHALL contain the setting with its default value. The choice SHALL take effect the next time the agent client starts the browser server.

#### Scenario: Default setting

- **WHEN** neither the developer's `.env` file nor the environment sets `BROWSER_HEADLESS` and an agent opens a page
- **THEN** no browser window appears on the developer's screen

#### Scenario: Developer wants to watch

- **WHEN** the developer sets `BROWSER_HEADLESS=false` in `.env` and the agent client restarts the browser server
- **THEN** the agent's browser opens as a visible window

#### Scenario: Environment overrides the file

- **WHEN** `.env` sets `BROWSER_HEADLESS=true` and the agent client is started with `BROWSER_HEADLESS=false` in its environment
- **THEN** the browser opens as a visible window

#### Scenario: Invalid value

- **WHEN** `.env` sets `BROWSER_HEADLESS=maybe` and the agent client starts the browser server
- **THEN** the server does not start
- **AND** the error the client reports names `BROWSER_HEADLESS`
