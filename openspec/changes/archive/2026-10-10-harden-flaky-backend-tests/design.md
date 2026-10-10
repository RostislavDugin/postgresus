## Context

See proposal.md - Why for the three observed failures. The current state behind each:

- **Container start.** Every test container goes through one of two helpers, `startContainer` (`backend/internal/util/testing/containers/containers.go:110`) and `startUnpublished` (`containers.go:83`). Both call `testcontainers.GenericContainer` once and `t.Fatalf` on any error. When creation or start fails, `GenericContainer` still returns the half-started container alongside the error (`testcontainers-go@v0.42.0/generic.go:79-97`), and today nobody terminates it. The MongoDB image's entrypoint boots `mongod` twice, first on loopback to create the root user and then on all interfaces (`containers/mongodb.go:24-35`). Under `go test -p=8` on a 4-vCPU runner the second boot can fail to bind and exit with code 48, which surfaces as `wait until ready: container exited with code 48`.
- **Bastion restart.** `startBastion` (`backend/internal/features/tests/physical/postgresql/shared/sshtunnel.go:88-93`) restarts the container and then `waitForBastionAccepting` (`sshtunnel.go:99-118`) polls a plain TCP dial. `Container.Start` already reran the container's own wait strategy, which is a listening-port check inside the container and on the published port (`containers/ssh_bastion.go:210`; `testcontainers-go@v0.42.0/docker.go:243-268`). Both TCP-level checks passed, and the very next SSH handshake from `OpenAdminConn` was reset (`bodies.go:521`). The logs show the supervisor's own tunnel opening successfully in the same second, so the bastion recovered moments later. The exact reason sshd reset that one handshake is not proven. What is proven is that TCP readiness did not imply SSH readiness.
- **Log rotation.** The rotating sink writes through `lumberjack` v2.2.1. After a rotation, `lumberjack` deletes surplus backups in a background goroutine (`mill`/`millRun`), and `Close` does not wait for it. The test pushes about 5.6 MB through a 1 MB limit with three backups allowed, shuts the writer down and immediately counts files (`backend/internal/util/logger/file_sink_test.go:31-56`). If the last cleanup has not run yet, it sees five files.

Constraints: `backend/AGENTS.md` Testing section (run the touched packages via `just test-package`, never the full suite), the principle that nothing observes cleanup by sleeping (stated for stored-file cleanup in `backend/AGENTS.md:269` and applied here to log-file cleanup), and ADR-0013, which keeps one container per engine version alive and makes container boots comparatively rare and expensive.

## Goals / Non-Goals

**Goals:**

- A transient container boot failure no longer fails a test, for every container the shared helpers start.
- A restarted bastion is used only after it completes an SSH handshake.
- The rotation test asserts the cap the sink actually guarantees: eventually at most four files.

**Non-Goals:**

- Retrying anything after a container has booted. Test logic, queries and tunnels opened by the code under test keep failing on the first error.
- Diagnosing why sshd reset the handshake. The fix makes the test wait for the condition it needs, whatever the cause.

## Decisions

### 1. Retry the whole boot in the shared start helpers

Both helpers delegate to one function that runs up to three attempts. A failed attempt terminates the container `GenericContainer` returned, if any, logs the attempt number, image and error with `t.Logf`, and moves on. After the last attempt the test fails with the last error and the attempt count. The loop lives in one shared function in `containers.go` and calls `GenericContainer` directly. It gets no injectable start call and no unit test of its own: `backend/AGENTS.md` reserves unit tests for complex model logic, and a three-iteration loop in a test helper does not qualify. Every test that boots a container exercises the success path. There is no pause between attempts: the failures seen are load spikes, and an attempt already spends seconds pulling state and booting.

Three attempts: one retry covers the observed failure, and a second keeps a back-to-back repeat under the same load spike from failing the test. More attempts only delay the report of a genuinely broken image.

Alternatives rejected:

- *Retry only MongoDB.* The failure mode, an entrypoint that boots twice or a container that dies under CPU contention, is not specific to MongoDB, and the shared helpers are the one place every engine passes through. Per-engine retries would duplicate the same loop.
- *Change MongoDB's readiness instead.* Exit 48 happens inside the container before any readiness check can help. Waiting longer cannot revive an exited process.
- *Suite-wide rerun of failed tests.* Rejected in proposal.md - Out of scope.
- *Retry only for specific errors (exit code 48, "wait until ready").* Matching testcontainers error strings is brittle across library upgrades. A deterministic failure such as a missing image costs at most two extra fast failures.

### 2. Bastion readiness is a full SSH handshake with the test credentials

`waitForBastionAccepting` becomes a wait that dials the bastion's published address with an SSH client, authenticates with `SshBastionUsername` and `SshBastionPassword`, and closes the client on success. Each try has a short timeout, tries repeat every 500 ms, and the existing 90-second deadline stays. The function is renamed to match what it now proves. It keeps dialling the address from before the outage, for the reason its current comment gives. Host key handling matches the production forwarder (`backend/internal/features/sshtunnel/forwarder.go:107-110`).

Alternatives rejected:

- *Read the SSH version banner only.* The banner arrives before key exchange, and the observed reset happened during the handshake. A banner check could pass and still leave the next handshake to fail.
- *Retry inside `OpenAdminConn`.* Every physical test uses that helper. A retry there would also hide a tunnel regression in tests that are not restarting anything.
- *Open a real tunnel through the production `sshtunnel` package as the probe.* It needs a target and more setup, and adds nothing over a successful authenticated handshake, which is the step that failed.

### 3. The rotation test polls the cap

The upper-bound assertion becomes `assert.Eventually` over the file count with a deadline of a few seconds, checked every 50 ms or so. The lower bound (more than one file, so rotation happened) and the per-file size check stay as they are, run once after the eventual condition holds.

Alternatives rejected:

- *Make `Shutdown` wait for `lumberjack`'s cleanup.* `lumberjack` exposes no way to wait for its background goroutine, and production does not need it: the cap holds within moments of every rotation.
- *Sleep before counting.* It breaks the no-sleeping principle that `backend/AGENTS.md:269` states for stored-file cleanup, and a fixed sleep is either too short on a slow runner or wasted on a fast one.
- *Relax the bound to five.* That would accept a sink that keeps one backup too many forever.

## Risks / Trade-offs

- [A broken image or misconfigured request now fails three times before the test reports it] → Fast failures (pull errors, bad commands) add seconds. A readiness timeout triples, up to 12 minutes for MongoDB's 240-second startup timeout. That only happens for a container that never becomes ready, which is already a red build. Each attempt's error is in the test log.
- [Retries hide a container that has become chronically flaky] → Every failed attempt is logged with its image, so a recurring retry shows up in CI output instead of disappearing.
- [The bastion's pinned host port could be taken between attempts] → The failed container is terminated first, which releases the binding. The same small window exists today between the port reservation and the first start.
- [The handshake probe passes but a later handshake still resets] → That would point at a real problem in sshd or the tunnel under reconnect load rather than readiness, which is worth seeing as a failure.
- [The root cause of the bastion reset stays unknown] → Accepted. The test needs SSH readiness and now waits for exactly that.
