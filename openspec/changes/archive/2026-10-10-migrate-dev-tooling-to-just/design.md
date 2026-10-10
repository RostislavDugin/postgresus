## Context

See proposal.md for the motivation. The facts below shape the approach.

- The backend execs `swag init` (`backend/cmd/main.go:484`) and `goose up` (`backend/cmd/main.go:499`) at startup outside production. Both must be binaries on `PATH`, not only Go tool dependencies.
- `backend/cmd/main.go:58` imports the generated `databasus-backend/swagger` package, and `backend/.gitignore` ignores `swagger/`. A fresh worktree cannot compile the backend until something runs `swag init`, even though the backend would generate the docs itself once running.
- `make test` (`backend/Makefile:9`) removes every container labelled `org.testcontainers`, pulls test images, runs `cleanup_test_db`, migrates eight slot databases with goose and runs `go test` with Ryuk disabled. It does not start the compose fixtures; it assumes they are already up. CI starts them with a plain `docker compose up -d` (`.github/workflows/ci-release.yml:289`).
- `test-fedora` (`backend/Makefile:30`) wraps `test` with the libedit shim (`backend/Makefile:23`), because the bundled clients link `libedit.so.2` and Fedora ships `libedit.so.0`. The same shim is needed to run the backend, because startup checks every bundled client.
- `docker-compose.yml` keeps the dev database (`backend-db`) next to the shared test fixtures, and `backend-db` stores its data in an anonymous volume. `docker compose down` on that file would stop other worktrees' fixtures, and removing the dev database container would lose its data.
- ADR 0013 makes the test fixtures long-lived and shared between runs.
- The hardcoded-string guard (`frontend/eslint.config.js:14`, configured type-aware at `frontend/eslint.config.js:211`) backs the `frontend-localization` requirement "New code cannot add hardcoded strings".
- `.agent-hooks/review_gate.py:114` calls `golangci-lint` and `pnpm` from `PATH`, and its comments name `make lint` and `make test`.

Module rules that constrain the approach: the root `AGENTS.md` and the `agent-instructions` spec allow a table of commands with their purpose in `AGENTS.md`, but not the internal steps of a command. The steps therefore live in the Justfiles and scripts. `backend/AGENTS.md` and `frontend/AGENTS.md` name lint and test commands, and those names change.

## Goals / Non-Goals

**Goals:**

- One command each for running the app, running the backend suite and running the filesystem suite, all from the repository root.
- An interrupt leaves no orphaned processes or test containers, while the shared fixtures and the dev database's data survive.
- Tool versions come from one file, which local hosts and CI read alike.

**Non-Goals:**

- Making the backend suite runnable on macOS or other Linux distributions.
- Isolating parallel test runs from each other. A full run still resets what other worktrees share, as it does today.

## Decisions

### Layout mirrors the team's other project

A root `Justfile` holds the cross-module recipes (`run`, `test`, `test-filesystem`, `pull-testcontainers`, `lint`, `format`). `backend/Justfile` and `agent/verification/Justfile` hold the module recipes, and the root delegates to them with `just --justfile <module>/Justfile`. Multi-step logic with traps lives in bash scripts under `scripts/`, because just recipes cannot hold a trap that spans several lines cleanly.

Alternatives rejected: a single root `Justfile` with every recipe, which mixes backend and agent concerns and breaks `cd backend && just lint`; keeping the Makefiles next to new Justfiles, which the "no unrequested backward compatibility" rule forbids.

### `just run` runs through process-compose, prepared by a script

`scripts/dev.sh` applies the libedit shim, starts `backend-db` with `docker compose up -d --wait`, runs `swag init` so a fresh worktree compiles, runs `pnpm --dir frontend install --frozen-lockfile` (a quick no-op when nothing changed), frees the backend port held by a stale run, and then starts process-compose. The processes are: a `links` entry printing the backend and frontend URLs, `backend` (`go run ./cmd` with the shim on `LD_LIBRARY_PATH`), `frontend` (`pnpm --dir frontend dev`) and `db` (the `backend-db` log). On exit, a trap runs `docker compose stop backend-db`. The stale-port logic moves from the `kill-stale` target (`backend/Makefile:74`) into the script unchanged.

Alternatives rejected: `docker compose down` on exit, as the team's other project does, because here it would stop other worktrees' test fixtures and drop the dev database's anonymous volume; a separate `just swagger` recipe, which the user asked to remove; starting `backend-victoria-logs` too, because log shipping is optional and the user can start it by hand.

### `just test` keeps the current suite steps and adds a cleanup trap

`scripts/test.sh` applies the shim, runs `docker compose up -d` for the non-profile services and waits as CI does (`.github/workflows/ci-release.yml:291`): `pg_isready` in `backend-db` and `backend-db-test`, then ports 5004, 5007 and 5008, removes leftover test containers, pulls test images, runs `cleanup_test_db`, migrates every slot with goose and runs `go test` with the current flags. A trap on `EXIT`, `INT` and `TERM` kills the remaining children and removes every container labelled `org.testcontainers`. Shared compose fixtures are left running.

The trap removes all labelled containers on the daemon, not only this run's. Testcontainers labels each test binary with its own session ID, which the script cannot learn, and the start of every run already removes all labelled containers. So the interrupt matches what a new run would do anyway.

Alternatives rejected: enabling Ryuk to clean up, which the suite disabled for a reason and CI mirrors (`.github/workflows/ci-release.yml:340`); stopping the compose fixtures on interrupt, which ADR 0013 and parallel worktrees rule out; running the filesystem suite in the same command, which the user excluded because it is slow and rarely changes.

### Migration and API-doc commands become documentation

The `migration-*` and `swagger` targets are deleted. The root `AGENTS.md` command table gains rows for creating a migration (`goose -dir backend/migrations create <name> sql`), applying and rolling back migrations, and regenerating the API docs (`swag init -g ./cmd/main.go -o swagger` from `backend/`). Agents write every migration, so a recipe adds nothing over the command itself.

Alternative rejected: keeping the recipes as thin wrappers, which duplicates the commands without hiding any complexity.

### mise pins the tools, including the binaries the backend execs

`mise.toml` pins Go 1.26.3 (from `backend/go.mod`), Node 24 (from the `Dockerfile`), pnpm 9.15.9 (from `frontend/package.json`), just, lefthook, process-compose, goose 3.27.3 (from the `Dockerfile`), swag 1.16.4 (matching `github.com/swaggo/swag` in `backend/go.mod`) and golangci-lint 2.11.3 (from CI). just, lefthook and process-compose get the latest release at implementation time, pinned exactly.

Alternative rejected: `go tool` dependencies, as the team's other project uses. The backend execs `goose` and `swag` by name, so they must be on `PATH`, and adding them to `go.mod` would pull their dependency trees into the backend module.

### lefthook replaces pre-commit, and the hook runs tools through mise

`lefthook.yml` sets `lefthook: mise exec -- lefthook`, so Git clients that start without a mise-activated shell still find the tools. The backend and agent jobs run `golangci-lint fmt` on staged Go files with `stage_fixed`, then the module's `just lint`; the backend job also runs `go mod tidy` as pre-commit does today. The frontend job runs `oxfmt` on staged files with `stage_fixed`, then `pnpm lint` and `pnpm typecheck`. Each job is gated by a glob on its module directory.

The module `lint` recipes only check and never write. They run `golangci-lint fmt --diff` and `golangci-lint run`, so a commit cannot reformat unstaged files, including another agent's edits, which the root `AGENTS.md` forbids and `.agent-hooks/review_gate.py:111` avoids for the same reason. Writing happens only in the `format` recipes and in the hook's staged-file step. The current `make lint` starts with a writing `golangci-lint fmt` (`backend/Makefile:51`), so this is a deliberate change.

Alternative rejected: keeping the frontend build in the hook, because a type check catches the same type errors without producing a bundle.

### The frontend moves to oxfmt and oxlint, and the hardcoded-string guard rides along as a JS plugin

`oxfmt.config.ts` keeps the current style: semicolons, single quotes, trailing commas, width 100, two-space indent and parentheses around arrow parameters. Import sorting uses oxfmt's `sortImports` with groups for third-party modules, assets, `@/` imports and relative imports. The numbered FSD groups in `.prettierrc.js` (`1_pages` and so on) match no current folder, so they go. Tailwind classes are sorted by `sortTailwindcss` against `src/index.css`.

`oxlint.config.ts` enables the built-in `react` and `typescript` plugins. Their rules cover today's ESLint set, including the hooks rules and `only-export-components`. `eslint-plugin-i18next` and the patched `no-literal-string` wrapper load through `jsPlugins`, with the same options as today, on the same files. JS plugins get no TypeScript type information, so literals typed as a union of string literals stop being exempt. New exclusions in the options or reasoned inline suppressions cover them. Rules oxlint cannot express are dropped, as the user allowed. Scripts become `lint: oxlint`, `format: oxfmt`, `format:check: oxfmt --check` and `typecheck: tsc -b`, which works because both TypeScript configs set `noEmit`.

The team's other project uses `oxc-config-mantine`. It is not used here, because it targets Mantine and this frontend uses Ant Design.

Alternatives rejected: keeping ESLint only for the i18n rule, which the user ruled out; dropping the i18n guard, which would break the `frontend-localization` requirement.

### CI reads tool versions from mise where a job changes anyway

The jobs this change has to touch (frontend lint, backend tests, filesystem suite, verification agent e2e) install tools with `jdx/mise-action`, pinned by commit SHA like every other action. They then call `just pull-testcontainers`, `just test-filesystem`, `just --justfile agent/verification/Justfile e2e` and `pnpm format:check`. Two comments in the backend test job name the Makefile's `test` target: the Ryuk comment (`.github/workflows/ci-release.yml:338`) and the timeout comment (`.github/workflows/ci-release.yml:347`). Both are reworded to name `scripts/test.sh`. The backend test job keeps its own migration loop and its 40-minute timeout, because CI runs Ubuntu and `just test` assumes Fedora. Jobs this change does not otherwise touch keep their setup actions.

Alternative rejected: moving every CI job to mise in this change, which is scope creep.

## Risks / Trade-offs

- [`eslint-plugin-i18next` may not run under oxlint's alpha JS plugin API, or the patched visitor may behave differently there.] → Verify it first, on a file with a known violation. If it cannot run, stop and ask the user before touching the `frontend-localization` requirement.
- [Losing the type-aware exemption may flood the frontend with new i18n errors.] → Prefer a few generic exclusions, such as attribute names, over many inline suppressions. Report the count if the fix grows large.
- [The one-time oxfmt reformat produces a large diff that conflicts with open frontend branches.] → Keep the reformat in its own task, so it can be committed separately.
- [Interrupting `just test` removes test containers that another worktree's run is using.] → Same behavior as starting a run today. `AGENTS.md` already tells agents to keep the full suite for the end of a change.
- [The review gate calls `golangci-lint` from `PATH` and fails when the agent's shell lacks mise activation.] → AGENTS.md tells contributors to activate mise in their shell. The gate itself keeps calling tools by name.
- [Contributors who relied on the Dev Container lose it.] → The proposal marks this BREAKING. mise plus a Fedora host is the documented environment.

## Migration Plan

Contributors install mise, run `mise install` at the repository root and run `lefthook install` once to replace the pre-commit hook. `pre-commit uninstall` removes the old hook if it was installed. Rollback is reverting the change.
