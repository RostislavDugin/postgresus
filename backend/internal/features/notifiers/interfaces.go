package notifiers

import "log/slog"

type NotificationSender interface {
	Send(logger *slog.Logger, heading string, message string) error

	Validate() error

	HideSensitiveData()
}
