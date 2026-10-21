## Context

See `proposal.md` for the motivation and `specs/` for the behavior. This section covers only the current state that shapes the approach.

- `just run` calls `scripts/dev.sh`. That script starts the dev containers and runs process-compose with `--no-server` (`scripts/dev.sh:49`). Nothing outside the TUI can restart a process.
- The backend process is a plain `go run ./cmd` (`process-compose.yaml:9-11`), so it never sees a saved edit.
- The frontend is the Vite dev server (`process-compose.yaml:13-14`), which already updates the page on its own.
- The backend applies migrations on every start (`backend/cmd/main.go:92`). It listens on the fixed `:4005` (`backend/cmd/main.go:71`), and the dev frontend calls that port (`frontend/src/constants.ts:22`).
- `free_backend_port` kills whatever listens on 4005 before a run (`scripts/dev.sh:20-33`). The EXIT trap stops the dev database, the WAL backup source and its load generator (`scripts/dev.sh:13-15`).
- The public API covers everything a seed needs, and in development nothing blocks sign-up:
  - Sign-up, sign-in and the user check are at `backend/internal/features/users/controllers/user_controller.go:26-31`.
  - The first account becomes the root admin (`backend/internal/features/users/services/user_services.go:728-786`).
  - Turnstile only runs when its secret is set.
  - Workspaces are created at `workspace_controller.go:22-30`, storages at `storages/controller.go:20-21`, and databases are created and listed at `databases/controller.go:23-27`.
- Creating a database tests the connection on the server itself. For physical databases that includes the replication readiness checks. The live WAL backup source in `docker-compose.yml` passes them.
- The web interface saves the database first, then its backup config, then triggers the first backup when backups are on (`frontend/src/features/databases/ui/CreateDatabaseComponent.tsx:87-120`).
- The physical backup config's defaults keep 7 chains (`backend/internal/features/backups/config/physical/constants.go:4-5`). They already pass the WAL-stream validation (`backend/internal/features/backups/config/physical/model.go:149-196`).
- Dev-only Go commands already live next to the server in `backend/cmd/`: `cleanup_test_db` runs from `scripts/test.sh:49`. The production image builds only `./cmd` (`Dockerfile:62`).
- `chrome-devtools-mcp` is registered only in one developer's personal Claude Code config, as `npx chrome-devtools-mcp@latest`. Codex has no browser server.
- The test fixtures on 5007 and 5008 are reset by `cleanup_test_db` (`backend/cmd/cleanup_test_db/main.go:104-106`). The physical backend tests also use them for replication slots.

Rules that constrain the approach:

- From the root `AGENTS.md`:
  - "Running commands": a long-lived process the agent started is stopped before the task ends.
  - "Documentation consistency" and the `agent-instructions` spec: a skill holds rules of agent behavior, and lists of what exists live next to the code.
- From `backend/AGENTS.md`:
  - "Prefer controller tests over unit tests", and the test cleanup rules.
- From the testing skill: test what a caller observes, and mock only external systems.

## Goals / Non-Goals

**Goals:**

- A just-started local app is ready to sign in to and holds databases with live backups, with no clicks needed.
- The backend picks up a saved Go edit without anyone touching the process window.
- Claude Code and Codex start the same pinned browser server from repository config.
- Every new piece is checked by an automated test or a local command the agent runs.

**Non-Goals:**

- Making the dev environment safe to expose on a network.
- Seeding a state that matches production data, or many databases for load.
- Recovering a seeded database whose replication slot was lost when its container was recreated. See Risks.

## Decisions

### 1. The seed is a dev-only Go command that calls the public HTTP API

`backend/cmd/seed_dev/main.go` is a separate main package, like `cleanup_test_db`. It reads the repository `.env` with `godotenv`, signs in or signs up, and then creates what is missing through `/api/v1`. It declares the few request and response fields it uses instead of importing the backend's types. Importing any feature package runs its dependency wiring, which loads the full backend configuration and exits when the bundled database clients cannot start. That happens on Fedora without the `libedit` shim `just run` applies, so a seed run on its own would fail before sending a request. The backup settings are read and sent back as a JSON object with only the seeded fields changed, so the server defaults survive without being restated. A renamed field still fails the seed test, which drives the real controllers. Its test sits in the same package. The test serves the real controllers through `httptest.NewServer`, against the test database and fixtures. Its router is built the way `backend/cmd/main.go:366-376` and `createE2ETestRouter` (`backend/internal/features/users/controllers/e2e_test.go:250-262`) build theirs. The user controller's public routes (sign-up, sign-in) and the healthcheck controller sit on the bare `/api/v1` group. Everything else sits on a group behind `AuthMiddleware`: the workspace, storage and database controllers, both backup config controllers, and the logical and physical backup controllers that trigger the first backup. The `workspaces_testing.CreateTestRouter` helper does not fit, because it puts every controller behind auth and registers no healthcheck. The seed takes its account, names and database endpoints as parameters. `main` fills them from `.env`, falling back to the defaults of `docker-compose.yml`, and the test passes a unique email and the test fixture ports.

Rejected alternatives:

- **A Python or Bash script.** Its test would have to fake our own backend, which the testing skill forbids. Bash also needs `jq`, and mise does not pin it.
- **Extending `--create-user` in the server binary** (`backend/cmd/main.go:175-180`). That puts fixture data and known credentials into the production binary, against the requirement "Seed credentials exist only in local development".
- **Writing to the metadata database directly.** The direct path skips validation, version detection and the server-side connection check. It also skips the replication slot name the physical model assigns on create, and it bypasses the running backend's in-memory cache.

### 2. Items are found by name, created when missing and never changed

The seed uses these names:

- the account `test@email.com`;
- the workspace `Dev`;
- the storage `Local`;
- the databases `Logical PG16` and `WAL PG18`.

The seed lists what exists in the workspace and creates only the missing items. An item that exists is left exactly as it is, including its backup config. This keeps whatever a developer changed while working on a feature.

Rejected alternatives:

- **Reconciling every item to the seeded state on each run.** Every `just run` would undo the developer's edits, and on a physical database it could change the backup type mid-chain.
- **A marker row recording that seeding happened.** It needs a migration for dev tooling. It would also stop the seed from restoring a deleted database, which the "One seeded database was deleted" scenario requires.

### 3. Sign in first, sign up only when sign-in fails

The seed signs in with `test@email.com` and `Q1234567`. Sign-in has three outcomes. A token means success. A pending two-factor sign-in, returned when the instance requires two-factor authentication (`backend/internal/features/users/services/user_services.go:198-204`), makes the seed exit non-zero with a message that names the account and says two-factor sign-in is required. A failed sign-in leads to sign-up. When sign-up is refused for any reason, the seed exits non-zero with the account and the backend's reason. Possible reasons are that the account already exists or that the registration policy closes sign-up for a non-first account (`user_services.go:728-786`). It never calls the password reset path (`--new-password`), so it cannot change an existing account, as the security requirement demands. Sign-in is rate-limited to 10 attempts per minute per email (`user_controller.go:107-113`). One attempt per run stays far below that.

On an empty database the dev account becomes the root admin. On a database where someone signed up first, it becomes a member. Creating a workspace then depends on the workspace-creation setting. If the backend refuses this or any other create call, the seed exits non-zero with the name of the item and the backend's reason, and it never retries by changing an existing item. The developer then either signs in as that admin or allows members to create workspaces.

### 4. Seeded databases and their backup settings

| Name | Type | Source | Backups |
|---|---|---|---|
| `Logical PG16` | `POSTGRES_LOGICAL` | shared logical fixture, `TEST_LOGICAL_POSTGRES_16_PORT`, `testdb` | daily, `COUNT` retention of 3 |
| `WAL PG18` | `POSTGRES_PHYSICAL`, `FULL_INCREMENTAL_WAL_STREAM` | WAL backup source, `WAL_BACKUP_POSTGRES_18_PORT` and `WAL_BACKUP_DB_*` | server defaults, retention `CHAINS` with 2 chains |

For each database the seed fetches the default backup config, switches backups on, sets the `Local` storage as `storage.id` and the retention above, and saves it. It then triggers the first backup the same way the web interface does after its wizard. Everything else comes from the server's defaults, so the seed does not restate the WAL lag threshold or the interval rules.

The logical source stays on 5004 even though `just test` resets that container. The reset recreates `testdb` with the same user (`backend/cmd/cleanup_test_db/main.go:104`), so later logical backups keep working, and `pg_dump` only reads.

Rejected alternatives:

- **The physical fixtures on 5007 or 5008.** The supervisor of a dev WAL-streaming database keeps a replication slot on its source, and the physical backend tests own those containers.
- **A separate dev container for logical backups.** The shared fixture already serves this purpose, and one more always-on container costs memory for nothing.
- **Leaving backups off.** The user chose backups on, so the Monitoring tab and the backup list show live data right after start.

### 5. The seed runs as a process-compose process gated on the backend port

`process-compose.yaml` gains:

- a readiness probe on the backend that checks that port 4005 accepts connections. process-compose keeps probing for the whole run, and every request the backend serves writes an access log line (`backend/internal/middleware/access_log.go`), so an HTTP probe would fill the backend output with a health line every few seconds. The seed asks the public health endpoint (`backend/internal/features/system/healthcheck/controller.go:14`) itself, once;
- a `seed` process that runs `go run ./cmd/seed_dev` in `backend/`, with `depends_on: backend: condition: process_healthy` and no restart.

A failed seed ends only its own process, as the spec requires. To seed an app that is already running, the same command runs on its own as `go run ./cmd/seed_dev` in `backend/`. It always targets `http://localhost:4005`, because the backend port is fixed. When the health endpoint does not answer, the seed exits at once with the message the spec requires, and it never waits for an app that is not there. When the endpoint answers with an error status, the seed prints the reason from the response body, for example the disk usage check.

Rejected alternatives:

- **A `just seed` recipe with a backend address parameter.** The backend port is fixed and the agent never has a backend of its own on another port, so the parameter has no user today. `just run` seeds on its own, and the rare rerun goes through the command directly. The seed already takes the base URL as a parameter for its test, so a flag can be added if ports per worktree ever appear.
- **Seeding from `scripts/dev.sh`** before process-compose starts. The backend is not up yet, so the script needs its own wait loop, and its output would land outside the process window.
- **A seeding hook inside the backend's startup.** That is production code.

### 6. The backend runs under air, pinned in mise

`mise.toml` adds `"aqua:air-verse/air"` at the latest release, 1.67.4 at the time of writing. `process-compose.yaml` runs `air` in `backend/` instead of `go run ./cmd`. `backend/.air.toml` sets the following:

- **Build.** `go build -o ./tmp/databasus ./cmd`. `tmp/` is added to `backend/.gitignore`.
- **What triggers a rebuild.** `.go` files only. Files ending in `_test.go` are excluded, and so are `tmp`, `swagger`, `ui`, `migrations` and the dev-only commands `cmd/seed_dev` and `cmd/cleanup_test_db`.
- **Debounce.** About one second, so an editor's multi-file save causes one rebuild.
- **On a failed build.** `stop_on_error = false`, so the old binary keeps serving.
- **On restart.** The old process gets an interrupt and a short grace period, so the backend shuts down the way it does on Ctrl+C.

Each restart reruns the migrations, just as `just run` does today.

air does not exit when its child dies: it keeps watching and rebuilds on the next save. The current `free_backend_port` (`scripts/dev.sh:17-33`) relies on the opposite, that killing the listener also ends its parent `go run`. So an air left behind by a killed run would survive the port cleanup and later fight the new run for :4005. `scripts/dev.sh` therefore gains a step before process-compose starts. It stops every `air` process whose working directory is this worktree's `backend/` and only then frees the port. The comment above `free_backend_port` changes to match. Watchers of other worktrees are left alone.

Rejected alternatives:

- **watchexec around `go run`.** It kills the old process before building, so a compile error leaves no backend on 4005.
- **Removing `--no-server`** and restarting the backend through the process-compose CLI. This keeps the restart manual, while the user chose automatic restarts.
- **gow.** Not in the mise registry, so it cannot be pinned the same way.

### 7. The browser server is chrome-devtools-mcp, pinned in mise and started through one launcher

`mise.toml` adds `"npm:chrome-devtools-mcp" = "1.11.0"`. The launcher `scripts/browser-mcp.sh` works as follows:

1. It finds the repository root from its own location and changes into it. mise resolves the pinned version from the `mise.toml` of the current directory, and the client may start the launcher from anywhere.
2. It reads `BROWSER_HEADLESS` from the environment, then from the root `.env`, and defaults to `true`. It parses `.env` without sourcing it, because `.env` holds quoted DSNs.
3. A value other than `true` or `false` makes it exit with an error on stderr. The client then reports that the server failed to start.
4. It runs `exec mise exec -- chrome-devtools-mcp --isolated --viewport 1440x900`, adding `--headless` when the value is `true`. `mise exec` resolves the pinned version whatever `PATH` the agent client was started with.

`.mcp.json` and `.codex/config.toml` both register the server as `chrome-devtools`. Each calls the launcher through `bash -c` and `git rev-parse --show-toplevel`, the same pattern the hooks in `.claude/settings.json` and `.codex/hooks.json` use. `.env.example` gets `BROWSER_HEADLESS=true` under a `# agent browser` heading, and the local `.env` gets the same line, as the user asked.

`--isolated` gives every session a temporary profile, so two agents never fight over `~/.cache/chrome-devtools-mcp/chrome-profile`. The cost is that each session signs in again, which takes one form with the seeded account. The name `chrome-devtools` keeps the tool names agents already know.

Rejected alternatives:

- **Playwright MCP.** It downloads its own Chromium, and Playwright does not officially support Fedora. Its debugging tools for console, network and performance are thinner. Registering both servers would give the agent two near-identical toolsets.
- **`npx chrome-devtools-mcp@latest`.** Unpinned, and it fetches from npm whenever a session starts.
- **Expanding `${BROWSER_HEADLESS}` inside `.mcp.json`.** Claude Code supports that, but Codex does not expand variables in arguments. The launcher gives both clients one behavior.
- **A persistent shared profile.** The second concurrent session fails because the profile is locked.

### 8. The skill holds rules, and the facts live next to the seed

`.agents/skills/run-app/SKILL.md` describes when and how the agent checks the UI, and nothing else:

- reuse the app only when both ports answer and the process listening on the backend port has its working directory inside the agent's own worktree;
- when the app serves another worktree, leave it alone, skip the browser check and tell the user it was skipped and why;
- when only one of the two ports answers, start nothing, skip the browser check and tell the user which port is held;
- when neither port answers, start `just run -t=false` in the background;
- sign in with the seeded account;
- stop the run with an interrupt and confirm that both ports are free;
- a browser check never replaces the tests a task names.

The account, the names and the ports are a list of what exists, so the `agent-instructions` spec keeps them out of the skill. The account and the database names live in the seed command's package comment, and the ports in the `links` process of `process-compose.yaml`; the skill links to both as its sources of values. `.claude/skills/run-app/SKILL.md` is a symlink, like the other repository skills. The skill also tells the agent that it may rerun the seed command when the app of its own worktree lacks the dev account or a seeded database. The root `AGENTS.md` updates its `just run` row to mention the seed and the restart on change. It also gains a two-line "Checking the app in a browser" section that points to the skill, the same way "Tests" points to the testing skill.

The skill is named `run-app` so that the built-in `run` skill, which first looks for a project skill about launching the app, finds it.

Rejected alternative:

- **Putting the account and database list straight into `AGENTS.md` or the skill.** The `agent-instructions` spec forbids lists of what exists there, and the diff reviewer would reject them.

## Risks / Trade-offs

- **[Risk]** Each backend restart interrupts a full backup or a WAL receive in progress. → The user accepted this. The receiver already resumes after a restart, and the next scheduled run repeats an interrupted full backup.
- **[Risk]** An agent's background `just run` killed with SIGKILL skips the EXIT trap. That can leave a Vite server on 5173, and the next Vite then silently moves to 5174. → The skill tells the agent to stop the run with an interrupt and to confirm that 4005 and 5173 are free afterwards. If one port is still held, the next check finds only one port answering, and the skill makes the agent report it instead of starting a second run. A stale backend and a stale air of the same worktree are stopped by `scripts/dev.sh` (decision 6).
- **[Risk]** Recreating the WAL backup source container drops the slot of `WAL PG18`, and its backups start failing. In the other direction, recreating `backend-db` orphans the slot on the source, which then keeps WAL without limit. → The seed does not repair either case, because existing items stay untouched. The developer deletes `WAL PG18` and reruns the seed command. An orphaned slot is dropped on the source with `pg_drop_replication_slot`. The seed command's package comment records both fixes.
- **[Risk]** Codex reads a project `.codex/config.toml` only for a trusted project. → Codex already trusts this repository for `.codex/hooks.json`. The task verifies the registration with `codex mcp list` from the repository root.
- **[Risk]** The health endpoint also fails when disk usage reaches 95% (`backend/internal/features/system/healthcheck/controller.go:36-50`). On a nearly full dev disk the seed would create nothing. → The seed prints the health endpoint's reason in its process window, under `just run` and on its own alike. The seed command's package comment names this case.
- **[Trade-off]** A seed test against the real API needs the shared test fixtures and the test database. It is slower than a faked test, but it catches a changed API.
- **[Trade-off]** The account `test@email.com` / `Q1234567` exists on any dev instance that ran `just run`, and the dev backend listens on all interfaces. The dev database already exposes its port with the same password, so the exposure does not grow. Binding to localhost stays out of scope.

## Migration Plan

1. After pulling the change, run `mise install` to get `air` and `chrome-devtools-mcp`.
2. Remove the personal `chrome-devtools` entry from `~/.claude.json` with `claude mcp remove chrome-devtools -s local`. A local-scope server overrides a project-scope server of the same name, so the old `npx …@latest` entry would otherwise keep running.
3. Approve the project `.mcp.json` server when Claude Code asks for it for the first time.
4. Rollback is a revert. The seeded items stay in the dev database and can be deleted in the UI.
