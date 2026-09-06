## 1. Planning checkpoint

- [x] 1.1 Run the mandatory plan reviewer against `proposal.md`, `design.md`, all delta specs, `AGENTS.md`, `backend/AGENTS.md`, `website/AGENTS.md` and `adr/AGENTS.md`; resolve every `CHANGES REQUIRED` finding before editing implementation files

## 2. Local cache

- [x] 2.1 Replace the Valkey client in `backend/internal/util/cache` with the `Store` contract and bounded `MemoryStore`; verify expiration, copied payloads, clear, capacity rejection, canceled operations and concurrent create-if-absent and read-and-delete through `go test -race ./internal/util/cache`
- [x] 2.2 Add the namespaced `JSONStore[T]` wrapper with explicit serialization and provider errors; verify its default and custom TTL, namespace isolation, malformed payload and error cases through `go test -race ./internal/util/cache`
- [x] 2.3 Move restore metadata, restore tokens and stream locks to the new cache contract, make lock acquisition atomic and verify fail-open metadata reload plus fail-closed token and lock behavior through `go test -race ./internal/features/backups/backups/download/... ./internal/features/restores/...`

## 3. Local publish-subscribe

- [x] 3.1 Create `backend/internal/util/pubsub` with the `Broker` contract and `LocalBroker`; verify immediate subscription readiness, fan-out, ordering, cancellation, publication errors, idempotent close and panic isolation through `go test -race ./internal/util/pubsub`
- [x] 3.2 Split the generic task-cancellation manager into its registration, request and listener responsibilities, inject `pubsub.Broker`, delete the Valkey-specific readiness handshake and replace timing sleeps with synchronization; verify two cancellation components sharing one broker through `go test -race ./internal/features/tasks/cancellation`

## 4. Local rate limiting

- [x] 4.1 Create `backend/internal/util/ratelimiter` with the `Counter` contract and cache-backed `MemoryCounter`; verify sliding-window boundaries, independent scopes, rejected-attempt accounting, invalid settings, expiration and one remaining concurrent allowance through `go test -race ./internal/util/ratelimiter`
- [x] 4.2 Inject `ratelimiter.Counter` into user and verification-agent request handling, propagate limiter failures as rejected requests without logging user identifiers and verify the behavior through `go test -race ./internal/features/users/controllers ./internal/features/verification/agents`

## 5. Remove Valkey infrastructure

- [x] 5.1 Delete Valkey fields and examples from `backend/internal/config`, `.env.example` and `.gitignore`; remove logical database selection, test cache namespaces, startup checks, healthcheck probes and external test cleanup while retaining PostgreSQL worker-slot isolation; verify the affected packages through `go test -race ./internal/config ./internal/features/system/healthcheck ./internal/util/testing/...`
- [x] 5.2 Remove `github.com/valkey-io/valkey-go`, run `go mod tidy` and verify active backend code has no Valkey or Redis imports or client types
- [x] 5.3 Remove the Valkey package repository and binary from `Dockerfile`, remove its startup function from `docker/start.sh` and remove the Compose service, CI readiness wait and remaining Valkey comments from `.github/workflows/ci-release.yml`; verify with `docker run --rm -v "$PWD:/mnt" koalaman/shellcheck:v0.10.0 docker/start.sh e2e/docker-storage/run.sh` and `docker run --rm -v "$PWD:/repo" -w /repo rhysd/actionlint:1.7.7 -color`
- [x] 5.4 Update the Docker filesystem suite to expect only Databasus and PostgreSQL processes and no cache readiness stage; build the candidate image and verify `make test-filesystems` passes

## 6. Single-instance deployment and documentation

- [x] 6.1 Delete the Helm `replicaCount` value and documentation, remove Valkey references from `deploy/helm/values.yaml` and `deploy/helm/README.md` and render one StatefulSet replica directly; run `docker run --rm -v "$PWD:/repo" -w /repo alpine/helm:3.18.6 lint deploy/helm` and verify the output of `docker run --rm -v "$PWD:/repo" -w /repo alpine/helm:3.18.6 template databasus deploy/helm` contains exactly one application replica and no supported replica-count override
- [x] 6.2 In the English Advanced Config page, replace `Databasus, PostgreSQL, and Valkey use the same non-root operating-system account named <code>databasus</code>.` with `Databasus and PostgreSQL use the same non-root operating-system account named <code>databasus</code>.` Remove Valkey from the corresponding sentence in all five translated copies. Keep English copy free of Oxford commas. Verify `npm run lint` and `npm run build` pass in `website/`
- [x] 6.3 Append dated replacement notes to ADR-0006 and ADR-0013 stating that Valkey-backed state was replaced by process-local cache, publish-subscribe and rate-limiting providers; preserve their decision bodies and verify no new ADR file is added
- [x] 6.4 Apply the delta requirements to the current OpenSpec capabilities and verify them with `npx --yes @fission-ai/openspec@1.11.0 validate --specs --strict`

## 7. Final verification

- [x] 7.1 Run `go test -race ./internal/util/cache ./internal/util/pubsub ./internal/util/ratelimiter ./internal/features/tasks/cancellation ./internal/features/backups/backups/download/... ./internal/features/restores/... ./internal/features/users/controllers ./internal/features/verification/agents` in `backend/`; resolve every failure
- [x] 7.2 Run `make test` and `make lint` in `backend/`; resolve every test, formatting and lint failure
- [x] 7.3 Build the production image and verify it starts healthy without Valkey configuration, packages, processes, ports or readiness logs
- [x] 7.4 Run the mandatory post-implementation reviewer against the complete diff and every governing module document, resolve every `CHANGES REQUIRED` finding and leave the working tree ready to commit
