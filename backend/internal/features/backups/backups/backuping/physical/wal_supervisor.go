package backuping_physical

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"slices"
	"sync"
	"sync/atomic"
	"time"

	"github.com/google/uuid"

	"databasus-backend/internal/config"
	backups_core_enums "databasus-backend/internal/features/backups/backups/core/enums"
	physical_repositories "databasus-backend/internal/features/backups/backups/core/physical/repositories"
	postgresql_executor "databasus-backend/internal/features/backups/backups/usecases/physical/postgresql"
	backups_config_physical "databasus-backend/internal/features/backups/config/physical"
	"databasus-backend/internal/features/databases"
	postgresql_physical "databasus-backend/internal/features/databases/databases/postgresql/physical"
	encryption_secrets "databasus-backend/internal/features/encryption/secrets"
	"databasus-backend/internal/features/storages"
	tasks_cancellation "databasus-backend/internal/features/tasks/cancellation"
	util_encryption "databasus-backend/internal/util/encryption"
	"databasus-backend/internal/util/walmath"
)

// PhysicalWalStreamSupervisor is the background service that owns long-running
// pg_receivewal streamers. Ownership and restart recovery run through the
// physical_wal_streamers heartbeat table: every tick it discovers WAL_STREAM
// databases, CAS-claims any that are unclaimed / FAILED / stale, runs one
// WalStreamSupervisor goroutine per owned DB, and heartbeats the rows it owns.
type PhysicalWalStreamSupervisor struct {
	databaseService     *databases.DatabaseService
	backupConfigService *backups_config_physical.BackupConfigService
	storageService      *storages.StorageService
	walSegmentRepo      *physical_repositories.PhysicalWalSegmentRepository
	historyRepo         *physical_repositories.PhysicalWalHistoryRepository
	walStreamerRepo     *physical_repositories.PhysicalWalStreamerRepository
	notificationSender  NotificationSender
	taskCancelManager   *tasks_cancellation.TaskCancelManager
	secretKeyService    *encryption_secrets.SecretKeyService
	fieldEncryptor      util_encryption.FieldEncryptor
	logger              *slog.Logger

	mu      sync.Mutex
	running map[uuid.UUID]*runningStreamer

	lastTickTime atomicTime

	hasRun  atomic.Bool
	isReady atomic.Bool
}

// runningStreamer is the handle to one locally-owned streamer goroutine.
type runningStreamer struct {
	cancel               context.CancelFunc
	done                 chan struct{}
	watchDir             string
	shouldRemoveWatchDir atomic.Bool
}

func (s *PhysicalWalStreamSupervisor) IsRunning() bool {
	return s.isReady.Load()
}

func (s *PhysicalWalStreamSupervisor) IsSupervisorHealthy() bool {
	return s.lastTickTime.Load().After(time.Now().UTC().Add(-schedulerHealthcheckThreshold))
}

func (s *PhysicalWalStreamSupervisor) Run(ctx context.Context) {
	if s.hasRun.Swap(true) {
		panic(fmt.Sprintf("%T.Run() called multiple times", s))
	}

	s.logger = s.logger.With("job_id", uuid.New(), "job_name", walStreamSupervisorJobName)

	s.lastTickTime.Store(time.Now().UTC())

	if err := s.recoverStreamersOnStartup(); err != nil {
		s.logger.Error(
			"failed to recover wal streamers on startup",
			"error",
			err,
			"job_name",
			walStreamSupervisorJobName,
		)

		panic(err)
	}

	s.isReady.Store(true)

	defer func() {
		s.isReady.Store(false)
		s.stopAllStreamers()
	}()

	ticker := time.NewTicker(walStreamSupervisorTickInterval)
	defer ticker.Stop()

	s.reconcile(ctx)

	for {
		select {
		case <-ctx.Done():
			return

		case <-ticker.C:
			s.reconcile(ctx)

			s.lastTickTime.Store(time.Now().UTC())
		}
	}
}

func (s *PhysicalWalStreamSupervisor) recoverStreamersOnStartup() error {
	failed, err := s.walStreamerRepo.MarkStaleRunningFailed(int(streamerHeartbeatStaleness.Seconds()))
	if err != nil {
		return err
	}

	if failed > 0 {
		s.logger.Warn(
			fmt.Sprintf("recovered %d stale wal streamers on startup", failed),
			"job_name", walStreamSupervisorJobName,
		)
	}

	return nil
}

// reconcile starts streamers for newly-claimable WAL_STREAM databases, stops
// local streamers whose database is no longer a candidate (disabled or demoted),
// and heartbeats the rows we own.
func (s *PhysicalWalStreamSupervisor) reconcile(ctx context.Context) {
	configs, err := s.backupConfigService.GetBackupConfigsWithEnabledBackups()
	if err != nil {
		s.logger.Error(
			"wal supervisor: failed to load enabled configs",
			"error",
			err,
			"job_name",
			walStreamSupervisorJobName,
		)

		return
	}

	candidates := make(map[uuid.UUID]bool)

	for _, backupConfig := range configs {
		if !isWalStreamCandidate(backupConfig) {
			continue
		}

		logger := s.logger.With("database_id", backupConfig.DatabaseID, "job_name", walStreamSupervisorJobName)

		candidates[backupConfig.DatabaseID] = true

		s.ensureStreamerRunning(ctx, logger, backupConfig)
	}

	s.stopNonCandidates(candidates)
	s.heartbeatOwnedStreamers()
}

// isWalStreamCandidate reports whether a config should have a running streamer:
// WAL_STREAM backup type with storage configured.
func isWalStreamCandidate(backupConfig *backups_config_physical.PhysicalBackupConfig) bool {
	if backupConfig.PostgresqlPhysical == nil ||
		backupConfig.PostgresqlPhysical.BackupType != postgresql_physical.BackupTypeFullIncrementalAndWalStream {
		return false
	}

	return backupConfig.StorageID != nil
}

// ensureStreamerRunning starts a streamer for backupConfig when we don't already
// run one locally and we can win the heartbeat-table claim.
func (s *PhysicalWalStreamSupervisor) ensureStreamerRunning(
	ctx context.Context,
	logger *slog.Logger,
	backupConfig *backups_config_physical.PhysicalBackupConfig,
) {
	s.mu.Lock()
	_, alreadyRunning := s.running[backupConfig.DatabaseID]
	s.mu.Unlock()

	if alreadyRunning {
		return
	}

	claimed, err := s.walStreamerRepo.ClaimIfClaimable(
		backupConfig.DatabaseID, int(streamerHeartbeatStaleness.Seconds()),
	)
	if err != nil {
		logger.Error("wal supervisor: claim failed", "error", err)

		return
	}

	if !claimed {
		return
	}

	s.startStreamer(ctx, logger, backupConfig)
}

func (s *PhysicalWalStreamSupervisor) startStreamer(
	ctx context.Context,
	logger *slog.Logger,
	backupConfig *backups_config_physical.PhysicalBackupConfig,
) {
	db, err := s.databaseService.GetDatabaseByID(backupConfig.DatabaseID)
	if err != nil || db.PostgresqlPhysical == nil {
		logger.Error("wal supervisor: failed to load database for streamer", "error", err)
		s.releaseClaim(logger, backupConfig.DatabaseID)

		return
	}

	storage, err := s.storageService.GetStorageByID(*backupConfig.StorageID)
	if err != nil {
		logger.Error("wal supervisor: failed to load storage for streamer", "error", err)
		s.releaseClaim(logger, backupConfig.DatabaseID)

		return
	}

	masterKey, ok := s.resolveMasterKey(logger, backupConfig)
	if !ok {
		s.releaseClaim(logger, backupConfig.DatabaseID)

		return
	}

	// Derive from the supervisor's run ctx so a process shutdown cancels every
	// streamer; the per-DB cancel (registered with TaskCancelManager + stored on
	// runningStreamer) handles targeted teardown on disable / demote / db-remove.
	streamerCtx, cancel := context.WithCancel(ctx)
	done := make(chan struct{})
	streamer := &runningStreamer{
		cancel:   cancel,
		done:     done,
		watchDir: filepath.Join(config.GetEnv().DataFolder, "wal-queue", db.ID.String()),
	}

	s.taskCancelManager.RegisterTask(db.ID, func() {
		streamer.shouldRemoveWatchDir.Store(true)
		cancel()
	})

	supervisor := postgresql_executor.NewWalStreamSupervisor(postgresql_executor.WalStreamSpec{
		DatabaseID:           db.ID,
		SourceDB:             db.PostgresqlPhysical,
		StorageID:            storage.ID,
		Storage:              storage,
		Encryption:           backupConfig.Encryption,
		MasterKey:            masterKey,
		FieldEncryptor:       s.fieldEncryptor,
		WalSegmentRepo:       s.walSegmentRepo,
		HistoryRepo:          s.historyRepo,
		WatchDirRoot:         config.GetEnv().DataFolder,
		WalLagThresholdBytes: backupConfig.WalLagThresholdBytes,
		OnGapDetected:        s.gapNotifier(db, backupConfig),
		OnSlotRebuilt:        s.slotRebuildFullRequester(logger, backupConfig),
		Logger:               s.logger,
	})

	s.mu.Lock()
	s.running[db.ID] = streamer
	s.mu.Unlock()

	logger.Info("wal stream supervisor started for database")

	go func() {
		defer close(done)
		defer s.taskCancelManager.UnregisterTask(db.ID)
		defer s.removeWatchDirIfRequested(logger, streamer)

		if err := supervisor.Run(streamerCtx); err != nil {
			logger.Error("wal stream supervisor exited with error", "error", err)

			// Mark FAILED so a later tick can reclaim it. A clean
			// ctx-cancel (shutdown / lifecycle stop) does not reach here with an
			// error, so the row is left for the cancelling path to handle.
			if markErr := s.walStreamerRepo.MarkFailed(db.ID); markErr != nil {
				logger.Error("wal supervisor: failed to mark streamer failed", "error", markErr)
			}
		}

		s.mu.Lock()
		delete(s.running, db.ID)
		s.mu.Unlock()
	}()
}

func (s *PhysicalWalStreamSupervisor) resolveMasterKey(
	logger *slog.Logger,
	backupConfig *backups_config_physical.PhysicalBackupConfig,
) (string, bool) {
	if backupConfig.Encryption != backups_core_enums.BackupEncryptionEncrypted {
		return "", true
	}

	key, err := s.secretKeyService.GetSecretKey()
	if err != nil {
		logger.Error("wal supervisor: failed to fetch master key", "error", err)

		return "", false
	}

	return key, true
}

// gapNotifier builds the post-upload gap-probe callback: it sends a BackupFailed
// notification (when the config opts in) to each of the database's notifiers.
func (s *PhysicalWalStreamSupervisor) gapNotifier(
	db *databases.Database,
	backupConfig *backups_config_physical.PhysicalBackupConfig,
) func(gapStart, gapEnd walmath.LSN) {
	return func(gapStart, gapEnd walmath.LSN) {
		if !slices.Contains(backupConfig.SendNotificationsOn, backups_config_physical.NotificationWalGap) {
			return
		}

		title := fmt.Sprintf("Physical WAL gap detected for %q", db.Name)
		message := fmt.Sprintf("database_id=%s gap=[%s, %s)", db.ID, gapStart.String(), gapEnd.String())

		for _, notifier := range db.Notifiers {
			s.notificationSender.SendNotification(&notifier, title, message)
		}
	}
}

func (s *PhysicalWalStreamSupervisor) slotRebuildFullRequester(
	logger *slog.Logger,
	backupConfig *backups_config_physical.PhysicalBackupConfig,
) func(context.Context, string) error {
	return func(_ context.Context, reason string) error {
		if err := s.backupConfigService.RequestFullBackupNow(backupConfig.DatabaseID); err != nil {
			return err
		}

		logger.Warn("requested out-of-cadence full backup after wal slot rebuild", "reason", reason)

		return nil
	}
}

func (s *PhysicalWalStreamSupervisor) stopNonCandidates(candidates map[uuid.UUID]bool) {
	s.mu.Lock()
	var toStop []uuid.UUID

	for databaseID := range s.running {
		if !candidates[databaseID] {
			toStop = append(toStop, databaseID)
		}
	}
	s.mu.Unlock()

	for _, databaseID := range toStop {
		s.stopStreamer(databaseID, true)
	}
}

func (s *PhysicalWalStreamSupervisor) stopStreamer(databaseID uuid.UUID, shouldRemoveWatchDir bool) {
	s.mu.Lock()
	streamer := s.running[databaseID]
	s.mu.Unlock()

	if streamer == nil {
		return
	}

	if shouldRemoveWatchDir {
		streamer.shouldRemoveWatchDir.Store(true)
	}

	streamer.cancel()

	select {
	case <-streamer.done:
	case <-time.After(streamerStopTimeout):
		s.logger.Warn("wal supervisor: streamer stop timed out", "database_id", databaseID)
	}

	s.mu.Lock()
	delete(s.running, databaseID)
	s.mu.Unlock()

	if err := s.walStreamerRepo.MarkFailed(databaseID); err != nil {
		s.logger.Error(
			"wal supervisor: failed to mark stopped streamer failed",
			"database_id",
			databaseID,
			"error",
			err,
		)
	}
}

func (s *PhysicalWalStreamSupervisor) heartbeatOwnedStreamers() {
	s.mu.Lock()
	owned := make([]uuid.UUID, 0, len(s.running))
	for databaseID := range s.running {
		owned = append(owned, databaseID)
	}
	s.mu.Unlock()

	for _, databaseID := range owned {
		if err := s.walStreamerRepo.Heartbeat(databaseID); err != nil {
			s.logger.Error("wal supervisor: heartbeat failed", "database_id", databaseID, "error", err)
		}
	}
}

func (s *PhysicalWalStreamSupervisor) stopAllStreamers() {
	s.mu.Lock()
	owned := make([]uuid.UUID, 0, len(s.running))
	for databaseID := range s.running {
		owned = append(owned, databaseID)
	}
	s.mu.Unlock()

	for _, databaseID := range owned {
		s.stopStreamer(databaseID, false)
	}
}

func (s *PhysicalWalStreamSupervisor) removeWatchDirIfRequested(logger *slog.Logger, streamer *runningStreamer) {
	if !streamer.shouldRemoveWatchDir.Load() {
		return
	}

	if err := os.RemoveAll(streamer.watchDir); err != nil {
		logger.Warn("failed to remove wal queue directory", "watch_dir", streamer.watchDir, "error", err)
	}
}

// releaseClaim marks a just-won streamer row FAILED when we could not actually
// start the streamer, so the slot is immediately reclaimable rather than looking
// alive until the heartbeat goes stale.
func (s *PhysicalWalStreamSupervisor) releaseClaim(logger *slog.Logger, databaseID uuid.UUID) {
	if err := s.walStreamerRepo.MarkFailed(databaseID); err != nil {
		logger.Error("wal supervisor: failed to release streamer claim", "error", err)
	}
}
