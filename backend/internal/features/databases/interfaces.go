package databases

import (
	"context"

	"github.com/google/uuid"

	postgresql_physical "databasus-backend/internal/features/databases/databases/postgresql/physical"
)

type DatabaseCreationListener interface {
	OnDatabaseCreated(ctx context.Context, databaseID uuid.UUID)
}

type DatabaseRemoveListener interface {
	OnBeforeDatabaseRemove(ctx context.Context, databaseID uuid.UUID) error
}

type DatabaseCopyListener interface {
	OnDatabaseCopied(ctx context.Context, originalDatabaseID, newDatabaseID uuid.UUID)
}

type BackupTypeChange struct {
	DatabaseID    uuid.UUID
	OldBackupType postgresql_physical.BackupType
	NewBackupType postgresql_physical.BackupType
}

type DatabaseBackupTypeChangeListener interface {
	OnBackupTypeChanged(ctx context.Context, change BackupTypeChange)
}
