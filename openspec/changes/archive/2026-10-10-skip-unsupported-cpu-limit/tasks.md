## 1. Read host capabilities from one `docker info` call

- [x] 1.1 In `agent/verification/internal/features/container`, add a pure function that translates a `docker info` response into host capabilities: whether user-namespace remapping is on, and whether the CPU limit is supported (`CPUCfsPeriod` and `CPUCfsQuota` both true). Verify with table-driven unit tests covering both CFS flags true, each one false, and the userns security option present or absent. Run them with `go test -race -count=1 -run Test_ToHostCapabilities ./internal/features/container/` in `agent/verification`.
- [x] 1.2 Replace `dockerEngine.UserNSRemapEnabled` with one engine method that calls `docker info` once and returns those capabilities through the function from 1.1. Delete `UserNSRemapEnabled`. Verify that `grep -rnw UserNSRemapEnabled agent/verification` finds nothing and `make build` in `agent/verification` succeeds.

## 2. Apply the CPU limit only when supported

- [x] 2.1 Give `Manager` a boolean for CPU limit support, set to true in `NewManager`, and make `buildSpec` set `NanoCPUs` to 0 when it is false. Verify with a `buildSpec` test in `postgres_test.go` that asserts `NanoCPUs` is 0 while the memory limit, the process limit, the dropped capabilities, the minimal capability set and no-new-privileges are unchanged. Keep the existing test asserting the per-job `NanoCPUs` when support is true. Run them with `go test -race -count=1 -run Test_BuildSpec ./internal/features/container/` in `agent/verification`.
- [x] 2.2 Update `StartupSelfCheck` to use the method from 1.2. Keep the user-namespace warning. Store CPU limit support on the Manager. When support is false, log one warning that restore containers run without a CPU limit and can use every CPU core on the host. When `docker info` fails, keep the existing warning and leave the CPU limit on. Verify that `make build` in `agent/verification` succeeds and that reading `StartupSelfCheck` shows each of the three outcomes.

## 3. Lint and review

- [x] 3.1 Run `make lint` in `agent/verification` and fix every finding until it passes.
- [x] 3.2 Run the mandatory post-implementation review from `AGENTS.md` on the diff and these artifacts, and resolve every `CHANGES REQUIRED` finding.
