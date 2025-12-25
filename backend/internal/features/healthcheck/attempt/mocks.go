package healthcheck_attempt

import (
	"databasus-backend/internal/features/databases"
	"databasus-backend/internal/features/notifiers"

	"github.com/google/uuid"
	"github.com/stretchr/testify/mock"
)

type MockHealthcheckAttemptSender struct {
	mock.Mock
}

func (m *MockHealthcheckAttemptSender) SendNotification(
	notifier *notifiers.Notifier,
	title string,
	message string,
) {
	m.Called(notifier, title, message)
}

type MockDatabaseService struct {
	mock.Mock
}

func (m *MockDatabaseService) TestDatabaseConnectionDirect(
	database *databases.Database,
) error {
	return m.Called(database).Error(0)
}

func (m *MockDatabaseService) SetHealthStatus(
	databaseID uuid.UUID,
	healthStatus *databases.HealthStatus,
) error {
	return m.Called(databaseID, healthStatus).Error(0)
}

func (m *MockDatabaseService) GetDatabaseByID(
	id uuid.UUID,
) (*databases.Database, error) {
	args := m.Called(id)

	if args.Get(0) == nil {
		return nil, args.Error(1)
	}

	database, ok := args.Get(0).(*databases.Database)
	if !ok {
		return nil, args.Error(1)
	}

	return database, args.Error(1)
}
