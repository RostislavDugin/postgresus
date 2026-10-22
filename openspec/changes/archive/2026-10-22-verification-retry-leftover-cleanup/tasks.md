## 1. Label-filtered removal routine

- [x] 1.1 Replace the engine's `ListManaged(ctx, agentID)` with a container listing that takes a label set, add a network listing with the same signature, and pass every label as a separate `label` filter value. Both return only the resource ID and its verification ID, so cut `ManagedContainer` down to those fields and delete the network lookup. Until task 1.2 replaces it, the old purge compiles against the new container listing and skips its network step. Verify with `just build` in `agent/verification`.
- [x] 1.2 In one step, replace the container-driven startup purge with one routine and the test fake with a fake Docker host, then rewrite the existing purge tests against it. The routine lists containers by labels and removes them, then lists networks by labels and removes them, so a failure in one half does not skip the other. It treats a removal error that `IsNotFound` from `github.com/containerd/errdefs` recognizes as success and accepts the narrow engine interface. Make `github.com/containerd/errdefs` a direct dependency in `agent/verification/go.mod`. The fake host in `fake_engine_test.go` stores containers and networks with their labels and applies a label filter with AND semantics. It also implements `CreateNetwork` and records listing filters, removals and network creations in one ordered log, and it can be told to fail a listing, fail a removal or return `errdefs.ErrNotFound` for a removal. The fake's setup methods stay beside it in `fake_engine_test.go`, because a `testing.go` file is compiled without the test files and cannot see the fake. The label builders the tests share are the production helpers `Spawn` and the purge use, and the log-capturing helper joins `DiscardLogger` in the module's shared `internal/testutil`, so the package gets no `testing.go`. Rewrite the comments that tie the labels to the startup purge alone: the purge comment that says a leftover never collides with a fresh run, `dockerengine.go:26-27` and `manager.go:146-148`. Verify by rewriting the existing tests in `internal/features/container/purge_test.go` against the fake host and running `just test` in `agent/verification`.

## 2. Startup purge tests

Each test lives in `internal/features/container/purge_test.go` and is verified with `just test` in `agent/verification`.

- [x] 2.1 `Test_PurgeAgentContainers_WhenNetworkHasNoContainer_RemovesNetwork` pins that a network labeled with the agent's ID and no attached container is removed.
- [x] 2.2 `Test_PurgeAgentContainers_WhenContainerAndNetworkExist_RemovesContainerBeforeNetwork` pins the removal order.
- [x] 2.3 `Test_PurgeAgentContainers_WhenResourcesBelongToOtherAgentOrNoAgent_LeavesThemInPlace` pins that a network and a container labeled with another agent ID, and a network with no agent labels, remain on the fake host.
- [x] 2.4 `Test_PurgeAgentContainers_FiltersByAgentLabelOnly` pins that both listings request exactly the agent label filter.
- [x] 2.5 `Test_PurgeAgentContainers_WhenNetworkListFails_LogsAndRemovesContainers` pins that a failed network listing does not stop the purge and does not return an error.
- [x] 2.6 `Test_PurgeAgentContainers_WhenRemovalReturnsNotFound_ContinuesWithRemainingResources` pins that a not-found removal is treated as removed and the purge goes on.
- [x] 2.7 `Test_PurgeAgentContainers_WhenContainerListFails_LogsAndRemovesNetworks` pins that a failed container listing still lets the purge remove the agent's networks.

## 3. Cleanup before an attempt

- [x] 3.1 Move the step of `Spawn` that clears leftovers and creates the network into a function that takes the narrow engine interface plus `CreateNetwork`, runs the removal routine with the agent and verification labels, and returns an error saying the leftovers of an earlier attempt could not be removed when the routine fails. Make `Spawn` call it. Verify with `just build` in `agent/verification`.

## 4. Cleanup-before-attempt tests

Each test lives in a new `internal/features/container/leftovers_test.go`, or beside the function from 3.1 if it lands in another file, and is verified with `just test` in `agent/verification`.

- [x] 4.1 `Test_PrepareAttemptNetwork_WhenPreviousAttemptLeftNetwork_RemovesItAndCreatesNetwork` pins the first scenario of the retry requirement. A comment on the test names issue #832, which the project rules allow for a test that pins an issue's behavior.
- [x] 4.2 `Test_PrepareAttemptNetwork_WhenPreviousAttemptLeftContainerAndNetwork_RemovesContainerThenNetworkThenCreates` pins the order across removal and creation.
- [x] 4.3 `Test_PrepareAttemptNetwork_WhenLeftoverAlreadyRemoved_CreatesNetwork` pins that a removal returning `errdefs.ErrNotFound` lets the attempt proceed.
- [x] 4.4 `Test_PrepareAttemptNetwork_WhenNoLeftovers_CreatesNetworkWithoutRemovals` pins the first-attempt scenario.
- [x] 4.5 `Test_PrepareAttemptNetwork_WhenLeftoverRemovalFails_ReturnsLeftoverErrorAndCreatesNoNetwork` pins the error message and that `CreateNetwork` is never called.
- [x] 4.6 `Test_PrepareAttemptNetwork_WhenListingFails_ReturnsLeftoverErrorAndCreatesNoNetwork` pins the same behavior for a failed listing.
- [x] 4.7 `Test_PrepareAttemptNetwork_WhenOtherVerificationRuns_LeavesItsResourcesInPlace` pins that the container and network of another verification of the same agent remain on the fake host.
- [x] 4.8 `Test_PrepareAttemptNetwork_WhenOtherAgentHasSameVerification_LeavesItsResourcesInPlace` pins that resources with the same verification label but another agent ID remain on the fake host.
- [x] 4.9 `Test_PrepareAttemptNetwork_FiltersByAgentAndVerificationLabels` pins that both listings request exactly the agent label and the verification label.

## 5. End-to-end checks on real Docker

- [x] 5.1 Add `agent/verification/e2e/scripts/test-restore-retry-leftover.sh`, modeled on `test-restore-success.sh`. After the agent has started and its startup purge has finished, and before the claim, it creates a network named `databasus-verif-<id>` labeled with the agent ID and that verification ID, so the startup purge cannot remove it first. It asserts with `docker network ls --filter name=databasus-verif-<id>` that the network exists right before it assigns the same verification ID through `POST /mock/set-claim`, then asserts a successful report and runs `leak_check`. A comment names issue #832. Register the script in `e2e/scripts/run-all.sh`. Verify with `just e2e` in `agent/verification`.
- [x] 5.2 Extend `agent/verification/e2e/scripts/test-startup-purge.sh` to also seed a network labeled with the agent ID and no container, and a network labeled with another agent ID. Assert that the first is gone after the agent starts and the second remains, then remove the second in the script's cleanup. Verify with `just e2e` in `agent/verification`.

## 6. Module checks

- [x] 6.1 Run `just lint` in `agent/verification` and fix every finding.

## Change verification

```sh
just --justfile agent/verification/Justfile test
```
