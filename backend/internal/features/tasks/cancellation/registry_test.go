package task_cancellation

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

func Test_Registry_CancelRegisteredTask_CancelsAndRemovesTask(t *testing.T) {
	registry := NewRegistry()
	taskID := uuid.New()
	taskContext, cancelTask := context.WithCancel(t.Context())
	registry.RegisterTask(taskID, cancelTask)

	registry.CancelRegisteredTask(taskID)
	assert.ErrorIs(t, taskContext.Err(), context.Canceled)

	secondContext, secondCancelTask := context.WithCancel(t.Context())
	defer secondCancelTask()
	registry.RegisterTask(taskID, secondCancelTask)
	registry.UnregisterTask(taskID)
	registry.CancelRegisteredTask(taskID)
	assert.NoError(t, secondContext.Err())
}
