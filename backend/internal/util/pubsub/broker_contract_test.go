package pubsub_test

import (
	"io"
	"log/slog"
	"testing"

	"databasus-backend/internal/util/pubsub"
	"databasus-backend/internal/util/pubsub/pubsubtest"
)

func Test_LocalBroker_ProviderContract(t *testing.T) {
	pubsubtest.RunBrokerContract(t, func(*testing.T) pubsub.Broker {
		logger := slog.New(slog.NewTextHandler(io.Discard, nil))

		return pubsub.NewLocalBroker(logger)
	})
}
