## Context

See proposal.md for why. The relevant current state:

- `Manager.buildSpec` always sets `NanoCPUs` to the per-job CPU budget (`agent/verification/internal/features/container/manager.go:197`), and `dockerEngine.CreateContainer` passes it to Docker unchanged (`agent/verification/internal/features/container/dockerengine.go:154-158`).
- The per-job budget is at least 1 by validation (`agent/verification/internal/config/config.go:123`, `:136-144`). It also sizes `pg_restore` parallelism (`agent/verification/internal/features/runner/runner.go:272`), so it has to stay meaningful even when no CPU limit is applied.
- The Docker daemon rejects any non-zero `NanoCPUs` when its `sysInfo.CPUCfs` is false (`github.com/docker/docker@v28.5.2/daemon/daemon_unix.go:495`). `docker info` reports the same value as both `CpuCfsPeriod` and `CpuCfsQuota` (`daemon/info_unix.go:40-41`). Older daemons tracked the two separately and rejected `NanoCPUs` unless both were true.
- `Manager.StartupSelfCheck` (`manager.go:75`) already calls `docker info` once, through `dockerEngine.UserNSRemapEnabled` (`dockerengine.go:67`), to warn when user-namespace remapping is off. `start.go:76-79` calls it before the runner starts, so nothing spawns a container until the check has finished.

## Goals / Non-Goals

**Goals:**

- Decide once per agent process whether the daemon can enforce a CPU limit, and drop the limit only when it cannot.
- Keep the decision testable without a Docker host that lacks CFS.

**Non-Goals:**

- Re-checking support during the process lifetime. The kernel under the daemon does not change while the agent runs.

## Decisions

### Detect support from the `docker info` CFS flags

The agent treats the CPU limit as supported only when `docker info` reports both `CPUCfsPeriod` and `CPUCfsQuota` as true. This is the condition the daemon itself checks before rejecting `NanoCPUs`, in current and older daemon versions, so the agent and the daemon cannot disagree.

Rejected alternatives:

- **Retry container creation without `NanoCPUs` after this specific error.** The agent would have to match the daemon's error text, which can change between Docker versions, and every job on an affected host would fail once before succeeding.
- **Read the cgroup filesystem from the agent process.** The agent can run in a container or talk to a remote daemon. What matters is the daemon's host, which only the daemon can report.
- **A manual `--disable-cpu-limit` flag.** The user would learn about the problem only after verifications fail, and then has to find the flag. Detection fixes the affected hosts with no configuration.

### Read host capabilities with one `docker info` call

`dockerEngine.UserNSRemapEnabled` is replaced by one method that calls `docker info` once and returns both answers, user-namespace remapping and CPU limit support. The translation from the `docker info` response to those two booleans is a pure function, so a unit test covers it by feeding it info structs.

Rejected alternative: a second engine method with its own `docker info` call. It leaves `UserNSRemapEnabled` untouched, but sends a duplicate request to the daemon at every startup and spreads host inspection across two methods.

### Keep the CPU limit when support cannot be determined

If `docker info` fails, the agent logs the existing warning and keeps applying the CPU limit. On a host that does support the limit, this preserves today's behavior. On a host that does not, container creation fails with the daemon's own error, which is what happens today.

Rejected alternative: drop the limit whenever support is unknown. A transient `docker info` failure would then silently remove the CPU limit on hosts that support it.

### Store the answer on the Manager

The Manager holds whether the CPU limit is supported. `NewManager` sets it to true, `StartupSelfCheck` sets it from `docker info`, and `buildSpec` sets `NanoCPUs` to 0 when it is false. To Docker, 0 means no CPU limit. `SpawnSpec`, the runner and the config stay as they are.

Rejected alternatives:

- **Carry the flag in the agent's capacity or config.** Config is built before Docker is contacted and describes what the operator configured, not what the host supports.
- **Ask the daemon on every spawn.** That repeats a request whose answer does not change while the agent runs.

### Announce the missing limit in the startup log only

When the limit is unsupported, `StartupSelfCheck` logs one warning saying that restore containers run without a CPU limit and can use every core on the host. That follows the existing user-namespace warning and `agent/verification/AGENTS.md` (Logging, Warn level: degraded but recoverable). The heartbeat and the backend are not changed. See proposal.md, Out of scope.

## Module rules

- `agent/verification/AGENTS.md`: Docker SDK calls stay confined to `dockerengine.go`, test names follow `Test_WhatWeDo_WhichConditions_WhatWeExpect`, and tests sit next to the code (`postgres_test.go` holds the `buildSpec` tests). Logs follow the Logging rules.
- `AGENTS.md`: names state intent, so booleans take an `is` prefix. There is no backward-compatibility shim, so `UserNSRemapEnabled` is deleted rather than kept beside the new method.

## Risks / Trade-offs

- [A restore on an unsupported host can use every CPU core, starving other services on a NAS] → The startup warning makes this visible. The memory limit, the process limit and the privilege controls still apply. A relative-priority fallback is out of scope.
- [No CI host lacks CFS, so the real Docker path is not exercised automatically] → Unit tests cover the `docker info` translation and `buildSpec`. Confirmation on an affected host has to come from the issue reporter.
- [A daemon could report CFS support it does not have] → Container creation then fails with the daemon's error, the same as today. No new failure mode is introduced.

## Migration Plan

None. The agent decides at startup with no configuration, and reverting the change restores the current behavior.
