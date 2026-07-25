package usecases_physical_postgresql

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"syscall"
	"time"

	postgresql_physical "databasus-backend/internal/features/databases/databases/postgresql/physical"
	postgresql_shared "databasus-backend/internal/features/databases/databases/postgresql/shared"
)

const receivewalApplicationNamePrefix = "databasus_wal_receiver_"

// receiverExit is the disposition of one pg_receivewal run, decided by
// spawnAndSupervise and acted on by runReceivewalSupervision.
type receiverExit int

const (
	receiverCtxCancelled    receiverExit = iota // ctx cancelled — stop the loop, no error
	receiverInternalRestart                     // our own SIGTERM (back pressure / slot stall) — respawn promptly
	receiverRetryable                           // non-zero exit that a local respawn may fix (network)
	receiverFatal                               // non-retryable exit — escalate so the supervisor reclaims it on a later tick
)

// spawnAndSupervise starts one pg_receivewal process and blocks until it exits,
// the restart signal fires, or ctx is cancelled. It reports the run's
// disposition, how long the process streamed, and (for receiverFatal) the error.
func (s *WalStreamSupervisor) spawnAndSupervise(
	ctx context.Context,
	logger *slog.Logger,
	pgBin string,
) (receiverExit, time.Duration, error) {
	password, err := postgresql_shared.DecryptFieldIfNeeded(s.spec.SourceDB.Password, s.spec.FieldEncryptor)
	if err != nil {
		logger.Error("decrypt source password for pg_receivewal", "error", err)

		return receiverRetryable, 0, nil
	}

	creds, err := postgresql_shared.WriteCredentialFilesToTempDir(
		s.spec.SourceDB.CredentialSpec(), password, s.spec.FieldEncryptor,
	)
	if err != nil {
		logger.Error("write pg_receivewal credentials", "error", err)

		return receiverRetryable, 0, nil
	}
	defer creds.Remove()

	procCtx, procCancel := context.WithCancel(ctx)
	defer procCancel()

	cmd, err := newReceivewalCommand(procCtx, pgBin, s.spec.SourceDB, creds, s.watchDir, s.slotName)
	if err != nil {
		logger.Error("build pg_receivewal command", "error", err)

		return receiverRetryable, 0, nil
	}

	stderrPipe, err := cmd.StderrPipe()
	if err != nil {
		logger.Error("pg_receivewal stderr pipe", "error", err)

		return receiverRetryable, 0, nil
	}

	if err := cmd.Start(); err != nil {
		logger.Error("start pg_receivewal", "error", err)

		return receiverRetryable, 0, nil
	}

	stderr := newStderrCapture(stderrPipe)
	startedAt := time.Now().UTC()

	logger.Info("pg_receivewal started", "watch_dir", s.watchDir)

	exited := make(chan error, 1)

	go func() { exited <- cmd.Wait() }()

	select {
	case <-ctx.Done():
		procCancel()
		<-exited
		stderr.stop()

		return receiverCtxCancelled, 0, nil

	case <-s.restartSignal:
		logger.Info("restarting pg_receivewal on internal signal (back pressure or slot stall)")
		procCancel()
		<-exited
		stderr.stop()

		return receiverInternalRestart, time.Since(startedAt), nil

	case waitErr := <-exited:
		stderr.stop()
		ranFor := time.Since(startedAt)

		if waitErr == nil || procCtx.Err() != nil {
			return receiverRetryable, ranFor, nil
		}

		stderrText := stderr.contents()

		if isFatalReceivewalError(stderrText) {
			logger.Error("pg_receivewal exited with a non-retryable error; marking streamer for reassignment",
				"error", waitErr, "stderr", truncateStderr(stderrText))

			return receiverFatal, ranFor, fmt.Errorf(
				"pg_receivewal fatal error: %w; stderr: %s", waitErr, truncateStderr(stderrText),
			)
		}

		logger.Warn("pg_receivewal exited; will respawn",
			"error", waitErr, "stderr", truncateStderr(stderrText))

		return receiverRetryable, ranFor, nil
	}
}

// isFatalReceivewalError reports whether a pg_receivewal stderr indicates a
// condition a local respawn can never clear — a full / unwritable local disk,
// rejected credentials, or the replication slot being held by another consumer.
// These escalate to streamer-FAILED so the supervisor can reclaim it on a later
// tick instead of this process crash-looping on an unfixable cause.
func isFatalReceivewalError(stderr []byte) bool {
	// Lower-case both sides: OS errno strings vary in case ("Permission denied")
	// while PG messages are lower-case, and we want to match either.
	text := strings.ToLower(string(stderr))

	for _, needle := range []string{
		"no space left on device",
		"could not write",
		"authentication failed",
		"no pg_hba.conf entry",
		"permission denied",
		"is active for", // replication slot "<name>" is active for PID <n>
	} {
		if strings.Contains(text, needle) {
			return true
		}
	}

	return false
}

// newReceivewalCommand builds the pg_receivewal invocation. WAL is left
// uncompressed locally (no --compress) because the uploader re-compresses with
// zstd on upload; --no-loop makes the process exit on connection loss so the
// supervision loop owns retry; --synchronous flushes each segment promptly. SSL
// is supplied through the same PGSSL* env path pg_basebackup uses, so mTLS needs
// no extra handling here.
func newReceivewalCommand(
	ctx context.Context,
	pgBin string,
	sourceDB *postgresql_physical.PostgresqlPhysicalDatabase,
	creds *postgresql_shared.CredentialTempFiles,
	watchDir string,
	slotName string,
) (*exec.Cmd, error) {
	if _, err := exec.LookPath(pgBin); err != nil {
		return nil, fmt.Errorf("pg_receivewal binary not found at %s: %w", pgBin, err)
	}

	args := []string{
		"--directory=" + watchDir,
		"--slot=" + slotName,
		"--no-loop",
		"--synchronous",
		"--verbose",
		"--no-password",
		"-h", sourceDB.Host,
		"-p", strconv.Itoa(sourceDB.Port),
		"-U", sourceDB.Username,
	}

	cmd := exec.CommandContext(ctx, pgBin, args...)

	cmd.Env = append(os.Environ(),
		"PGPASSFILE="+creds.PgpassPath,
		"PGAPPNAME="+receivewalApplicationName(sourceDB),
		"PGCLIENTENCODING=UTF8",
		"PGCONNECT_TIMEOUT=30",
		"LC_ALL=C.UTF-8",
		"LANG=C.UTF-8",
	)

	sslMode := sourceDB.SslMode
	if sslMode == "" {
		sslMode = postgresql_shared.PostgresSslModeDisable
	}

	cmd.Env = append(cmd.Env,
		"PGSSLMODE="+string(sslMode),
		"PGSSLCERT="+creds.ClientCertPath,
		"PGSSLKEY="+creds.ClientKeyPath,
		"PGSSLROOTCERT="+creds.RootCertPath,
		"PGSSLCRL=",
	)

	cmd.Cancel = func() error {
		return signalForGracefulCancel(cmd.Process)
	}

	cmd.WaitDelay = killAfterCancel
	setReceivewalProcessAttributes(cmd)

	return cmd, nil
}

func receivewalApplicationName(sourceDB *postgresql_physical.PostgresqlPhysicalDatabase) string {
	if sourceDB.DatabaseID == nil {
		return receivewalApplicationNamePrefix + sourceDB.ID.String()
	}

	return receivewalApplicationNamePrefix + sourceDB.DatabaseID.String()
}

// setReceivewalProcessAttributes makes the kernel send pg_receivewal a SIGTERM
// if the Databasus process dies, so a crashed supervisor never leaks an orphaned
// receiver that keeps holding the replication slot. Pdeathsig is Linux-only;
// Databasus ships Linux containers exclusively.
func setReceivewalProcessAttributes(cmd *exec.Cmd) {
	cmd.SysProcAttr = &syscall.SysProcAttr{Pdeathsig: syscall.SIGTERM}
}
