package task_cancellation

import (
	"context"
	"log/slog"
	"sync"

	"github.com/google/uuid"

	"databasus-backend/internal/util/pubsub"
)

type Listener struct {
	mutex        sync.Mutex
	broker       pubsub.Broker
	registry     *Registry
	logger       *slog.Logger
	subscription pubsub.Subscription
}

func NewListener(broker pubsub.Broker, registry *Registry, logger *slog.Logger) *Listener {
	return &Listener{
		broker:   broker,
		registry: registry,
		logger:   logger,
	}
}

func (l *Listener) Start(ctx context.Context) error {
	l.mutex.Lock()
	defer l.mutex.Unlock()

	subscription, err := l.broker.Subscribe(ctx, taskCancelChannel, l.cancelTask)
	if err != nil {
		return err
	}
	l.subscription = subscription

	l.logger.InfoContext(ctx, "subscribed to task cancellations")

	return nil
}

func (l *Listener) Close() error {
	l.mutex.Lock()
	defer l.mutex.Unlock()

	if l.subscription == nil {
		return nil
	}

	err := l.subscription.Close()
	l.subscription = nil

	return err
}

func (l *Listener) cancelTask(message string) {
	taskID, err := uuid.Parse(message)
	if err != nil {
		l.logger.Error("received invalid task cancellation", "error", err)

		return
	}

	l.registry.CancelRegisteredTask(taskID)
}
