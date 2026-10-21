## Context

See proposal.md for why. The flaky failures come from five separate causes, so this design has one decision per cause. Each was found in CI logs from the last two months, and none reproduced in three local full runs on an idle machine.

**Error output of child processes.** Ten places start a database tool, read its standard error through `cmd.StderrPipe()` in a goroutine, and call `cmd.Wait()` before that goroutine has finished reading:

- Restores: `restores/usecases/mysql/restore_backup_uc.go:205-221`, `mariadb/restore_backup_uc.go:224-240`, `mongodb/restore_backup_uc.go:213-229`, and `postgresql/restore_backup_uc.go:297-333` and `:693-712`.
- Logical backups: `backups/backups/usecases/logical/postgresql/create_backup_uc.go:165,217,242`, `mysql/create_backup_uc.go:220,275,304`, `mariadb/create_backup_uc.go:223,278,307` and `mongodb/create_backup_uc.go:165,216,241`. The stdout copy finishes before `Wait`, which narrows the window, but a tool that fails before writing any data still loses its message.
- The WAL receiver: `backups/backups/usecases/physical/postgresql/receivewal_process.go:83-107` starts `cmd.Wait()` in a goroutine right after `cmd.Start()`. The fatal-error check at `:144` and the resume-mismatch check at `:131` then read whatever the capture got before `Wait` closed the pipe.

The `os/exec` documentation forbids this order: `Wait` closes the read end of the pipe, and output not yet read is lost. The CI failure `"mysql-stub failed: exit status 1 – stderr: " does not contain "SQL syntax"` in `restores/usecases/mysql/restore_backup_uc_test.go:103` is this race. It failed four times. `pg_basebackup.go:229` and `stream.go:204-217` already drain the output correctly before `Wait` and are not touched.

**Test container readiness.** `util/testing/containers/ssh_bastion.go:210`, `samba.go:39` and `ftp.go:56` wait with `wait.ForListeningPort` only. Docker's port proxy accepts on the mapped port before the service inside listens, so the first connection is reset during the handshake. SFTP had the same failure and now also waits for its server's log line (`sftp.go:23-31`). The FTP failure `failed to dial the container after adding egress delay: connection refused` (`netem.go:35`) dials the container's own address, which bypasses the proxy, and has the same cause: the server was not listening yet.

**Storage failure before the dump starts.** Each logical backup starts the storage write, `storage_files.StartBackgroundWrite`, before it starts the dump tool, for example `logical/mysql/create_backup_uc.go:250` and `:258`. A storage that fails at once cancels the backup context, and `exec.CommandContext` then refuses to start the tool. The backup fails with `start mysqldump: context canceled` instead of the storage error. The shared assertion at `backups/backups/core/logical/testing.go:79-90` (`Test_Execute_WhenStorageFailsBeforeReading_ReturnsStorageError`) caught this once in CI.

**Fixed waits in tests.**

- `telemetry/background_service_test.go:180-202` starts `Run` in a goroutine, sleeps 10 ms and expects a second `Run` to panic. If the goroutine has not reached `hasRun.Swap(true)` (`background_service.go:55`) by then, the test's own call becomes the first one and blocks. The goroutine's call then panics outside `require.Panics`, and that kills the whole test binary of the package.
- `agent/verification/internal/features/api/idle_timeout_reader_test.go:14-71` uses a 100–200 ms idle timeout against writes every 50 ms and sleeps of 200 ms. That leaves 100–150 ms of slack for a loaded runner.

**Log order in the Docker filesystem suite.** `e2e/docker-storage/run.sh:1024-1040` compares line numbers in `docker logs 2>&1`. The supervisor writes `Databasus application exited with status <n>` to standard output (`docker/start.sh:686`), and PostgreSQL writes `received fast shutdown request` to standard error, its default log destination, because `start.sh:497-503` passes no log settings. Docker copies the two streams in separate goroutines and does not order them against each other.

## Goals / Non-Goals

**Goals:**

- Every failure listed in the proposal becomes deterministic: it passes on a loaded runner because the code no longer depends on timing, not because a limit was raised.
- Each bug fix starts with a test that fails on today's code for the reason described above.

**Non-Goals:**

- Bounding the size of captured error output. Today's capture is unbounded too.
- Merging the four logical backup usecases or the restore usecases into shared code beyond what each decision below needs.
- Changing `pg_basebackup.go` or `stream.go`, which already drain their output correctly.

## Decisions

### 1. Capture error output with `cmd.Stderr` set to a buffer

Each of the ten places sets `cmd.Stderr` to a `bytes.Buffer` and drops the pipe and its goroutine. `os/exec` then copies the output itself, and `cmd.Wait()` returns only after the copy has finished. Reading the buffer after `Wait` therefore always sees all the output. The WAL receiver keeps its `cmd.Wait()` goroutine and its `select`, reads the buffer after `exited` delivers, and drops `newStderrCapture` from its file.

When `Wait` waits for that copy, a grandchild process holding the inherited stderr open can keep `Wait` blocked after the tool itself exited. So every place that switches to the buffer also sets `cmd.WaitDelay` to 10 seconds, as `pg_basebackup.go:121` and `receivewal_process.go:248` already do with `killAfterCancel` (`pg_basebackup.go:27`). The restore and logical backup packages each get their own 10-second constant, because the physical backup package is not a dependency they should take. When the delay expires, `Wait` closes the pipes and returns `exec.ErrWaitDelay` even for a tool that exited with status 0. No special handling is added for that case, because none of these tools leaves a process behind after a successful exit. `pg_dump -j` and `pg_restore -j` wait for their workers before they exit, and `mysqldump`, `mysql`, the MariaDB tools, `mongodump`, `mongorestore` and `pg_receivewal` start no long-lived children.

The PostgreSQL restore over standard input (`postgresql/restore_backup_uc.go:291-359`) needs one more change. It pumps the backup through `cmd.StdinPipe()`. When `pg_restore` exits before reading everything, that copy fails with a broken pipe, and `:357-359` returns `failed to stream backup data to pg_restore` before the code reaches `resolvePgRestoreOutcome` with the tool's output. After this change, a non-nil `waitErr` goes to `resolvePgRestoreOutcome` first, and the copy error is reported only when `pg_restore` itself succeeded. Read failures of the backup stream are not affected. They are caught earlier by `GetBackupStreamFailure` (`:351-355`), because the failure-tracking reader sits on the read side of that copy. The MySQL, MariaDB and MongoDB restores pass the stream through `cmd.Stdin`, where `os/exec` ignores a broken pipe and `Wait` reports the tool's exit status, so they need no such change.

Alternatives rejected:

- **Receive from the channel before calling `Wait`.** It is a one-line move in each place, but it keeps ten copies of the goroutine boilerplate. It also makes the caller wait for end of output with no bound, while `WaitDelay` gives a bound.
- **Move `stderrCapture` from `pg_basebackup.go:196` to a shared package.** It exists for a caller that needs a drain with a timeout and a forced stop. None of the ten places reads output while the tool runs, so its extra surface buys them nothing.

### 2. Readiness checks that speak each protocol

Each of the three containers waits until a probe from the host gets a valid protocol answer through the mapped port, alongside the existing port wait:

- **SSH bastion:** read the server's identification line and require the `SSH-2.0-` prefix.
- **FTP:** read the greeting and require the `220` reply code.
- **Samba:** an SMB session through the SMB client library the NAS storage already uses (`github.com/hirochachacha/go-smb2`, `storages/models/nas/model.go:247`), logged in as `SambaUsername` with `SambaPassword` from `samba.go`, which then mounts `SambaShare`. That library runs negotiation and login together and refuses anonymous sessions, and a mounted share is also what the NAS tests need.

The probe is a `wait.Strategy` in `util/testing/containers`, written once for the line-greeting case and reused by SSH and FTP.

Alternatives rejected:

- **Wait for a log line, as `sftpReady` does.** The bastion is built from our own Dockerfile, and its sshd logging target would need checking. The Samba and FTP images are third-party, and their log lines are not a contract. A protocol answer is what the tests actually need.
- **Retry the connection inside the tests or the storage code.** That spreads a test-fixture problem across production code and many call sites, and a retry in production code would hide real connection failures.

### 3. Start the dump tool before the storage write

Each logical backup calls `cmd.Start()` before `storage_files.StartBackgroundWrite`. A storage that fails at once, before reading any backup data, can then no longer stop the tool from starting. A storage failure after the start already reaches the `save to storage` path that the after-the-dump scenario tests (`create_backup_uc.go:219-226` in PostgreSQL and the same block in the other three).

Alternative rejected: keep the order, and when `cmd.Start()` fails with a cancelled context, wait for the storage write's error and report it instead. That adds a branch to four files to explain a failure the reordering removes. The reordering also lets the start-failure path drop its storage cleanup.

The reordering is safe because the tool's stdout is an OS pipe that nothing reads until the copy goroutine starts. The tool blocks on a full pipe until then, which is the same backpressure it has today.

### 4. Tests wait for a condition or remove the race

- **Telemetry:** the test calls `Run` synchronously first with an already cancelled context. `Run` sets `hasRun`, and `sleep` at `background_service.go:97` returns at once on a cancelled context, so the call returns. The second `Run` then must panic, and no goroutine or sleep is involved. A rejected alternative polled `hasRun` with `require.Eventually`. It still needs a goroutine, and it reads private state.
- **Idle-timeout reader:** the checks that expect cancellation use `require.Eventually` with a bound of seconds. The check that expects no cancellation while bytes flow raises the idle timeout to seconds while keeping the 50 ms write interval. That keeps the test fast, because the writer still finishes in 250 ms, and leaves a margin no scheduler pause reaches. The check that `Stop` prevents cancellation keeps its fixed sleep. After `Stop` no cancellation can happen, so the sleep only gives a broken `Stop` the chance to show itself.

### 5. The supervisor's exit line goes to standard error

`docker/start.sh:686` writes its line with `>&2`. PostgreSQL inherits the same standard error, and the supervisor sends the shutdown signal only after the `echo` returned. The two lines therefore reach Docker through one pipe in order. The PostgreSQL-exit error at `:680` already goes to standard error.

Alternative rejected: compare `docker logs --timestamps`. The daemon stamps each stream's lines as its copier reads them, so the timestamps can invert just as the line order does.

## Risks / Trade-offs

- [A process that writes huge error output fills memory] → The buffer is unbounded, as `io.ReadAll` is today. No behavior gets worse.
- [A grandchild outlives the tool and holds its error output open] → After 10 seconds `Wait` closes the pipes and returns `exec.ErrWaitDelay`, so a tool that exited with status 0 would be reported as failed. Decision 1 explains why none of these tools leaves such a process behind. If one ever does, the failure names `exec.ErrWaitDelay` and points straight at the cause.
- [A readiness probe is wrong for one image and the container never becomes ready] → The probe has the existing 120-second startup timeout and fails with the probe's last error, so a wrong probe shows up as a clear setup failure in the first local run of the touched package.
- [The reordering in decision 3 changes behavior when the tool fails to start for its own reasons] → The storage write has not started yet, so there is nothing to clean up. The start error is returned as today.
- [The tests for the races still pass on broken code if the race window does not open] → Repeating a fast-exiting stub does not open it reliably: even on one processor the output got lost about once in 1500 runs. So each error-output test uses a stub that writes its error from a background process shortly after the stub itself exited, which is how a fast exit looks when the reader is scheduled late. Today's code loses that output on every run, and each task verifies the failure before the fix.
- [The storage-before-start race of decision 3 has no test that fails on today's code] → `exec.Cmd.Start` checks the context right after the storage write's goroutine is started, with no point in between where the scheduler can switch, so the store won the race in at most 6 of 3000 runs. A hook between the two calls would make it deterministic, but only for tests and in four files. The reordering removes the race by construction, and the existing `Test_Execute_WhenStorageFailsBeforeReading_ReturnsStorageError` keeps the scenario covered.

## Migration Plan

Nothing to migrate. The change ships with the next release, and rollback is a revert.
