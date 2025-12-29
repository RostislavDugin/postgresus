package notifiers

import (
	"databasus-backend/internal/util/encryption"
	"log/slog"
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
