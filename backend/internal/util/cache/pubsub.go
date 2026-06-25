package cache_utils

import (
	"context"
	"fmt"
	"log/slog"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/valkey-io/valkey-go"

	"databasus-backend/internal/util/logger"
)

// readyMarkerPrefix tags the self-published probe that Subscribe uses to
// confirm the SUBSCRIBE command has been accepted by Valkey before
// returning. The wrapped handler intercepts these markers and never
// forwards them to the real handler.
const readyMarkerPrefix = "__databasus_subscribe_ready__:"

// subscribeReadyTimeout caps how long Subscribe waits for its own ready
// marker to round-trip. 5s is generous enough to absorb a stressed CI host
// without making subscriber-setup hang dominate startup time.
const subscribeReadyTimeout = 5 * time.Second

// subscribeReadyPollInterval is how often we re-publish the ready marker
// while waiting. Each publish before the subscriber is registered is lost
// (pubsub is fire-and-forget); we keep republishing until the subscriber
// catches one.
const subscribeReadyPollInterval = 25 * time.Millisecond

type PubSubManager struct {
	client        valkey.Client
	subscriptions map[string]context.CancelFunc
	mu            sync.RWMutex
	logger        *slog.Logger
}

func NewPubSubManager() *PubSubManager {
	return &PubSubManager{
		client:        getCache(),
		subscriptions: make(map[string]context.CancelFunc),
		logger:        logger.GetLogger(),
	}
}

func (m *PubSubManager) Subscribe(
	ctx context.Context,
	channel string,
	handler func(message string),
) error {
	m.mu.Lock()
	if _, exists := m.subscriptions[channel]; exists {
		m.mu.Unlock()
		return fmt.Errorf("already subscribed to channel: %s", channel)
	}

	subCtx, cancel := context.WithCancel(ctx)
	m.subscriptions[channel] = cancel
	m.mu.Unlock()

	readyMarker := readyMarkerPrefix + uuid.New().String()
	ready := make(chan struct{})
	readyOnce := sync.Once{}

	wrappedHandler := func(message string) {
		if strings.HasPrefix(message, readyMarkerPrefix) {
			if message == readyMarker {
				readyOnce.Do(func() { close(ready) })
			}
			return
		}

		handler(message)
	}

	go m.subscriptionLoop(subCtx, channel, wrappedHandler)

	if err := m.waitForSubscribeReady(ctx, channel, readyMarker, ready); err != nil {
		cancel()

		m.mu.Lock()
		delete(m.subscriptions, channel)
		m.mu.Unlock()

		return err
	}

	return nil
}

func (m *PubSubManager) Publish(ctx context.Context, channel, message string) error {
	cmd := m.client.B().Publish().Channel(channel).Message(message).Build()
	result := m.client.Do(ctx, cmd)

	if err := result.Error(); err != nil {
		m.logger.Error("Failed to publish message to Redis", "channel", channel, "error", err)
		return fmt.Errorf("failed to publish message: %w", err)
	}

	return nil
}

func (m *PubSubManager) Close() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	for channel, cancel := range m.subscriptions {
		cancel()
		delete(m.subscriptions, channel)
	}

	return nil
}

func (m *PubSubManager) subscriptionLoop(
	ctx context.Context,
	channel string,
	handler func(message string),
) {
	defer func() {
		if r := recover(); r != nil {
			m.logger.Error("Panic in subscription handler", "channel", channel, "panic", r)
		}
	}()

	m.logger.Info("Starting subscription", "channel", channel)

	err := m.client.Receive(
		ctx,
		m.client.B().Subscribe().Channel(channel).Build(),
		func(msg valkey.PubSubMessage) {
			defer func() {
				if r := recover(); r != nil {
					m.logger.Error("Panic in message handler", "channel", channel, "panic", r)
				}
			}()

			handler(msg.Message)
		},
	)

	if err != nil && ctx.Err() == nil {
		m.logger.Error("Subscription error", "channel", channel, "error", err)
	} else if ctx.Err() != nil {
		m.logger.Info("Subscription cancelled", "channel", channel)
	}

	m.mu.Lock()
	delete(m.subscriptions, channel)
	m.mu.Unlock()
}

// waitForSubscribeReady republishes a unique marker on the channel until
// the subscriber's wrapped handler observes it. This proves that the
// SUBSCRIBE command has been accepted by Valkey and our handler is in the
// receive path. Without this handshake, callers race the SUBSCRIBE ack and
// can publish-then-miss the next message; tests papered over it with
// time.Sleep heuristics that broke under load.
func (m *PubSubManager) waitForSubscribeReady(
	ctx context.Context,
	channel string,
	readyMarker string,
	ready <-chan struct{},
) error {
	deadline := time.NewTimer(subscribeReadyTimeout)
	defer deadline.Stop()

	tick := time.NewTicker(subscribeReadyPollInterval)
	defer tick.Stop()

	publishCtx, cancelPublish := context.WithTimeout(ctx, subscribeReadyTimeout)
	defer cancelPublish()

	publish := func() {
		_ = m.client.Do(
			publishCtx,
			m.client.B().Publish().Channel(channel).Message(readyMarker).Build(),
		).Error()
	}

	publish()

	for {
		select {
		case <-ready:
			return nil

		case <-tick.C:
			publish()

		case <-deadline.C:
			return fmt.Errorf("subscribe ready timeout on channel %q after %s", channel, subscribeReadyTimeout)

		case <-ctx.Done():
			return ctx.Err()
		}
	}
}
