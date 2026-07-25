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
	// receivewalRespawnBackoff — initial pause between a pg_receivewal exit and respawn so
	// a hard-failing source (auth, pg_hba) is not hammered. --no-loop makes
	// pg_receivewal exit on connection loss; this loop is its supervision.
	receivewalRespawnBackoff = 2 * time.Second

	receivewalRespawnMaxBackoff = 30 * time.Minute

	// receivewalMinHealthyUptime — a receiver that streamed at least this long
	// before exiting counts as a transient blip (network drop, slot resend) and
	// resets the crash-loop counter; a shorter run counts toward escalation.
	receivewalMinHealthyUptime = 15 * time.Second

	// receivewalMaxRapidFailures — this many back-to-back sub-uptime exits escalate
	// to a fatal supervisor error so the streamer row is marked FAILED and the
	// supervisor can reclaim it on a later tick, instead of crash-looping locally
	// forever on a condition a local respawn can never fix (ENOSPC, bad creds, a
	// slot held by a thief).
	receivewalMaxRapidFailures = 5

	pausePollInterval = 1 * time.Second
)

// WalStreamSpec is the immutable configuration of one database's WAL streamer.
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

	// WatchDirRoot is config.DataFolder; the per-DB queue lives under
	// <root>/wal-queue/<database_id>/. It must survive a process restart so crash
	// recovery can re-process finalized-but-not-uploaded segments.
	WatchDirRoot string

	// WalLagThresholdBytes drives the lag monitor (lag_monitor.go): a slot lag over
	// this many bytes triggers a slot rebuild.
	WalLagThresholdBytes int64

	// OnGapDetected fires once per newly-observed WAL gap (see WalUploader); nil
	// disables notification.
	OnGapDetected func(gapStart, gapEnd walmath.LSN)

	// OnSlotRebuilt fires after the persistent slot has been recreated. Callers use
	// it to request a fresh base backup that anchors the new WAL chain.
	OnSlotRebuilt func(ctx context.Context, reason string) error

	Logger *slog.Logger
}

// WalStreamSupervisor runs and supervises one pg_receivewal process per database:
// it spawns the receiver, archives every fully-rotated segment via the
// insert-first WalUploader, applies disk back pressure, restarts a stalled
// receiver, forwards .history files, and (lag_monitor.go) rebuilds the slot on
// lag/loss. Run blocks until ctx is cancelled.
type WalStreamSupervisor struct {
	spec     WalStreamSpec
	uploader *WalUploader
	watchDir string
	slotName string

	// Back-pressure watermarks derived once from the source's (immutable)
	// wal_segment_size; recomputing them on every poll tick would be wasted work.
	highWatermarkBytes int64
	lowWatermarkBytes  int64

	// restartSignal asks the supervision loop to SIGTERM the current
	// pg_receivewal and respawn (sent by the back-pressure monitor and the
	// slot-LSN watcher). Buffered size 1; sends are non-blocking and coalesced.
	restartSignal chan struct{}

	// isPaused holds the supervision loop between pg_receivewal runs so a slot
	// rebuild can drop+recreate the slot without the receiver re-attaching.
	isPaused atomic.Bool

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

	// HIGH scales up for clusters with a non-default wal_segment_size so one
	// segment can never single-handedly stop the receiver; LOW is HIGH/5 for the
	// 5x hysteresis that prevents flapping on the boundary.
	highWatermarkBytes := max(walLocalMinHighWatermarkBytes, 4*walSegmentSizeBytes(spec.SourceDB))

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

// Run starts the uploader, the back-pressure monitor, the slot-LSN watcher, the
// lag monitor, and the pg_receivewal supervision loop, blocking until ctx is
// cancelled. The persistent slot is created if missing; torn *.partial files are
// cleared before the first spawn.
func (s *WalStreamSupervisor) Run(ctx context.Context) error {
	logger := s.spec.Logger.With("database_id", s.spec.DatabaseID, "slot_name", s.slotName)

	// pg_receivewal finalizes a segment by writing a marker into <dir>/archive_status/
	// and refuses to start (or errors mid-stream) if that subdirectory is absent — it
	// does not create it itself. Create both up front.
	if err := os.MkdirAll(filepath.Join(s.watchDir, "archive_status"), 0o700); err != nil {
		return fmt.Errorf("create wal watch dir: %w", err)
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

// runReceivewalSupervision is the pg_receivewal lifecycle loop: drain back
// pressure, clear partials, spawn, and react to the run's disposition. It returns
// a non-nil error only when the receiver is unrecoverable here (fatal exit or a
// crash loop), so Run can mark the streamer FAILED for reclaim on a later tick.
func (s *WalStreamSupervisor) runReceivewalSupervision(ctx context.Context, logger *slog.Logger) error {
	pgBin := tools.GetPostgresqlExecutable(s.spec.SourceDB.Version, tools.PostgresqlExecutablePgReceivewal)
	respawnBackoff := receivewalRespawnBackoff
	rapidFailures := 0

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

		outcome, ranFor, fatalErr := s.spawnAndSupervise(ctx, logger, pgBin)

		switch outcome {
		case receiverCtxCancelled:
			return nil

		case receiverFatal:
			return fatalErr

		case receiverInternalRestart:
			// Our own SIGTERM (back pressure or slot stall): respawn promptly; the
			// top-of-loop backlog/pause gates already throttle the cause.
			respawnBackoff = receivewalRespawnBackoff
			rapidFailures = 0

			continue
		}

		// receiverRetryable: a run that streamed for a healthy span is a transient
		// blip (network) — reset the crash-loop counter; a string of sub-uptime
		// exits is a crash loop a local respawn cannot fix, so escalate.
		if ranFor >= receivewalMinHealthyUptime {
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

// waitWhilePaused blocks the supervision loop while a slot rebuild holds the
// receiver down. Returns false if ctx is cancelled while paused.
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

// walSegmentSizeBytes returns the source cluster's captured wal_segment_size, or
// the 16 MB default when it has not been captured yet.
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
