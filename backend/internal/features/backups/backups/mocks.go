package backups

import (
	"postgresus-backend/internal/features/notifiers"

	"github.com/stretchr/testify/mock"
)

type MockNotificationSender struct {
	mock.Mock
}

func (m *MockNotificationSender) SendNotification(
	notifier *notifiers.Notifier,
	vars map[string]string,
) {
	m.Called(notifier, vars)
}
