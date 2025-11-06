package healthcheck_attempt

import (
	"postgresus-backend/internal/features/databases"
	"postgresus-backend/internal/features/notifiers"

	"github.com/google/uuid"
)

type HealthcheckAttemptSender interface {
	SendNotification(
		notifier *notifiers.Notifier,
		vars map[string]string,
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
