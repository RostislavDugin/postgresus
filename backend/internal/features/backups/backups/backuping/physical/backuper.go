package backuping_physical

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"slices"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	backups_core_enums "databasus-backend/internal/features/backups/backups/core/enums"
	physical_enums "databasus-backend/internal/features/backups/backups/core/physical/enums"
	physical_models "databasus-backend/internal/features/backups/backups/core/physical/models"
	physical_repositories "databasus-backend/internal/features/backups/backups/core/physical/repositories"
	postgresql_executor "databasus-backend/internal/features/backups/backups/usecases/physical/postgresql"
	backups_config_physical "databasus-backend/internal/features/backups/config/physical"
	"databasus-backend/internal/features/databases"
	encryption_secrets "databasus-backend/internal/features/encryption/secrets"
	"databasus-backend/internal/features/storages"
	tasks_cancellation "databasus-backend/internal/features/tasks/cancellation"
	workspaces_services "databasus-backend/internal/features/workspaces/services"
	"databasus-backend/internal/storage"
	util_encryption "databasus-backend/internal/util/encryption"
	"databasus-backend/internal/util/walmath"
)

// PhysicalBackuper drives a FULL or INCR through the postgresql executor.
// The scheduler invokes MakeBackup directly in a goroutine.
type PhysicalBackuper struct {
	databaseService     *databases.DatabaseService
	fieldEncryptor      util_encryption.FieldEncryptor
	workspaceService    *workspaces_services.WorkspaceService
	fullRepo            *physical_repositories.PhysicalFullBackupRepository
	incrRepo            *physical_repositories.PhysicalIncrementalBackupRepository
	inFlightRepo        *physical_repositories.PhysicalInFlightBackupRepository
	historyRepo         *physical_repositories.PhysicalWalHistoryRepository
	backupConfigService *backups_config_physical.BackupConfigService
	storageService      *storages.StorageService
	notificationSender  NotificationSender
	taskCancelManager   *tasks_cancellation.TaskCancelManager
	secretKeyService    *encryption_secrets.SecretKeyService
	logger              *slog.Logger
	fullExecutor        FullBackupExecutor
	incrExecutor        IncrementalBackupExecutor
}

func (b *PhysicalBackuper) MakeBackup(backupID uuid.UUID, isCallNotifier bool) {
	logger := b.logger.With("backup_id", backupID)

	fullBackup, err := b.fullRepo.FindByID(backupID)
	if err != nil {
		logger.Error("failed to look up full backup row", "error", err)
		return
	}

	if fullBackup != nil {
		b.runFullBackup(logger, fullBackup, isCallNotifier)
		return
	}

	incrBackup, err := b.incrRepo.FindByID(backupID)
	if err != nil {
		logger.Error("failed to look up incremental backup row", "error", err)

		return
	}

	if incrBackup != nil {
		b.runIncrementalBackup(logger, incrBackup, isCallNotifier)
		return
	}

	logger.Warn("backup not found in either typed table; ignoring assignment")
}

func (b *PhysicalBackuper) runFullBackup(
	logger *slog.Logger,
	fullBackup *physical_models.PhysicalFullBackup,
	isCallNotifier bool,
) {
	backupCtx, ok := b.loadBackupContext(logger, fullBackup.DatabaseID)
	if !ok {
		b.finalizeFullAsError(fullBackup, physical_enums.PhysicalBackupErrorPgBasebackupFailed,
			"failed to load backup context")

		return
	}

	ctx, cancel := context.WithCancel(context.Background())
	b.taskCancelManager.RegisterTask(fullBackup.ID, cancel)
	defer b.taskCancelManager.UnregisterTask(fullBackup.ID)

	fullBackupSpec := postgresql_executor.FullBackupSpec{
		CommonBackupSpec: postgresql_executor.CommonBackupSpec{
			SourceDB:       backupCtx.Database.PostgresqlPhysical,
			DatabaseName:   backupCtx.Database.Name,
			StorageID:      backupCtx.Storage.ID,
			Storage:        backupCtx.Storage,
			Encryption:     backupCtx.Config.Encryption,
			MasterKey:      backupCtx.MasterKey,
			FieldEncryptor: b.fieldEncryptor,
			FullRepo:       b.fullRepo,
			HistoryRepo:    b.historyRepo,
			Logger:         logger,
			ProgressListener: func(completedMb float64, elapsedMs int64) {
				if err := b.fullRepo.UpdateProgress(fullBackup.ID, completedMb, elapsedMs); err != nil {
					logger.Error("failed to update full backup progress", "error", err)
				}
			},
		},
		Backup: fullBackup,
	}

	backupResult, err := b.fullExecutor.Execute(ctx, fullBackupSpec)
	if err != nil {
		logger.Error("full executor returned error", "error", err)

		b.finalizeFullAsError(fullBackup, physical_enums.PhysicalBackupErrorPgBasebackupFailed, err.Error())

		return
	}

	if backupResult.Status != physical_enums.PhysicalBackupStatusCompleted {
		logger.Warn("full executor returned non-COMPLETED result",
			"status", backupResult.Status,
			"reason", reasonOrEmpty(backupResult.ErrorReason),
			"message", backupResult.ErrorMessage)
	}

	if err := b.persistFullResult(fullBackup, backupResult); err != nil {
		logger.Error("failed to persist full result", "error", err)

		return
	}

	if isCallNotifier {
		b.sendFullBackupNotification(backupCtx.Config, backupCtx.Database, fullBackup, backupResult)
	}
}

func (b *PhysicalBackuper) runIncrementalBackup(
	logger *slog.Logger,
	incrBackup *physical_models.PhysicalIncrementalBackup,
	isCallNotifier bool,
) {
	backupCtx, ok := b.loadBackupContext(logger, incrBackup.DatabaseID)
	if !ok {
		b.finalizeIncrAsError(incrBackup, physical_enums.PhysicalBackupErrorPgBasebackupFailed,
			"failed to load backup context")

		return
	}

	parentRef, err := b.resolveParentManifest(incrBackup)
	if err != nil {
		logger.Error("failed to resolve parent manifest", "error", err)

		b.finalizeIncrAsChainBroken(incrBackup,
			physical_enums.PhysicalBackupErrorParentManifestMissing, err.Error())

		return
	}

	ctx, cancel := context.WithCancel(context.Background())
	b.taskCancelManager.RegisterTask(incrBackup.ID, cancel)
	defer b.taskCancelManager.UnregisterTask(incrBackup.ID)

	incrBackupSpec := postgresql_executor.IncrementalBackupSpec{
		CommonBackupSpec: postgresql_executor.CommonBackupSpec{
			SourceDB:       backupCtx.Database.PostgresqlPhysical,
			DatabaseName:   backupCtx.Database.Name,
			StorageID:      backupCtx.Storage.ID,
			Storage:        backupCtx.Storage,
			Encryption:     backupCtx.Config.Encryption,
			MasterKey:      backupCtx.MasterKey,
			FieldEncryptor: b.fieldEncryptor,
			FullRepo:       b.fullRepo,
			HistoryRepo:    b.historyRepo,
			Logger:         logger,
			ProgressListener: func(completedMb float64, elapsedMs int64) {
				if err := b.incrRepo.UpdateProgress(incrBackup.ID, completedMb, elapsedMs); err != nil {
					logger.Error("failed to update incremental backup progress", "error", err)
				}
			},
		},
		Backup:             incrBackup,
		ParentManifest:     parentRef,
		IncrRepo:           b.incrRepo,
		IncrementalCadence: backupCtx.Config.IncrementalBackupInterval.ApproxPeriod(),
	}

	backupResult, err := b.incrExecutor.Execute(ctx, incrBackupSpec)
	if err != nil {
		logger.Error("incremental executor returned error", "error", err)

		b.finalizeIncrAsError(incrBackup, physical_enums.PhysicalBackupErrorPgBasebackupFailed, err.Error())

		return
	}

	if backupResult.Status != physical_enums.PhysicalBackupStatusCompleted {
		logger.Warn("incremental executor returned non-COMPLETED result",
			"status", backupResult.Status,
			"reason", reasonOrEmpty(backupResult.ErrorReason),
			"message", backupResult.ErrorMessage)
	}

	if err := b.persistIncrResult(incrBackup, backupResult); err != nil {
		logger.Error("failed to persist incremental result", "error", err)

		return
	}

	if isCallNotifier {
		b.sendIncrBackupNotification(backupCtx.Config, backupCtx.Database, incrBackup, backupResult)
	}
}

func (b *PhysicalBackuper) loadBackupContext(
	logger *slog.Logger,
	databaseID uuid.UUID,
) (*backupContext, bool) {
	cfg, err := b.backupConfigService.GetBackupConfigByDbId(databaseID)
	if err != nil {
		logger.Error("failed to fetch physical backup config", "error", err)

		return nil, false
	}

	if cfg.StorageID == nil {
		logger.Error("physical backup config has no storage id")

		return nil, false
	}

	db, err := b.databaseService.GetDatabaseByID(databaseID)
	if err != nil {
		logger.Error("failed to fetch database by id", "error", err)

		return nil, false
	}

	if db.PostgresqlPhysical == nil {
		logger.Error("database is not a physical postgres database")

		return nil, false
	}

	storage, err := b.storageService.GetStorageByID(*cfg.StorageID)
	if err != nil {
		logger.Error("failed to fetch storage", "error", err)

		return nil, false
	}

	masterKey := ""
	if cfg.Encryption == backups_core_enums.BackupEncryptionEncrypted {
		key, secretErr := b.secretKeyService.GetSecretKey()
		if secretErr != nil {
			logger.Error("failed to fetch master key", "error", secretErr)

			return nil, false
		}

		masterKey = key
	}

	return &backupContext{cfg, db, storage, masterKey}, true
}

func (b *PhysicalBackuper) resolveParentManifest(
	incrBackup *physical_models.PhysicalIncrementalBackup,
) (postgresql_executor.ParentManifestRef, error) {
	if incrBackup.ParentIncrementalBackupID != nil {
		parent, lookupErr := b.incrRepo.FindByID(*incrBackup.ParentIncrementalBackupID)
		if lookupErr != nil {
			return postgresql_executor.ParentManifestRef{}, fmt.Errorf("look up parent incr: %w", lookupErr)
		}

		if parent == nil || parent.ManifestFileName == nil || parent.StopLSN == nil {
			return postgresql_executor.ParentManifestRef{}, errors.New(
				"parent incremental row missing or has no manifest_file_name / stop_lsn",
			)
		}

		return postgresql_executor.ParentManifestRef{
			BackupID:   parent.ID,
			FileName:   *parent.ManifestFileName,
			Encryption: parent.Encryption,
			Salt:       derefString(parent.ManifestEncryptionSalt),
			IV:         derefString(parent.ManifestEncryptionIV),
			StopLSN:    *parent.StopLSN,
		}, nil
	}

	parent, lookupErr := b.fullRepo.FindByID(incrBackup.RootFullBackupID)
	if lookupErr != nil {
		return postgresql_executor.ParentManifestRef{}, fmt.Errorf("look up root full: %w", lookupErr)
	}

	if parent == nil || parent.ManifestFileName == nil || parent.StopLSN == nil {
		return postgresql_executor.ParentManifestRef{}, errors.New(
			"root full row missing or has no manifest_file_name / stop_lsn")
	}

	return postgresql_executor.ParentManifestRef{
		BackupID:   parent.ID,
		FileName:   *parent.ManifestFileName,
		Encryption: parent.Encryption,
		Salt:       derefString(parent.ManifestEncryptionSalt),
		IV:         derefString(parent.ManifestEncryptionIV),
		StopLSN:    *parent.StopLSN,
	}, nil
}

func (b *PhysicalBackuper) persistFullResult(
	fullBackup *physical_models.PhysicalFullBackup,
	backupResult postgresql_executor.PhysicalBackupResult,
) error {
	fullBackup.Status = backupResult.Status
	fullBackup.ErrorReason = backupResult.ErrorReason
	fullBackup.FailMessage = nilOrPtr(backupResult.ErrorMessage)

	if backupResult.Status == physical_enums.PhysicalBackupStatusCompleted {
		fullBackup.TimelineID = backupResult.TimelineID
		fullBackup.StartLSN = lsnPtr(backupResult.StartLSN)
		fullBackup.StopLSN = lsnPtr(backupResult.StopLSN)
		fullBackup.BackupSizeMb = &backupResult.BackupSizeMb
		fullBackup.BackupDurationMs = &backupResult.BackupDurationMs
		fullBackup.Encryption = backupResult.EncryptionAlgo
		fullBackup.EncryptionSalt = nilOrPtr(backupResult.EncryptionSalt)
		fullBackup.EncryptionIV = nilOrPtr(backupResult.EncryptionIV)

		fullBackup.Compression = backupResult.Compression
		fullBackup.ManifestFileName = nilOrPtr(backupResult.ManifestFileName)
		fullBackup.ManifestEncryptionSalt = nilOrPtr(backupResult.ManifestEncryptionSalt)
		fullBackup.ManifestEncryptionIV = nilOrPtr(backupResult.ManifestEncryptionIV)

		completed := backupResult.CompletedAt
		if completed.IsZero() {
			completed = time.Now().UTC()
		}

		fullBackup.CompletedAt = &completed
	}

	return b.saveTerminalResultIfInProgress(
		fullBackup.DatabaseID,
		fullBackup.ID,
		func(tx *gorm.DB) (physical_enums.PhysicalBackupStatus, error) {
			var current physical_models.PhysicalFullBackup

			if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
				Select("status").
				Where("id = ?", fullBackup.ID).
				First(&current).Error; err != nil {
				return "", err
			}

			return current.Status, nil
		},
		func(tx *gorm.DB) error { return tx.Save(fullBackup).Error },
	)
}

func (b *PhysicalBackuper) persistIncrResult(
	incrBackup *physical_models.PhysicalIncrementalBackup,
	backupResult postgresql_executor.PhysicalBackupResult,
) error {
	incrBackup.Status = backupResult.Status
	incrBackup.ErrorReason = backupResult.ErrorReason
	incrBackup.FailMessage = nilOrPtr(backupResult.ErrorMessage)

	if backupResult.Status == physical_enums.PhysicalBackupStatusCompleted {
		incrBackup.TimelineID = backupResult.TimelineID
		incrBackup.StartLSN = lsnPtr(backupResult.StartLSN)
		incrBackup.StopLSN = lsnPtr(backupResult.StopLSN)
		incrBackup.BackupSizeMb = &backupResult.BackupSizeMb
		incrBackup.BackupDurationMs = &backupResult.BackupDurationMs
		incrBackup.Encryption = backupResult.EncryptionAlgo
		incrBackup.EncryptionSalt = nilOrPtr(backupResult.EncryptionSalt)
		incrBackup.EncryptionIV = nilOrPtr(backupResult.EncryptionIV)

		incrBackup.Compression = backupResult.Compression
		incrBackup.ManifestFileName = nilOrPtr(backupResult.ManifestFileName)
		incrBackup.ManifestEncryptionSalt = nilOrPtr(backupResult.ManifestEncryptionSalt)
		incrBackup.ManifestEncryptionIV = nilOrPtr(backupResult.ManifestEncryptionIV)

		completed := backupResult.CompletedAt
		if completed.IsZero() {
			completed = time.Now().UTC()
		}

		incrBackup.CompletedAt = &completed
	}

	return b.saveTerminalResultIfInProgress(
		incrBackup.DatabaseID,
		incrBackup.ID,
		func(tx *gorm.DB) (physical_enums.PhysicalBackupStatus, error) {
			var current physical_models.PhysicalIncrementalBackup

			if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
				Select("status").
				Where("id = ?", incrBackup.ID).
				First(&current).Error; err != nil {
				return "", err
			}

			return current.Status, nil
		},
		func(tx *gorm.DB) error { return tx.Save(incrBackup).Error },
	)
}

// saveTerminalResultIfInProgress writes the mutated backup row and releases its
// in-flight claim in one transaction, but only while the row is still
// IN_PROGRESS. A backuper can return after a restart-recovery sweep already
// moved the row to a terminal status — and possibly handed the database to a
// fresh backup. Persisting then would resurrect a superseded backup, so the
// guarded read (locked FOR UPDATE to serialize against the sweep's conditional
// update) skips the write instead. The claim delete is
// scoped to backupID so it can never remove the newer backup's claim.
func (b *PhysicalBackuper) saveTerminalResultIfInProgress(
	databaseID, backupID uuid.UUID,
	loadStatus func(tx *gorm.DB) (physical_enums.PhysicalBackupStatus, error),
	save func(tx *gorm.DB) error,
) error {
	superseded := false

	err := storage.GetDb().Transaction(func(tx *gorm.DB) error {
		status, err := loadStatus(tx)
		if err != nil {
			return err
		}

		if status != physical_enums.PhysicalBackupStatusInProgress {
			superseded = true

			return nil
		}

		if err := save(tx); err != nil {
			return err
		}

		return tx.Delete(
			&physical_models.PhysicalInFlightBackup{},
			"database_id = ? AND backup_id = ?",
			databaseID,
			backupID,
		).Error
	})
	if err != nil {
		return err
	}

	if superseded {
		b.logger.Warn("backup row no longer in progress; skipping terminal persist", "backup_id", backupID)
	}

	return nil
}

func (b *PhysicalBackuper) finalizeFullAsError(
	fullBackup *physical_models.PhysicalFullBackup,
	reason physical_enums.PhysicalBackupErrorReason,
	message string,
) {
	r := reason

	fullBackup.Status = physical_enums.PhysicalBackupStatusError
	fullBackup.ErrorReason = &r
	fullBackup.FailMessage = nilOrPtr(message)

	if err := b.fullRepo.Save(fullBackup); err != nil {
		b.logger.Error("failed to flip full row to ERROR", "backup_id", fullBackup.ID, "error", err)
	}

	_ = b.inFlightRepo.ReleaseOwned(fullBackup.DatabaseID, fullBackup.ID)
}

func (b *PhysicalBackuper) finalizeIncrAsError(
	incrBackup *physical_models.PhysicalIncrementalBackup,
	reason physical_enums.PhysicalBackupErrorReason,
	message string,
) {
	b.finalizeIncrWithStatus(incrBackup, physical_enums.PhysicalBackupStatusError, reason, message)
}

// finalizeIncrAsChainBroken closes the chain instead of marking a transient
// failure. Use it only for irreversible conditions (a missing parent manifest,
// expired summaries) where retrying the same INCR is futile: CHAIN_BROKEN forces
// the next scheduler tick to open a fresh FULL, whereas ERROR would keep the
// chain extendable and retry the doomed INCR forever.
func (b *PhysicalBackuper) finalizeIncrAsChainBroken(
	incrBackup *physical_models.PhysicalIncrementalBackup,
	reason physical_enums.PhysicalBackupErrorReason,
	message string,
) {
	b.finalizeIncrWithStatus(incrBackup, physical_enums.PhysicalBackupStatusChainBroken, reason, message)
}

func (b *PhysicalBackuper) finalizeIncrWithStatus(
	incrBackup *physical_models.PhysicalIncrementalBackup,
	status physical_enums.PhysicalBackupStatus,
	reason physical_enums.PhysicalBackupErrorReason,
	message string,
) {
	r := reason

	incrBackup.Status = status
	incrBackup.ErrorReason = &r
	incrBackup.FailMessage = nilOrPtr(message)

	if err := b.incrRepo.Save(incrBackup); err != nil {
		b.logger.Error("failed to flip incr row to terminal status",
			"status", status, "backup_id", incrBackup.ID, "error", err)
	}

	_ = b.inFlightRepo.ReleaseOwned(incrBackup.DatabaseID, incrBackup.ID)
}

func (b *PhysicalBackuper) sendFullBackupNotification(
	cfg *backups_config_physical.PhysicalBackupConfig,
	db *databases.Database,
	fullBackup *physical_models.PhysicalFullBackup,
	backupResult postgresql_executor.PhysicalBackupResult,
) {
	notificationType, title, message := classifyFullBackupNotification(db, fullBackup, backupResult, b.workspaceService)
	if notificationType == "" {
		return
	}

	if !slices.Contains(cfg.SendNotificationsOn, notificationType) {
		return
	}

	for _, notifier := range db.Notifiers {
		b.notificationSender.SendNotification(&notifier, title, message)
	}
}

func (b *PhysicalBackuper) sendIncrBackupNotification(
	cfg *backups_config_physical.PhysicalBackupConfig,
	db *databases.Database,
	incrBackup *physical_models.PhysicalIncrementalBackup,
	backupResult postgresql_executor.PhysicalBackupResult,
) {
	notificationType, title, message := classifyIncrBackupNotification(db, incrBackup, backupResult, b.workspaceService)
	if notificationType == "" {
		return
	}

	if !slices.Contains(cfg.SendNotificationsOn, notificationType) {
		return
	}

	for _, notifier := range db.Notifiers {
		b.notificationSender.SendNotification(&notifier, title, message)
	}
}

func classifyFullBackupNotification(
	db *databases.Database,
	fullBackup *physical_models.PhysicalFullBackup,
	backupResult postgresql_executor.PhysicalBackupResult,
	workspaceService *workspaces_services.WorkspaceService,
) (backups_config_physical.BackupNotificationType, string, string) {
	workspaceName := "unknown"
	if db.WorkspaceID != nil {
		if ws, err := workspaceService.GetWorkspaceByID(*db.WorkspaceID); err == nil {
			workspaceName = ws.Name
		}
	}

	switch fullBackup.Status {
	case physical_enums.PhysicalBackupStatusCompleted:
		return backups_config_physical.NotificationBackupSuccess,
			fmt.Sprintf("Physical FULL completed for %q (workspace %q)", db.Name, workspaceName),
			fmt.Sprintf("backup_id=%s size=%.2f MB duration=%dms",
				fullBackup.ID, backupResult.BackupSizeMb, backupResult.BackupDurationMs)

	case physical_enums.PhysicalBackupStatusError:
		return backups_config_physical.NotificationBackupFailed,
			fmt.Sprintf("Physical FULL failed for %q (workspace %q)", db.Name, workspaceName),
			fmt.Sprintf("backup_id=%s reason=%s message=%s",
				fullBackup.ID, reasonOrEmpty(fullBackup.ErrorReason), backupResult.ErrorMessage)

	case physical_enums.PhysicalBackupStatusChainBroken:
		return backups_config_physical.NotificationChainBroken,
			fmt.Sprintf("Physical FULL chain-broken for %q (workspace %q)", db.Name, workspaceName),
			fmt.Sprintf("backup_id=%s reason=%s message=%s",
				fullBackup.ID, reasonOrEmpty(fullBackup.ErrorReason), backupResult.ErrorMessage)
	}

	return "", "", ""
}

func classifyIncrBackupNotification(
	db *databases.Database,
	incrBackup *physical_models.PhysicalIncrementalBackup,
	backupResult postgresql_executor.PhysicalBackupResult,
	workspaceService *workspaces_services.WorkspaceService,
) (backups_config_physical.BackupNotificationType, string, string) {
	workspaceName := "unknown"
	if db.WorkspaceID != nil {
		if ws, err := workspaceService.GetWorkspaceByID(*db.WorkspaceID); err == nil {
			workspaceName = ws.Name
		}
	}

	switch incrBackup.Status {
	case physical_enums.PhysicalBackupStatusCompleted:
		return backups_config_physical.NotificationBackupSuccess,
			fmt.Sprintf("Physical INCR completed for %q (workspace %q)", db.Name, workspaceName),
			fmt.Sprintf("backup_id=%s size=%.2f MB duration=%dms",
				incrBackup.ID, backupResult.BackupSizeMb, backupResult.BackupDurationMs)

	case physical_enums.PhysicalBackupStatusError:
		return backups_config_physical.NotificationBackupFailed,
			fmt.Sprintf("Physical INCR failed for %q (workspace %q)", db.Name, workspaceName),
			fmt.Sprintf("backup_id=%s reason=%s message=%s",
				incrBackup.ID, reasonOrEmpty(incrBackup.ErrorReason), backupResult.ErrorMessage)

	case physical_enums.PhysicalBackupStatusChainBroken:
		return backups_config_physical.NotificationChainBroken,
			fmt.Sprintf("Physical INCR chain-broken for %q (workspace %q)", db.Name, workspaceName),
			fmt.Sprintf("backup_id=%s reason=%s message=%s",
				incrBackup.ID, reasonOrEmpty(incrBackup.ErrorReason), backupResult.ErrorMessage)
	}

	return "", "", ""
}

func reasonOrEmpty(r *physical_enums.PhysicalBackupErrorReason) string {
	if r == nil {
		return ""
	}

	return string(*r)
}

func derefString(s *string) string {
	if s == nil {
		return ""
	}

	return *s
}

func nilOrPtr(s string) *string {
	if s == "" {
		return nil
	}

	return &s
}

func lsnPtr(v walmath.LSN) *walmath.LSN {
	out := v

	return &out
}
