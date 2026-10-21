## Why

Every `docker stop` kills the embedded PostgreSQL server instead of shutting it down. Startup ends with `exec gosu databasus ./main`, so the Databasus application becomes process 1 and the only receiver of the stop signal. When it exits, the kernel kills every other process in the container, the PostgreSQL server included. Each following start therefore runs crash recovery. A long recovery used to run into the 30-second readiness limit, and the startup script then reset the write-ahead log of a healthy cluster. The open step of `remove-embedded-postgresql-socket` removes that limit, but every restart still depends on recovery, and the automatic reset stays reserved for real damage such as a full disk or a power loss.

## Governing docs

This change answers to [`AGENTS.md`](../../../AGENTS.md). It changes `docker/start.sh` and the Docker filesystem suite in `e2e/docker-storage/`. It changes no code under `backend/`, `frontend/`, `agent/` or `website/`.

It builds on `remove-embedded-postgresql-socket`, which rewrites the same startup functions, and is applied after that change.

## What Changes

- The startup script stays process 1 for the life of the container instead of replacing itself with the application. After the steps that need root, it replaces itself with a copy running under the `databasus` account, so process 1 stays non-root. That copy runs PostgreSQL and the application as child processes.
- On `SIGTERM` or `SIGINT`, the script forwards the signal to the application and waits until the application exits. It then shuts the embedded PostgreSQL server down in fast mode and waits for that shutdown. The container exits with the application's exit status.
- The application can still write to the metadata database while it shuts down, because PostgreSQL stops only after the application has exited.
- When the application exits on its own, the script shuts PostgreSQL down cleanly and the container exits with the application's status.
- When the PostgreSQL server exits while the application runs, the script stops the application and the container exits with a non-zero status, so a restart policy can start it again.
- A stop signal during startup ends startup. A PostgreSQL process that is already running is shut down, the application never starts, and the container exits with a non-zero status.
- The Docker filesystem suite gains two cases. They stop and break the container in these ways and read the cluster state of the stopped volume to prove a clean shutdown.

## Capabilities

### New Capabilities

- `container-shutdown`: how the container stops its application and the embedded PostgreSQL server, in which order, and with which exit status.

### Modified Capabilities

None.

## Impact

- `docker/start.sh`: `main` splits into a root phase and a runtime phase under `databasus`, entered with `--as-databasus`. The PostgreSQL steps drop their `gosu databasus` prefix, the single-user steps run in the background, and a new supervision step with signal handling replaces the final `exec`.
- `.trivyignore.yaml`: the justification for the root entrypoint describes the two-phase start.
- `e2e/docker-storage/run.sh`: two new cases for shutdown behavior.
- No backend, frontend, API, database migration, configuration variable or dependency change.

## Out of scope

- Raising the stop timeout. Docker's default of 10 seconds and the Helm chart's pod termination period stay as they are, and the installation documentation does not change. When the timeout runs out, the container is killed as it is today.
- Making the application wait for its background tasks after a stop signal. Today it exits once the HTTP server has drained (`backend/cmd/main.go:330-361`).
- Adding an init process such as `tini` to the image.
- Changing the automatic write-ahead log reset.
