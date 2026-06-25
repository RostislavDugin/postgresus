package backuping_logical

import (
	"context"
	"errors"
	"sync/atomic"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/mock"

	backups_core_enums "databasus-backend/internal/features/backups/backups/core/enums"
	backups_core_logical "databasus-backend/internal/features/backups/backups/core/logical"
	backups_config_logical "databasus-backend/internal/features/backups/config/logical"
	"databasus-backend/internal/features/databases"
	"databasus-backend/internal/features/notifiers"
	"databasus-backend/internal/features/storages"
)

type MockNotificationSender struct {
	mock.Mock
}

func (m *MockNotificationSender) SendNotification(
	notifier *notifiers.Notifier,
	title string,
	message string,
) {
	m.Called(notifier, title, message)
}

type CreateFailedBackupUsecase struct{}

func (uc *CreateFailedBackupUsecase) Execute(
	ctx context.Context,
	backup *backups_core_logical.LogicalBackup,
	backupConfig *backups_config_logical.LogicalBackupConfig,
	database *databases.Database,
	storage *storages.Storage,
	backupProgressListener func(completedMBs float64),
) (*backups_core_logical.BackupMetadata, error) {
	backupProgressListener(10)
	return nil, errors.New("backup failed")
}

type CreateSuccessBackupUsecase struct{}

func (uc *CreateSuccessBackupUsecase) Execute(
	ctx context.Context,
	backup *backups_core_logical.LogicalBackup,
	backupConfig *backups_config_logical.LogicalBackupConfig,
	database *databases.Database,
	storage *storages.Storage,
	backupProgressListener func(completedMBs float64),
) (*backups_core_logical.BackupMetadata, error) {
	backupProgressListener(10)
	return &backups_core_logical.BackupMetadata{
		EncryptionSalt: nil,
		EncryptionIV:   nil,
		Encryption:     backups_core_enums.BackupEncryptionNone,
	}, nil
}

// CreateLargeBackupUsecase simulates a large backup (10000 MB)
type CreateLargeBackupUsecase struct{}

func (uc *CreateLargeBackupUsecase) Execute(
	ctx context.Context,
	backup *backups_core_logical.LogicalBackup,
	backupConfig *backups_config_logical.LogicalBackupConfig,
	database *databases.Database,
	storage *storages.Storage,
	backupProgressListener func(completedMBs float64),
) (*backups_core_logical.BackupMetadata, error) {
	backupProgressListener(10000)
	return &backups_core_logical.BackupMetadata{
		EncryptionSalt: nil,
		EncryptionIV:   nil,
		Encryption:     backups_core_enums.BackupEncryptionNone,
	}, nil
}

// CreateProgressiveBackupUsecase simulates progressive size updates that exceed limit
type CreateProgressiveBackupUsecase struct{}

func (uc *CreateProgressiveBackupUsecase) Execute(
	ctx context.Context,
	backup *backups_core_logical.LogicalBackup,
	backupConfig *backups_config_logical.LogicalBackupConfig,
	database *databases.Database,
	storage *storages.Storage,
	backupProgressListener func(completedMBs float64),
) (*backups_core_logical.BackupMetadata, error) {
	// Simulate progressive backup that grows beyond limit
	backupProgressListener(1)
	if ctx.Err() != nil {
		return nil, ctx.Err()
	}

	backupProgressListener(3)
	if ctx.Err() != nil {
		return nil, ctx.Err()
	}

	backupProgressListener(5)
	if ctx.Err() != nil {
		return nil, ctx.Err()
	}

	backupProgressListener(10) // This exceeds the 5 MB limit
	if ctx.Err() != nil {
		return nil, ctx.Err()
	}

	// Should not reach here due to cancellation
	return &backups_core_logical.BackupMetadata{
		EncryptionSalt: nil,
		EncryptionIV:   nil,
		Encryption:     backups_core_enums.BackupEncryptionNone,
	}, nil
}

// CreateMediumBackupUsecase simulates a 50 MB backup
type CreateMediumBackupUsecase struct{}

func (uc *CreateMediumBackupUsecase) Execute(
	ctx context.Context,
	backup *backups_core_logical.LogicalBackup,
	backupConfig *backups_config_logical.LogicalBackupConfig,
	database *databases.Database,
	storage *storages.Storage,
	backupProgressListener func(completedMBs float64),
) (*backups_core_logical.BackupMetadata, error) {
	backupProgressListener(50)
	return &backups_core_logical.BackupMetadata{
		EncryptionSalt: nil,
		EncryptionIV:   nil,
		Encryption:     backups_core_enums.BackupEncryptionNone,
	}, nil
}

// MockTrackingBackupUsecase tracks backup use case calls for testing parallel execution
type MockTrackingBackupUsecase struct {
	callCount       atomic.Int32
	calledBackupIDs chan uuid.UUID
}

func NewMockTrackingBackupUsecase() *MockTrackingBackupUsecase {
	return &MockTrackingBackupUsecase{
		calledBackupIDs: make(chan uuid.UUID, 10),
	}
}

func (m *MockTrackingBackupUsecase) Execute(
	ctx context.Context,
	backup *backups_core_logical.LogicalBackup,
	backupConfig *backups_config_logical.LogicalBackupConfig,
	database *databases.Database,
	storage *storages.Storage,
	backupProgressListener func(completedMBs float64),
) (*backups_core_logical.BackupMetadata, error) {
	m.callCount.Add(1)

	// Send backup ID to channel (non-blocking)
	select {
	case m.calledBackupIDs <- backup.ID:
	default:
	}

	// Simulate backup work
	time.Sleep(100 * time.Millisecond)
	backupProgressListener(10)

	return &backups_core_logical.BackupMetadata{
		EncryptionSalt: nil,
		EncryptionIV:   nil,
		Encryption:     backups_core_enums.BackupEncryptionNone,
	}, nil
}

func (m *MockTrackingBackupUsecase) GetCallCount() int32 {
	return m.callCount.Load()
}

func (m *MockTrackingBackupUsecase) GetCalledBackupIDs() []uuid.UUID {
	ids := []uuid.UUID{}
	for {
		select {
		case id := <-m.calledBackupIDs:
			ids = append(ids, id)
		default:
			return ids
		}
	}
}
