## 1. Log rotation test

- [x] 1.1 In `backend/internal/util/logger/file_sink_test.go`, turn the "at most four files" assertion of `Test_RotatingFileWriter_WhenSizeExceeded_KeepsAtMostMaxBackupsPlusActive` into an `assert.Eventually` poll with a deadline of a few seconds (design.md - Decision 3). Keep the "more than one file" and per-file size checks, run after the poll. Verify with `just test-package -count=50 -run Test_RotatingFileWriter ./internal/util/logger/` from `backend/`.

## 2. Container boot retry

- [x] 2.1 In `backend/internal/util/testing/containers/containers.go`, move the `GenericContainer` call into one shared function that both `startContainer` and `startUnpublished` call. It makes up to three attempts, terminates the container a failed attempt returned, logs each failed attempt with `t.Logf` (attempt number, image, error), and fails the test with the last error and the attempt count (design.md - Decision 1). Verify that real containers boot through the new path with `just test-package ./internal/features/databases/databases/mongodb/... ./internal/features/databases/databases/mysql/...` from `backend/`, and check the terminate-and-log branch by reading the diff.

## 3. Bastion readiness after restart

- [x] 3.1 In `backend/internal/features/tests/physical/postgresql/shared/sshtunnel.go`, replace the TCP dial in `waitForBastionAccepting` with an authenticated SSH handshake using `SshBastionUsername` and `SshBastionPassword`, retried every 500 ms under the existing 90-second deadline. Rename the function after what it now proves and update its comment (design.md - Decision 2). Verify with `just test-package -run Test_PhysicalWalStream_WhenTheBastionDiesMidStream ./internal/features/tests/physical/postgresql/...` from `backend/`.

## 4. Lint

- [x] 4.1 Run `just lint` in `backend/` and fix every finding in the touched files.
