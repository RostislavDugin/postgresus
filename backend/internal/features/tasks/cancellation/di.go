package task_cancellation

import (
	"context"
	"sync"

	"databasus-backend/internal/util/logger"
	"databasus-backend/internal/util/pubsub"
)

var (
	taskCancellationRegistry  = NewRegistry()
	taskCancellationRequester = NewRequester(pubsub.GetBroker(), logger.GetLogger())
	taskCancellationListener  = NewListener(
		pubsub.GetBroker(),
		taskCancellationRegistry,
		logger.GetLogger(),
	)
)

func GetRegistry() *Registry {
	return taskCancellationRegistry
}

func GetRequester() *Requester {
	return taskCancellationRequester
}

func GetListener() *Listener {
	return taskCancellationListener
}

var SetupDependencies = sync.OnceFunc(func() {
	if err := taskCancellationListener.Start(context.Background()); err != nil {
		panic(err)
	}
})
