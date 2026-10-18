Backend commands run in `backend/`. `<pg>` stands for `./internal/features/backups/backups/usecases/physical/postgresql/...`.

## 1. Run start time

- [x] 1.1 Replace `isReceiverRunning` with one atomic run-start time, set from `startedAt` in `receivewal_process.go` and cleared on exit, and derive "running" from it in the lag monitor and the slot-LSN watcher (design D1). Build the classifier's sample in a small supervisor method, and replace `IsReceiverRunning` in `slotBreakSample` with `ReceiverRunStartedAt`; the lag rule keeps its current behavior, reading a non-zero start as running. Update the existing classifier tests to set `ReceiverRunStartedAt`. Verify with `just test-package <pg>` passing, including a new test in `wal_stream_test.go` that runs a receiver against the test source and checks through the sample-building method that the sample carries a start time while it runs and, waiting with `require.Eventually` up to a limit, a zero one after it exits.

## 2. Reproduce the bug

- [x] 2.1 Add `Test_RecordSampleAndClassifyBreak_WhenReceiverResumesWithLagAboveThreshold_DoesNotRebuild` in `lag_monitor_test.go` for the spec scenario "Resume after a back-pressure pause": one sample with a zero `ReceiverRunStartedAt` and the lag at twice the threshold, then samples with a new non-zero start time and the lag shrinking. Verify that `just test-package <pg>` fails on this test before the fix, because the first running sample asks for a rebuild.

## 3. Fix

- [x] 3.1 Replace the one-sample lag rule with the lag window over running time from design D2, using `warningSlotStatusHoldPeriod` as its length and a new 5% tolerance constant (D3), and update the comment above it. Rewrite `Test_RecordSampleAndClassifyBreak_WhenLagOverThresholdAndReceiverRunning_Rebuilds`, which pins the old behavior, as the "Receiver cannot keep up" case. Verify with `just test-package <pg>` passing, the test from 2.1 included, and with new classifier tests in `lag_monitor_test.go` covering the spec scenarios:
  - "Start after Databasus was down" and "Slow catch-up longer than 5 minutes";
  - "Short runs between pauses": a new start time every 3 minutes, lag growing between runs and shrinking within them;
  - "Receiver cannot keep up": the rebuild comes on the first sample after 5 minutes of running time, not before;
  - "Lag fluctuates above the threshold": swings below 5% of the threshold, rebuild on the first sample after 5 minutes of running time;
  - "Stalled receiver restarted repeatedly": a new start time every 70 seconds with 30-second samples and the lag growing reaches a rebuild on the sample where the intervals between samples of the same run add up to 5 minutes, and not before;
  - "Receiver stopped", "Lag drops below the threshold" and "No threshold";
  - "Extended episode ends above the threshold" and "Extended slot above the lag threshold";
  - "Lost slot" and "Foreign consumer", whose existing tests must still pass.

  The scenarios "Hourly limit reached" and the foreign consumer's refusal to drop are pinned by the existing `Test_WalStream_RebuildAttemptCap_StopsFourthAttemptInHour` and the `Test_IsOwnedReceiverBackend_*` tests in `slot_rebuild_test.go`, which must still pass; this change does not touch either path (design D4).

## 4. Wrap-up

- [x] 4.1 Run the formatter and the linters. Verify with `just format` and `just lint` at the repository root finishing clean.

## Change verification

```sh
just test-package ./internal/features/backups/backups/usecases/physical/postgresql/...
```
