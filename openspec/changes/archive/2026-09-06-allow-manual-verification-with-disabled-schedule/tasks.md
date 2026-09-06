## 1. Regression coverage

- [x] 1.1 Keep the pending manual-verification reproduction in `scheduler_test.go`, add a running manual-verification case, and confirm with `cd backend && go test ./internal/features/verification/runs -run 'Test_SweepCanceledByDisabledConfig_WhenManualVerification' -count=1` that the cases fail before the query fix.
- [x] 1.2 Cover disabled-schedule cancellation for `SCHEDULED` and `AFTER_BACKUP` verifications in both non-terminal states, including coexistence with a manual verification, and verify the API-visible statuses match the `backup-verification` scenarios.

## 2. Cancellation selection

- [x] 2.1 Rename `FindNonTerminalForDisabledConfigs` to `GetNonTerminalAutomaticVerificationsForDisabledSchedules` without an alias and restrict it to the explicit `SCHEDULED` and `AFTER_BACKUP` trigger allowlist.
- [x] 2.2 Rename `sweepCanceledByDisabledConfig` to `cancelAutomaticVerificationsForDisabledSchedules` without an alias, update its callers and tests, and verify the focused scheduler tests pass with neither old name left in the backend.

## 3. Final verification

- [x] 3.1 Run `cd backend && go test ./internal/features/verification/runs -count=1` and resolve every unexpected failure.
- [x] 3.2 Run `cd backend && make lint` and resolve every issue.
- [x] 3.3 Run the mandatory reviewer against the implementation diff, resolve every `CHANGES REQUIRED` finding, and leave the working tree ready to commit with a `PASS` verdict.
