package usecases_physical_postgresql

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"sync"
	"sync/atomic"
	"time"

	"github.com/google/uuid"

	backups_core_enums "databasus-backend/internal/features/backups/backups/core/enums"
	physical_repositories "databasus-backend/internal/features/backups/backups/core/physical/repositories"
	postgresql_physical "databasus-backend/internal/features/databases/databases/postgresql/physical"
	"databasus-backend/internal/features/storages"
	util_encryption "databasus-backend/internal/util/encryption"
	"databasus-backend/internal/util/tools"
	"databasus-backend/internal/util/walmath"
)

const (
	// Initial pause between a pg_receivewal exit and respawn so a hard-failing
	// source (auth, pg_hba) is not hammered. --no-loop makes pg_receivewal exit
	// on connection loss; this loop is its supervision.
	receivewalRespawnBackoff = 2 * time.Second

	receivewalRespawnMaxBackoff = 30 * time.Minute

	// A receiver that streamed at least this long before exiting counts as a
	// transient blip (network drop, slot resend) and resets the crash-loop
	// counter; a shorter run counts toward escalation.
	receivewalMinHealthyUptime = 15 * time.Second

	// This many back-to-back sub-uptime exits escalate to a fatal supervisor
	// error so the streamer row is marked FAILED and the supervisor can reclaim
	// it on a later tick, instead of crash-looping locally forever on a condition
	// a local respawn can never fix (ENOSPC, bad creds, a slot held by a thief).
	receivewalMaxRapidFailures = 5

	// A realign empties the resume path, so the very next spawn starts from the
	// slot. A second mismatch means the slot itself no longer covers what the
	// server retains — only a rebuild clears that.
	receivewalMaxResumeMismatches = 2

	pausePollInterval = 1 * time.Second
)

type resumeMismatchAction int

const (
	resumeMismatchActionRetry resumeMismatchAction = iota
	resumeMismatchActionRebuildSlot
)

type resumeMismatchEscalator struct {
	mismatchCount int
}

func (e *resumeMismatchEscalator) recordMismatchAndDecideEscalation() resumeMismatchAction {
	e.mismatchCount++

	if e.mismatchCount >= receivewalMaxResumeMismatches {
		e.mismatchCount = 0

		return resumeMismatchActionRebuildSlot
	}

	return resumeMismatchActionRetry
}

func (e *resumeMismatchEscalator) reset() {
	e.mismatchCount = 0
}

type WalStreamSpec struct {
	DatabaseID     uuid.UUID
	SourceDB       *postgresql_physical.PostgresqlPhysicalDatabase
	StorageID      uuid.UUID
	Storage        storages.StorageFileSaver
	Encryption     backups_core_enums.BackupEncryption
	MasterKey      string
	FieldEncryptor util_encryption.FieldEncryptor
	WalSegmentRepo *physical_repositories.PhysicalWalSegmentRepository
	HistoryRepo    *physical_repositories.PhysicalWalHistoryRepository

	// config.DataFolder; the per-DB queue lives under
	// <root>/wal-queue/<database_id>/. It must survive a process restart so crash
	// recovery can re-process finalized-but-not-uploaded segments.
	WatchDirRoot string

	// A slot lag over this many bytes triggers a rebuild (lag_monitor.go).
	WalLagThresholdBytes int64

	// How often a source that is being written to is asked to close its current WAL
	// segment, so the newest WAL reaches storage without waiting for the segment to
	// fill (wal_rotation.go). Zero disables it.
	ForcedRotationInterval time.Duration

	// How far the archived recovery point may fall behind a source that keeps
	// writing before the operator is alerted (wal_staleness.go). Zero disables it.
	ArchiveStalenessThreshold time.Duration

	// Fires once per newly-observed WAL gap (see WalUploader); nil disables
	// notification.
	OnGapDetected func(gapStart, gapEnd walmath.LSN)

	// Fires after the persistent slot has been recreated. Callers use it to
	// request a fresh base backup that anchors the new WAL chain.
	OnSlotRebuilt func(ctx context.Context, reason string) error

	// Fires while the chain is still intact but degrading (slot retention
	// warnings, a wedged receiver); nil disables notification.
	OnChainAtRisk func(report ChainRiskReport)

	Logger *slog.Logger
}

// One supervisor owns one database's pg_receivewal process. Run blocks until ctx
// is cancelled.
type WalStreamSupervisor struct {
	spec     WalStreamSpec
	uploader *WalUploader
	watchDir string
	slotName string

	// Derived once from the source's (immutable) wal_segment_size; recomputing
	// them on every poll tick would be wasted work.
	highWatermarkBytes int64
	lowWatermarkBytes  int64

	// Asks the supervision loop to SIGTERM the current pg_receivewal and respawn
	// (sent by the back-pressure monitor and the slot-LSN watcher). Buffered size
	// 1; sends are non-blocking and coalesced.
	restartSignal chan struct{}

	// Holds the supervision loop between pg_receivewal runs so a slot rebuild can
	// drop+recreate the slot without the receiver re-attaching.
	isPaused atomic.Bool

	// Lets the slot-LSN watcher and the lag monitor tell a real problem from lag
	// we are causing ourselves by holding the receiver down.
	isReceiverRunning atomic.Bool

	// rebuildMu serializes slot rebuilds in this process; rebuildTimestamps powers
	// the per-hour loop-protection cap. One supervisor owns a DB at a time (the
	// physical_wal_streamers heartbeat claim), so this is the only guard needed.
	rebuildMu         sync.Mutex
	rebuildTimestamps []time.Time
}

func NewWalStreamSupervisor(spec WalStreamSpec) *WalStreamSupervisor {
	watchDir := filepath.Join(spec.WatchDirRoot, "wal-queue", spec.DatabaseID.String())

	uploader := NewWalUploader(WalUploadDeps{
		DatabaseID:          spec.DatabaseID,
		StorageID:           spec.StorageID,
		Storage:             spec.Storage,
		Encryption:          spec.Encryption,
		MasterKey:           spec.MasterKey,
		FieldEncryptor:      spec.FieldEncryptor,
		WalSegmentRepo:      spec.WalSegmentRepo,
		WalSegmentSizeBytes: walSegmentSizeBytes(spec.SourceDB),
		Logger:              spec.Logger,
		OnGapDetected:       spec.OnGapDetected,
	})

	highWatermarkBytes := max(walLocalMinHighWatermarkBytes, 8*walSegmentSizeBytes(spec.SourceDB))

	return &WalStreamSupervisor{
		spec:               spec,
		uploader:           uploader,
		watchDir:           watchDir,
		slotName:           spec.SourceDB.ReplicationSlotName,
		highWatermarkBytes: highWatermarkBytes,
		lowWatermarkBytes:  highWatermarkBytes / 5,
		restartSignal:      make(chan struct{}, 1),
	}
}

func (s *WalStreamSupervisor) Run(ctx context.Context) error {
	logger := s.spec.Logger.With("database_id", s.spec.DatabaseID, "slot_name", s.slotName)

	// pg_receivewal finalizes a segment by writing a marker into <dir>/archive_status/
	// and refuses to start (or errors mid-stream) if that subdirectory is absent — it
	// does not create it itself. Create both up front.
	if err := os.MkdirAll(filepath.Join(s.watchDir, "archive_status"), 0o700); err != nil {
		return fmt.Errorf("create wal watch dir: %w", err)
	}

	if err := os.MkdirAll(filepath.Join(s.watchDir, pendingUploadDirName), 0o700); err != nil {
		return fmt.Errorf("create pending upload dir: %w", err)
	}

	if err := s.spec.SourceDB.VerifyWalSlot(ctx, logger, s.spec.FieldEncryptor); err != nil {
		return fmt.Errorf("verify persistent replication slot: %w", err)
	}

	// Crash recovery: clear torn *.partial files (the slot resends them) and take
	// over any finalized-but-not-uploaded segments left by a previous crash, so
	// recovery does not wait on the cleaner's grace sweep. Runs before the receiver
	// spawns, so there is no concurrent writer in watch_dir.
	s.removePartials(logger)
	s.recoverLocalSegmentsOnStartup(ctx, logger)

	// A fatal pg_receivewal exit (disk full, auth, stolen slot, crash loop) must
	// tear down the whole supervisor — not just the receiver — so the streamer
	// row is marked FAILED and reclaimed on a later supervisor tick. Derive a cancelable
	// ctx the auxiliary loops share and cancel it when supervision returns fatal.
	runCtx, cancelRun := context.WithCancel(ctx)
	defer cancelRun()

	var wg sync.WaitGroup

	for _, loop := range []func(context.Context, *slog.Logger){
		s.runUploaderLoop,
		s.runBackpressureMonitor,
		s.runSlotLsnWatcher,
		s.runLagMonitor,
		s.runForcedWalRotation,
	} {
		wg.Go(func() { loop(runCtx, logger) })
	}

	fatalErr := s.runReceivewalSupervision(runCtx, logger)

	cancelRun()
	wg.Wait()

	if fatalErr != nil {
		logger.Error("wal stream supervisor stopping with fatal error", "error", fatalErr)

		return fatalErr
	}

	logger.Info("wal stream supervisor stopped")

	return nil
}

// Returns a non-nil error only when the receiver is unrecoverable here (fatal
// exit or a crash loop), so Run can mark the streamer FAILED for reclaim on a
// later tick.
func (s *WalStreamSupervisor) runReceivewalSupervision(ctx context.Context, logger *slog.Logger) error {
	pgBin := tools.GetPostgresqlExecutable(s.spec.SourceDB.Version, tools.PostgresqlExecutablePgReceivewal)
	respawnBackoff := receivewalRespawnBackoff
	rapidFailures := 0

	var mismatchEscalator resumeMismatchEscalator

	for {
		if ctx.Err() != nil {
			return nil
		}

		if !s.waitWhilePaused(ctx) {
			return nil
		}

		if !s.waitForBacklogBelowLow(ctx, logger) {
			return nil
		}

		// Clear any stale restart signal so a spawn does not get cancelled by a
		// signal raised while no process was running.
		s.drainRestartSignal()
		s.removePartials(logger)
		s.realignResumePath(ctx, logger)

		receiverRun := s.spawnAndSupervise(ctx, logger, pgBin)

		switch receiverRun.Exit {
		case receiverCtxCancelled:
			return nil

		case receiverFatal:
			return receiverRun.FatalErr

		case receiverInternalRestart:
			// Our own SIGTERM (back pressure or slot stall): respawn promptly; the
			// top-of-loop backlog/pause gates already throttle the cause.
			respawnBackoff = receivewalRespawnBackoff
			rapidFailures = 0

			mismatchEscalator.reset()

			continue

		case receiverResumeMismatch:
			if mismatchEscalator.recordMismatchAndDecideEscalation() == resumeMismatchActionRebuildSlot {
				if err := s.rebuildSlot(ctx, logger, breakReasonSlotLost); err != nil {
					return fmt.Errorf("rebuild slot after repeated wal resume mismatch: %w", err)
				}
			}

			if !sleepCtx(ctx, receivewalRespawnBackoff) {
				return nil
			}

			continue
		}

		mismatchEscalator.reset()

		// receiverRetryable: a run that streamed for a healthy span is a transient
		// blip (network) — reset the crash-loop counter; a string of sub-uptime
		// exits is a crash loop a local respawn cannot fix, so escalate.
		if receiverRun.RanFor >= receivewalMinHealthyUptime {
			rapidFailures = 0
			respawnBackoff = receivewalRespawnBackoff
		} else {
			rapidFailures++
		}

		if rapidFailures >= receivewalMaxRapidFailures {
			return fmt.Errorf(
				"pg_receivewal crash-looped: %d rapid failures, escalating for reassignment", rapidFailures,
			)
		}

		if !sleepCtx(ctx, respawnBackoff) {
			return nil
		}

		respawnBackoff = min(respawnBackoff*2, receivewalRespawnMaxBackoff)
	}
}

func (s *WalStreamSupervisor) waitWhilePaused(ctx context.Context) bool {
	if !s.isPaused.Load() {
		return true
	}

	ticker := time.NewTicker(pausePollInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return false

		case <-ticker.C:
			if !s.isPaused.Load() {
				return true
			}
		}
	}
}

func (s *WalStreamSupervisor) signalRestart() {
	select {
	case s.restartSignal <- struct{}{}:
	default:
	}
}

func (s *WalStreamSupervisor) drainRestartSignal() {
	select {
	case <-s.restartSignal:
	default:
	}
}

func walSegmentSizeBytes(sourceDB *postgresql_physical.PostgresqlPhysicalDatabase) int64 {
	if sourceDB.WalSegmentSizeBytes != nil && *sourceDB.WalSegmentSizeBytes > 0 {
		return *sourceDB.WalSegmentSizeBytes
	}

	return int64(walmath.WalSegmentSize)
}

func sleepCtx(ctx context.Context, d time.Duration) bool {
	timer := time.NewTimer(d)
	defer timer.Stop()

	select {
	case <-ctx.Done():
		return false

	case <-timer.C:
		return true
	}
}
