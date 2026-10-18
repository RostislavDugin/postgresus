## Why

Databasus drops and recreates the replication slot when the slot falls behind by more than `WalLagThresholdBytes`, 256 MiB by default. The rebuild leaves a gap in the WAL chain and forces a new full backup. The code intends this as the operator's "sacrifice the chain to protect the primary" knob, and promises in a comment that it "never fires against lag we caused by holding the receiver down ourselves" (`backups/backups/usecases/physical/postgresql/lag_monitor.go:64-71`).

The check does not keep that promise. It only asks whether the receiver is running at the moment of the 30-second tick (`lag_monitor.go:110-113`). Back pressure stops the receiver while the local queue drains, and the slot falls behind during the pause. When the receiver starts again, the next tick sees a lag above the threshold with the receiver running, and drops the slot, although the receiver is just about to catch up. The same happens after Databasus was down for a few minutes in front of a busy source. A busy database writing to a distant storage, the case `parallel-wal-segment-uploads` was built for, can therefore lose its chain on every long pause.

## What Changes

- The lag threshold asks for a rebuild only when the running receiver fails to reduce the lag: over 5 minutes of time the receiver spent running, the lag stayed above the threshold and shrank by less than 5% of it. Time while the receiver is stopped does not count, and receiver restarts do not reset the count.
- A receiver that is catching up after a back-pressure pause, a receiver restart or a Databasus restart no longer costs the chain, however large the lag it starts from.
- A receiver that runs and still falls behind, or only keeps pace while far behind, is rebuilt as before after 5 minutes of running time, also when the stall watcher keeps restarting it.
- A lost slot and a slot held by a foreign consumer are still handled on the first sample. Every rebuild keeps the existing limit of attempts per hour.

## Capabilities

### New Capabilities

- `physical-backups/wal-slot-rebuild`: when Databasus drops and recreates the WAL streaming replication slot, and when it must not.

### Modified Capabilities

None.

## Impact

- Module rules: `AGENTS.md` and `backend/AGENTS.md`.
- Backend:
  - the slot break classifier in `lag_monitor.go` gains a lag window over running time;
  - the receiver's running flag becomes the start time of its current run, zero while stopped;
  - the classifier tests change, because one sample above the threshold no longer rebuilds.
- The Monitoring tab from `wal-stream-monitoring-tab`, already implemented, estimates the time to a slot rebuild only up to the moment the lag reaches the threshold (`fillEstimates` in `wal_health/keeping_up.go`). After this change the rebuild comes at least 5 minutes of running time later, and only when the lag does not shrink, so the estimate runs early until its follow-up lands.
- Source disk: while back pressure keeps pausing the receiver, the threshold no longer drops the slot, so the source keeps the WAL until the receiver catches up. `max_slot_wal_keep_size` on the source and the existing slot retention alert remain the protection for that case, as the comment already intends.

## Out of scope

- Changing the default threshold or making the hold period configurable.
- Showing the threshold in the backup settings UI.
- The slot stall watcher (`slot_lsn_watcher.go`) and the rebuild attempt cap.
- Back pressure itself and the watermark sizes.
- Requesting a full backup after the silent slot recreate at session start.
- Follow-up: adding the hold of running time to the Monitoring tab's slot rebuild estimate (`EstimatedSlotRebuildSeconds`) and its `rebuildConsequence` text.
