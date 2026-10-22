## Context

See proposal.md for the motivation. The current state that shapes the approach:

- `Spawn` names the network of every attempt `databasus-verif-<verification ID>` and creates it with no check for an existing one (`agent/verification/internal/features/container/manager.go:155`). Both the network and the container carry `LabelAgentID` and `LabelVerificationID` (`manager.go:149-152`, `dockerengine.go:26-29`).
- `Terminate` removes the container and then the network under a 30-second deadline (`container.go:95-108`, `container.go:17`). When the container removal fails or overruns, the network removal fails too, because the container is still attached. The runner only logs that error (`runner/runner.go:183-187`).
- The runner reports a failure from inside the job run, and the teardown runs later as a deferred call (`runner/runner.go:178`, `runner/runner.go:183`). The pool releases the job slot only after the run returns (`runner/pool.go:34`). With more than one slot, another slot can therefore claim the requeued attempt while the previous attempt is still tearing down. This mechanism is inferred from the code and was not reproduced.
- The startup purge lists containers by `LabelAgentID` and removes each container's first attached network (`purge.go:26-58`, `dockerengine.go:316-345`). A network with no container attached is invisible to it, which contradicts its own comment at `purge.go:21-23`.
- `Manager` holds the concrete `*dockerEngine` (`manager.go:55-61`). The purge stays unit-testable because it accepts the narrow `purgeEngine` interface (`purge.go:11-15`), faked in `fake_engine_test.go`.

The approach answers to `agent/verification/AGENTS.md`: test naming `Test_WhatWeDo_WhichConditions_WhatWeExpect`, tests beside the code, shared test helpers in `internal/testutil` (a package `testing.go` cannot see a fake defined in a `_test.go` file, and the lint skips test files, so unexported test-only helpers there would fail it), IDs as key-value pairs in logs, and the `slices` helpers instead of manual loops. No ADR in `adr/` covers the verification agent's Docker resources.

## Goals / Non-Goals

**Goals:**

- One removal routine serves both the startup purge and the cleanup before an attempt, and the two differ only in the label filter and in how they treat a failure.
- The removal is idempotent: a resource that is already gone counts as removed, so a racing teardown cannot fail the new attempt.

**Non-Goals:**

- Detecting why a previous attempt leaked. The cleanup makes the retry independent of the cause.
- Changing the teardown deadline or the order of reporting and teardown in the runner.

## Decisions

### Clear leftovers by label before every attempt

`Spawn` removes the containers and then the networks labeled with this agent's ID and the attempt's verification ID, before `CreateNetwork`.

Alternatives considered:

- Remove only an existing network with the same name. Rejected: in the suspected leak the old container is still attached, so Docker refuses to remove the network and the retry fails exactly as before.
- Reuse an existing network instead of creating one. Rejected: it keeps a half-torn-down container on the network and makes the attempt depend on the state of the previous one.
- Add the attempt number to the network name. Rejected: the agent never learns the attempt number from the assignment, and every attempt would leak another network instead of the first one blocking the rest.

### One label-filtered removal routine for both paths

A single function lists this agent's containers and networks that match a label filter, removes the containers, then removes the networks. The startup purge calls it with the agent label only. `Spawn` adds the verification label. The purge no longer derives networks from containers. `ListManaged(ctx, agentID)` (`dockerengine.go:316-345`) is replaced by a container listing that takes the label set, beside a matching network listing, so no agent-only listing is kept beside the label-filtered one. The listings return only what the routine reads, the resource ID and its verification ID for logging, so `ManagedContainer` loses `NetworkID` and `CreatedUnix`, and its network lookup goes away. The two comments that tie the labels to the startup purge alone (`dockerengine.go:26-27`, `manager.go:146-148`) are rewritten, because the cleanup before an attempt depends on both labels too.

At startup the routine lists and removes containers, then lists and removes networks, and a failure in one half does not skip the other. After a failed container listing, a network that still has a container attached fails its removal and is logged, which is the purge's existing best-effort behavior.

Alternatives considered:

- Keep the container-driven purge and add a second pass over orphaned networks. Rejected: it removes the same networks through two paths and keeps the field that hides orphaned networks.
- Write a separate cleanup for `Spawn`. Rejected: it would duplicate the list-and-remove logic and its not-found handling.

### Failure handling differs by path

The startup purge stays best-effort: it logs a failed listing or removal and moves on, so a transient Docker error never blocks startup (`purge.go:17-24`). The cleanup before an attempt returns an error, and `Spawn` fails with a message about the leftovers of an earlier attempt. That failure is still a pre-restore setup failure, so the backend requeues it as before.

Alternative considered: log the failure and let `CreateNetwork` proceed. Rejected: when the leftover survived, the attempt fails anyway with the misleading "already exists" error that this change removes.

### Treat "not found" as removed

The removal of a container or network that Docker reports as missing counts as success. Docker reports this condition as an error that `github.com/containerd/errdefs` classifies with `IsNotFound`. The removal routine applies this check, not the Docker engine, so a fake engine returning `errdefs.ErrNotFound` exercises it in unit tests. The module is already in `agent/verification/go.mod` as an indirect dependency and becomes a direct one, with no new download.

Alternative considered: match the error text. Rejected: the text is not a stable contract of the Docker API.

### Test through the narrow engine interface

The removal routine takes an interface with the listing and removal methods it uses, which extends the existing `purgeEngine` pattern. The step of `Spawn` that clears leftovers and then creates the network moves into a function that takes the same interface plus `CreateNetwork`, because `Spawn` holds the concrete engine and cannot be unit-tested. That lets a test pin that the cleanup runs before the network is created, and that no network is created when the cleanup fails.

The fake engine stores containers and networks with their labels and applies the requested label filter with AND semantics, as Docker does. A test then asserts which resources remain on the fake host, so the scenarios that leave other agents' and other verifications' resources untouched check the outcome rather than the filter arguments. One test pins the label filter values that the routine requests, since the fake's matching only proves the filter is right if the routine asks for the right labels. The Docker engine's new listing method stays thin, and the existing tests do not exercise the engine against a real daemon either.

Real Docker is covered by the agent's existing end-to-end suite rather than by unit tests. `agent/verification/e2e` runs the built agent against a mock backend with the host's `/var/run/docker.sock` mounted, the mock takes a chosen verification ID through `POST /mock/set-claim`, `test-startup-purge.sh` already seeds a labeled leftover, and `leak_check` in `e2e/scripts/lib.sh` already checks networks. Two scenarios join it: a retry against a pre-created `databasus-verif-<id>` network, and an orphaned network plus a foreign-agent network at startup. They check the label filter semantics and the not-found errors of a real daemon, which the fake only imitates.

Alternative considered: a Docker-backed test inside `just test`. Rejected: `just test` runs no Docker-backed tests today, and the e2e suite already provides that environment. The e2e checks stay out of the change verification because `just e2e` rebuilds its images without cache and runs every scenario, and the change verification section allows only the unit test command.

## Risks / Trade-offs

- [The previous attempt of the same verification is still running in another slot when the cleanup removes its container] → That attempt is stale, since the backend already requeued the verification, so ending it is the desired outcome. Its failure report may reach the backend after the retry starts, and this change leaves the backend's handling of such a report as it is.
- [Docker applies label filters with AND semantics only when each label is a separate filter value] → The listing passes the agent label and the verification label as two values of the `label` filter, which Docker combines with AND. A unit test pins the filter the routine requests.
- [A leftover labeled with a different agent ID still collides by name] → Out of scope per proposal.md. The failure then keeps today's message.

## Migration Plan

No data or config migration. The first startup after the upgrade removes networks that earlier agent versions orphaned. Rolling back restores the previous behavior with no cleanup needed.
