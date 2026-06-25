package start

import (
	"bufio"
	"context"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"runtime"
	"sync"
	"syscall"
	"time"

	"databasus-verification-agent/internal/config"
	"databasus-verification-agent/internal/features/api"
	"databasus-verification-agent/internal/features/container"
	"databasus-verification-agent/internal/features/heartbeat"
	"databasus-verification-agent/internal/features/restore"
	"databasus-verification-agent/internal/features/runner"
	"databasus-verification-agent/internal/features/upgrade"
	"databasus-verification-agent/internal/features/verifier"
)

func Start(cfg *config.Config, agentVersion string, isDev bool, log *slog.Logger) error {
	if _, err := cfg.Validate(); err != nil {
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
	capacity, err := cfg.Validate()
	if err != nil {
		return err
	}

	if err := cfg.ValidateTransport(isStdinTTY(), bufio.NewReader(os.Stdin)); err != nil {
		return err
	}

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

	apiClient := api.NewClient(cfg.DatabasusHost, cfg.Token, cfg.AgentID, log)

	engine, err := container.NewDockerEngine()
	if err != nil {
		return err
	}

	containerManager := container.NewManager(engine, cfg.AgentID, cfg.GetVerificationPgImageRepo(), log)
	if err := containerManager.StartupSelfCheck(ctx); err != nil {
		return err
	}

	// Start fresh: remove any container leaked by a prior ungraceful death
	// before claiming the first job. Safe to wipe unconditionally because the
	// lock guarantees a single agent owns no live container the instant it
	// starts.
	containerManager.PurgeContainers(ctx)

	pool := runner.NewPool(capacity.MaxConcurrentJobs)
	heartbeater := heartbeat.NewHeartbeater(apiClient, capacity, log)
	verificationRunner := runner.NewRunner(
		apiClient, capacity, pool,
		containerManagerSpawner{containerManager: containerManager},
		restore.NewRestorer(log),
		verifier.NewVerifier(log),
		heartbeater,
		log,
	)

	var backgroundUpgrader *upgrade.BackgroundUpgrader
	if agentVersion != "dev" && !isDev {
		backgroundUpgrader = upgrade.NewBackgroundUpgrader(apiClient, agentVersion, isDev, cancel, log)
		go backgroundUpgrader.Run(ctx)
	}

	log.Info("Agent started")

	var wg sync.WaitGroup
	wg.Go(func() { heartbeater.Run(ctx) })
	wg.Go(func() { verificationRunner.Run(ctx) })

	<-ctx.Done()

	// verificationRunner.Run drains in-flight jobs (pool.Wait) before it
	// returns, so wg.Wait only completes once every container is torn down —
	// strictly before any re-exec on self-update.
	wg.Wait()

	if backgroundUpgrader != nil {
		backgroundUpgrader.WaitForCompletion(30 * time.Second)

		if backgroundUpgrader.IsUpgraded() {
			// The deferred ReleaseLock runs before this returns to the caller,
			// freeing the flock and removing the lock file. The caller then
			// syscall.Exec's the same PID with os.Args unchanged, so the new
			// image re-enters via the same subcommand (`run` or `_run`) and
			// re-acquires the lock cleanly with no contender.
			return upgrade.ErrUpgradeRestart
		}
	}

	log.Info("Agent stopped")

	return nil
}

func isStdinTTY() bool {
	info, err := os.Stdin.Stat()
	if err != nil {
		return false
	}

	return info.Mode()&os.ModeCharDevice != 0
}
