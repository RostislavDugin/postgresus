## 1. Request an unprivileged container

- [x] 1.1 In `agent/verification/internal/features/container/postgres_test.go` and `dockerengine_test.go`, change the hardening assertions to the new request: the spec and the engine config start as `postgres`, drop all capabilities, add none (`CapAdd` empty) and carry no security options (`SecurityOpt` empty). Run `just test` in `agent/verification` and see these tests fail on the current code for those reasons.
- [x] 1.2 In `agent/verification/internal/features/container/dockerengine.go`, move the construction of the exec request in `dockerEngine.Exec` into a pure function, as `toContainerConfigs` does for the container request, and add a test in `dockerengine_test.go` that asserts the request carries the command, the environment and the stdin attachment and names no user, so the agent's commands run as the container's user. Verify with `just test` in `agent/verification`.
- [x] 1.3 Delete `NoNewPrivileges` and `CapAdd` from `SpawnSpec`, `minimalCaps` from `manager.go`, and their use in `Manager.buildSpec` and `toContainerConfigs`, keeping `CapDropAll` and `User`. Rewrite the comments on `containerUser`, on `buildSpec`'s hardening and on `CreateNetwork` so they name the controls that remain and say why the container starts as `postgres` (the AppArmor signal denial under no-new-privileges, per design.md). Verify that `just test` and `just lint` in `agent/verification` pass and that `grep -rn "minimalCaps\|NoNewPrivileges\|no-new-priv" agent/verification --include=*.go` finds nothing.

## 2. Check the applied privileges end to end

- [x] 2.1 In `agent/verification/e2e/scripts/test-container-privileges.sh`, require the first `Uid` field of `/proc/1/status` to be non-zero and `CapBnd` to be `0000000000000000`, drop the `NoNewPrivs` check, and update the header comment and the failure and success messages, which no longer mention no-new-privileges. Verify that `just e2e` in `agent/verification` reports every test passed, and that `E2E_ENGINE_SOCKET=$XDG_RUNTIME_DIR/podman/podman.sock just e2e-podman` does too.

## 3. Confirm the fix on rootful Podman in CI

- [x] 3.1 Commit the work so far and push the branch `fix/podman-port-diagnostics`, which runs only `Debug Podman e2e`. The user approved this commit and push for this task: the failure reproduces only on rootful Podman with AppArmor, and the user asked for the CI check inside the tasks (design.md, "Verify on CI as part of the tasks"). Then wait for the run with `gh run watch <run-id> --exit-status`. The step `Run e2e tests on Podman` must report `Results: 3 passed, 0 failed`. If it fails, read the run's log with `gh run view <run-id> --log`, fix the cause in groups 1 and 2, and repeat this task.

## 4. Remove the temporary diagnostics and restore CI

- [x] 4.1 Delete `agent/verification/e2e/diagnose-podman-ports.sh` and `.github/workflows/debug-podman.yml`, remove the step `Diagnose Podman port publishing` from `e2e-verification-agent-podman`, and restore the push trigger of `.github/workflows/ci-release.yml` to `branches: ["**"]` without the temporary comment. Verify that `git diff origin/develop -- .github/workflows/ci-release.yml` prints nothing, that `actionlint .github/workflows/ci-release.yml` reports no findings, and that `grep -rn "diagnose-podman-ports\|debug-podman\|podman-port-diagnostics" . --exclude-dir=.git --exclude-dir=openspec` finds nothing.
- [x] 4.2 Commit the cleanup and push the branch, which now runs the full `CI and Release` workflow; the user approved this commit and push for the same reason as in 3.1. Wait for the run with `gh run watch <run-id> --exit-status`. Every job that runs on a branch push must pass, `e2e-verification-agent` and `e2e-verification-agent-podman` included.

## 5. Hand over the open section of `run-verification-on-podman`

- [x] 5.1 In `openspec/changes/run-verification-on-podman/tasks.md`, replace the open tasks 6.3 and 6.4 with one line saying that `start-restore-containers-unprivileged` fixes the cause and removes the diagnostics, so that change has no open task. In its `design.md`, update the risk entry about the red Podman job and the paragraph on the temporary diagnostic step the same way. Verify that `openspec validate run-verification-on-podman --strict` passes and that `openspec list` shows all of its tasks complete.

## 6. Final checks

- [x] 6.1 Run `just lint` at the repository root and fix every finding. Verify that `grep -rn 'NoNewPrivileges\|NoNewPrivs\|"no-new-privileges"' agent/verification --exclude-dir=artifacts` finds nothing, so neither the code, the unit tests nor the end-to-end test still sets or checks the flag, and that `openspec validate start-restore-containers-unprivileged --strict` passes.

## Change verification

```sh
just --justfile agent/verification/Justfile test
```
