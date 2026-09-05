## Context

See [proposal.md](proposal.md#why) for the attack paths and [the capability spec](specs/internal-postgresql-protection/spec.md) for required behavior.

PostgreSQL connection strings are assembled centrally in `backend/internal/features/databases/databases/postgresql/shared/credentials.go:82-120`, while logical dumps pass their database argument separately in `backend/internal/features/backups/backups/usecases/logical/postgresql/create_backup_uc.go:363-371`. Connection creation also has more than one entry point: the database-level tunnel dispatcher in `backend/internal/features/databases/tunneled_database.go:34-71` and engine-specific tunnel functions.

The direct connection-test endpoint accepts both unsaved configurations and partial updates for saved database IDs. Its service path is split from trusted healthchecks at `backend/internal/features/databases/service.go:475-493`, and saved-target authorization happens at `backend/internal/features/databases/service.go:1017-1075`.

The container starts PostgreSQL and the application in one process tree. `Dockerfile:330-456` owns PostgreSQL initialization, authentication policy, password rotation, and the connection string inherited by the application. Custom `PUID` and `PGID` values remain supported, subject to the account separation check at `Dockerfile:232-252`.

The approach is constrained by [`AGENTS.md`](../../../../AGENTS.md) and [`backend/AGENTS.md`](../../../../backend/AGENTS.md), especially their rules on layered security, controller-level tests, sanitized errors, explicit names, and removal of obsolete call signatures without compatibility shims.

## Goals / Non-Goals

**Goals:**

- Stop each confirmed path from a workspace-controlled PostgreSQL request to the embedded metadata database.
- Authorize saved credentials before any caller-controlled endpoint can use them.
- Preserve legitimate local databases, remote SSH targets, healthchecks, external metadata databases, and custom non-conflicting PostgreSQL UIDs.
- Make the container's PostgreSQL authentication safe even if a target classifier misses a future local hostname form.

**Non-Goals:**

- Resolve arbitrary hostnames during request validation.
- Redesign workspace roles, database persistence, or the general API error format.
- Extend this PostgreSQL-specific target policy to other database engines.
- Add a data migration or a long-lived compatibility path for the old service method signature.

## Decisions

### 1. Combine semantic target checks with runtime isolation

PostgreSQL targets are classified before validation succeeds and again immediately before a connection opens. The shared classifier treats absolute and abstract socket names as local, examines every comma-separated libpq host, recognizes configured container aliases, and parses IPv4 forms accepted by libc before checking loopback or unspecified ranges. A loopback database address behind a genuinely remote SSH bastion remains remote in meaning.

The connection-time check is required because persisted rows can predate the validation rule or enter a connection path that does not call full model validation. The container authentication changes provide the second boundary when an unrecognized hostname still resolves locally.

Alternatives rejected:

- Extend the old string allowlist only. It cannot cover Unix sockets, multi-host fallback, or resolver-equivalent numeric addresses reliably.
- Harden the container without application checks. That makes access depend on the secrecy of one credential and still permits repeated probes of the internal service.
- Resolve DNS during validation. It adds latency and availability dependencies, and the result can change between validation and connection through DNS rebinding.

### 2. Quote each conninfo value at construction

The shared PostgreSQL credential builder quotes and escapes every string field before parsing it. The logical dump receives a one-field conninfo expression whose quoted `dbname` contains the complete database name. This handles parameter-looking input as data instead of trying to reject selected substrings.

Alternatives rejected:

- Reject whitespace or the text `dbname=`. Valid PostgreSQL identifiers can contain unusual characters, and a denylist would miss other parameter names and escaping forms.
- Quote only passwords. Host, username, database, TLS mode, and certificate paths are all parser inputs and need the same invariant.
- Pass the raw database name to the dump tool. That tool interprets values containing conninfo syntax as connection strings.

### 3. Authorize the saved record before merging request fields

An unsaved direct test requires `workspaceId` and checks the caller's existing database-management permission for that workspace. A saved-ID request first loads the persisted database, authorizes its stored workspace, and only then overlays the submitted update. The saved password is therefore unavailable to an unauthorized request.

Database storage and workspace access are represented by narrow internal interfaces in `backend/internal/features/databases/interfaces.go:12-31`. This permits controller tests to inject repository and membership failures without replacing production dependency wiring.

Alternatives rejected:

- Trust the request's `workspaceId`. A caller can claim a workspace they control while referencing a database stored elsewhere.
- Merge first and clear the password on denial. The unsafe credential has already crossed the authorization boundary, and later code can accidentally use it.
- Check authorization only in the controller. Trusted internal callers and future entry points would bypass the invariant.

### 4. Separate user-authorized and trusted connection tests

The HTTP path takes an authenticated user and resolves an authorized target. Scheduled healthchecks call a separate trusted method with a database already loaded by the service, while both methods share the connection implementation and the runtime embedded-target check.

Alternatives rejected:

- Accept a nullable user in one method. A missing principal would silently become an authorization bypass available to any caller.
- Give the healthcheck a synthetic administrator. That invents user identity for a system action and makes audit behavior misleading.
- Keep the old method name for both meanings. It hides the security boundary between an HTTP request and a trusted background operation.

### 5. Make authorization infrastructure failures explicit and sanitized

Unknown and unauthorized saved IDs map to one permission sentinel and HTTP 403 response. A repository failure and a workspace-permission lookup failure use separate wrapped sentinels, preserve the original error for server-side handling, and expose only fixed HTTP 500 messages.

Alternatives rejected:

- Return every failure as HTTP 400 with `err.Error()`. That exposes storage and membership details and misclassifies server failures as invalid input.
- Return HTTP 404 for unknown IDs. A different response would let a caller distinguish an existing protected UUID from an unused one.
- Discard underlying errors entirely. Wrapping retains diagnostic context without sending it to the client.

### 6. Use a private peer-authenticated socket to bootstrap random TCP credentials

The PostgreSQL socket lives under `/databasus-data/pgsocket`, owned by `postgres` with mode `0700`. Local HBA rules allow peer authentication only for the PostgreSQL operating-system account and reject other local users. Startup administration uses that socket to assign a new random password. The application and legacy-data checks use loopback TCP with SCRAM, and replication is rejected.

When `DATABASE_DSN` is absent from the container environment, startup exports a DSN containing the generated password. An explicitly supplied value wins, matching the existing environment precedence contract.

Alternatives rejected:

- Keep a fixed password because the port binds to loopback. Workspace-controlled connection features run inside the same container, so loopback is within their reach.
- Keep `local all all trust` and rely on socket filesystem placement. Authentication should still reject access if directory permissions are weakened later.
- Disable Unix sockets entirely. Startup would need another bootstrap credential path, while a private peer-authenticated socket already gives PostgreSQL a standard local administration boundary.
- Store one generated password permanently in the volume. Rotation at startup avoids creating another secret file and limits the lifetime of a disclosed value.

### 7. Reject UID collisions while preserving custom ownership

Before remapping `postgres`, startup resolves the requested `PUID`. If another image account already owns it, startup exits before changing the account. An unused UID is still accepted and applied.

Alternatives rejected:

- Continue using `usermod -o` for every value. Duplicate UIDs erase the filesystem boundary between `postgres` and `databasus`.
- Remove custom UID support. Bind-mounted NAS and host directories rely on operator-selected ownership.
- Reject only the known `databasus` UID. Root or another existing account would create the same class of collision.

## Risks / Trade-offs

- [A legitimate local database named `databasus` is rejected] -> Preserve the established conservative rule for local logical targets; a remote database with that name remains valid.
- [A hostname not recognized as local can pass classification] -> Do not rely on classification alone; private socket permissions, HBA rejection, generated credentials, and replication rejection remain active.
- [Comma splitting rejects a Unix-socket path containing a comma] -> Favor preventing libpq fallback to the embedded instance. Such paths can be renamed or reached through a remote tunnel.
- [Password rotation interrupts a process that retained the previous DSN] -> Rotation happens before the application starts, and the new DSN is exported to that process tree.
- [Clients of the direct endpoint omit `workspaceId`] -> Treat this as an intentional API break; clients must supply the workspace already selected in the UI or API workflow.
- [A custom `PUID` collides after an image adds another system account] -> Fail with the conflicting username before changing ownership, giving the operator a deterministic configuration error.
- [Rolling back to an older image can fail authentication against a volume already upgraded] -> Take a volume snapshot before deployment. Before rollback, either restore that snapshot or use the current image's peer-authenticated socket to set the internal role to the credential expected by the older image.

## Migration Plan

1. Remove an explicit `DATABASE_DSN` that targets the embedded database. Keep it only when it targets an external PostgreSQL database with its own credentials.
2. Update direct connection-test clients to send `workspaceId` for unsaved configurations and accept the documented 403 and 500 responses.
3. Build and run the backend tests, lint, Dockerfile validation, and image smoke tests before publishing the image.
4. Snapshot `/databasus-data` before replacing an existing container.
5. Start the new image. Startup rewrites the internal HBA file, creates the private socket directory, rotates the internal password, and launches the application with the matching DSN.
6. Confirm application health, verify that the application account cannot enter the private socket directory, and verify that the previously published fixed password fails over TCP.
7. For rollback, stop the new container and restore the volume snapshot before starting the prior image. If no snapshot is available, reset the internal role through the current image's peer-authenticated socket to the credential expected by the prior image, then roll back.

No database schema migration is required.
