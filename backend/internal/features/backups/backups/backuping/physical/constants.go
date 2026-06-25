package backuping_physical

import "time"

const (
	// Scheduler tick and recovery sweep share this cadence. Each tick is a handful
	// of cheap indexed queries per enabled DB, so a 1 s cadence keeps out-of-cadence
	// backup triggers and crash recovery near-immediate without meaningful load
	// (user-configured FULL/INCR intervals still span hours to days — the tick only
	// decides whether one is due).
	schedulerTickInterval = 1 * time.Second

	// IsSchedulerRunning() reports unhealthy when no tick completed within this
	// window. Mirrors logical's healthcheck threshold.
	schedulerHealthcheckThreshold = 5 * time.Minute

	// Stable log job_name for the scheduler; never the struct type name, since a
	// rename would silently break log queries.
	schedulerJobName = "physical_backup_scheduler"

	// Cleaner ticks every 3 s so retention decisions become visible almost
	// immediately, while the per-tick WAL byte budget keeps a single tick bounded
	// even at this cadence. Mirrors logical.
	cleanerTickInterval = 3 * time.Second

	// Never delete a chain whose end timestamp is younger than
	// max(full, incr cadence) × 2 — protects a chain that just completed or is
	// still being extended from premature retention eviction.
	chainGraceIntervalMultiplier = 2

	// A WAL row with file_name still NULL past 1 h is an abandoned insert-first
	// claim (the live owner finishes in seconds); reap it so the
	// (database_id, timeline_id, start_lsn) slot is free to re-receive.
	walClaimGracePeriod = 1 * time.Hour

	cleanerJobName = "physical_backup_retention_cleanup"

	// The WAL-stream supervisor reconciles owned streamers on the same 15 s
	// cadence as the scheduler — cadence-driven work is cheap, and the tighter
	// bound comes from streamer reclaim, where 15 s + streamerHeartbeatStaleness
	// keeps detect-to-reclaim under ~2 min after a crash.
	walStreamSupervisorTickInterval = 15 * time.Second

	// A streamer row whose heartbeat is older than this is assumed dead and
	// reclaimable by another process. Heartbeats fire every tick (15 s), so 90 s =
	// 6 missed beats — tolerant of GC pauses without leaving a dead streamer
	// unclaimed for long.
	streamerHeartbeatStaleness = 90 * time.Second

	// Bounds how long a reconcile waits for one streamer goroutine to drain on
	// stop so a stuck pg_receivewal teardown can't stall the whole supervisor.
	streamerStopTimeout = 30 * time.Second

	walStreamSupervisorJobName = "physical_wal_stream_supervisor"
)

// Per-tick WAL deletion budget anchors to the latest FULL's size (a cluster
// producing a 10 GB FULL produces O(10 GB) of WAL between FULLs, so "one FULL's
// worth" is a self-scaling chunk), with a 256 MB floor so clusters with tiny
// FULLs but heavy WAL don't crawl.
const minWalDeleteBudgetMB float64 = 256

// Conservative fallback floor in cleaner grace logic, mirroring logical's
// 60-minute floor on individual backups.
const recentBackupGracePeriod = 60 * time.Minute
