## Context

See proposal.md for why. The relevant current state, on the branch `fix/podman-port-diagnostics`:

- `Manager.buildSpec` requests every restore container with `NoNewPrivileges: true`, `CapDropAll: true`, `CapAdd: minimalCaps` and `User: containerUser` (`agent/verification/internal/features/container/manager.go:215-218`). `minimalCaps` holds `CHOWN`, `DAC_OVERRIDE`, `FOWNER`, `SETGID` and `SETUID` (`manager.go:34`), the set the official entrypoint needs to prepare the data directory and switch from root with `gosu`. `containerUser` is `postgres` (`manager.go:41`); it was added on this branch (commit a8fe37a6) as a first attempt and did not fix CI by itself.
- `toContainerConfigs` turns `NoNewPrivileges` into `SecurityOpt: ["no-new-privileges"]` and `CapDropAll` into `CapDrop: ["ALL"]`, and copies `CapAdd` and `User` (`dockerengine.go:155-190`). `SpawnSpec` carries the fields (`spawnspec.go:40-43`). The unit tests pin all of it (`postgres_test.go:18-21`, `:38-40`, `dockerengine_test.go:92-93`).
- `dockerEngine.Exec` sets no user (`dockerengine.go`, `Exec`), so the agent's commands in the container (`dd` writing the dump to `/tmp`, `pg_restore`, `du`) run as the container's user.
- `test-container-privileges.sh` reads `/proc/1/status` of a running restore container and requires `CapBnd` `00000000000000cb` and `NoNewPrivs` `1` (`agent/verification/e2e/scripts/test-container-privileges.sh:19`, `:66-69`).
- The diagnostics behind the root cause are temporary: `agent/verification/e2e/diagnose-podman-ports.sh`, the step `Diagnose Podman port publishing` in `e2e-verification-agent-podman` (`.github/workflows/ci-release.yml:265-267`), the workflow `.github/workflows/debug-podman.yml`, which runs only that job on the debugging branch, and the exclusion of that branch from the push trigger of `ci-release.yml` (`ci-release.yml:6`).
- `waitForReady` reports the last ping error on a readiness timeout (`manager.go:239`). That belongs to `run-verification-on-podman` (its task 6.1) and stays.
- `run-verification-on-podman` is not archived. Its section 6 tracks the Podman CI failure with two open tasks, 6.3 (fix the cause) and 6.4 (remove the diagnostics), and its delta spec adds the two requirements this change modifies.

Diagnostic runs on `ubuntu-24.04` with Podman 4.9.3, `crun` 1.14.1 and AppArmor enabled (runs 39831270927 and 39835622435) showed:

| Container request | Postgres came up |
|---|---|
| root start, no-new-privileges (the agent before this branch) | no: `gosu` spins in `sched_yield` |
| `postgres` start, no-new-privileges (the branch now) | no: a backend sleeps in `authentication` forever |
| root start, no-new-privileges, AppArmor unconfined | yes |
| root start, no-new-privileges, seccomp unconfined | no |
| root start, no no-new-privileges | yes |
| `postgres` start, no no-new-privileges | yes |

The kernel log held AppArmor `DENIED` signal lines for `gosu` (`rtmin+1`, glibc's thread signal for `setuid`) and for `postgres` (`urg`, `usr1`, `term`), each with the peer label `containers-default-0.57.4-apparmor1//&crun`.

Constraints come from `agent/verification/AGENTS.md`: pure translation functions get unit tests, engine-dependent behavior is covered in `e2e/`, and tests mirror the source split.

## Goals / Non-Goals

**Goals:**

- Verification completes on rootful Podman with `crun` on an AppArmor host, and stays green on Docker and on rootless Podman.
- The restore container starts as an unprivileged user with an empty capability bounding set, so even a process that reaches user ID 0 inside it holds no capability.

**Non-Goals:**

- Changing anything on the host: AppArmor profiles, Podman's configuration or `crun`.
- A runtime check of the applied privileges; the end-to-end test keeps that role, as `run-verification-on-podman` decided.

## Decisions

### Start as `postgres` with no capabilities, and drop no-new-privileges

The container request sets `User: postgres`, `CapDrop: ["ALL"]`, no `CapAdd` and no `no-new-privileges`. The official entrypoint skips `gosu` and the ownership fixes when it does not start as root, so `initdb` and the server run as `postgres` from the first instruction. No step in the container needs a capability, so the bounding set can be empty.

no-new-privileges exists to stop a process from gaining privileges through a setuid or file-capability binary. With an empty bounding set, such a binary can still give a process user ID 0, but the kernel grants it no capability, so it cannot pass permission checks, load modules, change namespaces or do anything else root's capabilities allow. What it keeps is ownership of root-owned files inside a container that is deleted after the job. AppArmor, seccomp, the namespaces and the limits stay in force, and they are what stands between the container and the host.

`SpawnSpec` loses `NoNewPrivileges` and `CapAdd`, and `minimalCaps` is deleted, since nothing sets them any more. `CapDropAll` and `User` stay.

Rejected alternatives:

- **Keep no-new-privileges and start as `postgres`.** Tested: Postgres's own processes signal each other, and AppArmor denies those signals under the stacked label too.
- **Request `apparmor=unconfined`.** It fixes the hang by removing the stronger control, and the agent would ask for less confinement than the engine applies by default.
- **Drop no-new-privileges only on Podman hosts with AppArmor.** The agent would have to detect the host's security modules and runtime through the engine API and carry two container configurations. A Docker host configured with `crun` would hit the same stacking without matching the detection.
- **Drop no-new-privileges and keep the five capabilities.** A process that gains user ID 0 through a setuid binary would then hold `DAC_OVERRIDE`, `FOWNER`, `SETUID` and the others inside the container, for no benefit, because nothing needs them once the container starts unprivileged.
- **Ship an AppArmor profile, or tell users to change theirs.** The agent does not own the host's AppArmor configuration, and a manual step per host is what this change avoids.

### The agent's commands run as `postgres` too

`Exec` keeps not setting a user, so `dd`, `pg_restore` and `du` run as the container's user. `dd` writes the dump to `/tmp`, which any user may write. `pg_restore` connects over the local socket as the database superuser, which does not depend on the operating-system user. `du` reads the data directory and `/tmp`, both owned by `postgres`. The full Docker suite passed locally with this on the branch, TimescaleDB and the disk-budget test included.

The exec request moves into a pure function, as the container request did in `run-verification-on-podman`, and a unit test pins that it names no user.

Rejected alternative: run execs as root through the exec request. Root without capabilities gains nothing for these commands, and it would put a root process back into the container.

### Check the new privileges end to end

`test-container-privileges.sh` keeps its approach: it holds the backup stream, reads `/proc/1/status` of the running container and then lets the job finish. It now requires the first `Uid` field to be non-zero and `CapBnd` to be `0000000000000000`. It no longer checks `NoNewPrivs`. The unit tests assert that the request sets the user, drops all capabilities, adds none and sets no security options.

### Verify on CI as part of the tasks, then restore the CI files

The failure needs rootful Podman with `crun` on an AppArmor host, which no local check reproduces: the development host runs Fedora with SELinux and rootless Podman. The user asked for the CI run to be part of the tasks, which overrides the project's default of keeping CI runs out of tasks. The apply phase therefore pushes to `fix/podman-port-diagnostics`, where `debug-podman.yml` runs only the Podman job, and reads its result with `gh`. Once that job is green, it removes the diagnostics, restores `ci-release.yml` to its state before the debugging, deletes `debug-podman.yml`, pushes again and checks that the full `CI and Release` workflow passes on the branch.

### Hand over section 6 of `run-verification-on-podman`

Task 6.3 of that change is the fix this change makes, and task 6.4 is its cleanup. Both are replaced by a line that points here, and its risk entry about the red Podman job is updated the same way. `run-verification-on-podman` keeps its delta as it was proposed, with five capabilities and no-new-privileges, and this change's MODIFIED delta replaces that wording. So `run-verification-on-podman` is archived first and this change right after it, in the same step, so the main spec never ships the superseded wording.

## Risks / Trade-offs

- [An image set through `verification-pg-image-repo` without a `postgres` user, or with an entrypoint that needs root] → Container creation or startup fails, and the job is reported as a setup failure with the engine's error. Every image built on the official Postgres image works, and the proposal states the expectation.
- [The data directory in the anonymous volume is not writable by `postgres` on some engine] → The engines copy the image directory's ownership into a new anonymous volume, and the official image creates it owned by `postgres`. The end-to-end tests cover Docker and rootful Podman, and rootless Podman passed locally.
- [The TimescaleDB image is Alpine-based, with a `postgres` user of a different ID] → The user is resolved by name inside the image; the TimescaleDB end-to-end test already passed on Docker with the user switch.
- [Without no-new-privileges, a flaw in a setuid binary of the image gives user ID 0 inside the container] → It comes with no capabilities, inside an ephemeral container under AppArmor, seccomp and namespaces.
- [The diagnosis rests on `ubuntu-24.04`; other AppArmor hosts may stack labels differently] → The change removes the dependency on signals crossing a stacked label altogether, rather than matching one host's labels.

## Migration Plan

None. The change ships with the next agent release; agents on Docker keep working, and agents on rootful Podman with AppArmor start completing verifications. Rollback is reverting the commit; the agent stores no state this change touches.
