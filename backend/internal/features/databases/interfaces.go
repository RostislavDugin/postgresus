package databases

import (
	"context"

	"github.com/google/uuid"
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
