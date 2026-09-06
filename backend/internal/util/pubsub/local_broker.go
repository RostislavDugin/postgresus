package pubsub

import (
	"context"
	"log/slog"
	"sync"
)

type LocalBroker struct {
	mutex            sync.Mutex
	subscriptions    map[string]map[*localSubscription]struct{}
	publicationGates map[string]chan struct{}
	isClosed         bool
	logger           *slog.Logger
}

type localSubscription struct {
	broker              *LocalBroker
	channel             string
	handler             MessageHandler
	subscriptionContext context.Context
	mailbox             chan string
	stopped             chan struct{}
	done                chan struct{}
	stateMutex          sync.Mutex
	isClosed            bool
	pendingCount        int
}

func NewLocalBroker(logger *slog.Logger) *LocalBroker {
	return &LocalBroker{
		subscriptions:    make(map[string]map[*localSubscription]struct{}),
		publicationGates: make(map[string]chan struct{}),
		logger:           logger,
	}
}

func (b *LocalBroker) Subscribe(
	ctx context.Context,
	channel string,
	handler MessageHandler,
) (Subscription, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	subscription := &localSubscription{
		broker:              b,
		channel:             channel,
		handler:             handler,
		subscriptionContext: ctx,
		mailbox:             make(chan string),
		stopped:             make(chan struct{}),
		done:                make(chan struct{}),
	}

	b.mutex.Lock()
	if b.isClosed {
		b.mutex.Unlock()

		return nil, ErrBrokerClosed
	}
	if b.subscriptions[channel] == nil {
		b.subscriptions[channel] = make(map[*localSubscription]struct{})
	}
	b.subscriptions[channel][subscription] = struct{}{}
	b.mutex.Unlock()

	go subscription.run()

	return subscription, nil
}

func (b *LocalBroker) Publish(ctx context.Context, publication Publication) error {
	if err := ctx.Err(); err != nil {
		return err
	}

	publicationGate, err := b.getPublicationGate(publication.Channel)
	if err != nil {
		return err
	}

	select {
	case publicationGate <- struct{}{}:
		defer func() { <-publicationGate }()
	case <-ctx.Done():
		return ctx.Err()
	}

	subscriptions, err := b.reserveSubscriptions(publication.Channel)
	if err != nil {
		return err
	}

	for subscriptionIndex, subscription := range subscriptions {
		select {
		case subscription.mailbox <- publication.Message:
			subscription.releaseDelivery()
		case <-ctx.Done():
			for _, pendingSubscription := range subscriptions[subscriptionIndex:] {
				pendingSubscription.releaseDelivery()
			}

			return ctx.Err()
		}
	}

	return nil
}

func (b *LocalBroker) Close() error {
	b.mutex.Lock()
	if b.isClosed {
		b.mutex.Unlock()

		return nil
	}
	b.isClosed = true

	subscriptions := make([]*localSubscription, 0)
	for _, channelSubscriptions := range b.subscriptions {
		for subscription := range channelSubscriptions {
			subscriptions = append(subscriptions, subscription)
		}
	}
	b.subscriptions = make(map[string]map[*localSubscription]struct{})
	b.mutex.Unlock()

	for _, subscription := range subscriptions {
		if err := subscription.Close(); err != nil {
			return err
		}
	}

	return nil
}

func (b *LocalBroker) getPublicationGate(channel string) (chan struct{}, error) {
	b.mutex.Lock()
	defer b.mutex.Unlock()

	if b.isClosed {
		return nil, ErrBrokerClosed
	}

	publicationGate := b.publicationGates[channel]
	if publicationGate == nil {
		publicationGate = make(chan struct{}, 1)
		b.publicationGates[channel] = publicationGate
	}

	return publicationGate, nil
}

func (b *LocalBroker) reserveSubscriptions(channel string) ([]*localSubscription, error) {
	b.mutex.Lock()
	defer b.mutex.Unlock()

	if b.isClosed {
		return nil, ErrBrokerClosed
	}

	channelSubscriptions := b.subscriptions[channel]
	subscriptions := make([]*localSubscription, 0, len(channelSubscriptions))
	for subscription := range channelSubscriptions {
		if subscription.reserveDelivery() {
			subscriptions = append(subscriptions, subscription)
		}
	}

	return subscriptions, nil
}

func (b *LocalBroker) closeSubscription(subscription *localSubscription) {
	b.mutex.Lock()
	subscription.stateMutex.Lock()
	if !subscription.isClosed {
		subscription.isClosed = true
		channelSubscriptions := b.subscriptions[subscription.channel]
		delete(channelSubscriptions, subscription)
		if len(channelSubscriptions) == 0 {
			delete(b.subscriptions, subscription.channel)
		}
		if subscription.pendingCount == 0 {
			close(subscription.stopped)
		}
	}
	subscription.stateMutex.Unlock()
	b.mutex.Unlock()
}

func (s *localSubscription) Close() error {
	s.broker.closeSubscription(s)

	return nil
}

func (s *localSubscription) reserveDelivery() bool {
	s.stateMutex.Lock()
	defer s.stateMutex.Unlock()

	if s.isClosed || s.subscriptionContext.Err() != nil {
		return false
	}

	s.pendingCount++

	return true
}

func (s *localSubscription) releaseDelivery() {
	s.stateMutex.Lock()
	defer s.stateMutex.Unlock()

	s.pendingCount--
	if s.isClosed && s.pendingCount == 0 {
		close(s.stopped)
	}
}

func (s *localSubscription) run() {
	defer close(s.done)

	for {
		select {
		case message := <-s.mailbox:
			s.invoke(s.subscriptionContext, message)
		case <-s.subscriptionContext.Done():
			s.broker.closeSubscription(s)
			s.drain(s.subscriptionContext)

			return
		case <-s.stopped:
			return
		}
	}
}

func (s *localSubscription) drain(ctx context.Context) {
	for {
		select {
		case message := <-s.mailbox:
			s.invoke(ctx, message)
		case <-s.stopped:
			return
		}
	}
}

func (s *localSubscription) invoke(ctx context.Context, message string) {
	defer func() {
		if recovered := recover(); recovered != nil {
			s.broker.logger.ErrorContext(
				ctx,
				"panic in publish-subscribe handler",
				"channel",
				s.channel,
				"panic",
				recovered,
			)
		}
	}()

	s.handler(message)
}
