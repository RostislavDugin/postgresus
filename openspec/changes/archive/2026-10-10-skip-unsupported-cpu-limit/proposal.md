## Why

Restore verification cannot run on hosts whose kernel lacks the CPU CFS quota controller, such as older Synology NAS models and fnOS. On every job the agent asks Docker for a CPU limit, and the Docker daemon on those hosts rejects the container outright, so every verification fails before the restore starts. No flag avoids this, because `--max-cpu` must be at least 1.

## What Changes

- At startup the agent asks the Docker daemon whether it can enforce a CPU limit. The answer comes from the same `docker info` call the agent already makes to check user-namespace remapping.
- If the daemon cannot enforce a CPU limit, the agent creates verification containers without one and logs a warning at startup. The warning says that a restore can then use every CPU core on the host.
- If the daemon can enforce it, or if the agent cannot find out, containers keep the current per-job CPU limit.
- `--max-cpu`, the per-job CPU budget, and the `pg_restore` parallelism derived from it are unchanged. The memory limit, the process limit and the other container hardening controls are unchanged.

Nothing breaks. No flag, config field or API changes.

## Capabilities

### New Capabilities

- `verification-container-resources`: the resource limits the verification agent applies to each restore container, and how the agent adapts them to what the Docker host supports.

### Modified Capabilities

None. `backup-verification` covers how verification runs are scheduled, not how the agent runs the restore container.

## Impact

- Code: `agent/verification` only. The Docker engine wrapper and the container manager in `internal/features/container`.
- Rules this change answers to: `AGENTS.md` and `agent/verification/AGENTS.md`.
- Backend, frontend, the agent-to-backend API and the heartbeat payload are untouched.

## Out of scope

- A manual flag to turn the CPU limit off on hosts that support it.
- Reporting to the backend or showing in the UI that an agent runs without a CPU limit.
- Using CPU shares or another relative-priority mechanism as a replacement when the hard limit is unavailable.
- Detecting other unsupported limits, such as memory or process limits.
- Changing how `--max-cpu` is validated or how the per-job CPU budget is computed.
