package databases

import (
	"databasus-backend/internal/util/encryption"
	"log/slog"

	"github.com/google/uuid"
)

type DatabaseValidator interface {
	Validate() error
}

type DatabaseConnector interface {
	TestConnection(
		logger *slog.Logger,
		encryptor encryption.FieldEncryptor,
		databaseID uuid.UUID,
	) error

	HideSensitiveData()
}

type DatabaseCreationListener interface {
	OnDatabaseCreated(databaseID uuid.UUID)
}

type DatabaseRemoveListener interface {
	OnBeforeDatabaseRemove(databaseID uuid.UUID) error
}

type DatabaseCopyListener interface {
	OnDatabaseCopied(originalDatabaseID, newDatabaseID uuid.UUID)
}
