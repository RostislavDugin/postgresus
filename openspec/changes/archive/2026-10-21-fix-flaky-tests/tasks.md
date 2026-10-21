## 1. Error output of restore tools

- [x] 1.1 Make the MySQL restore regression test fail reliably on today's code: in `restores/usecases/mysql/restore_backup_uc_test.go`, let the client stub write its error from a background process shortly after the stub itself exited, which is how a fast exit looks when the output reader is scheduled late, and confirm that `just test-package ./internal/features/restores/usecases/mysql/...` in `backend/` fails with an empty `stderr:` before any fix.
- [x] 1.2 Switch the MySQL restore to `cmd.Stderr` set to a `bytes.Buffer` with `cmd.WaitDelay`, as design decision 1 describes, and verify that `just test-package ./internal/features/restores/usecases/mysql/...` passes.
- [x] 1.3 Add the same late-writing-stub regression test to the MariaDB, MongoDB and PostgreSQL restore packages, covering both PostgreSQL restore paths, and confirm each fails on today's code. Then apply design decision 1 in `mariadb/restore_backup_uc.go`, `mongodb/restore_backup_uc.go` and `postgresql/restore_backup_uc.go`, and verify that `just test-package ./internal/features/restores/...` passes.
- [x] 1.4 Add a regression test for the PostgreSQL restore over standard input in which a `pg_restore` stub writes an error and exits before reading the whole backup, and confirm it fails on today's code with `failed to stream backup data to pg_restore`. Then send a non-nil `waitErr` to `resolvePgRestoreOutcome` before the copy error, as design decision 1 describes, and verify that `just test-package ./internal/features/restores/usecases/postgresql/...` passes, including the pre-existing `public` schema tests in that package.

## 2. Error output of dump tools and the WAL receiver

- [x] 2.1 Add a regression test, shared through `backups/backups/core/logical/testing.go` the way the storage-timeout assertion is, in which a dump tool stub writes an error and exits before producing data. Run it for all four engines, confirm it fails on today's code, apply design decision 1 to the four logical `create_backup_uc.go` files, and verify that `just test-package ./internal/features/backups/backups/usecases/logical/...` passes.
- [x] 2.2 Add a regression test in `backups/backups/usecases/physical/postgresql/` in which a receiver stub writes `FATAL:  password authentication failed` and exits at once, and assert the receiver run ends as fatal with that text in its error. Confirm it fails on today's code, apply design decision 1 to `receivewal_process.go`, remove `newStderrCapture` from that file, and verify that `just test-package ./internal/features/backups/backups/usecases/physical/postgresql/...` passes.

## 3. Storage failure before the dump starts

- [x] 3.1 Move `cmd.Start()` before `storage_files.StartBackgroundWrite` in the four logical `create_backup_uc.go` files, as design decision 3 describes, and verify that `just test-package ./internal/features/backups/backups/usecases/logical/...` passes.

## 4. Test container readiness

- [x] 4.1 Add a greeting-probe `wait.Strategy` to `util/testing/containers` and use it for the SSH bastion (`SSH-2.0-`) and FTP (`220`) next to their port waits. Verify that `just test-package ./internal/util/testing/containers/... ./internal/features/sshtunnel/... ./internal/features/storages/...` passes.
- [x] 4.2 Add a Samba readiness probe that logs in with `SambaUsername` and `SambaPassword` through the NAS storage's SMB client library and mounts `SambaShare`, as design decision 2 describes, and verify that `just test-package ./internal/util/testing/containers/... ./internal/features/storages/...` passes.

## 5. Fixed waits in tests

- [x] 5.1 Rewrite `Test_Run_WhenCalledTwice_Panics` in `telemetry/background_service_test.go` to call `Run` first with a cancelled context and drop the goroutine and the sleep, as design decision 4 describes, and verify that `just test-package ./internal/features/telemetry/...` passes.
- [x] 5.2 Rewrite the idle-timeout reader tests in `agent/verification/internal/features/api/idle_timeout_reader_test.go` as design decision 4 describes, and verify that `just --justfile agent/verification/Justfile test` passes.

## 6. Log order in the Docker filesystem suite

- [x] 6.1 Send the `Databasus application exited with status` line in `docker/start.sh` to standard error, as design decision 5 describes. Verify that `bash -n docker/start.sh` succeeds and that `git grep -n 'Databasus application exited with status' docker/start.sh` shows the line ending in `>&2`.

## Change verification

```sh
just test-package ./internal/features/restores/...
just test-package ./internal/features/backups/backups/usecases/logical/...
just test-package ./internal/features/backups/backups/usecases/physical/postgresql/...
just test-package ./internal/util/testing/containers/... ./internal/features/sshtunnel/... ./internal/features/storages/...
just test-package ./internal/features/telemetry/...
just --justfile agent/verification/Justfile test
```
