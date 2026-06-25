package start

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"os/exec"
	"syscall"
	"time"
)

const (
	logFileName        = "databasus-verification.daemon.log"
	stopTimeout        = 30 * time.Second
	stopPollInterval   = 500 * time.Millisecond
	daemonStartupDelay = 500 * time.Millisecond
)

func Stop(log *slog.Logger) error {
	pid, err := ReadLockFilePID()
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return errors.New("agent is not running (no lock file found)")
		}

		return fmt.Errorf("failed to read lock file: %w", err)
	}

	if !isProcessAlive(pid) {
		_ = os.Remove(lockFileName)
		return fmt.Errorf("agent is not running (stale lock file removed, PID %d)", pid)
	}

	log.Info("Sending SIGTERM to agent", "pid", pid)

	if err := syscall.Kill(pid, syscall.SIGTERM); err != nil {
		return fmt.Errorf("failed to send SIGTERM to PID %d: %w", pid, err)
	}

	deadline := time.Now().Add(stopTimeout)
	for time.Now().Before(deadline) {
		if !isProcessAlive(pid) {
			log.Info("Agent stopped", "pid", pid)
			return nil
		}

		time.Sleep(stopPollInterval)
	}

	return fmt.Errorf("agent (PID %d) did not stop within %s — process may be stuck", pid, stopTimeout)
}

func Status(log *slog.Logger) error {
	pid, err := ReadLockFilePID()
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			fmt.Println("Agent is not running")
			return nil
		}

		return fmt.Errorf("failed to read lock file: %w", err)
	}

	if isProcessAlive(pid) {
		fmt.Printf("Agent is running (PID %d)\n", pid)
	} else {
		fmt.Println("Agent is not running (stale lock file)")
		_ = os.Remove(lockFileName)
	}

	return nil
}

func spawnDaemon(log *slog.Logger) (int, error) {
	execPath, err := os.Executable()
	if err != nil {
		return 0, fmt.Errorf("failed to resolve executable path: %w", err)
	}

	args := []string{"_run"}

	logFile, err := os.OpenFile(logFileName, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
	if err != nil {
		return 0, fmt.Errorf("failed to open log file %s: %w", logFileName, err)
	}

	cwd, err := os.Getwd()
	if err != nil {
		_ = logFile.Close()
		return 0, fmt.Errorf("failed to get working directory: %w", err)
	}

	cmd := exec.CommandContext(context.Background(), execPath, args...)
	cmd.Dir = cwd
	cmd.Stderr = logFile
	cmd.SysProcAttr = &syscall.SysProcAttr{Setsid: true}

	if err := cmd.Start(); err != nil {
		_ = logFile.Close()
		return 0, fmt.Errorf("failed to start daemon process: %w", err)
	}

	pid := cmd.Process.Pid

	// Detach — we don't wait for the child
	_ = logFile.Close()

	time.Sleep(daemonStartupDelay)

	if !isProcessAlive(pid) {
		return 0, fmt.Errorf("daemon process (PID %d) exited immediately — check %s for details", pid, logFileName)
	}

	log.Info("Daemon spawned", "pid", pid, "log", logFileName)

	return pid, nil
}
