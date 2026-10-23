package upgrade

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"os/exec"
	"runtime"
	"strings"
	"time"

	"databasus-verification-agent/internal/features/api"
)

// verifyBinaryTimeout bounds the freshly-downloaded binary's `version` probe so
// a corrupt or wrong-arch binary that hangs cannot wedge startup or the
// background-upgrade goroutine. A var so tests can shrink it.
var verifyBinaryTimeout = 30 * time.Second

// binaryDownloadTimeout bounds the agent self-download. The streaming HTTP
// client has no client-level timeout (it is shared with the long-lived backup
// stream), so the deadline is scoped per call here.
const binaryDownloadTimeout = 10 * time.Minute

// CheckAndUpdate checks if a new version is available and upgrades the binary on disk.
// Returns (true, nil) if the binary was upgraded, (false, nil) if already up to date,
// or (false, err) on failure. Callers are responsible for re-exec or restart signaling.
func CheckAndUpdate(apiClient *api.Client, currentVersion string, isDev bool, log *slog.Logger) (bool, error) {
	if isDev {
		log.Info("Skipping update check (development mode)")

		return false, nil
	}

	serverVersion, err := apiClient.FetchServerVersion(context.Background())
	if err != nil {
		log.Warn("Could not reach server for update check", "error", err)

		return false, fmt.Errorf(
			"unable to check version, please verify Databasus server is available: %w",
			err,
		)
	}

	if serverVersion == currentVersion {
		log.Info("Agent version is up to date", "version", currentVersion)

		return false, nil
	}

	log.Info("Updating agent...", "current", currentVersion, "target", serverVersion)

	selfPath, err := os.Executable()
	if err != nil {
		return false, fmt.Errorf("failed to determine executable path: %w", err)
	}

	tempPath := selfPath + ".update"

	defer func() {
		_ = os.Remove(tempPath)
	}()

	downloadCtx, cancelDownload := context.WithTimeout(context.Background(), binaryDownloadTimeout)
	defer cancelDownload()

	downloadedVersion, err := apiClient.DownloadVerificationAgentBinary(
		downloadCtx, runtime.GOARCH, tempPath, serverVersion,
	)
	if err != nil {
		return false, fmt.Errorf("failed to download update: %w", err)
	}

	if err := os.Chmod(tempPath, 0o755); err != nil {
		return false, fmt.Errorf("failed to set permissions on update: %w", err)
	}

	if err := verifyBinary(tempPath, downloadedVersion); err != nil {
		return false, fmt.Errorf("update verification failed: %w", err)
	}

	if downloadedVersion == currentVersion {
		log.Info("Agent version is up to date", "version", currentVersion)
		return false, nil
	}

	if err := os.Rename(tempPath, selfPath); err != nil {
		return false, fmt.Errorf("failed to replace binary (try --skip-update if this persists): %w", err)
	}

	log.Info("Agent binary updated", "version", downloadedVersion)

	return true, nil
}

func verifyBinary(binaryPath, expectedVersion string) error {
	ctx, cancel := context.WithTimeout(context.Background(), verifyBinaryTimeout)
	defer cancel()

	// Runs the downloaded agent binary to check it before it replaces the current one.
	// nosemgrep: go.lang.security.audit.dangerous-exec-command.dangerous-exec-command
	cmd := exec.CommandContext(ctx, binaryPath, "version")

	output, err := cmd.Output()
	if err != nil {
		return fmt.Errorf("binary failed to execute: %w", err)
	}

	reportedVersion := strings.TrimSpace(string(output))
	if reportedVersion != expectedVersion {
		return fmt.Errorf("version mismatch: expected %q, got %q", expectedVersion, reportedVersion)
	}

	return nil
}
