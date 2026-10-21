## 1. Seed command

- [x] 1.1 Create `backend/cmd/seed_dev/main.go` as a dev-only main package, following design decisions 1 to 4:
  - The seed takes the account, the names and both database endpoints as parameters, and `main` fills them from the repository `.env`, falling back to the defaults in `docker-compose.yml`. It imports no backend package, per design decision 1, so it starts without the `libedit` shim.
  - It checks `GET /api/v1/system/health` first. It exits non-zero with "start the app with `just run`" when the backend does not answer, and with the reason from the response body when the endpoint returns an error status.
  - It signs in, or signs up when sign-in fails, and stops naming the account when both fail.
  - It finds or creates the workspace `Dev`, the storage `Local` and the databases `Logical PG16` and `WAL PG18`. When the backend refuses a create call, the seed exits non-zero with the item's name and the backend's reason.
  - For each database it creates, it saves the backup config with backups on, the storage and the retention (3 logical backups, 2 WAL chains), then triggers the first backup the way `CreateDatabaseComponent.tsx` does.
  - The package comment lists the account, the names, the sources, the two slot recovery steps and the disk usage case from design.md "Risks".
  - Verify: in `backend/`, `go build ./cmd/seed_dev` and `just lint` pass, and `go list -deps ./cmd | grep seed_dev` prints nothing. At the root, `grep -n 'go build' Dockerfile` shows that the server binary is built only from `./cmd`.
- [x] 1.2 Add `backend/cmd/seed_dev/main_test.go`. It drives the seed against `httptest.NewServer` over a router built per design decision 1. The user controller's public routes and the healthcheck controller go on `/api/v1` without auth, as `backend/cmd/main.go` registers them. The workspace, storage and database controllers, both backup config controllers, and the logical and physical backup controllers go on a group behind `AuthMiddleware`. It uses the test database, a unique email per test, the logical fixture on `TEST_LOGICAL_POSTGRES_16_PORT` and the physical fixture on `TEST_PHYSICAL_POSTGRES_18_PORT`, and cleans up everything it created as `backend/AGENTS.md` "Clean up test data" requires. The tests cover:
  - a fresh account gets the workspace, the storage and both databases with backups enabled, a first backup started, a logical retention of 3 backups and a physical retention of 2 chains;
  - a second run creates nothing and leaves a database the test edited unchanged;
  - a deleted database is recreated while the rest stays unchanged;
  - an account with another password makes the seed fail with the account in the message, creates nothing and leaves that account's password and role unchanged;
  - a separate account created before the seed runs keeps its password, role and settings afterwards;
  - with two-factor sign-in required by the instance settings, the seed exits non-zero with the account and the two-factor reason and creates nothing;
  - a failed sign-in or sign-up exits non-zero, which the test asserts for every failure case;
  - a dev account that is a member, with member workspace creation switched off, makes the seed fail with the workspace name and the backend's reason, and changes no existing item;
  - an unreachable base URL makes the seed fail with the `just run` hint.
  - a stub `httptest.NewServer` whose `/api/v1/system/health` answers 503 with a `status` reason makes the seed exit non-zero with that reason in the message, and no other request reaches the stub. A full disk is a failure the testing skill allows to fake, because it is unsafe to reproduce.
  - Verify: `just test-package ./cmd/seed_dev/...` in `backend/`.

## 2. Backend restart on change and seeding in `just run`

- [x] 2.1 Add `"aqua:air-verse/air" = "1.67.4"` to `mise.toml`. Verify: `mise install` succeeds and `mise exec -- air -v` prints 1.67.4.
- [x] 2.2 Add `backend/.air.toml` per design decision 6, and add `tmp/` to `backend/.gitignore`. Verify: in `backend/`, `go build -o ./tmp/databasus ./cmd` (the configured build command) succeeds, and `git status --short --ignored tmp` shows `tmp/` as ignored. Then `grep -nE '^\s*(include_ext|exclude_regex|stop_on_error|send_interrupt|kill_delay)\b' .air.toml` shows `include_ext = ["go"]`, an `exclude_regex` with `_test\.go`, `stop_on_error = false`, `send_interrupt = true` and a non-zero `kill_delay`.
- [x] 2.3 In `process-compose.yaml`:
  - run the backend with `air` in `backend/`;
  - give the backend a readiness probe that checks that port 4005 accepts connections, per design decision 5;
  - add a `seed` process that runs `go run ./cmd/seed_dev` in `backend/` after the backend is healthy, without restarts.
  - In `scripts/dev.sh`, before `free_backend_port`, stop every `air` process whose working directory is this worktree's `backend/`, and update the comment above `free_backend_port` per design decision 6.
  - Verify: `process-compose --config process-compose.yaml up --dry-run` exits 0; `grep -n -A8 '^  seed:' process-compose.yaml` shows `depends_on` with `backend` and `condition: process_healthy` and `restart: "no"`; `grep -nE 'exit_on_end|exit_on_skipped|exit_on_failure' process-compose.yaml` prints nothing, so a failed seed does not end `just run`; `bash -n scripts/dev.sh` passes; and `grep -n 'air' scripts/dev.sh` shows the step comparing each process's working directory with `$root/backend`.

## 3. Browser server for both agent clients

- [x] 3.1 Add `"npm:chrome-devtools-mcp" = "1.11.0"` to `mise.toml`. Verify: `mise install` succeeds and `mise exec -- chrome-devtools-mcp --version` prints 1.11.0.
- [x] 3.2 Add `BROWSER_HEADLESS=true` under a `# agent browser` comment to `.env.example` and to the local `.env`, which the user approved. Verify: `grep -x 'BROWSER_HEADLESS=true' .env .env.example` matches in both files.
- [x] 3.3 Add the executable `scripts/browser-mcp.sh` per design decision 7: change into the repository root; read the environment first, then `.env` without sourcing it, then the default `true`; error on any other value; `exec mise exec -- chrome-devtools-mcp --isolated --viewport 1440x900`, adding `--headless` when the value is `true`. Verify with `bash -n scripts/browser-mcp.sh`, then with a copy of the launcher at `$tmp/scripts/browser-mcp.sh`, a temporary `$tmp/.env` and a stub `mise` first on `PATH` that prints its arguments:
  - with no `BROWSER_HEADLESS` in `.env` or the environment, the printed arguments contain `--headless` and `--isolated`;
  - with `BROWSER_HEADLESS=false` in `.env`, they do not contain `--headless`;
  - with `BROWSER_HEADLESS=true` in `.env` and `BROWSER_HEADLESS=false` in the environment, they do not contain `--headless`;
  - with `BROWSER_HEADLESS=maybe` in `.env`, the launcher exits non-zero, the stub is not called, and stderr names `BROWSER_HEADLESS`.
- [x] 3.4 Register the server as `chrome-devtools` in `.mcp.json` and in `.codex/config.toml`, both calling the launcher through `bash -c` and `git rev-parse --show-toplevel`. Verify: `python3 -m json.tool .mcp.json` succeeds, and `codex mcp list`, run from the repository root, lists `chrome-devtools` with the launcher command.

## 4. Agent instructions

- [x] 4.1 Add `.agents/skills/run-app/SKILL.md` with the rules from design decision 8 and spec requirement "The agent can see and drive the local app". It includes the rule to rerun `go run ./cmd/seed_dev` in `backend/` when the app of the agent's own worktree lacks the dev account or a seeded database. It links to the seed command's package comment for the account, names and ports instead of listing them. Add the symlink `.claude/skills/run-app/SKILL.md` -> `../../../.agents/skills/run-app/SKILL.md`. Verify: `readlink .claude/skills/run-app/SKILL.md` prints that target, `grep -c 'test@email.com' .agents/skills/run-app/SKILL.md` prints 0, and `grep -n 'cmd/seed_dev' .agents/skills/run-app/SKILL.md` matches the rerun rule.
- [x] 4.2 In the root `AGENTS.md`, update the `just run` row to mention the seed and the backend restart on change, and add a short "Checking the app in a browser" section that points to the `run-app` skill. Search the repository's Markdown for other descriptions of `just run` and bring them in line. Verify: `grep -rn "just run\|just seed" --include=*.md . | grep -v openspec/` finds no `just seed` and every `just run` mention consistent with the new behavior.
- [x] 4.3 Run the root linters for the touched modules. Verify: `just lint` exits 0.

## Change verification

```sh
just test-package ./cmd/seed_dev/...
```
