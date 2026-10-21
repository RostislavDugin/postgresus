## Context

See proposal.md for the motivation. Paths below are under `backend/internal/features/backups/backups/usecases/physical/postgresql/`.

- The lag monitor ticks every 30 seconds and passes one `slotBreakSample` to `slotBreakClassifier.recordSampleAndClassifyBreak` (`lag_monitor.go:121-170`). The sample carries the slot state, `WalLagThresholdBytes`, `IsReceiverRunning` and the observation time (`lag_monitor.go:54-59`).
- The classifier checks, in order: a foreign consumer, `lost`, `extended`/`unreserved` with a 5-minute hold, and then the lag threshold. The lag rule is one comparison on the current sample: `LagBytes > WalLagThresholdBytes && IsReceiverRunning` (`lag_monitor.go:110-113`). The comment above it promises that lag Databasus caused by holding the receiver down never costs the chain (`lag_monitor.go:64-71`).
- `isReceiverRunning` is set right after `pg_receivewal` starts and cleared when it exits (`receivewal_process.go:96-97`). Back pressure stops the receiver and the supervision loop waits for the queue to drain before it spawns a new one (`wal_backpressure.go:41-64`, `wal_stream.go:340`). The first tick after the spawn therefore sees the lag built during the pause, with the receiver running.
- The slot-LSN watcher reads the same flag for its stall check and restarts a receiver whose `restart_lsn` has been frozen for 60 seconds (`slot_lsn_watcher.go:112-150`). Back pressure restarts it at the high watermark (`wal_backpressure.go:34-35`). This change keeps both, and the lag rule must survive their restarts.
- The classifier is covered by pure unit tests on single samples (`lag_monitor_test.go:20-150`). `Test_RecordSampleAndClassifyBreak_WhenLagOverThresholdAndReceiverRunning_Rebuilds` pins the current one-sample behavior.

Constraints from `backend/AGENTS.md`: unit tests are allowed for complex model logic with no API surface, which the classifier is; test names follow `Test_WhatWeDo_WhichConditions_WhatWeExpect`; `time.Now().UTC()`.

## Goals / Non-Goals

**Goals:**

- The lag rule fires only for lag the running receiver does not reduce.
- The decision stays a pure function of the samples, testable without a database.

**Non-Goals:**

- Changing the `lost`, foreign consumer or `extended`/`unreserved` rules.
- Persisting classifier state across sessions. A new session starts with an empty window, which only delays a rebuild.

## Decisions

### D1. One run-start timestamp instead of the running flag

`isReceiverRunning atomic.Bool` (`wal_stream.go:208`) becomes one atomic run-start time, set from the existing `startedAt` in `receivewal_process.go:100` and cleared when the process exits. Zero means stopped. The lag monitor and the slot-LSN watcher derive "running" from a non-zero value, so the two can never disagree. `slotBreakSample` replaces `IsReceiverRunning` with `ReceiverRunStartedAt`; a zero value means the receiver was stopped at that tick.

The sample is built by a small method on the supervisor, so a test can check the wiring through what the classifier receives rather than through the atomic itself.

Alternative rejected: *a second atomic next to the flag.* Two fields stored separately let a reader see "running" with no start time.

### D2. A lag window over running time

The classifier keeps a lag window: the added-up running time, the added-up lag change, and the previous sample's run start and lag. On each sample that passes the earlier rules:

1. When the threshold is zero, or the lag is at or below the threshold, the window is cleared.
2. When the receiver is stopped, nothing is added, and the previous sample is forgotten, so the next pair starts inside a run.
3. When the previous sample belongs to the same run, the time between the two samples is added to the running time and the lag difference to the change. A pair that spans two runs adds nothing, because the lag between them grew while no receiver ran.
4. When the running time reaches 5 minutes:
   - if the lag shrank by less than 5% of the threshold, the classifier asks for a rebuild with `breakReasonWalLag` and clears the window;
   - otherwise the receiver is draining, and the window starts again from this sample.

The `lost` and `extended`/`unreserved` branches clear the window as well, so lag seen during such an episode never counts afterwards.

Restarts no longer reset the window, so a receiver the slot-LSN watcher restarts every 60 to 70 seconds (`slot_lsn_watcher.go:112-150`), or that back pressure stops often, still reaches the rule when its runs do not reduce the lag. Back pressure's own runs drain the queue at network speed, so their lag shrinks and the window keeps restarting.

The 5% tolerance makes "not reducing" robust to a lag that wobbles on a busy source. For the default 256 MiB threshold it is about 13 MiB over 5 minutes, roughly 43 KiB/s of drain, which is effectively a receiver that only keeps pace.

Alternatives rejected:

- *A window per receiver run.* A watcher or back-pressure restart every minute would reset it, so a stuck receiver would never be rebuilt.
- *A fixed grace period after each receiver start.* A backlog that takes longer than the grace to drain still costs the chain, and the right grace depends on the source's write rate and the network.
- *Rebuilding only when the lag grew between two samples.* A receiver that keeps exact pace while far behind holds the excess WAL forever, and the operator's threshold says that excess is too much.
- *Restarting the window on any decrease.* A lag that wobbles by a few kilobytes would restart it forever.

### D3. The hold reuses the slot status hold

The window length is `warningSlotStatusHoldPeriod`, 5 minutes, the hold the classifier already applies to `extended` and `unreserved`. One hold for both rules keeps "how long Databasus waits before acting on a slot condition" a single value. The 5% tolerance is a new named constant next to it.

Alternative rejected: *a separate hold constant.* No observed need for a different value, and two holds of the same length drift apart in later edits.

### D4. Rebuild requests keep their existing guards

The classifier only asks; `rebuildSlot` still applies the cap of three attempts an hour (`slot_rebuild.go:17-27`) and refuses to drop a slot held by a consumer it cannot attribute to its own receiver (`slot_rebuild.go:41-53`). This change touches neither.

## Risks / Trade-offs

- [A receiver that genuinely cannot keep up is rebuilt after 5 minutes of running time instead of at once.] → The source holds at most that much more WAL than the threshold. `max_slot_wal_keep_size` and the slot retention alert still bound the worst case.
- [While back pressure keeps pausing the receiver and each run drains the lag, the lag rule never fires and the source keeps WAL until archiving catches up.] → This is what the comment always promised. The source is protected by `max_slot_wal_keep_size` and warned by the slot retention alert, and the Monitoring tab shows the backlog.
- [A lag that shrinks slowly but by more than 5% of the threshold per 5 minutes is never rebuilt.] → It ends by itself. If it slows below that, the next window asks for a rebuild.

## Migration Plan

No data or schema change. The classifier state lives in memory per session, so a deploy simply starts with empty windows. A rollback restores the old one-sample rule.
