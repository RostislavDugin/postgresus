// Package testutil holds helpers shared across the verification agent's
// package tests.
package testutil

import (
	"bytes"
	"log/slog"
)

// DiscardLogger returns a no-op logger so tests don't write to the rotating
// log file or stdout.
func DiscardLogger() *slog.Logger {
	return slog.New(slog.DiscardHandler)
}

// CaptureLogger returns a logger that writes text records to the returned
// buffer, so a test can assert what was logged and at which level.
func CaptureLogger() (*slog.Logger, *bytes.Buffer) {
	buf := &bytes.Buffer{}

	return slog.New(slog.NewTextHandler(buf, &slog.HandlerOptions{Level: slog.LevelDebug})), buf
}
