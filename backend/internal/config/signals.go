package config

import (
	"os"
	"os/signal"
	"syscall"

	"databasus-backend/internal/util/logger"
)

var isShutDownSignalReceived = false

func StartListeningForShutdownSignal() {
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, os.Interrupt, syscall.SIGTERM)

	go func() {
		<-quit

		// Nothing else records why the process began shutting down.
		logger.GetLogger().Info("shutdown signal received")

		isShutDownSignalReceived = true
	}()
}

func IsShouldShutdown() bool {
	return isShutDownSignalReceived
}
