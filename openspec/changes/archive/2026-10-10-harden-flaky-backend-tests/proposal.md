## Why

The `test-backend` CI job takes about 16 minutes, and one flaky test anywhere in it turns the job red and costs a full rerun. Two attempts of the same `main` run (`38072212295`, PR #824) failed on three unrelated flakes, none of them a product bug:

- A `mongo:7.0` test container exited with code 48 while booting. `mongod` failed to bind its port during the image entrypoint's second start, so the test never got a database.
- `Test_PhysicalWalStream_WhenTheBastionDiesMidStream_KeepsTheSlotAndResumes` restarted its SSH bastion, waited until the port accepted TCP, and then had its first SSH handshake reset.
- `Test_RotatingFileWriter_WhenSizeExceeded_KeepsAtMostMaxBackupsPlusActive` counted log files while the rotation library was still deleting old backups in the background, and saw five files instead of at most four.

Each failure has a known cause in test code or test infrastructure, so each is fixed where it starts. A blanket rerun of failed tests would hide the next real race in the WAL-stream resilience tests, which exist to catch exactly that.

## What Changes

- Starting a test container retries a failed boot up to three attempts in total. The failed container is removed before the next attempt, and every failed attempt is logged in the test output so a recurring boot problem stays visible. This covers every engine and auxiliary container the shared container helpers start, not just MongoDB.
- After the bastion restart, the WAL-stream outage test treats the bastion as ready only once a full SSH handshake with the test credentials succeeds through the published port. An accepted TCP connection no longer counts.
- The log rotation test waits, with a bounded deadline, for the file count to drop to the cap instead of checking it once right after shutdown. The check that rotation happened at all stays as it is.

No production code changes. Nothing is **BREAKING**.

## Capabilities

### New Capabilities

None.

### Modified Capabilities

None. The change touches only test code and test helpers, so no spec-level behavior changes and the change sets `skip_specs: true`.

## Impact

- `backend/internal/util/testing/containers/containers.go`: both container start paths gain the retry.
- `backend/internal/features/tests/physical/postgresql/shared/sshtunnel.go`: the bastion readiness wait after a restart.
- `backend/internal/util/logger/file_sink_test.go`: the rotation cap assertion.
- Governing rules: root `AGENTS.md` and `backend/AGENTS.md`, notably its Testing section and the no-sleeping principle its Stored files section states for cleanup.

## Out of scope

- Rerunning failed tests suite-wide (`gotestsum --rerun-fails` or similar). It would mask real races, and the observed flakes all have fixable causes.
- Splitting `test-backend` into several CI jobs to make a rerun cheaper. Every shard would repeat the ~22 GB image pull and setup.
- Changing `-failfast`, `-p`, timeouts or any other CI workflow setting.
- Reworking readiness checks of containers that have not flaked (Mailpit, FTP, Samba, Azurite and others that wait only for a listening port).
- Retrying inside shared connection helpers such as the physical fixture's admin connection opener. A retry there would also hide tunnel regressions in every other test that uses it.
- Any change to the production SSH tunnel, WAL streamer or file log sink.
