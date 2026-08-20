package verification_runs

import (
	"context"

	"databasus-backend/internal/features/notifiers"
	notifier_models "databasus-backend/internal/features/notifiers/models"
)

type NotificationSender interface {
	SendNotification(ctx context.Context, notifier *notifiers.Notifier, notification notifier_models.Notification)
}
