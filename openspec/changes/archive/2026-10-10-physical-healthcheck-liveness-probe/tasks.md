## 1. Test fixture

- [x] 1.1 Add a `WithMaxWalSenders(n)` option to the physical PostgreSQL test container beside `WithMaxConnections` in `backend/internal/util/testing/containers/postgres_physical.go`, overriding the hard-coded `max_wal_senders=10`. Verify with `go build ./...` from `backend/`.

## 2. Liveness probe on the physical model

- [x] 2.1 Add `PostgresqlPhysicalDatabase.ProbeLiveness(ctx, logger, encryptor)` in `postgresql/physical/model.go`, next to `TestReplicationConnection`: an ordinary connection through `openConn`, `SELECT 1`, a 30-second timeout derived from `ctx`, and the driver error returned wrapped and unclassified. Verify with `go build ./...` from `backend/`.
- [x] 2.2 In `postgresql/physical/model_test.go`, add `Test_ProbeLiveness_WhenWalSendersExhausted_Succeeds`: start a source with `WithMaxWalSenders(1)`, hold the slot with an open replication connection, then assert that `ProbeLiveness` succeeds and that `TestReplicationConnection` fails, so the fixture is proven to have exhausted the slot. Verify with `go test ./internal/features/databases/databases/postgresql/physical/... -run Test_ProbeLiveness -count=1`.
- [x] 2.3 In the same file, add `Test_ProbeLiveness_WhenPasswordIsWrong_FailsWithoutLeakingPassword`: the probe returns an error, the error carries the server's authentication refusal, and the error text does not contain the wrong password. Verify with the same command as 2.2.

## 3. Type dispatch

- [x] 3.1 Add `Database.ProbeLiveness(ctx, logger, encryptor)` in `databases/model.go`: a physical source with a nil `PostgresqlPhysical` returns "postgresql physical config is not set", a physical source calls `PostgresqlPhysical.ProbeLiveness`, and every other type runs `TestConnection`. The doc comment explains why non-physical types still run their full connection test: they share no replication capacity with backups, so nothing needs protecting there. Verify with `go build ./...` from `backend/`.
- [x] 3.2 In `databases/model_test.go`, add `Test_ProbeLiveness_ForEveryDatabaseType_WithoutEngineConfig_ReturnsErrorInsteadOfPanicking`, mirroring the existing `TestConnection` case table. Verify with `go test ./internal/features/databases/ -run Test_ProbeLiveness -count=1`.

## 4. Wire the healthcheck to the probe

- [x] 4.1 Rename `DatabaseService.TestTrustedDatabaseConnection` to `ProbeDatabaseLiveness` and give it its own body: open the tunnel through `OpenTunnel`, then call `ProbeLiveness` on the tunneled copy. Leave `testDatabaseConnection` serving the workspace-authorized path only. Delete the old name. Verify that `grep -rn TestTrustedDatabaseConnection backend/` returns nothing and that `go build ./...` passes.
- [x] 4.2 Rename the method on the healthcheck `DatabaseService` interface, on `MockDatabaseService`, at the call site in `CheckDatabaseHealthUseCase`, and in `check_database_health_uc_test.go`. Verify with `go test ./internal/features/healthcheck/... -count=1`.
- [x] 4.3 In a new `databases/service_test.go`, add `Test_ProbeDatabaseLiveness_WithEmbeddedPhysicalTarget_ReturnsError`, which pins that the trusted path still refuses an embedded target before any connection opens. Verify with `go test ./internal/features/databases/ -run Test_ProbeDatabaseLiveness -count=1`.
- [x] 4.4 In the same file, add `Test_ProbeDatabaseLiveness_WhenSourceIsBehindSshBastion_Succeeds`, using `containers.StartPhysicalPostgresBehindSshBastion`. Verify with the same command as 4.3.

## 5. Verification and review

- [x] 5.1 Run `make lint` in `backend/` and fix every finding in the touched files.
- [x] 5.2 Run the touched packages from `backend/` and confirm that they pass: `go test ./internal/features/databases/... ./internal/features/healthcheck/... -count=1`.
- [x] 5.3 Run the reviewer subagent on the working-tree diff and resolve every CHANGES REQUIRED finding.
- [x] 5.4 Run `openspec validate physical-healthcheck-liveness-probe --strict` and confirm that it passes.
