## Why

Databasus supports one application process per installation, but its transient cache, rate limiting and task-cancellation messaging still require a separate Valkey process. This adds configuration, image size, startup failure modes and test infrastructure without providing useful cross-instance coordination.

## What Changes

- Replace the Valkey-backed cache with a bounded, process-local cache behind a provider interface. Preserve expiration, invalidation, atomic consume and atomic reservation semantics.
- Move publish-subscribe behavior into a dedicated local package with its own provider interface and contract tests.
- Move rate limiting into a dedicated local package with its own provider interface, sliding-window behavior and contract tests. Feature callers no longer express limits as generic cache keys; the local implementation may privately use the typed cache provider for bounded storage and expiration.
- Keep cache, publish-subscribe and rate-limiting callers independent of their local implementations so a distributed provider can be added later without changing feature code.
- Remove the Valkey server, client dependency, health probe, environment configuration, startup orchestration, test cleanup and CI service.
- Make the supplied deployments run exactly one application process so local transient state cannot diverge between replicas.
- Update container tests and Helm documentation. Update all six Advanced Config pages to describe the single-process runtime. The English page will say `Databasus and PostgreSQL use the same non-root operating-system account named <code>databasus</code>.` English copy added or revised by this change will not use Oxford commas.
- Add short notes to relevant existing ADRs that Valkey-backed runtime state was replaced by local cache, publish-subscribe and rate-limiting providers. Preserve the original decisions as historical context.
- **BREAKING**: delete `VALKEY_HOST`, `VALKEY_PORT`, `VALKEY_USERNAME`, `VALKEY_PASSWORD` and `VALKEY_IS_SSL`. External Valkey connections and the old client-facing cache constructors are removed without aliases or fallback handling.
- **BREAKING**: delete the Helm `replicaCount` setting and render one application replica. Multi-replica installations are no longer representable by the supplied chart.

## Capabilities

### New Capabilities

- `local-cache`: Define the bounded, concurrent, process-local cache contract, including expiration and atomic operations used for locks and single-use values.
- `local-publish-subscribe`: Define process-local publication, subscription readiness, delivery, cancellation and failure isolation.
- `local-rate-limiting`: Define concurrent sliding-window request limits with deterministic expiration and independent caller scopes.
- `single-instance-runtime`: Define one application process as the supported installation topology for supplied deployment artifacts.

### Modified Capabilities

- `docker-storage-compatibility`: Remove Valkey from the bundled runtime services and from startup ordering and identity checks.
- `internal-postgresql-protection`: Describe PostgreSQL as the only embedded server and remove the obsolete Valkey port requirement.

## Impact

The change affects the backend cache and its callers, task cancellation, authentication and verification-agent rate limiting, backend configuration and dependencies, the main command, test database cleanup, Docker image construction and startup, Compose development services, release CI, Docker filesystem tests, Helm documentation, current OpenSpec contracts, ADRs and all six website Advanced Config translations.

The change answers to `AGENTS.md`, `backend/AGENTS.md`, `website/AGENTS.md` and `adr/AGENTS.md`.

Out of scope: support for multiple application processes or replicas in one installation, persistence of transient state across restarts, implementing a Valkey adapter, compatibility shims for deleted environment variables, constructors or Helm settings, changes to public HTTP APIs or configured rate-limit thresholds, creating a new ADR and rewriting historical OpenSpec archives or ADR decision bodies. Updating the two existing ADRs instead of creating a replacement ADR is an explicit scope decision for this change.
