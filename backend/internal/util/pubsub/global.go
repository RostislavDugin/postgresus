package pubsub

import "databasus-backend/internal/util/logger"

var defaultBroker Broker = NewLocalBroker(logger.GetLogger())

func GetBroker() Broker {
	return defaultBroker
}
