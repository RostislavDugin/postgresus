## 1. Pin the container request with a unit test

- [x] 1.1 In `agent/verification/internal/features/container/dockerengine.go`, move the construction of `container.Config` and `container.HostConfig` out of `CreateContainer` into a pure function that takes a `SpawnSpec`, and make `CreateContainer` call it. Add a table-driven test in `dockerengine_test.go` that asserts, for a hardened spec: `CapDrop` is exactly `["ALL"]`, `CapAdd` equals `minimalCaps`, `SecurityOpt` contains `no-new-privileges`, `Binds` and `Mounts` are empty, the memory, process and CPU limits are copied, a zero `NanoCPUs` stays zero, and port 5432 is published only on `127.0.0.1`. Verify with `just test` in `agent/verification`.

## 2. Delete the runtime read-back

- [x] 2.1 Delete `Manager.assertSecurity` and its call in `Manager.Spawn`, `dockerEngine.InspectSecurity` and `securitystate.go`. Leave `buildSpec`, `minimalCaps` and the hardening fields of `SpawnSpec` unchanged. Update any comment that names the read-back. Verify that `grep -rn "assertSecurity\|InspectSecurity\|SecurityState\|hardening regression" agent/verification` finds nothing, and that `just build` and `just test` in `agent/verification` pass.

## 3. Check the applied privileges end to end

- [x] 3.1 Add `POST /mock/set-stream-delay` with a body `{"seconds": N}` to `agent/verification/e2e/mock-server/main.go`. `handleBackupStream` waits that long before writing the response, and `/mock/reset` clears the delay. Verify with `just lint` in `agent/verification`, which covers `./e2e/...`.
- [x] 3.2 Add `agent/verification/e2e/scripts/test-container-privileges.sh`. It sets a 30-second stream delay, claims a job for `good-pg18.dump`, waits until a running container labeled with the test's agent ID exists, and reads `/proc/1/status` through `docker exec`. It fails unless `CapBnd` is `00000000000000cb` and `NoNewPrivs` is `1`. It then waits for the `COMPLETED` report and runs `leak_check`. Add the script to `TESTS` in `run-all.sh`. Verify with `just e2e` in `agent/verification`, which must report every test passed.

## 4. Run the end-to-end subset on Podman

- [x] 4.1 In `agent/verification/e2e/docker-compose.yml`, mount the runner's engine socket from `${E2E_ENGINE_SOCKET:-/var/run/docker.sock}` and update the runner's comments. Add `agent/verification/e2e/scripts/run-podman.sh`, which runs `test-container-privileges.sh`, `test-restore-success.sh 18` and `test-restore-broken.sh` and reports results the way `run-all.sh` does. Add an `e2e-podman` recipe to `agent/verification/Justfile` that prepares artifacts as `e2e` does and runs the runner with `E2E_ENGINE_SOCKET` defaulting to `/run/podman/podman.sock` and the command `bash /opt/agent/scripts/run-podman.sh`. Verify with `E2E_ENGINE_SOCKET=$XDG_RUNTIME_DIR/podman/podman.sock just e2e-podman` in `agent/verification`, which must report every test passed, and with `just e2e` still passing.
- [x] 4.2 Add the job `e2e-verification-agent-podman` to `.github/workflows/ci-release.yml`, with the same trigger condition, runner image, timeout, `needs` and tool setup as `e2e-verification-agent`. Before `just e2e-podman` it runs `sudo systemctl start podman.socket` and prints `podman version` and `podman info --format '{{.Host.OCIRuntime.Name}}'`. Its cleanup step also removes leftover Podman containers labeled by the agent. Add the job to the `needs` of `determine-version`. Verify with `actionlint .github/workflows/ci-release.yml`, which must report no new findings compared with the file before the change.

## 5. Final checks

- [x] 5.1 Run `just lint` in `agent/verification` and fix every finding until it passes.
- [x] 5.2 Search the repository for mentions of the deleted read-back and of the e2e entry points (`run-all.sh`, `just e2e`, `docker.sock` in the agent's e2e) and bring every affected comment and document in line with the change. Verify that `grep -rn "hardening regression" .` outside `openspec/` finds nothing.

## Change verification

```sh
just --justfile agent/verification/Justfile test
```
