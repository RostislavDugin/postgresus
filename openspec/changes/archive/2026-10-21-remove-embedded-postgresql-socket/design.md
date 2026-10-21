## Context

See `proposal.md` for the motivation. The embedded PostgreSQL server is started with `-k /databasus-data/pgsocket` and waits for readiness through that socket (`docker/start.sh:398-419`). Bootstrap writes a peer mapping from the `databasus` account to role `postgres` (`write_bootstrap_postgresql_authentication`, `docker/start.sh:374-396`). It then sets the generated password through the socket (`configure_postgresql_database`, `docker/start.sh:472-490`), replaces the authentication rules with `local all all reject` and reloads the server (`activate_runtime_postgresql_authentication`, `docker/start.sh:492-509`). Finally it checks that the socket refuses a login (`verify_postgresql_runtime_authentication`, `docker/start.sh:511-532`). The socket therefore carries one administrative statement per start. Every other connection, including the application's, uses loopback TCP with SCRAM.

The socket location has moved twice. An earlier image used the default `/var/run/postgresql`. ZFS-backed overlay storage on TrueNAS Scale ignores ownership changes on directories from image layers, so PostgreSQL could not create its lock file there. Commit `27d891fb` moved the socket to `/tmp`. Commit `61fe8dd8` then moved it under the data root, because `/tmp` is reachable by every process in the container. The archived `protect-internal-postgresql-access` design (decision 6) states the placement without comparing locations. It rejected disabling sockets because "Startup would need another bootstrap credential path". Single-user mode was not considered.

Since `simplify-docker-runtime-user`, PostgreSQL and the application share one operating-system account. The `0700` mode on the socket directory therefore separates PostgreSQL from other UIDs only, not from the application. The authentication rules already provide the effective barrier.

Current startup also exposes the generated password in process arguments. `psql -v internal_postgres_password=...` (`docker/start.sh:474-476`) and the `gosu databasus env PGPASSWORD=...` calls (`docker/start.sh:522`, `docker/start.sh:561`, `docker/start.sh:572`) put it into the argument list of a running process.

The approach answers to the root `AGENTS.md`: no compatibility shims, refactoring of the touched startup code within the same change, and the test rules of the testing skill. ADR-0003 keeps PostgreSQL embedded in the image, and this change stays inside that decision. No module `AGENTS.md` governs `docker/` or `e2e/`.

## Goals / Non-Goals

**Goals:**

- The embedded server creates no Unix socket at any point, so no mount property can block startup.
- No startup step admits a login without the generated password.
- The generated password appears in no process argument list and in no container log line.
- The filesystem suite reproduces the macOS failure on a Linux CI runner.

**Non-Goals:**

- Hardening of the password in process environments. The application account can already read the published connection string, so hiding the environment from the same account protects nothing.
- Changing how the backend classifies socket hosts.

## Decisions

### 1. Administer the cluster in single-user mode before the server starts

Startup runs `postgres --single` under the `databasus` account against the data directory. It feeds `ALTER ROLE postgres WITH PASSWORD '<generated>'` through standard input. Single-user mode opens the cluster files directly and listens on nothing. Nothing else can connect while the password changes, and the server never runs with an old or unknown password.

A spike against PostgreSQL 17.11 confirmed the behavior this decision relies on:

- The statement succeeds without any listener. No `.s.PGSQL` file appears in `/tmp` or `/var/run/postgresql`.
- After an unclean shutdown caused by `kill -9`, single-user mode replays the write-ahead log and commits the statement. All 100000 rows written before the kill survived.
- Single-user mode refuses to run while a postmaster holds `postmaster.pid`.
- With a missing checkpoint record it aborts with `PANIC: could not locate a valid checkpoint record`.

Alternatives rejected:

- Move the socket to `/dev/shm`, which is tmpfs, writable under a read-only root filesystem and already holds the published connection string (`docker/start.sh:13`). This fixes macOS with fewer changes. It keeps the peer-mapping window and a socket file whose permissions protect nothing against the shared account.
- Move the socket to `/run/databasus`. A directory baked into the image repeats the TrueNAS ownership failure. A directory created at startup needs `chown`, which the hardened Helm values lack because they drop every capability except `SETUID` and `SETGID` (`deploy/helm/README.md:398-409`). A read-only root filesystem also makes `/run` unwritable without an extra volume.
- Use an abstract Unix socket. It has no file-system permissions. With `--network host` it lives in the host's network namespace, where any host process can reach it and two containers collide on the name.
- Trust loopback TCP during bootstrap. With `--network host`, loopback belongs to the host, so any host process could log in without a password during that window.
- Persist a password with `initdb --pwfile`. It covers only new clusters, and the archived design deliberately avoided storing a secret on the volume.
- Keep the socket under the data root and document a tmpfs mount for macOS. Every macOS user would hit the failure before finding the workaround.

### 2. Split single-user work into an open step and a password step

The first single-user invocation receives empty input. It only opens the cluster, which replays the write-ahead log when needed, and then exits with a shutdown checkpoint. A failure of this step is the only trigger for `pg_resetwal`, after which the open step runs once more. Before the reset, startup removes `postmaster.pid`. A single-user backend that panics, as it does on a missing checkpoint record, leaves its lock file behind, and `pg_resetwal` refuses to run while that file exists. No PostgreSQL process runs at that point, so the file is stale. When `pg_resetwal` or that second attempt fails, startup prints the existing recovery-failure message from `report_postgresql_recovery_failure` and exits. The second invocation sets the password with `-c exit_on_error=on`, so a failing statement ends the backend with exit code 1 instead of 0. In the spike, exit code 0 after a failed statement was the default behavior. A failure of the password step stops the container without a reset, because the cluster has already opened successfully.

Today any start failure triggers `pg_resetwal`, including a socket failure on a healthy cluster. That is what happened in the reported macOS log. After this change a reset happens only when the cluster cannot be opened at all. A regular server start that fails after both steps succeeded stops the container with an error.

Alternatives rejected:

- One invocation that opens the cluster and sets the password. Its exit code cannot tell a broken log from a failed statement. Telling them apart would require parsing PostgreSQL output.
- Keep the reset around the regular server start. Once single-user mode has opened the cluster, the log is consistent, and a reset at that point can only discard data.

### 3. Keep the password out of arguments and the log

The shell writes the statement with the `printf` builtin into a pipe to the single-user backend. A builtin spawns no process, so the password never enters an argument list. The password step runs with `-c log_min_error_statement=panic`. With the default setting `error`, the spike showed the failing `ALTER ROLE` statement, password included, in the `STATEMENT:` log line. With `panic` the statement was absent. On failure, the password step prints the captured output to standard error. PostgreSQL's error text can name the command, as in `cannot execute ALTER ROLE in a read-only transaction`, but it contains neither the statement nor the password. `-c password_encryption=scram-sha-256` is passed to the password step as well, so the stored verifier matches the server's SCRAM rules. The spike confirmed a `SCRAM-SHA-256` verifier.

`psql` calls that need the password get `PGPASSWORD` as a shell assignment prefix in front of `gosu`, for example `PGPASSWORD="${internal_postgres_password}" gosu databasus psql ...`. `gosu` preserves the environment, so the password reaches `psql` without appearing in the arguments of `gosu` or `env`.

Alternatives rejected:

- `psql -v` variables, as today. They are argument values.
- A temporary `PGPASSFILE`. It adds a secret file and a cleanup path for no gain over the environment of a short-lived process.

### 4. Write the authentication rules once and start without sockets

Before any PostgreSQL process starts, startup writes the final `pg_hba.conf`: SCRAM on `127.0.0.1/32` and `::1/128`, and reject for replication. It also writes an empty `pg_ident.conf`. Both files get mode `0600` as today. The regular server starts with `-c unix_socket_directories=''` beside the existing `-p 5437` and `-c password_encryption=scram-sha-256`. The server also starts with `-c max_wal_senders=0`. The `host replication` reject rules match only physical replication. A logical replication connection, requested with `replication=database`, is matched by database name, so it falls under the SCRAM rule, and the superuser `postgres` may replicate. With no WAL senders, PostgreSQL refuses every replication connection after authentication. The command line applies these settings to new and existing clusters alike, because `postgresql.conf` is only appended on `initdb` (`docker/start.sh:366-371`). Readiness polls `pg_isready -h localhost -p 5437`, which needs no authentication.

The `local` rules disappear from `pg_hba.conf`, since no socket exists to match them. The bootstrap peer map, the rule rewrite, `pg_ctl reload` and the check that a socket login fails are deleted. The check that the generated password works over loopback stays.

The metadata database is created after the server starts, through `psql -h localhost` with the generated password, using the existing `\gexec` statement.

Alternatives rejected:

- Keep `local all all reject` as defense in depth. With sockets disabled the rule can never match, and it would suggest a socket exists.
- Revoke replication with `ALTER ROLE postgres NOREPLICATION`. A superuser bypasses that attribute.
- Write `unix_socket_directories` into `postgresql.conf`. Existing clusters would need an edit of a file that startup otherwise only appends on initialization.

### 5. Stop preparing the socket directory

`create_required_data_directories`, `normalize_data_permissions` and `verify_service_directories` drop `/databasus-data/pgsocket` (`docker/start.sh:182-213`, `docker/start.sh:327-339`). A directory left by an earlier image is neither touched nor deleted.

Alternative rejected: delete the leftover directory. Removal is a mutation of user data with no benefit, and it would fail on mounts that deny it.

### 6. Reproduce the macOS failure with seccomp

A new filesystem case starts the candidate with a seccomp profile. The profile allows every system call except `socket` with address family `AF_UNIX`, which returns `EAFNOSUPPORT`. In the spike, a locally built image with the current startup script failed under it with the same `could not create any Unix-domain sockets` error as on macOS. PostgreSQL 17 started under the profile without sockets and answered over TCP.

A shared assertion checks that `/proc/net/unix` inside the container lists no `.s.PGSQL` entry. It runs in every case through `assert_running_container`, so every layout proves that no socket exists without hard-coding a directory. The NFS case loses its tmpfs mounts on `/databasus-data/pgsocket`. `prepare_bind_tree` and the NFS share setup stop creating the directory. The runtime authentication assertion drops its socket-login probe and keeps the loopback probe without a password.

The upgrade cases add `v3.60.0`, the latest release that bootstraps through the socket.

Alternatives rejected:

- Run Docker Desktop on macOS in CI. Runners would need nested virtualization and a licensed Docker Desktop, and the job would test one platform instead of the general property.
- Mount a FUSE or vfat file system as the data root. Such a mount needs host privileges and brings unrelated failures, such as missing ownership, into the case.

## Risks / Trade-offs

- [A bundled component other than PostgreSQL needs `AF_UNIX`, so the seccomp case fails for an unrelated reason] -> The current image dies at the PostgreSQL start under the profile, so such a component can only surface in task 2.5, when the rewritten bootstrap lets the application run. If it does, the case narrows the profile or is redesigned there, before the filesystem suite changes in group 3.
- [Single-user mode adds a shutdown checkpoint per invocation, two per start] -> The spike measured under 0.1 s per checkpoint on a fresh cluster. After an unclean shutdown, recovery runs once in the open step, as it ran in the regular start before.
- [`pg_resetwal` still runs for an open failure that has nothing to do with the log, such as a permission error] -> Storage validation runs earlier and covers the data directory. Narrowing the trigger further is out of scope.
- [Operators lose socket access for manual administration] -> They use `psql` with the connection string the container publishes, or single-user mode in a stopped container.
- [The single-user backend prints a banner and `backend>` prompts to the container log] -> Its output is discarded unless the step fails. On failure, startup prints the captured output. The statement is absent from it because of `log_min_error_statement=panic`. The error text may still name the command, such as `ALTER ROLE`.

## Migration Plan

1. Build the image and run the new no-socket case, the bind-root case and the upgrade cases locally.
2. Publish the image. Existing volumes start without manual steps. The leftover socket directory stays empty.
3. Rolling back to `v3.60.0` needs no volume change. That image recreates its socket directory and rewrites `pg_hba.conf` with its own peer rule before it starts the server (`v3.60.0:docker/start.sh:546-554`). Rollback to older images keeps the guidance of the archived `protect-internal-postgresql-access` design.
