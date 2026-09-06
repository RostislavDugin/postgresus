package task_cancellation

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"databasus-backend/internal/util/pubsub"
)

func Test_Listener_WithSharedBroker_CancelsTasksInEveryRegistry(t *testing.T) {
	broker := pubsub.NewLocalBroker(NewTestLogger())
	t.Cleanup(func() { require.NoError(t, broker.Close()) })

	firstRegistry := NewRegistry()
	secondRegistry := NewRegistry()
	firstListener := NewListener(broker, firstRegistry, NewTestLogger())
	secondListener := NewListener(broker, secondRegistry, NewTestLogger())
	require.NoError(t, firstListener.Start(t.Context()))
	require.NoError(t, secondListener.Start(t.Context()))
	t.Cleanup(func() { require.NoError(t, firstListener.Close()) })
	t.Cleanup(func() { require.NoError(t, secondListener.Close()) })

	taskID := uuid.New()
	firstContext, cancelFirstTask := context.WithCancel(t.Context())
	secondContext, cancelSecondTask := context.WithCancel(t.Context())
	firstRegistry.RegisterTask(taskID, cancelFirstTask)
	secondRegistry.RegisterTask(taskID, cancelSecondTask)

	taskCancellationRequester := NewRequester(broker, NewTestLogger())
	require.NoError(t, taskCancellationRequester.RequestCancellation(t.Context(), taskID))

	assertContextCanceled(t, firstContext)
	assertContextCanceled(t, secondContext)
}

func assertContextCanceled(t *testing.T, taskContext context.Context) {
	t.Helper()

	select {
	case <-taskContext.Done():
		assert.ErrorIs(t, taskContext.Err(), context.Canceled)
	case <-time.After(time.Second):
		t.Fatal("task context was not canceled")
	}
}
