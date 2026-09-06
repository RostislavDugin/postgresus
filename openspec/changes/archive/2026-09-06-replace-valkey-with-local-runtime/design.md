## Context

See `proposal.md` for the motivation and scope.

The cache package currently constructs a process-wide Valkey client from environment settings and exposes that client to callers (`backend/internal/util/cache/cache.go:13-48`). Its typed wrapper owns JSON serialization but also embeds the Valkey client and hides write and transport failures (`backend/internal/util/cache/utils.go:19-126`). Single-use restore tokens depend on atomic read-and-delete (`backend/internal/features/backups/backups/download/restore_token/store.go:40-61`), while the download lock currently performs a non-atomic read followed by a write (`backend/internal/features/backups/backups/download/stream_guard/tracker.go:34-45`).

Publish-subscribe code shares the cache package and contains a Valkey-specific readiness handshake (`backend/internal/util/cache/pubsub.go:18-200`). Its only production consumer is task cancellation (`backend/internal/features/tasks/cancellation/cancel_manager.go:13-68`). Rate limiting also shares the cache package and represents each attempt as a separate Valkey key before scanning the namespace (`backend/internal/util/cache/rate_limiter.go:12-82`).

Valkey is part of configuration validation, application startup, healthchecks, image construction, container startup, Compose and CI (`backend/internal/config/config.go:54-67`, `backend/internal/config/config.go:215-223`, `backend/cmd/main.go:92-100`, `backend/internal/features/system/healthcheck/service.go:31-44`, `Dockerfile:112-143`, `docker/start.sh:348-374`, `docker-compose.yml:60-73`, `.github/workflows/ci-release.yml:284-299`). The embedded server is capped at 256 MiB and uses all-keys LRU eviction (`docker/start.sh:350-356`).

The Helm chart defaults to one replica but currently exposes `replicaCount` and passes it directly to the StatefulSet (`deploy/helm/values.yaml:13-14`, `deploy/helm/templates/statefulset.yaml:10`). Local transient state is correct only when one application process owns an installation.

The approach is constrained by `AGENTS.md` naming, error-handling, no-shim, security and testing rules; `backend/AGENTS.md` file organization, dependency injection, controller-test, logging and concurrency rules; `website/AGENTS.md` six-language synchronization rules; and `adr/AGENTS.md` guidance to preserve historical decisions. The user's direction to update existing ADRs without creating a new ADR is part of this change's scope.

## Goals / Non-Goals

**Goals:**

- Give cache, publish-subscribe and rate limiting separate provider contracts that do not expose Valkey types.
- Make the default providers safe under concurrent access and test their contracts independently from feature tests.
- Preserve the existing TTLs, sliding-window thresholds, task-cancellation behavior and atomic single-use token consumption.
- Make stream-lock acquisition atomic instead of carrying the current read-then-write race into the local provider.
- Bound transient cache memory without evicting live locks or tokens.
- Make one application process an enforced property of supplied deployments.

**Non-Goals:**

- Coordinate state between application processes or Kubernetes replicas.
- Persist cache entries, subscriptions or counters across process restarts.
- Ship a Valkey adapter in this change.
- Preserve old environment variables, constructors, package aliases or the Helm replica setting.
- Change public HTTP payloads, rate-limit thresholds or backup formats.

## Decisions

### Keep three narrow provider contracts

`backend/internal/util/cache` will declare a byte-oriented `Store` contract and provide `MemoryStore`. A generic `JSONStore[T]` will own namespaces, JSON encoding and the default expiration used by typed callers. The package declaration will become `cache`; the existing `cache_utils` name and Valkey-based constructors will be deleted.

`backend/internal/util/pubsub` will declare a `Broker` contract and provide `LocalBroker`. Publications will use a named `Publication` value and subscriptions will accept a `MessageHandler`. `backend/internal/util/ratelimiter` will declare a `Counter` contract and provide `MemoryCounter`. Feature constructors will accept these contracts, while each package will expose one process-wide local provider through an explicit `Get...` accessor for composition roots.

The generic `TaskCancelManager` type will be deleted. Task cancellation will separate the local `Registry`, whose methods register, unregister and cancel registered tasks, from the `Requester`, which publishes cancellation requests and the `Listener`, which delivers subscribed requests to the registry. Variables and constructor fields will keep the full domain names, such as `taskCancellationRegistry`, so the short package-local type names do not hide their responsibility.

The rejected alternative is one broad transient-state interface. Cache storage, ordered message delivery and atomic request counting have different failure and concurrency rules, so a combined interface would expose unrelated methods to every caller. Keeping pub/sub and rate limiting inside `util/cache` was also rejected because it would preserve the current misplaced responsibilities.

### Use serialized values at the cache-provider boundary

`Store` operations will accept `context.Context` and return errors. Writes will use a named entry parameter containing the key, copied payload and positive lifetime. The contract will include get, set, create-if-absent, read-and-delete, delete and clear operations. `JSONStore[T]` will translate between typed values and bytes, accept custom lifetimes through a named `ExpiringValue[T]` parameter and return serialization or provider errors to its caller.

`MemoryStore` will copy byte slices on both write and read so callers cannot mutate shared memory outside the store lock. One mutex will make create-if-absent and read-and-delete indivisible. An injected time source will make expiration-boundary tests deterministic.

The rejected alternative is storing arbitrary Go values directly. That would share mutable pointers across goroutines and would make a later remote adapter behave differently. Preserving the current methods that swallow errors was rejected because security-sensitive callers must distinguish a missing value from a failed operation and a future remote provider can fail even when the local provider normally cannot.

### Reject writes at the cache budget instead of evicting live entries

The production `MemoryStore` will use a 256 MiB payload budget, matching the current embedded Valkey ceiling. Accounting will cover stored keys and serialized values; it will not claim to reproduce allocator overhead. Reads will remove an expired requested entry. Writes will sweep expired entries before checking the budget. If the new value still does not fit, the write will return a capacity error and leave all live entries unchanged.

No cleanup goroutine is needed. Lazy expiration plus a full expired-entry sweep only when capacity is needed keeps idle behavior simple, avoids lifecycle leaks in tests and keeps memory bounded.

The rejected alternative is LRU eviction, including the current Valkey policy. A cache miss is acceptable for derived data, but silently evicting a live stream lock could admit a second stream. An unbounded map was rejected because attacker-controlled identifiers and issued tokens could grow process memory. A third-party local-cache library was rejected because the required atomic and no-live-eviction contract is small enough to implement and test without another dependency.

### Let each cache consumer choose fail-open or fail-closed behavior

Derived restore metadata may log a provider error and use the original request configuration carried into the detached restore worker. Restore-token issuance and consumption, stream-lock acquisition and rate-limit evaluation will propagate errors because treating those failures as cache misses can weaken single-use, exclusivity or request-limit guarantees. The stream guard will use create-if-absent for lock acquisition and will keep its existing heartbeat TTL behavior.

The rejected alternative is one package-wide fallback policy. The cache package cannot know whether an entry is an optimization or an authorization and concurrency guard.

### Deliver local pub/sub messages through registered subscription loops

`LocalBroker` will register a subscription before `Subscribe` returns. Each subscription will own an unbuffered mailbox and a loop that invokes its handler sequentially. Publishing will reserve matching subscriptions while holding the broker map lock, then submit the message to each mailbox while selecting on the publication context. Closing or canceling a subscription will remove it from later snapshots and drain publications already reserved for it before stopping. Close operations will return after recording closure so handlers can close their own subscription or broker without waiting on their delivery loop. A per-channel publication lock will preserve order when publishers overlap. A successful publish means every subscription active at the snapshot accepted one copy; it does not require every handler to finish.

Handler panics will be recovered at the subscription boundary so one consumer cannot terminate the process or another subscription. Canceling the subscription context, closing the subscription or closing the broker will stop later delivery. The task-cancellation requester and listener will receive `pubsub.Broker` through their constructors and retain the existing setup hook, but the Valkey readiness markers and sleeps in cancellation tests will disappear.

The rejected alternative is calling handlers synchronously inside `Publish`, because a handler cannot then be interrupted through the publication context. Starting one goroutine per message was rejected because it gives no backpressure and can reorder messages. Buffered mailboxes with silent overflow were rejected because dropping a task-cancellation message would report success without canceling the task.

### Implement rate limiting as a domain provider over the local cache

`ratelimiter.Counter` will accept a named request containing scope, identifier, maximum attempts and window duration. Its operation will be named to state both effects: it records the attempt and reports whether the request is allowed. `MemoryCounter` will use a private `JSONStore` namespace and one mutex around read, prune, append, cap and write. It will keep at most `limit + 1` recent timestamps per scope and identifier. Older timestamps cannot change an allowed result once the newest `limit + 1` attempts are still in the window.

Using the cache gives rate-limit entries the same TTL cleanup and memory budget without adding cache-specific operations to the public rate-limiter contract. If the cache cannot record the attempt, protected endpoints will fail closed and log the error without logging the identifier when it contains user data.

The rejected alternative is exposing an atomic mutation callback on `cache.Store`; a remote provider cannot execute an arbitrary Go callback atomically. A separate unbounded map was rejected because it would duplicate expiration and memory accounting. Fixed windows and token buckets were rejected because they change the current sliding-window boundary behavior. A future Valkey-backed `Counter` can implement the same contract with a server-side atomic operation without using `cache.Store`.

### Test contracts and local concurrency separately

Each utility package will have a contract test suite parameterized by a provider factory. The local provider will run through that suite and a future Valkey provider can run the same cases. Local implementation tests will use a controllable time source for exact expiration boundaries. Concurrency tests will cover one-winner cache consume and reserve operations, one remaining rate-limit allowance, ordered pub/sub delivery, cancellation, close behavior and handler panic isolation.

Feature tests will keep coverage at the observable boundary. Task-cancellation tests will create separate `Registry`, `Requester` and `Listener` instances that share one local broker and will use channels or contexts instead of sleeps. Existing user-controller rate-limit tests will verify fail-closed error handling and download and restore tests will cover atomic lock and token behavior. Focused packages will run with the Go race detector before the full backend suite.

The rejected alternative is testing only concrete maps. Those tests would not define the behavior a future provider must preserve. Keeping timing sleeps was rejected because it hides registration races and makes failures depend on machine load.

### Remove Valkey from runtime and test orchestration

The backend will remove Valkey configuration fields, validation, logical database selection, test cache namespaces, startup connection checks and external cache cleanup. Test worker slots will continue isolating PostgreSQL metadata databases, while each Go test binary will naturally own separate local transient state.

The runtime image will remove the Valkey repository and package. Container startup will proceed from storage validation to PostgreSQL bootstrap without starting a cache server. Compose and CI will remove the Valkey service and readiness wait. The deep healthcheck will stop probing an external cache connection because the local providers have no network dependency. The filesystem suite will assert that only the application and PostgreSQL run under the selected account.

The rejected alternative is leaving an unused Valkey service available for rollback. That keeps the image, CVE surface, startup dependency and configuration that this change removes. Rollback uses an earlier image that still contains its own required service.

### Enforce one replica and update current architecture notes

The Helm StatefulSet will render one replica directly. The chart will delete `replicaCount` from values and documentation rather than accepting an unsupported topology. Docker already starts one application process per container, so no new Docker setting is needed.

ADR-0006 will receive a dated note that its Valkey-based multi-node coordination is historical and that current installations use process-local providers. ADR-0013 will receive a dated note that test binaries now isolate local cache, pub/sub and rate-limit state by process, while worker slots remain responsible for metadata-database isolation. Their original decision bodies will remain unchanged and no new ADR will be created. This is an explicit user-directed exception to the default `adr/AGENTS.md` guidance to record a replacement decision in a new ADR with supersession markers.

The rejected alternative is documenting `replicaCount: 1` as a recommendation while leaving higher values valid. That permits a deployment whose cancellation messages and rate limits diverge between pods. Rewriting the ADR bodies was rejected because they record the reasoning that applied when those decisions were made.

## Risks / Trade-offs

- [A manually assembled multi-process deployment has divergent transient state] -> State the single-process contract, hard-code one Helm replica and remove supported configuration that suggests otherwise.
- [The cache reaches its payload budget] -> Reclaim expired entries, reject the new write without evicting live entries, log capacity failures and fail closed for tokens, locks and rate limits.
- [A slow pub/sub subscriber delays publication] -> Keep explicit backpressure, honor publication cancellation while submitting and keep current handlers small. A future heavy consumer must hand work to its own bounded queue.
- [Concurrent access introduces races or mutable aliases] -> Serialize state changes under package-owned locks, copy cache payloads and run focused tests with `-race`.
- [Removing the external health probe hides transient-state failure] -> Cover provider construction and contracts in process; retain deep checks for disk, tools, PostgreSQL, schedulers and verification agents.
- [Operators keep deleted Valkey or Helm values] -> Document the breaking removals. Unknown old environment values are unused and the old Helm replica value no longer changes rendered topology.
- [Rollback restores the old runtime dependency] -> Roll back with the earlier image and chart, which contain their own Valkey package and startup logic. No transient data migration is required because the production Valkey configuration disables persistence.

## Migration Plan

1. Add the three provider contracts, local implementations, contract tests and deterministic concurrency tests.
2. Move callers to the new constructors and error contracts, including atomic stream locks and fail-closed rate-limit errors.
3. Remove Valkey configuration, dependencies, healthchecks, startup code, test cleanup, Compose service and CI wait.
4. Enforce one Helm replica, update container tests, Helm documentation, all six Advanced Config pages, current OpenSpec contracts and the two relevant ADR notes.
5. Run focused race tests, the full backend suite and linter, website lint and build, strict OpenSpec validation and the Docker filesystem suite.

Deployment requires no stored-data migration. Existing cache entries, rate-limit attempts, subscriptions, locks and tokens are transient and start empty with the new process.
