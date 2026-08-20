// Package logger builds the application's slog pipeline: a stdout console sink, an optional
// rotating file, and an optional OpenTelemetry exporter, fanned out from a single handler.
//
// internal/config imports this package, so this package must not import internal/config. That is
// why it loads .env and reads its own environment variables instead of going through EnvVariables.
//
// Redaction lives here rather than at call sites because every sink must be covered by it, and a
// call site cannot know that its line is about to be written to disk and shipped off-box.
package logger
