package runner

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"slices"
	"strings"
	"sync/atomic"
	"time"

	"github.com/google/uuid"

	"databasus-verification-agent/internal/config"
	"databasus-verification-agent/internal/features/api"
	"databasus-verification-agent/internal/features/dbconn"
	"databasus-verification-agent/internal/features/restore"
	"databasus-verification-agent/internal/features/verifier"
)

const (
	jobName = "verification_runner"

	claimErrorBackoff         = 5 * time.Second
	idleClaimBackoff          = 15 * time.Second
	maxBackupDownloadAttempts = 5
	streamIdleTimeout         = 60 * time.Second
	maxBackupDownloadBackoff  = 32 * time.Second

	// archive layout inside the per-job container's writable volume.
	inContainerArchiveDir = "/tmp"

	maxParallelRestoreJobs = 8
)

var supportedMajors = []string{"12", "13", "14", "15", "16", "17", "18"}

// errAbortNoReport signals the job was aborted (jobCtx cancelled, or the
// stream returned 410) — executeJob returns without POSTing a report.
var errAbortNoReport = errors.New("verification aborted; no report")

// backupDownloadBackoffFn is the download retry delay; tests swap it to avoid real sleeps.
var backupDownloadBackoffFn = backupDownloadBackoff

type SpawnRequest struct {
	PgMajor            string
	CPUPerJob          int
	RAMMbPerJob        int
	VerificationID     uuid.UUID
	TimescaledbVersion string
}

type Runner struct {
	api       APIClient
	capacity  config.Capacity
	pool      *Pool
	spawner   Spawner
	restorer  Restorer
	verifier  StatsCollector
	heartbeat Registrar
	connAlive func(ctx context.Context, conn dbconn.Conn) bool
	hasRun    atomic.Bool
	log       *slog.Logger
}

func NewRunner(
	apiClient APIClient,
	capacity config.Capacity,
	pool *Pool,
	spawner Spawner,
	restorer Restorer,
	statsCollector StatsCollector,
	heartbeat Registrar,
	log *slog.Logger,
) *Runner {
	return &Runner{
		api:       apiClient,
		capacity:  capacity,
		pool:      pool,
		spawner:   spawner,
		restorer:  restorer,
		verifier:  statsCollector,
		heartbeat: heartbeat,
		connAlive: verifier.ProbeConnAlive,
		log:       log,
	}
}

func (r *Runner) Run(ctx context.Context) {
	if r.hasRun.Swap(true) {
		panic(fmt.Sprintf("%T.Run() called multiple times", r))
	}

	logger := r.log.With("job_id", uuid.New(), "job_name", jobName)
	logger.Info("runner loop started")

	for ctx.Err() == nil {
		// Gate claiming on local capacity so we never hold a server-claimed
		// job we cannot start immediately.
		if r.pool.Saturated() {
			if !sleepOrDone(ctx, 1*time.Second) {
				break
			}

			continue
		}

		capacity := api.AgentCapacity{
			MaxCPU:            r.capacity.MaxCPU,
			MaxRAMMb:          r.capacity.MaxRAMMb,
			MaxDiskGb:         r.capacity.MaxDiskGb,
			MaxConcurrentJobs: r.capacity.MaxConcurrentJobs,
		}

		assignment, err := r.api.ClaimVerification(ctx, capacity)
		if err != nil {
			logger.Warn("claim failed", "error", err)

			if !sleepOrDone(ctx, claimErrorBackoff) {
				break
			}

			continue
		}

		if assignment == nil {
			if !sleepOrDone(ctx, idleClaimBackoff) {
				break
			}

			continue
		}

		r.pool.Go(func() { r.executeJob(ctx, assignment) })
	}

	logger.Info("runner loop draining in-flight jobs")
	r.pool.Wait()
	logger.Info("runner loop stopped")
}

func (r *Runner) executeJob(ctx context.Context, job *api.JobAssignment) {
	runLogger := r.log.With(
		"job_id", uuid.New(),
		"job_name", jobName,
		"verification_id", job.VerificationID,
	)

	jobCtx, cancel := context.WithCancel(ctx)
	defer cancel()

	// Register FIRST so the ID is in the heartbeat envelope before any
	// container exists: the backend must never see an artifact-bearing job
	// that is absent from this agent's reported in-flight set.
	r.heartbeat.TrackVerification(job.VerificationID, cancel)
	defer r.heartbeat.UntrackVerification(job.VerificationID)

	pgMajor, err := pgMajorFromDatabase(job.Database)
	if err != nil {
		r.reportFailure(ctx, job.VerificationID, nil, fmt.Sprintf("resolve pg major: %v", err), runLogger)

		return
	}

	jobContainer, err := r.spawner.Spawn(jobCtx, SpawnRequest{
		PgMajor:            pgMajor,
		CPUPerJob:          r.capacity.CPUPerJob,
		RAMMbPerJob:        r.capacity.RAMMbPerJob,
		VerificationID:     job.VerificationID,
		TimescaledbVersion: job.TimescaledbVersion,
	})
	if err != nil {
		if jobCtx.Err() != nil {
			return
		}

		r.reportFailure(ctx, job.VerificationID, nil, fmt.Sprintf("spawn container: %v", err), runLogger)

		return
	}

	defer func() {
		if termErr := jobContainer.Terminate(ctx); termErr != nil {
			runLogger.Error("failed to terminate container", "error", termErr)
		}
	}()

	// Server-authoritative per-job disk budget (downloaded archive + restored
	// DB + safe gap). The agent enforces it by watching the container's written
	// bytes and aborting the instant the budget is reached — uniform on every
	// storage driver, unlike a Docker --storage-opt cap.
	diskBudgetMb := int64(job.MaxContainerDiskMb)
	diskBudgetFailMessage := fmt.Sprintf("restore exceeded the per-job disk budget of %d MB", diskBudgetMb)

	var isDiskLimitHit atomic.Bool

	// A zero/absent budget (older backend) means no enforceable ceiling — skip
	// the watcher rather than trip instantly on used >= 0.
	if diskBudgetMb > 0 {
		watchCtx, watchStop := context.WithCancel(jobCtx)
		defer watchStop()

		go newDiskWatcher(
			jobContainer, diskBudgetMb*1024*1024,
			func() { isDiskLimitHit.Store(true); cancel() },
			runLogger,
		).run(watchCtx)
	} else {
		runLogger.Warn("no per-job disk budget from server; disk watcher disabled for this job")
	}

	archivePath := fmt.Sprintf("%s/%s.dump", inContainerArchiveDir, job.VerificationID)

	if err := r.downloadBackupIntoContainer(
		jobCtx,
		jobContainer,
		job.VerificationID,
		archivePath,
		runLogger,
	); err != nil {
		if isDiskLimitHit.Load() {
			r.reportDiskLimitExceeded(ctx, job.VerificationID, diskBudgetFailMessage, runLogger)

			return
		}

		if errors.Is(err, errAbortNoReport) {
			return
		}

		r.reportFailure(ctx, job.VerificationID, nil, fmt.Sprintf("download backup: %v", err), runLogger)

		return
	}

	parallelJobs := min(maxParallelRestoreJobs, r.capacity.CPUPerJob)

	// TimescaleDB restores single-threaded and in restoring mode. Single-threaded because parallel
	// pg_restore loads dependent _timescaledb_catalog rows before the rows they reference, tripping
	// the catalog's own foreign keys. post_restore runs on the success path below, before stats; on
	// failure the throwaway container is discarded, so nothing is left in restoring mode to repair
	// (no defer, unlike the backend's user-facing restore). The hooks use the verifier (host) conn
	// while pg_restore uses the in-container conn, but the database-level restoring GUC carries across.
	if job.TimescaledbVersion != "" {
		parallelJobs = 1

		if err := r.restorer.RunTimescalePreRestore(jobCtx, jobContainer.GetVerifierConn()); err != nil {
			if jobCtx.Err() != nil {
				return
			}

			r.reportFailure(ctx, job.VerificationID, nil,
				fmt.Sprintf("timescaledb_pre_restore: %v", err), runLogger)

			return
		}
	}

	restoreResult, err := r.restorer.RunPgRestore(
		jobCtx, jobContainer, archivePath, jobContainer.GetInContainerConn(), parallelJobs)
	if err != nil {
		if isDiskLimitHit.Load() {
			r.reportDiskLimitExceeded(ctx, job.VerificationID, diskBudgetFailMessage, runLogger)

			return
		}

		if jobCtx.Err() != nil {
			return
		}

		if !errors.Is(err, restore.ErrRestoreFailed) {
			// Exec-infrastructure failure (no usable exit code).
			r.reportFailure(ctx, job.VerificationID, nil,
				fmt.Sprintf("pg_restore exec: %v", err), runLogger)

			return
		}

		if restore.IsDiskExhausted(restoreResult.StderrTail) {
			// Hitting the estimate-derived ceiling is agent infra, not proof the
			// backup is corrupt — omit the exit code so the backend retries
			// (AgentSetupFailed), never BackupRejected.
			r.reportFailure(ctx, job.VerificationID, nil,
				fmt.Sprintf("pg_restore hit job disk ceiling: %v", err), runLogger)

			return
		}

		if !restore.IsMissingExtensionOnly(restoreResult.StderrTail) {
			runLogger.Error("pg_restore failed",
				"exit_code", restoreResult.PgRestoreExitCode,
				"stderr_tail", restoreResult.StderrTail)

			r.reportFailure(ctx, job.VerificationID, &restoreResult,
				fmt.Sprintf("pg_restore failed: %v", err), runLogger)

			return
		}

		// The only items pg_restore skipped are CREATE/COMMENT EXTENSION for
		// extensions this verification environment lacks — the data restored.
		// Treat the backup as restorable and fall through to stats; the backend's
		// restored-size check is the remaining safety net.
		runLogger.Warn(
			"pg_restore completed with only missing-extension errors; treating backup as restorable",
			"exit_code", restoreResult.PgRestoreExitCode,
			"skipped_extensions",
			strings.Join(restore.ExtractUnavailableExtensions(restoreResult.StderrTail), ","),
			"stderr_tail", restoreResult.StderrTail)
	}

	if job.TimescaledbVersion != "" {
		if err := r.restorer.RunTimescalePostRestore(jobCtx, jobContainer.GetVerifierConn()); err != nil {
			if jobCtx.Err() != nil {
				return
			}

			r.reportFailure(ctx, job.VerificationID, &restoreResult,
				fmt.Sprintf("timescaledb_post_restore: %v", err), runLogger)

			return
		}
	}

	verifierConn := jobContainer.GetVerifierConn()

	stats, err := r.verifier.CollectStats(jobCtx, verifierConn)
	if err != nil {
		if isDiskLimitHit.Load() {
			r.reportDiskLimitExceeded(ctx, job.VerificationID, diskBudgetFailMessage, runLogger)

			return
		}

		if jobCtx.Err() != nil {
			return
		}

		if !r.connAlive(jobCtx, verifierConn) {
			r.reportFailure(ctx, job.VerificationID, nil,
				fmt.Sprintf("verify (conn dead): %v", err), runLogger)

			return
		}

		r.reportFailure(ctx, job.VerificationID, &restoreResult,
			fmt.Sprintf("verify: %v", err), runLogger)

		return
	}

	if isDiskLimitHit.Load() {
		r.reportDiskLimitExceeded(ctx, job.VerificationID, diskBudgetFailMessage, runLogger)

		return
	}

	r.reportSuccess(ctx, job.VerificationID, restoreResult, stats, runLogger)
}

func (r *Runner) downloadBackupIntoContainer(
	jobCtx context.Context,
	jobContainer JobContainer,
	verificationID uuid.UUID,
	archivePath string,
	logger *slog.Logger,
) error {
	var lastErr error

	for attempt := 1; attempt <= maxBackupDownloadAttempts; attempt++ {
		if jobCtx.Err() != nil {
			return errAbortNoReport
		}

		err := r.downloadBackupIntoContainerOnce(jobCtx, jobContainer, verificationID, archivePath)
		if err == nil {
			return nil
		}

		var respErr *api.ResponseError
		if errors.As(err, &respErr) {
			if respErr.IsGone() {
				logger.Info("backup stream gone (410); dropping without report")

				return errAbortNoReport
			}

			if !respErr.Retryable() {
				return fmt.Errorf("backup stream non-retryable: %w", err)
			}
		}

		lastErr = err
		logger.Warn("backup stream attempt failed",
			"attempt", attempt, "max_attempts", maxBackupDownloadAttempts, "error", err)

		if !sleepOrDone(jobCtx, backupDownloadBackoffFn(attempt)) {
			return errAbortNoReport
		}
	}

	return fmt.Errorf("backup stream failed after %d attempts: %w", maxBackupDownloadAttempts, lastErr)
}

func (r *Runner) downloadBackupIntoContainerOnce(
	jobCtx context.Context,
	jobContainer JobContainer,
	verificationID uuid.UUID,
	archivePath string,
) error {
	body, err := r.api.DownloadBackup(jobCtx, verificationID)
	if err != nil {
		return err
	}
	defer func() { _ = body.Close() }()

	dlCtx, dlCancel := context.WithCancelCause(jobCtx)
	defer dlCancel(nil)

	idleReader := api.NewIdleTimeoutReader(body, streamIdleTimeout, dlCancel)
	defer idleReader.Stop()

	return r.restorer.StageBackupViaExec(dlCtx, jobContainer, idleReader, archivePath)
}

func (r *Runner) reportSuccess(
	ctx context.Context,
	verificationID uuid.UUID,
	restoreResult restore.Result,
	stats verifier.Stats,
	logger *slog.Logger,
) {
	tableStats := make([]api.ReportTableStat, 0, len(stats.TableStats))
	for _, ts := range stats.TableStats {
		tableStats = append(tableStats, api.ReportTableStat{
			SchemaName: ts.SchemaName,
			Name:       ts.Name,
			RowCount:   ts.RowCount,
		})
	}

	req := api.ReportRequest{
		Status:                  api.VerificationStatusCompleted,
		PgRestoreExitCode:       &restoreResult.PgRestoreExitCode,
		RestoreDurationMs:       &restoreResult.DurationMs,
		DBSizeBytesAfterRestore: &stats.DBSizeBytes,
		TableCount:              &stats.TableCount,
		SchemaCount:             &stats.SchemaCount,
		TableStats:              tableStats,
	}

	r.sendReport(ctx, verificationID, req, logger)
}

func (r *Runner) reportFailure(
	ctx context.Context,
	verificationID uuid.UUID,
	restoreResult *restore.Result,
	failMessage string,
	logger *slog.Logger,
) {
	if r.shouldSkipFailedReport(ctx, verificationID, logger) {
		return
	}

	message := failMessage
	req := api.ReportRequest{
		Status:      api.VerificationStatusFailed,
		FailMessage: &message,
	}

	if restoreResult != nil {
		req.PgRestoreExitCode = &restoreResult.PgRestoreExitCode
		req.RestoreDurationMs = &restoreResult.DurationMs
	}

	r.sendReport(ctx, verificationID, req, logger)
}

// reportDiskLimitExceeded posts the terminal disk-budget verdict. FailureKind
// makes the backend classify it DISK_LIMIT_EXCEEDED (terminal) instead of the
// retryable nil-exit-code path: the budget is server-computed, so retrying the
// same job against the same budget would fail identically.
func (r *Runner) reportDiskLimitExceeded(
	ctx context.Context,
	verificationID uuid.UUID,
	failMessage string,
	logger *slog.Logger,
) {
	if r.shouldSkipFailedReport(ctx, verificationID, logger) {
		return
	}

	message := failMessage
	kind := api.FailureKindDiskLimitExceeded
	r.sendReport(ctx, verificationID, api.ReportRequest{
		Status:      api.VerificationStatusFailed,
		FailMessage: &message,
		FailureKind: &kind,
	}, logger)
}

// shouldSkipFailedReport is the abort/report-race guard (the only agent-side
// lever with a frozen FAILED path): never POST FAILED for an aborted/cancelled
// ID — a spurious FAILED would flip CANCELED→FAILED or resurrect a cancelled
// row.
func (r *Runner) shouldSkipFailedReport(
	ctx context.Context, verificationID uuid.UUID, logger *slog.Logger,
) bool {
	if ctx.Err() != nil || r.heartbeat.IsAborted(verificationID) {
		logger.Info("skipping FAILED report: verification aborted/cancelled")

		return true
	}

	return false
}

func (r *Runner) sendReport(
	ctx context.Context,
	verificationID uuid.UUID,
	req api.ReportRequest,
	logger *slog.Logger,
) {
	err := r.api.Report(ctx, verificationID, req)
	if err == nil {
		logger.Info(fmt.Sprintf("report accepted: status=%s", req.Status))

		return
	}

	switch {
	case errors.Is(err, api.ErrReportGone):
		logger.Info("report dropped: verification no longer owned by this agent (410)")
	case errors.Is(err, api.ErrReportBudgetExhausted):
		logger.Warn("report abandoned: retry budget exhausted; backend will reclaim on next heartbeat")
	default:
		logger.Warn("report failed", "error", err)
	}
}

func pgMajorFromDatabase(db api.AssignedDatabase) (string, error) {
	if db.Type != "POSTGRES_LOGICAL" {
		return "", fmt.Errorf("unsupported database type %q (v1 is Postgres only)", db.Type)
	}

	if db.PostgresqlLogical == nil || db.PostgresqlLogical.Version == "" {
		return "", errors.New("assignment missing postgresql version")
	}

	if !slices.Contains(supportedMajors, db.PostgresqlLogical.Version) {
		return "", fmt.Errorf("unsupported postgres major %q", db.PostgresqlLogical.Version)
	}

	return db.PostgresqlLogical.Version, nil
}

func backupDownloadBackoff(attempt int) time.Duration {
	if attempt < 1 {
		attempt = 1
	}

	d := time.Duration(1<<attempt) * time.Second
	if d > maxBackupDownloadBackoff {
		return maxBackupDownloadBackoff
	}

	return d
}

func sleepOrDone(ctx context.Context, d time.Duration) bool {
	timer := time.NewTimer(d)
	defer timer.Stop()

	select {
	case <-ctx.Done():
		return false
	case <-timer.C:
		return true
	}
}
