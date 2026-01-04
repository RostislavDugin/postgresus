package notifiers

import (
	"databasus-backend/internal/util/encryption"
	"log/slog"

	"github.com/google/uuid"
)

type NotificationSender interface {
	Send(
		encryptor encryption.FieldEncryptor,
		logger *slog.Logger,
		heading string,
		message string,
	) error

	Validate(encryptor encryption.FieldEncryptor) error

	HideSensitiveData()

	EncryptSensitiveData(encryptor encryption.FieldEncryptor) error
}

type NotifierDatabaseCounter interface {
	GetNotifierAttachedDatabasesIDs(notifierID uuid.UUID) ([]uuid.UUID, error)
}
