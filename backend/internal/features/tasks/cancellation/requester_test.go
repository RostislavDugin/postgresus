package task_cancellation

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"

	"databasus-backend/internal/util/pubsub"
)

type failingBroker struct {
	err error
}

func (b failingBroker) Subscribe(context.Context, string, pubsub.MessageHandler) (pubsub.Subscription, error) {
	return nil, b.err
}

func (b failingBroker) Publish(context.Context, pubsub.Publication) error {
	return b.err
}

func (b failingBroker) Close() error {
	return b.err
}

func Test_Requester_WhenBrokerFails_ReturnsError(t *testing.T) {
	publicationError := errors.New("publication failed")
	taskCancellationRequester := NewRequester(failingBroker{err: publicationError}, NewTestLogger())

	err := taskCancellationRequester.RequestCancellation(t.Context(), uuid.New())
	assert.ErrorIs(t, err, publicationError)
}
