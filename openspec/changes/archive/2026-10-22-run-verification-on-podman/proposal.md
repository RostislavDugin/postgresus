## Why

Backup verification fails on every job when the agent talks to Podman through its Docker-compatible API (issue #833). After the restore container starts, the agent reads the container's configuration back and requires the literal `ALL` in the list of dropped capabilities. Podman does drop all capabilities except the minimal set, but it reports the dropped list differently: as the capabilities it removed from its own default set. The check therefore rejects a correctly hardened container, and the job ends with "hardening regression: CapDrop ALL not applied".

The check also protects nothing on Docker. Docker reports back the configuration the agent sent, so the check only confirms that the agent's own request contained `ALL`. It never looks at what the kernel applied. A unit test covers that request better, and a test against Podman catches this class of engine difference before a user does.

## What Changes

- The agent stops reading the container's security settings back after start. The read-back covered dropped capabilities, no-new-privileges and host bind mounts. All three branches only repeated the agent's own request, so the whole check goes: `Manager.assertSecurity`, `dockerEngine.InspectSecurity` and the `SecurityState` type are deleted, with no replacement at runtime.
- Restore containers keep the same hardening: every capability dropped except `CHOWN`, `DAC_OVERRIDE`, `FOWNER`, `SETGID` and `SETUID`, privilege escalation disabled, no host bind mounts, and the existing memory, CPU and process limits.
- A unit test checks that the container configuration the agent sends to the engine carries that hardening. The test covers the translation from the agent's container spec to the engine's host configuration, which nothing tests today.
- A new CI job runs a subset of the agent's end-to-end tests with restore containers on a rootful Podman engine, so the agent is tested against both engines it is used with.

Nothing breaks. No flag, config field, API or report field changes. The error "hardening regression: ..." disappears from verification failures.

## Capabilities

### New Capabilities

None.

### Modified Capabilities

- `verification-container-resources`: adds the hardening every restore container carries as its own requirement (today it appears only inside a CPU-limit scenario), and adds the requirement that verification runs on Podman through its Docker-compatible API.

## Impact

- Code: `agent/verification` only. The Docker engine wrapper and the container manager in `internal/features/container`, the end-to-end compose file, scripts and Justfile in `agent/verification/e2e` and `agent/verification/Justfile`.
- CI: one new job in `.github/workflows/ci-release.yml`, which `determine-version` also waits for.
- Rules this change answers to: `AGENTS.md` and `agent/verification/AGENTS.md`.
- Backend, frontend, the agent-to-backend API and the heartbeat payload are untouched.

## Out of scope

- Dropping or loosening any container hardening control. They stay as they are.
- Blocking outbound network access from restore containers.
- The hang at `gosu` that the issue reporter saw on Podman with the `crun` runtime. It did not reproduce locally on rootless Podman 5.8.7 with `crun`; if the new Podman CI job shows it, it becomes its own issue.
- Rootless Podman in CI, Podman on macOS or Windows, and other container engines.
- Running the full end-to-end suite, every PostgreSQL major included, on Podman.
- Documenting Podman as a supported engine on the website.
