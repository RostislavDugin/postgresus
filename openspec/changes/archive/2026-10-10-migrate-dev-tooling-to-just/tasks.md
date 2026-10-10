## 1. Toolchain

- [x] 1.1 Add `mise.toml` with the versions listed in design.md, pinning just, lefthook and process-compose to their latest release exactly. If mise is not installed on the host, stop and ask the user to install it. Verify that `mise install` succeeds and that `mise exec -- goose --version`, `mise exec -- swag --version` and `mise exec -- golangci-lint version` print the pinned versions.

## 2. Module Justfiles

- [x] 2.1 Add `backend/Justfile` with a check-only `lint` (`golangci-lint fmt --diff` and `golangci-lint run`), a writing `format`, and `pull-testcontainers` and `clean-testcontainers`, carrying over the image pattern and retry loop from `backend/Makefile`. Keep `backend/Makefile` until task 3.5. Verify that `just --justfile backend/Justfile lint` passes and leaves `git status --short backend` unchanged, that `just --justfile backend/Justfile pull-testcontainers` exits 0, and that `grep -n 'FAILED to pull' backend/Justfile` shows the message next to an `exit 1`.
- [x] 2.2 Add `agent/verification/Justfile` with `run`, `build`, `test`, a check-only `lint`, a writing `format`, `e2e` and `e2e-clean`, loading `.env` the way the Makefile did, then delete `agent/verification/Makefile`. Verify that `just lint`, `just build` and `just test` pass in `agent/verification` and that `just lint` leaves `git status --short agent` unchanged.

## 3. Root recipes and scripts

- [x] 3.1 Add a shared helper under `scripts/` that creates the libedit shim and prepends it to `LD_LIBRARY_PATH`, failing with the install hint when `libedit.so.0` is missing. Verify with `bash -n` on the helper and by sourcing it and checking that `ldd` on a bundled `assets/tools/x64` client resolves `libedit.so.2`.
- [x] 3.2 Add `scripts/test.sh` and the root `test` and `pull-testcontainers` recipes as design.md describes, with the `EXIT`/`INT`/`TERM` trap that kills remaining children and removes containers labelled `org.testcontainers`. Verify with `bash -n scripts/test.sh` and `just --dry-run test`, and by searching the script: `grep -n 'trap' scripts/test.sh` shows `EXIT`, `INT` and `TERM`, `grep -n 'label=org.testcontainers' scripts/test.sh` matches inside the cleanup function, and `grep -nE 'docker compose (down|stop)' scripts/test.sh` prints nothing.
- [x] 3.3 Add `scripts/dev.sh`, `process-compose.yaml` and the root `run` recipe as design.md describes, moving the stale-port logic from `backend/Makefile` and stopping only `backend-db` on exit. Verify with `bash -n scripts/dev.sh`, `just --dry-run run` and a process-compose dry run of `process-compose.yaml` that reports no config error. Then search the script: `docker compose stop backend-db` sits in the exit trap, `swag init` and `pnpm --dir frontend install --frozen-lockfile` come before the `process-compose` line, and `grep -n 'docker compose down' scripts/dev.sh` prints nothing.
- [x] 3.4 Add the root `test-filesystem` recipe with the same image and `CASE` handling as the root `Makefile`, then delete the root `Makefile`. Verify with `just --dry-run test-filesystem` and `CASE=bind-root just --dry-run test-filesystem`.
- [x] 3.5 Delete `backend/Makefile` now that the shim, the suite steps and the stale-port logic have moved. Verify that `ls Makefile backend/Makefile agent/verification/Makefile` fails for all three paths and that `grep -rn 'libedit.so.2\|/proc/net/tcp' scripts` finds the carried-over logic.

## 4. Frontend formatter and linter

- [x] 4.1 Add oxfmt with `frontend/oxfmt.config.ts` as design.md describes, set the `format` and `format:check` scripts, and remove Prettier, its plugins and `frontend/.prettierrc.js`. Verify that `pnpm install` succeeds and `pnpm format:check` runs and lists files rather than failing on configuration.
- [x] 4.2 Reformat the whole frontend with `pnpm format`, changing nothing else in this task. Verify that `pnpm format:check`, `pnpm test` and `pnpm build` pass.
- [x] 4.3 Add oxlint and load `eslint-plugin-i18next` with the patched `no-literal-string` wrapper through `jsPlugins`. Verify that oxlint reports a temporary file under `frontend/src` holding a hardcoded label, then delete that file. If the plugin cannot run under oxlint, stop and ask the user before going further.
- [x] 4.4 Complete `frontend/oxlint.config.ts` with the React and TypeScript rules, set the `lint` script to oxlint, remove ESLint, its plugins and `frontend/eslint.config.js`, and fix or reasonably suppress the new findings. Verify that `pnpm lint`, `pnpm test` and `pnpm build` pass.
- [x] 4.5 Add the `typecheck` script running `tsc -b`. Verify that `pnpm typecheck` passes.

## 5. Root lint and format

- [x] 5.1 Add the root `lint` and `format` recipes that delegate to the backend, the verification agent and the frontend, with frontend lint running `format:check`, `lint` and `typecheck`. Verify that `just format` followed by `just lint` passes at the root. Then add a temporary file under `frontend/src` with a hardcoded label, check that root `just lint` exits non-zero and names that file, and delete the file.

## 6. Commit hook

- [x] 6.1 Add `lefthook.yml` as design.md describes, delete `.pre-commit-config.yaml`, and replace the pre-commit entry in `.dockerignore` with the new tooling files. Verify that `mise exec -- lefthook validate` passes. Then stage a temporary unformatted file under `frontend/src`, run `mise exec -- lefthook run pre-commit`, and confirm that `git show :<path>` prints the formatted content and that the output lists the backend and agent jobs as skipped. Unstage and delete the file afterwards.

## 7. Dev Container removal

- [x] 7.1 Delete `.devcontainer/` and reword the Dev Container comment in `.gitattributes`. Verify that `git grep -n -i devcontainer -- ':!openspec/changes/archive'` matches only this change's artifacts.

## 8. CI

- [x] 8.1 Update the frontend lint, backend test, filesystem and verification agent e2e jobs in `.github/workflows/ci-release.yml` as design.md describes, pinning `jdx/mise-action` by commit SHA. Reword the two comments in the backend test job that name the Makefile's `test` target: the Ryuk comment and the timeout comment. Verify that the workflow parses with `python3 -c 'import yaml,sys; yaml.safe_load(open(sys.argv[1]))' .github/workflows/ci-release.yml` and that `git grep -n -E 'make |prettier|Makefile' .github/workflows` prints nothing.

## 9. Documentation and agent configuration

- [x] 9.1 Rewrite the development environment section of the root `AGENTS.md` for a Fedora host with mise. Add a command table with the purpose of `just run`, `just test`, `just test-filesystem`, `just pull-testcontainers`, `just lint`, `just format`, the goose create, up and down commands and `swag init`, without describing their internal steps. Update the linting section. Verify that `git grep -n -E 'make (test|lint|run|e2e|swagger|migration)|test-fedora|Dev Container' AGENTS.md` prints nothing.
- [x] 9.2 Replace the `make` references in `backend/AGENTS.md`, `agent/verification/AGENTS.md`, `backend/README.md` and `assets/tools/README.md`. Verify that `git grep -n -E '\bmake (test|lint|run|e2e|swagger|migration)|test-fedora|run-fedora' -- ':!openspec/changes/archive' ':!openspec/changes/migrate-dev-tooling-to-just'` prints nothing.
- [x] 9.3 Update `openspec/config.yaml` so its context names a Fedora host with mise and `just`, and its task rules name `just lint`, `just build` and `just test` in `agent/verification`, and exclude `just test` and `just test-filesystem` at the root instead of the `make` targets. Verify that `openspec validate migrate-dev-tooling-to-just --strict` passes and `grep -n make openspec/config.yaml` prints nothing.
- [x] 9.4 Replace `make` with `just` in the comments of `.agent-hooks/review_gate.py` and in the stubs and assertions of `.agent-hooks/review_gate_test.py`. Verify that `python3 -m unittest discover -s .agent-hooks -p '*_test.py'` passes.
