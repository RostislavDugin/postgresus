package task_cancellation

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/google/uuid"

	"databasus-backend/internal/util/pubsub"
)

const taskCancelChannel = "task:cancel"

type Requester struct {
	broker pubsub.Broker
	logger *slog.Logger
}

func NewRequester(broker pubsub.Broker, logger *slog.Logger) *Requester {
	return &Requester{broker, logger}
}

func (r *Requester) RequestCancellation(ctx context.Context, taskID uuid.UUID) error {
	if err := r.broker.Publish(ctx, pubsub.Publication{
		Channel: taskCancelChannel,
		Message: taskID.String(),
	}); err != nil {
		r.logger.ErrorContext(ctx, "failed to publish task cancellation", "task_id", taskID, "error", err)

		return fmt.Errorf("publish task cancellation: %w", err)
	}

	r.logger.InfoContext(ctx, "published task cancellation", "task_id", taskID)

	return nil
}
