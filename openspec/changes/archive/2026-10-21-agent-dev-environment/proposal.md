## Why

An agent working on the frontend cannot see what it built. No browser tool is registered in the repository: the only one is a `chrome-devtools` entry in one developer's personal Claude Code config, which pulls the latest package from npm on every start, and Codex has none at all. A fresh `just run` also opens an empty app: before anyone can look at a screen, someone has to sign up, create a workspace and a storage, and add a database through the wizard. The backend also ignores edits until someone restarts it by hand in the process window. The frontend already picks up edits on its own.

The result is that every UI check needs the developer's hands. This change makes the local app ready to look at as soon as it starts and keeps it running the latest code. It also gives Claude Code and Codex a browser to see and click through it, either visibly or headless.

## Governing docs

This change answers to the root [`AGENTS.md`](../../../AGENTS.md) and to [`backend/AGENTS.md`](../../../backend/AGENTS.md), because the seed command and the restart-on-change configuration live under `backend/`. It touches nothing under `frontend/`, `agent/` or `website/`.

## What Changes

- `just run` runs the backend under a file watcher. A saved Go source file rebuilds the backend and restarts it. When the new code fails to compile, the previous backend keeps serving and the compile error shows in the backend's output.
- `just run` seeds the local app once the backend answers. The seed creates whatever is missing among the following:
  - a dev account, `test@email.com` with password `Q1234567`;
  - a workspace with a local storage;
  - a PostgreSQL database with logical backups of the shared logical test fixture;
  - a PostgreSQL database with WAL-streaming backups of the live WAL backup source.
- Both databases have backups switched on, keep only a few backups, and get a first backup right away. The seed never modifies anything that already exists, so it runs on every start without touching data a developer changed.
- The seed command can also be run on its own against the app already running on the backend port, for example after a seeded database was deleted. There is no separate `just` recipe for it: the backend port is fixed, and `just run` covers the usual case.
- The seed is a dev-only command in the backend module, like the existing `cleanup_test_db`. The production image never builds it.
- `mise.toml` pins the backend file watcher and the Chrome DevTools MCP server.
- The repository registers that browser server for Claude Code (`.mcp.json`) and Codex (`.codex/config.toml`) through one launcher script. Each agent session gets its own temporary browser profile, so parallel sessions do not block each other.
- `BROWSER_HEADLESS` in `.env` and `.env.example` chooses headless or visible Chrome, with headless as the default. A value set in the environment wins over the file.
- A new repository skill tells the agent how to check the UI:
  - reuse the running app only when it serves the agent's own worktree;
  - skip the check and tell the user when the app serves another worktree or only one of its two ports answers;
  - start `just run` without its window when nothing runs;
  - rerun the seed command when the app of its own worktree lacks the dev account or a seeded database;
  - sign in with the seeded account;
  - stop what it started.
- The skill also says that a browser check adds to tests and never replaces them. The skill is exposed to Claude Code through a symlink in `.claude/skills`, like the others.
- The root `AGENTS.md` says in its `just run` row that the run seeds the app and restarts the backend on change, and points to the new skill.

### Out of scope

- Separate ports per worktree, and a backend address or port parameter for the seed. Only one worktree runs `just run` at a time, so the fixed backend and frontend ports stay.
- Playwright and its MCP server. One browser toolset is enough, and two would give the agent near-duplicate tools.
- Regenerating the API documentation when controller annotations change. It still happens once at the start of `just run`.
- Restarting the backend when a migration file alone changes. The next Go edit or the next `just run` applies it.
- Seeding MySQL, MariaDB, MongoDB, notifiers or the physical test fixtures on ports 5007 and 5008. Those fixtures belong to the physical backend tests, and a dev replication slot on them would interfere with those tests.
- Binding the dev servers to localhost only. The dev database already listens on all interfaces with a known password, and that does not change here.
- Changing the rule in `openspec/config.yaml` that keeps browser clicking out of task verification.
- Removing the developer's personal `chrome-devtools` entry from `~/.claude.json`. It lives outside the repository. The design explains why it has to go.

## Capabilities

### New Capabilities

None.

### Modified Capabilities

- `local-development-workflow`:
  - `just run` restarts the backend on source changes, updates the frontend without a restart and seeds the dev account and databases;
  - the seed command can be rerun against the running app;
  - the seed is dev-only;
  - the pinned tool set gains the file watcher and the browser server.
- `agent-workflow`: the agent can see and drive the local app in a browser, both in Claude Code and in Codex, and follows the rules for reusing, starting and stopping the app.

## Impact

- `backend/cmd/seed_dev/`: new dev-only command and its test, which drives the real API on the test database.
- `backend/.air.toml` and the backend's `.gitignore`: watcher configuration and its build directory.
- `process-compose.yaml`, `scripts/dev.sh`: the watcher, stopping a watcher left over from a killed run, the readiness probe and the seed process.
- `mise.toml`: two new pinned tools.
- `scripts/browser-mcp.sh`, `.mcp.json`, `.codex/config.toml`: browser server launcher and registration.
- `.env.example` and the local `.env`: `BROWSER_HEADLESS`.
- `.agents/skills/run-app/SKILL.md` and its `.claude/skills/run-app/SKILL.md` symlink.
- `AGENTS.md`: command table row and a pointer to the skill.
- No production code, API, schema or migration changes.
