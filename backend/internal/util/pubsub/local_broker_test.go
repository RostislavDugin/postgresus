package pubsub

import (
	"context"
	"io"
	"log/slog"
	"runtime"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newTestBroker() *LocalBroker {
	return NewLocalBroker(slog.New(slog.NewTextHandler(io.Discard, nil)))
}

func Test_LocalBroker_WhenPublicationContextEnds_ReturnsError(t *testing.T) {
	broker := newTestBroker()
	t.Cleanup(func() { require.NoError(t, broker.Close()) })

	handlerStarted := make(chan struct{})
	releaseHandler := make(chan struct{})
	_, err := broker.Subscribe(t.Context(), "tasks", func(string) {
		close(handlerStarted)
		<-releaseHandler
	})
	require.NoError(t, err)

	require.NoError(t, broker.Publish(t.Context(), Publication{Channel: "tasks", Message: "first"}))
	<-handlerStarted

	publicationContext, cancel := context.WithTimeout(t.Context(), 10*time.Millisecond)
	defer cancel()
	err = broker.Publish(publicationContext, Publication{Channel: "tasks", Message: "second"})
	assert.ErrorIs(t, err, context.DeadlineExceeded)
	close(releaseHandler)
}

func Test_LocalSubscription_ClosePreservesSnapshottedPublication(t *testing.T) {
	broker := newTestBroker()
	t.Cleanup(func() { require.NoError(t, broker.Close()) })

	handlerStarted := make(chan struct{})
	releaseHandler := make(chan struct{})
	messages := make(chan string, 2)
	subscription, err := broker.Subscribe(t.Context(), "tasks", func(message string) {
		messages <- message
		if message == "first" {
			close(handlerStarted)
			<-releaseHandler
		}
	})
	require.NoError(t, err)
	require.NoError(t, broker.Publish(t.Context(), Publication{Channel: "tasks", Message: "first"}))
	<-handlerStarted

	publicationFinished := make(chan error, 1)
	go func() {
		publicationFinished <- broker.Publish(
			t.Context(),
			Publication{Channel: "tasks", Message: "second"},
		)
	}()
	local := subscription.(*localSubscription)
	waitForPendingDeliveries(t, local, 1)

	require.NoError(t, subscription.Close())
	close(releaseHandler)
	require.NoError(t, <-publicationFinished)
	<-local.done

	assert.Equal(t, "first", <-messages)
	assert.Equal(t, "second", <-messages)
	require.NoError(t, broker.Publish(t.Context(), Publication{Channel: "tasks", Message: "third"}))
	select {
	case message := <-messages:
		t.Fatalf("closed subscription received %q", message)
	default:
	}
}

func Test_LocalSubscription_HandlerCanCloseItsSubscription(t *testing.T) {
	broker := newTestBroker()
	t.Cleanup(func() { require.NoError(t, broker.Close()) })

	closeFinished := make(chan error, 1)
	var subscription Subscription
	createdSubscription, err := broker.Subscribe(t.Context(), "tasks", func(string) {
		closeFinished <- subscription.Close()
	})
	require.NoError(t, err)
	subscription = createdSubscription

	require.NoError(t, broker.Publish(t.Context(), Publication{Channel: "tasks", Message: "cancel"}))
	require.NoError(t, <-closeFinished)
}

func Test_LocalSubscription_HandlerCanCloseBroker(t *testing.T) {
	broker := newTestBroker()

	closeFinished := make(chan error, 1)
	_, err := broker.Subscribe(t.Context(), "tasks", func(string) {
		closeFinished <- broker.Close()
	})
	require.NoError(t, err)

	require.NoError(t, broker.Publish(t.Context(), Publication{Channel: "tasks", Message: "cancel"}))
	require.NoError(t, <-closeFinished)
}

func Test_LocalSubscription_ContextCancellationDrainsSnapshottedPublication(t *testing.T) {
	broker := newTestBroker()
	t.Cleanup(func() { require.NoError(t, broker.Close()) })

	subscriptionContext, cancelSubscription := context.WithCancel(t.Context())
	handlerStarted := make(chan struct{})
	releaseHandler := make(chan struct{})
	messages := make(chan string, 2)
	subscription, err := broker.Subscribe(subscriptionContext, "tasks", func(message string) {
		messages <- message
		if message == "first" {
			close(handlerStarted)
			<-releaseHandler
		}
	})
	require.NoError(t, err)
	require.NoError(t, broker.Publish(t.Context(), Publication{Channel: "tasks", Message: "first"}))
	<-handlerStarted

	publicationFinished := make(chan error, 1)
	go func() {
		publicationFinished <- broker.Publish(
			t.Context(),
			Publication{Channel: "tasks", Message: "second"},
		)
	}()
	local := subscription.(*localSubscription)
	waitForPendingDeliveries(t, local, 1)
	cancelSubscription()
	close(releaseHandler)

	require.NoError(t, <-publicationFinished)
	<-local.done
	assert.Equal(t, "first", <-messages)
	assert.Equal(t, "second", <-messages)
}

func waitForPendingDeliveries(t *testing.T, subscription *localSubscription, pendingCount int) {
	t.Helper()

	deadline := time.After(time.Second)
	for {
		subscription.stateMutex.Lock()
		currentPendingCount := subscription.pendingCount
		subscription.stateMutex.Unlock()
		if currentPendingCount == pendingCount {
			return
		}

		select {
		case <-deadline:
			t.Fatalf("pending delivery count remained %d", currentPendingCount)
		default:
			runtime.Gosched()
		}
	}
}
