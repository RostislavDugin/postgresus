## Why

When a verification attempt fails for an agent-side reason, the backend requeues it under the same verification ID, and the agent names the isolated Docker network of every attempt after that ID. If the previous attempt left its network behind, each retry fails with "network with name databasus-verif-<id> already exists" until the retry limit runs out, and the user sees this network error instead of the cause of the first failure (issue #832). Restarting the agent does not help, because the startup purge finds networks only through the containers attached to them.

## What Changes

- Before it creates the network and container of an attempt, the agent removes whatever an earlier attempt of the same verification left on the Docker host under this agent's ID: the container first, then the network. A resource that is already gone counts as removed.
- If such a leftover cannot be removed, the attempt fails with an error that names the leftover cleanup, instead of the misleading "network already exists".
- The startup purge also removes this agent's networks that no container is attached to, found by the agent's label rather than through containers.
- The comment on the startup purge that says a leftover never collides with a fresh run is corrected, because a retry reuses the verification ID.

No caller, config, schema or API changes, and nothing is **BREAKING**.

## Out of scope

- Retrying the network removal inside the container teardown when the container removal has not finished in time.
- Backend changes: the requeue logic, the attempt limit, the verification ID reuse and the failure reporting.
- Reordering the failure report and the container teardown inside a job run.
- Leftovers labeled with a different agent ID that collide by name on a shared Docker host.
- UI changes, including how the first failure's cause is shown.

## Capabilities

### New Capabilities

- `verification-container-cleanup`: how the verification agent clears Docker containers and networks left by earlier attempts and earlier agent processes, so that they never block a new verification attempt.

### Modified Capabilities

None.

## Impact

- Code: the container feature of the verification agent (`agent/verification/internal/features/container/`), namely the job spawn, the startup purge and the Docker engine methods that list and remove managed resources.
- Rules: the change answers to `agent/verification/AGENTS.md` and the root `AGENTS.md`.
- Operations: the first startup after the upgrade also removes networks orphaned by earlier agent versions.
