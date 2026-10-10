package backups_core_logical

import (
	"context"
	"errors"
	"fmt"
	"time"

	"databasus-backend/internal/config"
)

const (
	backupTimeLimit       = 23 * time.Hour
	shutdownCheckInterval = 1 * time.Second
)

// NewBackupContext derives the context an engine runs one logical backup under. Besides ending
// with callerCtx, it ends with ErrBackupShutdown when the application starts shutting down and
// with ErrBackupTimeout once the backup time limit passes.
func NewBackupContext(callerCtx context.Context) (context.Context, context.CancelCauseFunc) {
	backupCtx, cancel := context.WithCancelCause(callerCtx)

	timeLimit := time.AfterFunc(backupTimeLimit, func() { cancel(ErrBackupTimeout) })

	go func() {
		defer timeLimit.Stop()

		ticker := time.NewTicker(shutdownCheckInterval)
		defer ticker.Stop()

		for {
			select {
			case <-backupCtx.Done():
				return
			case <-ticker.C:
				if config.IsShouldShutdown() {
					cancel(ErrBackupShutdown)
					return
				}
			}
		}
	}()

	return backupCtx, cancel
}

// DescribeBackupInterruption explains why backupCtx ended. A cancellation is reported only when
// callerCtx ended, because only the caller cancels on request. Any other cause comes from the
// storage write, which cancels backupCtx with its own error; that error can still match
// context.DeadlineExceeded or context.Canceled, so the shape of the cause decides nothing.
func DescribeBackupInterruption(callerCtx, backupCtx context.Context) error {
	cause := context.Cause(backupCtx)

	switch {
	case errors.Is(cause, ErrBackupShutdown):
		return ErrBackupShutdown
	case errors.Is(cause, ErrBackupTimeout):
		return ErrBackupTimeout
	case callerCtx.Err() != nil:
		return errors.New("backup cancelled")
	default:
		return fmt.Errorf("save to storage: %w", cause)
	}
}
