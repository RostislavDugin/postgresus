package pubsub

import "context"

type MessageHandler func(message string)

type Publication struct {
	Channel string
	Message string
}

type Subscription interface {
	Close() error
}

type Broker interface {
	Subscribe(ctx context.Context, channel string, handler MessageHandler) (Subscription, error)
	Publish(ctx context.Context, publication Publication) error
	Close() error
}
