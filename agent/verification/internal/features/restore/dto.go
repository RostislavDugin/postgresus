package restore

import "databasus-verification-agent/internal/features/dbconn"

// A non-zero ExitCode is not a Go error here — only failing to create, start,
// or attach the exec is.
type ExecResult struct {
	ExitCode int
	Stdout   string
	Stderr   string
}

type PgRestoreSpec struct {
	ArchivePath   string
	Conn          dbconn.Conn
	ParallelJobs  int
	IsTimescaledb bool
}

// Result is populated even on the error path (see ErrRestoreFailed).
type Result struct {
	PgRestoreExitCode int
	DurationMs        int64
	StderrTail        string
}
