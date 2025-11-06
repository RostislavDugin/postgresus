package notifiers

import "log/slog"

type NotificationSender interface {
	Send(logger *slog.Logger, vars map[string]string) error

	Validate() error
}
