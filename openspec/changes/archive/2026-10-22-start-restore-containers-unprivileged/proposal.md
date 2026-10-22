## Why

On Ubuntu hosts where rootful Podman runs containers through `crun`, no verification ever completes: the restore container starts, but Postgres in it never answers. The CI job `e2e-verification-agent-podman` (Podman 4.9.3, `crun` 1.14.1, `ubuntu-24.04`) fails all its tests for this reason, and it is most likely the hang the reporter of issue #833 saw.

The cause is the interaction of no-new-privileges with AppArmor. Under no-new-privileges, AppArmor cannot switch a process from `crun`'s profile to the container profile on exec, so it stacks the two, and the container's processes run under the label `containers-default-0.57.4-apparmor1//&crun`. The container profile allows signals only to its exact own label, so the kernel denies every signal between the container's processes. The official entrypoint's `gosu` then spins forever waiting for its threads to acknowledge a user switch, and when the container starts unprivileged, Postgres backends never wake from their latches. CI showed it directly: AppArmor `DENIED` lines for `gosu` and `postgres` signals, and every variant with no-new-privileges hung while the same container without the flag, or with AppArmor off, came up.

## What Changes

- Restore containers start as the image's unprivileged `postgres` user instead of root. The official entrypoint then never switches users, and the server, `initdb` and the agent's own commands inside the container (writing the dump, `pg_restore`, disk usage) all run as that user.
- Restore containers get no Linux capabilities at all. Today all are dropped and five are added back (`CHOWN`, `DAC_OVERRIDE`, `FOWNER`, `SETGID`, `SETUID`) because the entrypoint needed them to switch from root; starting as `postgres` makes them unnecessary.
- The agent stops requesting no-new-privileges for restore containers. With no capabilities left in the bounding set, a process that exploits a setuid binary in the image gets user ID 0 with no capabilities, so the flag no longer guards a privilege the container could gain. AppArmor, seccomp, namespaces and the memory, process and CPU limits stay as they are.
- The end-to-end privileges test checks the new state: the first process runs as a non-root user and its capability bounding set is empty.
- The temporary Podman diagnostics are removed once CI is green: the diagnostic script, its CI step, the debug-only workflow and the debugging branch's exclusion from the main CI workflow. The improved readiness-timeout message stays.
- The still-open change `run-verification-on-podman` hands its unfinished section on the Podman CI failure over to this change.

No flag, configuration field, API or report field changes. An image configured through `verification-pg-image-repo` now has to ship a `postgres` user whose entrypoint works without root, which every image built on the official Postgres image does, the bundled verification image included.

## Capabilities

### New Capabilities

None.

### Modified Capabilities

- `verification-container-resources`: the reduced-privileges requirement changes from "five capabilities and no-new-privileges" to "starts as an unprivileged user with no capabilities"; the Podman requirement extends to hosts that confine containers with AppArmor; the scenario that lists the controls kept without a CPU limit names the new controls.

## Impact

- Code: `agent/verification/internal/features/container` (the container request and its unit tests), `agent/verification/e2e/scripts/test-container-privileges.sh`, the verification image's Dockerfile comment.
- CI: `.github/workflows/ci-release.yml` returns to its state before the debugging, and `.github/workflows/debug-podman.yml` and `agent/verification/e2e/diagnose-podman-ports.sh` are deleted.
- OpenSpec: `openspec/changes/run-verification-on-podman` closes its section 6 by pointing here.
- Rules this change answers to: `AGENTS.md` and `agent/verification/AGENTS.md`.
- Backend, frontend, the agent-to-backend API and the verification image's contents are untouched.

## Out of scope

- Fixing or working around Ubuntu's AppArmor profiles or Podman's container profile on the host.
- Turning off AppArmor or seccomp for restore containers.
- Rootless Podman in CI, Podman on macOS or Windows, and other container engines.
- Blocking outbound network access from restore containers.
- Running the full end-to-end suite on Podman.
