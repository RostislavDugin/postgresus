package healthcheck_attempt

import (
	"context"

	"github.com/google/uuid"

	"databasus-backend/internal/features/databases"
	"databasus-backend/internal/features/notifiers"
	notifier_models "databasus-backend/internal/features/notifiers/models"
)

type HealthcheckAttemptSender interface {
	SendNotification(
		ctx context.Context,
		notifier *notifiers.Notifier,
		notification notifier_models.Notification,
	)
}

type DatabaseService interface {
	GetDatabaseByID(id uuid.UUID) (*databases.Database, error)

	TestDatabaseConnectionDirect(ctx context.Context, database *databases.Database) error

	SetHealthStatus(
		databaseID uuid.UUID,
		healthStatus *databases.HealthStatus,
	) error
}
