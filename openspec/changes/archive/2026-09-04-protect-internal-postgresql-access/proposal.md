## Why

Workspace users can reach the PostgreSQL instance embedded in the Databasus container through Unix sockets and libpq connection-string parsing that bypass the existing localhost check. The direct connection-test endpoint can also combine another workspace's saved password with a caller-controlled host because it loads and merges the saved database without checking access.

## Governing docs

This change answers to [`AGENTS.md`](../../../../AGENTS.md) and [`backend/AGENTS.md`](../../../../backend/AGENTS.md). It changes the root `Dockerfile` and backend code only.

## What Changes

- Treat every PostgreSQL connection field as a literal conninfo value, including the database name passed to `pg_dump`.
- Reject logical and physical backup targets that can address the embedded PostgreSQL instance through Unix sockets, local aliases, loopback or unspecified addresses, libc-compatible numeric IPv4 forms, and comma-separated libpq host lists.
- Enforce the embedded-target check both during model validation and immediately before opening a connection, so an older persisted row cannot bypass it.
- Authorize `test-connection-direct` against the requested workspace or the saved database's actual workspace before applying saved credentials. Unknown and unauthorized saved IDs return the same response.
- Keep trusted healthcheck calls separate from the user-facing direct connection-test path.
- Return sanitized server errors when saved-target or workspace-permission lookups fail.
- Move the embedded PostgreSQL Unix socket to a private directory, reject general local socket access, require SCRAM authentication over loopback TCP, and generate the internal password at container startup.
- Reject a PostgreSQL `PUID` that is already assigned to another account in the container.
- **BREAKING**: an ad hoc `POST /api/v1/databases/test-connection-direct` request without `workspaceId` is rejected with HTTP 400. Unauthorized or unknown saved targets return HTTP 403, and internal lookup failures return HTTP 500 instead of exposing an error through HTTP 400.
- **BREAKING**: `DatabaseService.TestDatabaseConnectionDirect` now requires the authenticated user. Trusted internal callers use `TestTrustedDatabaseConnection`; the old service signature is deleted without a compatibility shim.
- **BREAKING**: a container configured with a PostgreSQL `PUID` already owned by another account fails at startup. The conflicting configuration is rejected rather than preserved through duplicate UIDs.
- **BREAKING**: an explicitly supplied `DATABASE_DSN` that points to the embedded PostgreSQL instance with the previously published fixed password stops working because startup preserves the explicit value while rotating the internal role's password.

### Operator migration

- Remove an explicit `DATABASE_DSN` that points to the embedded database before upgrading, so startup can inject the generated credential. Keep `DATABASE_DSN` only when it points to an external PostgreSQL database with independently managed credentials.
- Update callers of the direct connection-test endpoint to include `workspaceId` for unsaved configurations and accept the documented 403 and 500 responses.
- Choose an unused PostgreSQL `PUID` if the configured value belongs to another account in the image.

### Out of scope

- Changing workspace roles or which existing roles may manage database connections.
- Applying the embedded-target policy to MySQL, MariaDB, or MongoDB.
- Resolving arbitrary DNS names during validation. Runtime socket isolation and generated credentials remain the protection when a hostname resolves to the container itself.
- Changing an explicitly supplied external `DATABASE_DSN`.
- Adding a new public error-code schema or changing generated Swagger files.

## Capabilities

### New Capabilities

- `internal-postgresql-protection`: prevents workspace-controlled connection and backup operations from reading the Databasus metadata database or reusing saved credentials outside their authorized workspace.

### Modified Capabilities

None. Existing physical-backup specifications do not define access to the embedded Databasus PostgreSQL instance or authorization for direct connection tests.

## Impact

- `Dockerfile`: embedded PostgreSQL socket placement, `pg_hba.conf`, generated credentials, and UID collision validation.
- `backend/internal/features/databases`: connection-test authorization, sanitized errors, trusted healthcheck entry point, and pre-connection target validation.
- `backend/internal/features/databases/databases/postgresql`: shared target classification and conninfo quoting for logical and physical connections.
- `backend/internal/features/backups/backups/usecases/logical/postgresql`: literal database-name handling for `pg_dump`.
- `backend/internal/features/healthcheck/attempt`: use of the trusted connection-test entry point.
- No new runtime dependency, database migration, or frontend change.
