## Why

Local development is spread across three Makefiles, a pre-commit config and a Dev Container that nobody uses. Starting the app takes several terminals, and the full backend suite runs only from `backend/` through `make test-fedora`. An interrupted run leaves test containers behind until the next run cleans them up. The frontend formats with Prettier and lints with ESLint, while the team's other project already runs on oxfmt and oxlint, and keeping two toolchains costs more than moving this one.

## What Changes

- **BREAKING** A root `Justfile` becomes the single entry point. `just run` starts the backend, the frontend and the dev database in one process-compose window, with every process listed on the left. Ctrl+C stops everything it started.
- **BREAKING** `just test` replaces `make test` and `make test-fedora` in `backend/`. It assumes a Fedora host, always applies the libedit shim, runs the full backend suite and removes the test containers it started when interrupted. It does not run the filesystem suite.
- **BREAKING** `just test-filesystem` replaces `make test-filesystems` in the root `Makefile`.
- `just pull-testcontainers`, `just lint` and `just format` move to the root. `backend/` and `agent/verification/` get their own `Justfile` with the module-level recipes.
- **BREAKING** All three Makefiles are deleted: `Makefile`, `backend/Makefile` and `agent/verification/Makefile`. So are the targets `run-fedora`, `migration-create`, `migration-up`, `migration-up-test`, `migration-down` and `swagger`. The goose and swag commands an agent needs are listed in the root `AGENTS.md` instead.
- `mise.toml` pins every tool the repository needs: Go, Node, pnpm, just, lefthook, process-compose, goose, swag and golangci-lint.
- **BREAKING** `.devcontainer/` is deleted. `AGENTS.md` and `openspec/config.yaml` stop calling the Dev Container the reference environment and name a Fedora host with mise instead.
- **BREAKING** lefthook replaces pre-commit. `.pre-commit-config.yaml` is deleted. The hook formats staged files and runs the module linters. The frontend job also runs the type check, which replaces the full frontend build.
- **BREAKING** The frontend switches from Prettier to oxfmt and from ESLint to oxlint. `.prettierrc.js`, `eslint.config.js` and every Prettier and ESLint package are deleted. Lint rules that oxlint cannot express are dropped. The hardcoded-string guard is the exception: it moves to oxlint and keeps working.
- The whole frontend is reformatted once by oxfmt.
- CI jobs that called `make` or Prettier call the new recipes and scripts.

## Out of scope

- `website/`, including its ESLint setup and npm lockfile.
- Hot reload for the backend (air or similar).
- Type-aware oxlint rules. They need TypeScript 7, and the frontend uses TypeScript 5.8.
- CI jobs that never call `make` or Prettier.
- Running the backend suite on any host other than Fedora.
- New lint rules beyond what ESLint enforces today.

## Capabilities

### New Capabilities

- `local-development-workflow`: the developer-facing commands for running the app, running the backend suite and checking a commit, including what an interrupt stops and what it leaves running for other worktrees.

### Modified Capabilities

- `docker-storage-compatibility`: the single entry point for the filesystem suite changes from `make test-filesystems` to `just test-filesystem`.

## Impact

- Module rules this change answers to: the root `AGENTS.md`, `backend/AGENTS.md`, `frontend/AGENTS.md` and `agent/verification/AGENTS.md`.
- New files: `Justfile`, `backend/Justfile`, `agent/verification/Justfile`, `mise.toml`, `lefthook.yml`, `process-compose.yaml`, scripts under `scripts/`, `frontend/oxfmt.config.ts` and `frontend/oxlint.config.ts`.
- Deleted files: the three Makefiles, `.pre-commit-config.yaml`, `.devcontainer/`, `frontend/.prettierrc.js` and `frontend/eslint.config.js`.
- `frontend/package.json` and `frontend/pnpm-lock.yaml` change: the ESLint and Prettier packages go and oxfmt and oxlint arrive. `eslint-plugin-i18next` stays, because oxlint loads it as a plugin.
- `.github/workflows/ci-release.yml`: the frontend lint job, the backend test job, the filesystem job and the verification agent e2e job.
- Documentation that names `make` targets: `AGENTS.md`, `backend/AGENTS.md`, `agent/verification/AGENTS.md`, `backend/README.md`, `assets/tools/README.md` and `openspec/config.yaml`.
- `.agent-hooks/review_gate.py` and its test name `make` in comments and stubs.
- Every contributor needs mise on the host. Anyone who still works in the Dev Container loses it.
