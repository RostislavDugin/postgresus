## Context

See `proposal.md` for the motivation. `main` in `docker/start.sh` ends with `exec gosu databasus ./main` (`docker/start.sh:603`). `gosu` replaces itself with the application, so the application takes over process ID 1. The PostgreSQL server was started in the background earlier (`start_postgresql`, `docker/start.sh:398-419`) and becomes a child of the application. The image declares no `STOPSIGNAL`, so `docker stop` sends `SIGTERM` to process 1 only. The application handles it (`backend/cmd/main.go:345`), drains HTTP requests for up to 10 seconds and returns. When process 1 exits, the kernel kills every remaining process in the PID namespace with `SIGKILL`, PostgreSQL included.

A spike with `postgres:17` reproduced the current pattern. A stand-in application ran as process 1 and PostgreSQL as its child. After `docker stop`, the next start logged `database system was not properly shut down; automatic recovery in progress`. With the supervisor described below, the same stop took 1.7 seconds and the container exited with `0`. The stand-in application still queried the database from its own `SIGTERM` handler, and the next start logged `database system was shut down at ...` with no recovery.

`docker/start.sh` runs with `set -e` (`docker/start.sh:2`).

The approach answers to the root `AGENTS.md`: no compatibility shims and refactoring of the touched startup code within the change. ADR-0003 embeds PostgreSQL as a process in the application container. This change keeps that topology and only orders its shutdown. No module `AGENTS.md` governs `docker/` or `e2e/`.

## Goals / Non-Goals

**Goals:**

- A stop within the timeout leaves the cluster cleanly shut down.
- The application keeps database access for its whole shutdown.
- The container's exit status reflects the application's outcome, and an unexpected PostgreSQL exit is visible to a restart policy.

**Non-Goals:**

- Guaranteeing a clean shutdown when the stop timeout expires first. That case stays a `SIGKILL`, as today.
- Supervising processes started with `docker exec`.

## Decisions

### 1. Keep the startup script as process 1 and supervise both services

The script runs in two phases. The root phase checks for a legacy volume, sets the runtime identity, writes the frontend configuration and prepares storage, because these steps change ownership, modes and accounts. It ends with `exec gosu databasus /app/start.sh` and a phase argument. The runtime phase runs under `databasus` in the same process 1. It runs every PostgreSQL step, starts the application in the background as `./main &` and records its process ID beside `postgres_pid`. It then waits with `wait -n` on both IDs. The script stays process 1, the process that receives the stop signal.

Process 1 therefore stays non-root, as `docker-storage-compatibility` requires and `assert_single_runtime_identity` checks through `/proc/1`. PostgreSQL and the application are started by the runtime phase itself, so they are real children of its shell and `wait` can collect them. The PostgreSQL steps no longer need a `gosu databasus` prefix.

Alternatives rejected:

- Keep the whole script under root as process 1. A root process would then live for the life of the container, which breaks the single non-root identity of all bundled services.
- Bootstrap PostgreSQL under root and `exec gosu databasus` into supervision afterwards. The running server would not be a child of the new shell, so `wait` and `wait -n` could not collect it.

Other alternatives rejected:

- Install `tini` or `dumb-init` as the entrypoint. With signal forwarding to the whole process group, PostgreSQL would receive `SIGTERM` at the same moment as the application. `SIGTERM` starts a smart shutdown, which refuses new connections while the application may still need one to finish. The entrypoint would also gain a dependency that solves only half of the problem, because the ordering logic still has to live in a script.
- Let the application stop PostgreSQL after its own shutdown. The backend also runs against an external metadata database in development and in deployments that set `DATABASE_DSN`. It would need to know whether the database is the embedded one and where its data directory lives, which couples it to the image layout.
- Rely on `docker run --init`. It is the operator's choice, not the image's, and it has the same ordering problem as `tini`.

### 2. Stop the application first, then PostgreSQL in fast mode

A `trap` on `TERM` and `INT` sends the received signal to the application. The supervisor then waits for the application to exit and logs `Databasus application exited with status <n>`. Next it sends `SIGINT` to the postmaster, which requests a fast shutdown. PostgreSQL disconnects remaining sessions and writes a shutdown checkpoint, and the supervisor waits until the postmaster process has exited. The supervisor signals its own child instead of calling `pg_ctl stop -m fast -w`, because `pg_ctl` finds the server through `postmaster.pid`, which a server that has just started may not have written yet. The container exits with the application's status.

Alternatives rejected:

- Smart shutdown (`SIGTERM` to the postmaster). It waits for every session to end. A session left by an application that was killed would hold the container until the stop timeout.
- Immediate shutdown (`-m immediate`). It skips the checkpoint, so the next start runs crash recovery, which is the behavior this change removes.
- Stop both services in parallel. The application's shutdown records interrupted backups (`openspec/specs/logical-backup-outcome/spec.md`, scenario "Application shuts down during a backup") and needs the database for that.

### 3. Survive interrupted waits and `set -e`

In Bash, a trapped signal interrupts `wait` immediately with a status above 128, before the awaited process has exited. The spike's first version stopped PostgreSQL while the application was still running because of this. The supervisor therefore repeats `wait -n -p exited_process_id` on both process IDs until Bash names the process that exited, and other waits repeat `wait` while `kill -0` succeeds. Every `wait` collects its status as `wait ... || status=$?`, so `set -e` does not end the script on a non-zero child status.

Alternative rejected: drop `set -e` for the whole script. The startup steps before supervision rely on it to stop at the first failure.

### 4. Install the stop trap at the start of `main`

Each phase sets the trap first. In the root phase no PostgreSQL process exists, so the handler only logs `Stop requested during startup` and exits. Between the `exec` and the runtime phase's trap, Bash briefly ignores a stop signal, because the kernel drops signals to process 1 that it does not handle. No PostgreSQL process runs in that moment, and Docker's stop timeout ends the container at worst. In the runtime phase, while no application runs, the handler first logs `Stop requested during startup`. It then stops the PostgreSQL process running at that moment and waits for it to exit, repeating `wait` while `kill -0` succeeds as in decision 3. The regular server receives `SIGINT` for a fast shutdown as in decision 2, a single-user step `SIGTERM`. Finally the handler exits with status `143` for `SIGTERM` or `130` for `SIGINT`. Exiting before the PostgreSQL process has ended would let the kernel kill it as soon as process 1 is gone, possibly during its shutdown checkpoint.

Docker delivers the stop signal to process 1 only, and Bash runs a trap only after the current foreground command returns. The single-user steps from `remove-embedded-postgresql-socket` therefore run in the background, followed by `wait` on their process ID. The handler sends them `SIGTERM`. In a spike, `SIGTERM` ended a single-user backend waiting for input within a second, while `SIGINT` did not. Recovery that was interrupted this way runs again on the next start, because the write-ahead log is unchanged.

Alternatives rejected:

- Ignore signals until the application runs. A stop during a long crash recovery would then wait for the full stop timeout and end in `SIGKILL`.
- Keep the single-user steps in the foreground. The trap would wait for a recovery of unbounded length.

### 5. Report an unexpected PostgreSQL exit by stopping the container

When `wait -n` returns because PostgreSQL ended first, the supervisor logs the event, sends `SIGTERM` to the application, waits for it and exits with status `1`. Nothing restarts PostgreSQL in place. A restart policy or the operator then starts a fresh container, which runs the normal recovery path.

Alternative rejected: restart PostgreSQL inside the running container. The application would see a database that disappeared and returned under it, and the supervisor would turn into a process manager.

### 6. Test shutdown in the filesystem suite

Two cases drive the real image.

`clean-shutdown` stops a healthy container with `docker stop` and with `docker kill --signal INT`, sends `SIGTERM` to the application process alone and kills the postmaster with `SIGKILL`. It checks exit status and duration. After each stop it reads the cluster state with `pg_controldata` from a helper container on the stopped volume. `Database cluster state: shut down` proves a clean stop, and `in production` proves a kill. The log of the next start cannot tell the two apart. The open step from `remove-embedded-postgresql-socket` replays the log, writes a shutdown checkpoint and discards its output, so the regular server always reports a clean previous shutdown. For the order of events, the supervisor logs `Databasus application exited with status <n>`, and the case asserts that this line precedes PostgreSQL's `received fast shutdown request`.

`startup-stop` holds the container inside a startup step without depending on timing. A `docker exec` loop inside the container waits for the target process and sends it `SIGSTOP`. The target is a `postgres --single` process in one part and the postmaster of the regular server in the other. With the step frozen, the case runs `docker stop` in the background. In the single-user part it asserts that `Stop requested during startup` appears while the target is still stopped. A foreground implementation would print that line only after the step had ended, so this part proves the background execution. The regular-server part skips that assertion. There the script may be waiting in a foreground `pg_isready` or `psql` call on the frozen postmaster, and Bash runs the trap only when that call returns. Both parts then send `SIGCONT`, so the pending signal reaches the target, and assert the exit status and the missing application start. The regular-server part also asserts the cluster state `shut down`.

Alternative rejected: a shell unit test with stand-in processes. It would test a copy of the supervision logic instead of the image the operator runs.

## Risks / Trade-offs

- [The application's HTTP drain takes up to 10 seconds, which equals Docker's default stop timeout, so PostgreSQL can still be killed] -> An idle application drains at once, and the spike's fast shutdown took about 0.1 seconds. Under load the result is no worse than today. Raising the timeout is out of scope.
- [The startup script now reaps orphaned processes as process 1] -> Bash collects any terminated child while it waits. Today the Go application is process 1 and does not reap processes it did not start.
- [A stop signal that arrives between the `exec` into the runtime phase and its trap is ignored] -> The window lasts until Bash has parsed the script, and no PostgreSQL process runs in it. Docker's stop timeout then kills a container that holds no database process.
- [`docker exec` sessions started during shutdown see the database disappear] -> They see the same thing today, earlier and without a checkpoint.
- [A single-user backend in the middle of write-ahead log replay may not act on `SIGTERM` at once; the spike only covered a backend waiting for input] -> The stop timeout then ends it with `SIGKILL`. The log stays unchanged, and the next start repeats the replay.
- [The open step of a clean cluster lasts about 0.1 seconds, so the `docker exec` loop could miss it] -> The loop starts as soon as the container runs and polls inside the container without a delay. Storage validation and `--test-storage` run before the first PostgreSQL step and take longer than the loop needs to start.

## Migration Plan

1. Apply after `remove-embedded-postgresql-socket`.
2. Build the image and run the `clean-shutdown` and `startup-stop` cases locally.
3. Publish. The first stop of an existing container still uses the old image's behavior. Its next start runs crash recovery once, and every later stop is clean.
4. Rollback restores the old behavior with no data change.
