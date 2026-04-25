package start

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/jackc/pgx/v5"

	"databasus-agent/internal/config"
	"databasus-agent/internal/features/api"
	full_backup "databasus-agent/internal/features/full_backup"
	"databasus-agent/internal/features/upgrade"
	"databasus-agent/internal/features/wal"
)

const (
	pgBasebackupVerifyTimeout = 10 * time.Second
	dbVerifyTimeout           = 10 * time.Second
	minPgMajorVersion         = 15
)

func Start(cfg *config.Config, agentVersion string, isDev bool, log *slog.Logger) error {
	if err := validateConfig(cfg); err != nil {
		return err
	}

	if err := verifyPgBasebackup(cfg, log); err != nil {
		return err
	}

	if err := verifyDatabase(cfg, log); err != nil {
		return err
	}

	if runtime.GOOS == "windows" {
		return RunDaemon(cfg, agentVersion, isDev, log)
	}

	pid, err := spawnDaemon(log)
	if err != nil {
		return err
	}

	fmt.Printf("Agent started in background (PID %d)\n", pid)

	return nil
}

func RunDaemon(cfg *config.Config, agentVersion string, isDev bool, log *slog.Logger) error {
	lockFile, err := AcquireLock(log)
	if err != nil {
		return err
	}
	defer ReleaseLock(lockFile)

	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()

	watcher, err := NewLockWatcher(lockFile, cancel, log)
	if err != nil {
		return fmt.Errorf("failed to initialize lock watcher: %w", err)
	}
	go watcher.Run(ctx)

	apiClient := api.NewClient(cfg.DatabasusHost, cfg.Token, cfg.InsecureSkipVerify, log)

	var backgroundUpgrader *upgrade.BackgroundUpgrader
	if agentVersion != "dev" && runtime.GOOS != "windows" {
		backgroundUpgrader = upgrade.NewBackgroundUpgrader(apiClient, agentVersion, isDev, cancel, log)
		go backgroundUpgrader.Run(ctx)
	}

	fullBackuper := full_backup.NewFullBackuper(cfg, apiClient, log)
	go fullBackuper.Run(ctx)

	streamer := wal.NewStreamer(cfg, apiClient, log)
	streamer.Run(ctx)

	if backgroundUpgrader != nil {
		backgroundUpgrader.WaitForCompletion(30 * time.Second)

		if backgroundUpgrader.IsUpgraded() {
			return upgrade.ErrUpgradeRestart
		}
	}

	log.Info("Agent stopped")

	return nil
}

func validateConfig(cfg *config.Config) error {
	if cfg.DatabasusHost == "" {
		return errors.New("argument databasus-host is required")
	}

	if cfg.DbID == "" {
		return errors.New("argument db-id is required")
	}

	if cfg.Token == "" {
		return errors.New("argument token is required")
	}

	if cfg.PgHost == "" {
		return errors.New("argument pg-host is required")
	}

	if cfg.PgPort <= 0 {
		return errors.New("argument pg-port must be a positive number")
	}

	if cfg.PgUser == "" {
		return errors.New("argument pg-user is required")
	}

	if cfg.PgType != "host" && cfg.PgType != "docker" {
		return fmt.Errorf("argument pg-type must be 'host' or 'docker', got '%s'", cfg.PgType)
	}

	if cfg.PgWalDir == "" {
		return errors.New("argument pg-wal-dir is required")
	}

	if cfg.PgType == "docker" && cfg.PgDockerContainerName == "" {
		return errors.New("argument pg-docker-container-name is required when pg-type is 'docker'")
	}

	return nil
}

func verifyPgBasebackup(cfg *config.Config, log *slog.Logger) error {
	switch cfg.PgType {
	case "host":
		return verifyPgBasebackupHost(cfg, log)
	case "docker":
		return verifyPgBasebackupDocker(cfg, log)
	default:
		return fmt.Errorf("unexpected pg-type: %s", cfg.PgType)
	}
}

func verifyPgBasebackupHost(cfg *config.Config, log *slog.Logger) error {
	binary := "pg_basebackup"
	if cfg.PgHostBinDir != "" {
		binary = filepath.Join(cfg.PgHostBinDir, "pg_basebackup")
	}

	ctx, cancel := context.WithTimeout(context.Background(), pgBasebackupVerifyTimeout)
	defer cancel()

	output, err := exec.CommandContext(ctx, binary, "--version").CombinedOutput()
	if err != nil {
		if cfg.PgHostBinDir != "" {
			return fmt.Errorf(
				"pg_basebackup not found at '%s': %w. Verify pg-host-bin-dir is correct",
				binary, err,
			)
		}

		return fmt.Errorf(
			"pg_basebackup not found in PATH: %w. Install PostgreSQL client tools or set pg-host-bin-dir",
			err,
		)
	}

	log.Info("pg_basebackup verified", "version", strings.TrimSpace(string(output)))

	return nil
}

func verifyPgBasebackupDocker(cfg *config.Config, log *slog.Logger) error {
	ctx, cancel := context.WithTimeout(context.Background(), pgBasebackupVerifyTimeout)
	defer cancel()

	output, err := exec.CommandContext(ctx,
		"docker", "exec", cfg.PgDockerContainerName,
		"pg_basebackup", "--version",
	).CombinedOutput()
	if err != nil {
		return fmt.Errorf(
			"pg_basebackup not available in container '%s': %w. "+
				"Check that the container is running and pg_basebackup is installed inside it",
			cfg.PgDockerContainerName, err,
		)
	}

	log.Info("pg_basebackup verified (docker)",
		"container", cfg.PgDockerContainerName,
		"version", strings.TrimSpace(string(output)),
	)

	return nil
}

func verifyDatabase(cfg *config.Config, log *slog.Logger) error {
	switch cfg.PgType {
	case "docker":
		return verifyDatabaseDocker(cfg, log)
	default:
		return verifyDatabaseHost(cfg, log)
	}
}

func verifyDatabaseHost(cfg *config.Config, log *slog.Logger) error {
	connStr := fmt.Sprintf(
		"host=%s port=%d user=%s password=%s dbname=postgres sslmode=disable",
		cfg.PgHost, cfg.PgPort, cfg.PgUser, cfg.PgPassword,
	)

	ctx, cancel := context.WithTimeout(context.Background(), dbVerifyTimeout)
	defer cancel()

	conn, err := pgx.Connect(ctx, connStr)
	if err != nil {
		return fmt.Errorf(
			"failed to connect to PostgreSQL at %s:%d as user '%s': %w",
			cfg.PgHost, cfg.PgPort, cfg.PgUser, err,
		)
	}
	defer func() { _ = conn.Close(ctx) }()

	if err := conn.Ping(ctx); err != nil {
		return fmt.Errorf("PostgreSQL ping failed at %s:%d: %w",
			cfg.PgHost, cfg.PgPort, err,
		)
	}

	var versionNumStr string
	if err := conn.QueryRow(ctx, "SHOW server_version_num").Scan(&versionNumStr); err != nil {
		return fmt.Errorf("failed to query PostgreSQL version: %w", err)
	}

	majorVersion, err := parsePgVersionNum(versionNumStr)
	if err != nil {
		return fmt.Errorf("failed to parse PostgreSQL version '%s': %w", versionNumStr, err)
	}

	if majorVersion < minPgMajorVersion {
		return fmt.Errorf(
			"PostgreSQL %d is not supported, minimum required version is %d",
			majorVersion, minPgMajorVersion,
		)
	}

	log.Info("PostgreSQL connection verified",
		"host", cfg.PgHost,
		"port", cfg.PgPort,
		"user", cfg.PgUser,
		"version", majorVersion,
	)

	return nil
}

func verifyDatabaseDocker(cfg *config.Config, log *slog.Logger) error {
	ctx, cancel := context.WithTimeout(context.Background(), dbVerifyTimeout)
	defer cancel()

	query := "SELECT current_setting('server_version_num')"

	cmd := exec.CommandContext(ctx,
		"docker", "exec",
		"-e", "PGPASSWORD="+cfg.PgPassword,
		cfg.PgDockerContainerName,
		"psql", "-h", "localhost", "-p", "5432", "-U", cfg.PgUser,
		"-d", "postgres", "-t", "-A", "-c", query,
	)

	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf(
			"failed to connect to PostgreSQL in container '%s' as user '%s': %w (output: %s)",
			cfg.PgDockerContainerName, cfg.PgUser, err, strings.TrimSpace(string(output)),
		)
	}

	versionNumStr := strings.TrimSpace(string(output))

	majorVersion, err := parsePgVersionNum(versionNumStr)
	if err != nil {
		return fmt.Errorf("failed to parse PostgreSQL version '%s': %w", versionNumStr, err)
	}

	if majorVersion < minPgMajorVersion {
		return fmt.Errorf(
			"PostgreSQL %d is not supported, minimum required version is %d",
			majorVersion, minPgMajorVersion,
		)
	}

	log.Info("PostgreSQL connection verified (docker)",
		"container", cfg.PgDockerContainerName,
		"user", cfg.PgUser,
		"version", majorVersion,
	)

	return nil
}

func parsePgVersionNum(versionNumStr string) (int, error) {
	versionNum, err := strconv.Atoi(strings.TrimSpace(versionNumStr))
	if err != nil {
		return 0, fmt.Errorf("invalid version number: %w", err)
	}

	if versionNum <= 0 {
		return 0, fmt.Errorf("invalid version number: %d", versionNum)
	}

	majorVersion := versionNum / 10000

	return majorVersion, nil
}
