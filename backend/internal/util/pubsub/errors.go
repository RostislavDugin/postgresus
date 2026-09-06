package pubsub

import "errors"

var ErrBrokerClosed = errors.New("publish-subscribe broker is closed")
