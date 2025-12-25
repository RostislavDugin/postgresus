package backups

import (
	"databasus-backend/internal/features/notifiers"

	"github.com/stretchr/testify/mock"
)

type MockNotificationSender struct {
	mock.Mock
}

func (m *MockNotificationSender) SendNotification(
	notifier *notifiers.Notifier,
	title string,
	message string,
) {
	m.Called(notifier, title, message)
}
