## 1. Security regressions

- [x] 1.1 Add conninfo tests proving that parameter-looking host, username, password, and database values remain literal, and verify the focused PostgreSQL shared-package test fails before the quoting fix
- [x] 1.2 Add shared-classifier tests for Unix sockets, local aliases, comma-separated host fallback, and libc-compatible numeric loopback forms, plus logical and physical model regressions for `/tmp`; verify they fail before target classification changes
- [x] 1.3 Add engine-level connection-time tests for persisted logical and physical embedded targets and a database-level logical regression, and verify the focused `OpenTunnel` tests fail before runtime enforcement
- [x] 1.4 Add controller tests for missing workspace, unauthorized and unknown saved IDs, authorized workspace members, repository failure, and ad hoc plus saved membership failures, and verify the vulnerable authorization and unsanitized-error cases fail
- [x] 1.5 Build the pre-fix image with `docker build` and verify that a container configured with `PUID=65532` becomes healthy instead of rejecting the UID collision

## 2. PostgreSQL target and conninfo protection

- [x] 2.1 Add one shared embedded-target classifier for logical and physical PostgreSQL configurations, including remote SSH semantics, and verify all classifier tests pass with `go test -count=1 ./internal/features/databases/databases/postgresql/shared`
- [x] 2.2 Call embedded-target validation from logical and physical model validation and immediately before opening engine-level and database-level tunnels, and verify the logical, physical, and database package tests pass
- [x] 2.3 Quote every string field used to build PostgreSQL conninfo and verify parsed connection fields exactly match the submitted literal values
- [x] 2.4 Pass the logical dump database as a quoted one-field conninfo value and verify the dump argument test preserves `postgres dbname=databasus` as one database name

## 3. Direct connection-test authorization

- [x] 3.1 Add internal database-store and workspace-service seams for controlled controller failure tests, and verify production dependency wiring compiles through `make lint`
- [x] 3.2 Require `workspaceId` and database-management permission for unsaved direct tests, verify the missing-workspace controller regression returns 400, and confirm the permission-denied branch returns before the connection path during implementation review
- [x] 3.3 Load a saved target, authorize its persisted workspace, and only then merge request fields with saved credentials, and verify a viewer or nonmember cannot redirect the saved password while an authorized member can test it
- [x] 3.4 Return the same 403 response for unknown and unauthorized saved IDs, and verify the controller responses are byte-equivalent JSON objects
- [x] 3.5 Wrap saved-target and permission lookup failures with separate sentinels, map them to sanitized HTTP 500 responses, and verify injected sensitive errors never appear in the response body
- [x] 3.6 Split the trusted healthcheck connection path from the user-authorized direct endpoint, update its interface and mocks, and verify `go test -count=1 ./internal/features/healthcheck/attempt` passes

## 4. Embedded PostgreSQL runtime isolation

- [x] 4.1 Create `/databasus-data/pgsocket` for the PostgreSQL account with mode `0700`, start PostgreSQL there, and verify the Databasus application account cannot connect through the socket
- [x] 4.2 Rewrite the internal HBA policy at startup to allow PostgreSQL peer administration, reject other local socket users and replication, and require SCRAM over loopback TCP; verify the running container reports the expected rules
- [x] 4.3 Generate a random internal PostgreSQL password at each startup, set it through the private socket, and export a matching `DATABASE_DSN` only when the operator did not supply one; verify the container becomes healthy and the password verifier changes after restart
- [x] 4.4 Update legacy WAL checks to use the generated credential without logging it, and verify startup completes against an existing metadata database
- [x] 4.5 Reject a requested PostgreSQL UID owned by another image account, and verify `PUID=65532` exits with code 1 while an unused `PUID=23456` starts healthy under that UID
- [x] 4.6 Verify the previously published fixed password fails over loopback TCP and no PostgreSQL socket is created under `/tmp`

## 5. Final verification

- [x] 5.1 Run `make lint` from `backend/` and verify it reports `0 issues`
- [x] 5.2 Run `make test-fedora` from `backend/` and verify the complete backend suite passes in the supported Fedora host-native environment
- [x] 5.3 Re-run all packages touched by the final review fixes with `go test -count=1` and verify the database, PostgreSQL shared/logical/physical, logical backup, and healthcheck packages pass
- [x] 5.4 Run `docker build --check .` and a full image build, then verify collision and non-conflicting custom-UID smoke scenarios
- [x] 5.5 Complete the mandatory planning and implementation reviews against `AGENTS.md` and `backend/AGENTS.md`, resolve every required change, and verify the final implementation review reports `PASS`
- [x] 5.6 Run `git diff --check`, remove test containers and images, and verify the working tree contains only the implementation and this completed OpenSpec change
