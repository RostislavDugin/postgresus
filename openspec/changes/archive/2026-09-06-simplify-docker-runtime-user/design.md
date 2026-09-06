## Context

See `proposal.md` for the motivation. The image currently creates distinct `postgres` and `databasus` accounts, publishes four identity variables, and remaps them before preparing mounted paths (`Dockerfile:162-179`, `Dockerfile:231-279`). PostgreSQL uses a private socket and peer authentication for startup administration, while the application connects through loopback with a generated password (`Dockerfile:281-367`). Local backup publication uses `LocalStorage.SaveFile`, including its rename and cross-filesystem fallback, and removal uses `LocalStorage.DeleteFile` (`backend/internal/features/storages/models/local/model.go`). The filesystem harness probes directories independently and assumes two accounts (`e2e/docker-storage/run.sh:135-215`).

The implementation follows the root `AGENTS.md`, `backend/AGENTS.md`, and `website/AGENTS.md`.

## Goals / Non-Goals

**Goals:**

- Make mounted-data ownership determine the default runtime identity for existing installations.
- Test the public local-storage save and delete operations at startup and in the filesystem suite.
- Keep the bundled processes non-root after startup preparation.
- Produce an error that gives operators the failed operation, path, effective IDs, and documentation link.

**Non-Goals:**

- Publish embedded PostgreSQL or Valkey outside the container.
- Rename the PostgreSQL database role or change the metadata schema.
- Add compatibility aliases for deleted variables.
- Change backup encryption, naming, or retention behavior.

## Decisions

### Use one operating-system account for bundled processes

The package-provided PostgreSQL account will become `databasus`, preserving numeric ID `999` as the fresh-install fallback. The application, PostgreSQL, and Valkey will all execute through that account. The PostgreSQL database role stays `postgres`.

### Use `PUID` and `PGID` as the complete public identity interface

Startup resolves each value independently. An explicit valid value wins. Otherwise it checks an existing PostgreSQL data directory, the backup mount, the data root, and then `999`. It skips root-owned candidates. The four separate variables are removed from the image and startup no longer reads them.

### Treat metadata normalization as preparation, not proof

Startup attempts `chown` and `chmod` on required paths. It then executes the actual operations under `databasus`. The backend `--test-storage` command exercises `LocalStorage.SaveFile` and `LocalStorage.DeleteFile`. Focused shell probes cover PostgreSQL paths, its socket directory, and Databasus control files.

### Keep the Dockerfile declarative

The `Dockerfile` will install packages, create the image account, copy runtime artifacts, and declare `docker/start.sh` as the entrypoint. The script will group identity resolution, mounted-storage preparation, service bootstrap, and final application execution into named functions. Its `main` function will show the runtime sequence without the escaping required by an inline Docker heredoc.

### Exercise local storage through its public methods

The main backend binary will accept `--test-storage`. It will create a unique probe name, call `LocalStorage.SaveFile`, then call `LocalStorage.DeleteFile`. Startup will start Valkey before invoking the command, while PostgreSQL and the Databasus application will remain stopped until storage validation passes. The command will report the failed operation, effective IDs, and documentation link. `LocalStorage.TestConnection` remains unchanged.

Valkey-backed rate limiters, cache utilities, and the publish-subscribe manager will retain explicit client injection. Starting Valkey first avoids adding deferred client providers only for startup command dispatch.

Configuration will recognize Go test binaries by their executable suffix and the `cleanup_test_db` command by its executable name. It will no longer infer test mode from arbitrary command-line arguments. This keeps `--test-storage` on production storage paths without requiring test database settings.

### Limit peer authentication to PostgreSQL bootstrap

Bootstrap writes a temporary identity map from operating-system user `databasus` to database role `postgres`. After setting the generated password and creating the metadata database, startup replaces the host authentication rules with socket rejection and reloads PostgreSQL. The application uses SCRAM over loopback.

### Test upgrades with persistent artifacts

Upgrade cases create data with each covered released image, stop it, start the candidate against the same persistent data, and verify a metadata marker, secret key, application log, backup marker, nested WAL queue, and PostgreSQL cluster marker. Startup recursively prepares an existing `backups/wal-queue` tree because its `0700` directories and `0600` segments must remain resumable when a released image used a different numeric account. It then traverses every queue directory and opens every segment for reading and writing under `databasus` before starting PostgreSQL or the Databasus application. Test names describe the storage behavior and contain no tracker references.

## Risks / Trade-offs

- [An automatically selected owner belongs to an unrelated image account] -> Reject collisions with another existing account and report the selected ID before services start.
- [A mount reports usable permissions but publication still differs] -> Call `LocalStorage.SaveFile` under the final runtime identity.
- [The startup probe conflicts with a real backup] -> Generate a unique probe name and delete only that file.
- [`--test-storage` is mistaken for a test binary] -> Detect test processes from the executable name and cover the storage flag with a regression test.
- [Storage validation fails after Valkey starts] -> Stop startup before PostgreSQL or the Databasus application and let container shutdown terminate Valkey.
- [Bootstrap authentication remains enabled after an interrupted start] -> Rewrite the temporary rules on every start and replace them before launching the application.
- [An update cannot use existing ownership and cannot change it] -> Stop with the exact operation, effective IDs, and the Advanced Config link.
- [An update leaves nested WAL queues owned by the previous image account] -> Prepare that tree recursively, verify directory creation and file opens as `databasus`, and preserve queued bytes in the upgrade fixtures.
- [Released images are unavailable during local development] -> Keep upgrade cases selectable and make the full CI job pull the pinned released tags before execution.

## Migration Plan

1. Build the image with one `databasus` account at `999:999` and the new startup resolver.
2. Correct test-process detection, start Valkey before storage validation, and add the backend `--test-storage` command with focused tests.
3. Replace the harness account assertions with public local-storage save, delete, and upgrade assertions.
4. Update documentation and synchronize validated delta specs into the main specifications.
5. Release notes will tell operators to remove the four deleted variables and use `PUID` and `PGID` only when automatic selection is unsuitable.
6. Replace the inline entrypoint heredoc with `docker/start.sh` without changing its runtime behavior or filesystem compatibility matrix.

Rollback uses the previous image against unchanged persistent volumes. The candidate does not rewrite backup contents, the secret key, or the PostgreSQL cluster beyond normal PostgreSQL startup and migrations.
