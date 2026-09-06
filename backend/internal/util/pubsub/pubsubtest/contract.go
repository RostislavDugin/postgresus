// Package pubsubtest provides reusable contract checks for message brokers.
package pubsubtest

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"databasus-backend/internal/util/pubsub"
)

type BrokerFactory func(t *testing.T) pubsub.Broker

func RunBrokerContract(t *testing.T, createBroker BrokerFactory) {
	t.Helper()

	t.Run("Test_Subscribe_WhenPublished_FansOutImmediately", func(t *testing.T) {
		broker := createBroker(t)
		t.Cleanup(func() { require.NoError(t, broker.Close()) })
		firstMessages := make(chan string, 1)
		secondMessages := make(chan string, 1)

		_, err := broker.Subscribe(t.Context(), "tasks", func(message string) { firstMessages <- message })
		require.NoError(t, err)
		_, err = broker.Subscribe(t.Context(), "tasks", func(message string) { secondMessages <- message })
		require.NoError(t, err)
		require.NoError(t, broker.Publish(t.Context(), pubsub.Publication{
			Channel: "tasks", Message: "cancel",
		}))

		assert.Equal(t, "cancel", <-firstMessages)
		assert.Equal(t, "cancel", <-secondMessages)
	})

	t.Run("Test_Publish_WithOverlappingMessages_PreservesOrder", func(t *testing.T) {
		broker := createBroker(t)
		t.Cleanup(func() { require.NoError(t, broker.Close()) })
		messages := make(chan string, 2)
		_, err := broker.Subscribe(t.Context(), "tasks", func(message string) { messages <- message })
		require.NoError(t, err)

		require.NoError(t, broker.Publish(t.Context(), pubsub.Publication{Channel: "tasks", Message: "first"}))
		require.NoError(t, broker.Publish(t.Context(), pubsub.Publication{Channel: "tasks", Message: "second"}))

		assert.Equal(t, "first", <-messages)
		assert.Equal(t, "second", <-messages)
	})

	t.Run("Test_Subscribe_WhenContextCanceled_StopsLaterDelivery", func(t *testing.T) {
		broker := createBroker(t)
		t.Cleanup(func() { require.NoError(t, broker.Close()) })
		subscriptionContext, cancelSubscription := context.WithCancel(t.Context())
		messages := make(chan string, 1)
		_, err := broker.Subscribe(
			subscriptionContext,
			"tasks",
			func(message string) { messages <- message },
		)
		require.NoError(t, err)
		cancelSubscription()

		require.NoError(t, broker.Publish(t.Context(), pubsub.Publication{
			Channel: "tasks", Message: "cancel",
		}))
		select {
		case message := <-messages:
			t.Fatalf("canceled subscription received %q", message)
		default:
		}
	})

	t.Run("Test_Publish_WhenHandlerPanics_IsolatesOtherSubscriptions", func(t *testing.T) {
		broker := createBroker(t)
		t.Cleanup(func() { require.NoError(t, broker.Close()) })
		healthyMessages := make(chan string, 2)
		_, err := broker.Subscribe(t.Context(), "tasks", func(string) { panic("handler failed") })
		require.NoError(t, err)
		_, err = broker.Subscribe(t.Context(), "tasks", func(message string) { healthyMessages <- message })
		require.NoError(t, err)

		require.NoError(t, broker.Publish(t.Context(), pubsub.Publication{Channel: "tasks", Message: "first"}))
		require.NoError(t, broker.Publish(t.Context(), pubsub.Publication{Channel: "tasks", Message: "second"}))
		assert.Equal(t, "first", <-healthyMessages)
		assert.Equal(t, "second", <-healthyMessages)
	})

	t.Run("Test_Close_WhenRepeated_RejectsLaterOperations", func(t *testing.T) {
		broker := createBroker(t)
		require.NoError(t, broker.Close())
		require.NoError(t, broker.Close())
		assert.ErrorIs(t, broker.Publish(t.Context(), pubsub.Publication{
			Channel: "tasks", Message: "cancel",
		}), pubsub.ErrBrokerClosed)
		_, err := broker.Subscribe(t.Context(), "tasks", func(string) {})
		assert.ErrorIs(t, err, pubsub.ErrBrokerClosed)
	})
}
