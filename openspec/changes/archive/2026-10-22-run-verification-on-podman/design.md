## Context

See proposal.md for why. The relevant current state:

- `Manager.buildSpec` sets `CapDropAll: true`, `CapAdd: minimalCaps` and `NoNewPrivileges: true` on every restore container (`agent/verification/internal/features/container/manager.go:203-226`, `minimalCaps` at `:35`). `postgres_test.go:19-20` and `:38-39` assert these fields on the spec.
- `dockerEngine.CreateContainer` translates the spec into the engine's host configuration inline (`dockerengine.go:136-176`): `CapDropAll` becomes `CapDrop: ["ALL"]`, `NoNewPrivileges` becomes `SecurityOpt: ["no-new-privileges"]`. No test covers this translation.
- After the container is ready, `Manager.Spawn` calls `assertSecurity` (`manager.go:194-197`), which reads the container back through `dockerEngine.InspectSecurity` (`dockerengine.go:215-245`) and fails the job unless `HostConfig.CapDrop` contains the literal `ALL`, `SecurityOpt` contains `no-new-privileges`, and `Binds` and `Mounts` are empty (`manager.go:237-253`). `SecurityState` (`securitystate.go`) exists only for this check; its `ReadonlyRootfs` field is read by nobody.
- Docker's inspect returns the host configuration the client sent, so on Docker the check compares the agent's request with itself. Podman's compat API returns `CapDrop` as the capabilities it removed from its own default set and an empty `CapAdd`. Reproduced locally on Podman 5.8.7: `CapDrop` was `["FSETID","KILL","NET_BIND_SERVICE","SETFCAP","SETPCAP","SYS_CHROOT"]`, while `/proc/1/status` in the same container showed `CapBnd: 00000000000000cb` (exactly the five minimal capabilities) and `NoNewPrivs: 1`.
- The runner spawns the container before it downloads the backup (`agent/verification/internal/features/runner/runner.go:166`, download at `:454`). The stream HTTP client has no timeout (`internal/features/api/api.go:86`), and the idle timeout of 60 seconds (`runner.go:28`) starts only once the body is being read.
- The end-to-end suite runs the agent binary in `e2e-agent-runner` with `network_mode: host` and the host's `/var/run/docker.sock` mounted (`agent/verification/e2e/docker-compose.yml`). The agent and the scripts' `docker` CLI both reach the engine through that socket. `just e2e` runs it (`agent/verification/Justfile`), and the CI job `e2e-verification-agent` calls `just e2e` (`.github/workflows/ci-release.yml:216-240`).
- The mock server serves the backup stream in `handleBackupStream` (`agent/verification/e2e/mock-server/main.go`). It has knobs for 503s and a torn stream, not for a delay.

The approach follows `agent/verification/AGENTS.md`: pure translation functions get table-driven unit tests, as `toHostCapabilities` already does in `dockerengine_test.go`, and behavior that needs a real engine is covered in `e2e/` rather than in `internal/` tests.

## Goals / Non-Goals

**Goals:**

- A Podman engine with a correctly hardened container no longer fails the job.
- Each hardening control has a test that can fail: the request the agent sends is covered by a unit test, and what the kernel applied is covered by an end-to-end test on Docker and on Podman.

**Non-Goals:**

- Detecting at runtime an engine that silently ignores the requested hardening.
- Changing how the agent talks to the engine beyond deleting the read-back.

## Decisions

### Delete the runtime read-back instead of fixing it

The agent stops reading the container's security settings back at all. `assertSecurity`, `InspectSecurity` and `SecurityState` are deleted, and `Spawn` returns the container once it is ready.

Rejected alternatives:

- **Accept either `ALL` or an explicit list in `CapDrop`.** A list would have to be compared with the engine's default capability set, which the agent cannot know. Podman reads it from `containers.conf`, which an administrator can change, and Podman also reports `CapAdd` as empty, so the agent cannot rebuild the effective set from inspect.
- **Read `CapBnd` and `NoNewPrivs` from `/proc/1/status` through an exec on every job.** This is the honest form of the check, and it works on any engine. It would only catch an engine that accepts the request and silently does not apply it, which none of the engines the agent runs on does. It costs an exec per job and a parser, and a false positive fails every verification on that host, as #833 shows. The end-to-end test below makes the same kernel-level check once per engine in CI instead.
- **Keep the no-new-privileges and bind-mount branches and drop only the capability branch.** Both remaining branches compare the agent's request with itself in the same way, and Podman's reporting of either can change in a future version with the same effect as #833.

### Test the request with a pure translation function

The body of `CreateContainer` that builds `container.HostConfig` and `container.Config` from a `SpawnSpec` moves into a pure function, and `CreateContainer` calls it. A table-driven unit test feeds it specs and asserts the dropped and added capabilities, the security options, the absence of binds and mounts, the resource limits and the loopback-only port binding. This mirrors `toHostCapabilities` and needs no engine.

Rejected alternative: a fake Docker API server over `httptest` that records the create request. It tests the same mapping through the SDK's JSON encoding, which adds a moving part without adding coverage of anything the agent decides.

### Check the applied privileges end to end, on both engines

A new end-to-end script, `test-container-privileges.sh`, makes the mock delay the backup stream by 30 seconds. While the job's container is running and waiting for its backup, the script finds the container by the agent's label and reads `/proc/1/status` through `docker exec`. It asserts `CapBnd` is `00000000000000cb` and `NoNewPrivs` is `1`, then lets the job finish and runs the usual leak check. At that point PID 1 is the Postgres server, which runs as a non-root user, so `CapEff` is zero for any configuration; the bounding set is what limits a root process in the container, such as the agent's own `pg_restore` exec.

The mock gets a `POST /mock/set-stream-delay` knob, cleared by `/mock/reset`. The delay applies before the response headers, so no agent timeout fires: the stream client has none, and the idle timeout starts with the body.

Rejected alternatives:

- **A Go test in `internal/` against a real engine.** `just test` and the CI job `test-verification-agent` run without engine access by design, and the module keeps engine-dependent tests in `e2e/`.
- **Polling for the container without a delay.** On a small fixture the container lives for a few seconds after it becomes ready, so the check would race the restore.

### Run a subset of the end-to-end suite on Podman in a separate CI job

The compose file takes the engine socket for `e2e-agent-runner` from `E2E_ENGINE_SOCKET`, defaulting to `/var/run/docker.sock`. The builder, the fixture generator and the mock server stay on Docker; only the agent and the scripts' `docker` CLI talk to Podman. A new script, `run-podman.sh`, runs `test-container-privileges.sh`, `test-restore-success.sh 18` and `test-restore-broken.sh`. A new Justfile recipe, `e2e-podman`, prepares artifacts as `e2e` does and runs the runner with the socket and script swapped. It defaults the socket to `/run/podman/podman.sock` and lets the developer override it, for example with a rootless socket.

A new CI job, `e2e-verification-agent-podman`, runs on `ubuntu-24.04`, starts the rootful Podman socket with `sudo systemctl start podman.socket`, prints `podman version` and the OCI runtime for the log, and calls `just e2e-podman`. `determine-version` waits for it as it waits for the Docker job. The existing Docker job also runs `test-container-privileges.sh` through `run-all.sh`.

Until the cause of the first failed run is fixed, the job also runs a temporary step, `Diagnose Podman port publishing`, before `just e2e-podman`. It runs `e2e/diagnose-podman-ports.sh` as root with `continue-on-error`, so it never fails the job. The script starts plain Postgres containers on a user-defined Podman network, one published on `127.0.0.1` as the agent publishes it and one on all interfaces. It checks them with `pg_isready` from the host network namespace through the loopback port, the host IP and the container IP. The first diagnostic run showed every one of these paths working while the agent got `connection refused`, so the script also creates containers through the Docker-compatible API with the agent's own request: the verification image, the tuned command, the hardening, the limits and `HostPort "0"`. Each further variant turns off one of the image, the capability drop and no-new-privileges, and the script prints each container's processes, capabilities and log. When the agent's path fails for plain containers, it prints the firewall rules, stops Docker, removes Docker's rules and repeats the checks. The diagnosis traced the hang to AppArmor denying signals between the container's processes under no-new-privileges. The change `start-restore-containers-unprivileged` fixes that and removes the step and the script.

Rejected alternatives:

- **The full suite on Podman.** The upgrade, start-stop and purge tests do not depend on the engine's reporting, and seven PostgreSQL majors would double the job time without exercising a different engine path. Major 18 matches the issue.
- **A matrix over the existing job.** The Podman run needs a different socket, a setup step and a different script list, so a matrix would need conditionals in every step.
- **Running the whole compose stack on Podman.** It tests `podman compose`, which the product does not use, and only the agent's engine calls matter here.

### Report why a restore container never became ready

On the first CI run on rootful Podman 4.9.3 (`ubuntu-24.04`), every restore container started but stayed unreachable through its published port, and the verification report said only `container never became ready: context deadline exceeded`. `waitForReady` polled once a second and returned only the deadline (`manager.go:232-249`), dropping every ping error. It now returns the deadline together with the error of the last ping that finished before the deadline. A ping the deadline itself cut short does not replace it, because its error only repeats the deadline. The report then tells a refused port from an unanswered connection and from a rejected login, on a user's host as well as in CI. pgx connection errors carry the user, database, host and port, not the password.

Rejected alternatives:

- **Log every failed ping.** Sixty lines per failed job, and the report the user sees would still say only "deadline exceeded".
- **Keep the error of the very last ping.** The unit test showed that the last ping usually starts just before the deadline and fails with a dial timeout caused by the deadline, which hides the real cause.

## Risks / Trade-offs

- [The `docker` CLI in the runner may read Podman's compat API slightly differently, for example in `network ls` label filters] → The leak check fails loudly rather than passing silently. If a filter is unsupported, the script switches to an output the CLI and Podman agree on, still from the runner.
- [Podman may resolve the agent's short image name `databasus/verification-postgres:18` differently from Docker] → Podman's compat API resolves short names to Docker Hub. If the CI run shows otherwise, that is a real user-facing bug and is fixed in the agent, not worked around in the test.
- [The `gosu` hang the reporter saw with `crun` may reproduce in CI] → It did: under no-new-privileges, AppArmor on `ubuntu-24.04` denies signals between the container's processes. `start-restore-containers-unprivileged` fixes it, and this change does not merge with a red Podman job.
- [The job does not run on `develop`, so its first run came after the merge to `main`, and it failed: Postgres in the restore containers never answered, because under no-new-privileges AppArmor denied signals between the container's processes] → `start-restore-containers-unprivileged` fixes the cause and tracks the green run on rootful Podman 4.9.3 in its tasks 3.1 and 4.2. That is the version `ubuntu-24.04` ships, which users on that LTS release also run. The runner is not switched to a newer Podman to make the job green, because that would hide a failure those users hit.
- [The 30-second delay lengthens each end-to-end run by about 30 seconds per engine] → Accepted; it buys a deterministic window instead of a race.
- [Without the read-back, an engine that silently ignores the requested hardening goes unnoticed at runtime] → No engine the agent supports does that, and the end-to-end test proves the applied state on both supported engines in CI.

## Migration Plan

None. The change ships with the next agent release. Agents on Docker behave as before, and agents on Podman start completing verifications. Rollback is reverting the commit; the agent stores no state this change touches.
