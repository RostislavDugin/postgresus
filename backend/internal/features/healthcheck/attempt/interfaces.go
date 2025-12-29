package healthcheck_attempt

import (
	"databasus-backend/internal/features/databases"
	"databasus-backend/internal/features/notifiers"

	"github.com/google/uuid"
)

type HealthcheckAttemptSender interface {
	SendNotification(
		notifier *notifiers.Notifier,
		title string,
		message string,
	)
}

type DatabaseService interface {
	GetDatabaseByID(id uuid.UUID) (*databases.Database, error)

	TestDatabaseConnectionDirect(database *databases.Database) error

	SetHealthStatus(
		databaseID uuid.UUID,
		healthStatus *databases.HealthStatus,
	) error
}
