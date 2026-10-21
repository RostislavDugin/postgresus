## Why

The embedded PostgreSQL server creates its Unix socket inside the mounted data directory, at `/databasus-data/pgsocket`. Docker Desktop on macOS shares a host folder through a file system that cannot set permissions on a socket file. A user who bind-mounts a host folder as `/databasus-data` therefore gets `could not create any Unix-domain sockets`, and the container never starts. The same failure applies to any data mount without Unix-socket support.

The socket serves one purpose. At startup it lets the image set a new random password on the internal `postgres` role through peer authentication. After that step, authentication rules reject every socket login. Setting the password in PostgreSQL single-user mode needs neither a socket nor a network listener, so the server can run without a socket at all.

## Governing docs

This change answers to [`AGENTS.md`](../../../AGENTS.md). It changes `docker/start.sh`, the Docker filesystem suite in `e2e/docker-storage/` and two OpenSpec capabilities. It changes no code under `backend/`, `frontend/`, `agent/` or `website/`.

## What Changes

- Startup sets the generated password on the internal `postgres` role in PostgreSQL single-user mode, before the server accepts any connection. Neither a socket nor a TCP listener exists during that step.
- The embedded PostgreSQL server runs with Unix sockets disabled. It listens only on loopback TCP, with SCRAM authentication and the generated password from its first accepted connection.
- Startup creates the metadata database over loopback TCP with the generated password.
- The embedded server starts with WAL senders disabled, so it refuses logical replication connections as well as physical ones. Authentication rules alone rejected only physical replication.
- The temporary peer mapping is deleted, together with its `pg_ident.conf` entry, the authentication rewrite and the reload that followed bootstrap. The authentication rules are written once, before the server starts.
- Startup no longer creates, prepares or verifies `/databasus-data/pgsocket`. An existing directory from an earlier image is left in place and ignored.
- The `pg_resetwal` recovery runs when the single-user step cannot open the cluster, because that step replays the write-ahead log first. A later failure of the regular server start stops the container without resetting the write-ahead log.
- The password no longer appears in the arguments of a startup process. It reaches PostgreSQL through standard input, and a failing statement is kept out of the container log.
- The Docker filesystem suite gains a case in which the container cannot create Unix sockets at all. The current image fails that case exactly as on macOS. The workaround mounts for the socket directory in the NFS case are deleted. The upgrade cases add `v3.60.0`, the latest release that bootstraps through the socket.
- **BREAKING**: an operator can no longer administer the embedded cluster through the image's peer-authenticated socket. The rollback procedure in the archived `protect-internal-postgresql-access` design relied on it. Operators use the generated connection string inside a running container instead, or single-user mode inside a stopped one.

## Capabilities

### New Capabilities

None.

### Modified Capabilities

- `internal-postgresql-protection`: the embedded server accepts no Unix-socket connections, and bootstrap administration happens with no listener at all instead of through temporary peer mapping.
- `docker-storage-compatibility`: storage validation no longer covers a PostgreSQL socket path. Startup succeeds when the data mount cannot hold Unix sockets, and the filesystem matrix covers that case. Upgrades cover `v3.60.0`. A write-ahead log reset happens only when the cluster cannot be opened.

## Impact

- `docker/start.sh`: PostgreSQL bootstrap, start, readiness check, recovery path and storage preparation.
- `e2e/docker-storage/run.sh`: runtime authentication assertions, bind-tree preparation, the NFS case and a new case without Unix-socket support.
- OpenSpec specs `internal-postgresql-protection` and `docker-storage-compatibility`.
- No backend, frontend, API, database migration, configuration variable or dependency change.

## Out of scope

- Moving temporary files, backups or PostgreSQL data off `/databasus-data`.
- Removing a leftover `/databasus-data/pgsocket` directory from existing volumes.
- Changing the Helm chart, its README or its `extraVolumes` guidance about `/var/run`.
- Narrowing when `pg_resetwal` runs beyond the move to the single-user step, such as matching specific PostgreSQL errors.
- Changing the embedded-target classifier in the backend. It keeps rejecting every Unix-socket host.
- Further compatibility work for Docker Desktop on macOS beyond the socket, such as file-system performance.
- Website or README documentation changes.
