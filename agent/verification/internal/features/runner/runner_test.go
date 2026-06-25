package runner

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"databasus-verification-agent/internal/config"
	"databasus-verification-agent/internal/features/api"
	"databasus-verification-agent/internal/features/dbconn"
	"databasus-verification-agent/internal/features/restore"
	"databasus-verification-agent/internal/features/verifier"
	"databasus-verification-agent/internal/testutil"
)

func testCapacity() config.Capacity {
	return config.Capacity{
		MaxCPU: 4, MaxRAMMb: 4096, MaxDiskGb: 50, MaxConcurrentJobs: 2,
		CPUPerJob: 2, RAMMbPerJob: 2048,
	}
}

func postgresJob() *api.JobAssignment {
	return &api.JobAssignment{
		VerificationID:     uuid.New(),
		BackupID:           uuid.New(),
		BackupSizeMb:       50,
		MaxContainerDiskMb: 200,
		Database: api.AssignedDatabase{
			Type:              "POSTGRES_LOGICAL",
			PostgresqlLogical: &api.AssignedPostgresql{Version: "16"},
		},
	}
}

func newTestRunner(
	apiClient APIClient, spawner Spawner, restorer Restorer,
	stats StatsCollector, reg Registrar,
) *Runner {
	return NewRunner(
		apiClient, testCapacity(), NewPool(2),
		spawner, restorer, stats, reg, testutil.DiscardLogger(),
	)
}

func testConn() dbconn.Conn {
	return dbconn.Conn{
		Host: "127.0.0.1", Port: 5432,
		User: "postgres", Password: "pw", Database: "postgres",
	}
}

func fakeContainerWith(conn dbconn.Conn) *fakeContainer {
	return &fakeContainer{inContainerConn: conn, verifierConn: conn}
}

func okSpawner() *fakeSpawner {
	return &fakeSpawner{container: fakeContainerWith(testConn())}
}

func Test_ExecuteJob_WhenPgMajorUnsupported_ReportsFailedWithoutExitCode(t *testing.T) {
	apiClient := &fakeAPI{}
	r := newTestRunner(apiClient, okSpawner(), &fakeRestorer{}, &fakeStats{}, newFakeRegistrar())

	job := postgresJob()
	job.Database.Type = "MYSQL"

	r.executeJob(t.Context(), job)

	report, ok := apiClient.lastReport()
	require.True(t, ok)
	assert.Equal(t, api.VerificationStatusFailed, report.Status)
	assert.Nil(t, report.PgRestoreExitCode)
}

func Test_ExecuteJob_WhenSpawnFails_ReportsFailedWithoutExitCode(t *testing.T) {
	apiClient := &fakeAPI{}
	spawner := &fakeSpawner{err: errors.New("image pull failed")}
	r := newTestRunner(apiClient, spawner, &fakeRestorer{}, &fakeStats{}, newFakeRegistrar())

	r.executeJob(t.Context(), postgresJob())

	report, ok := apiClient.lastReport()
	require.True(t, ok)
	assert.Equal(t, api.VerificationStatusFailed, report.Status)
	assert.Nil(t, report.PgRestoreExitCode)
}

func Test_ExecuteJob_WhenStreamReturns410_DoesNotReport(t *testing.T) {
	apiClient := &fakeAPI{downloadErr: &api.ResponseError{Op: "backup stream", StatusCode: 410}}
	r := newTestRunner(apiClient, okSpawner(), &fakeRestorer{}, &fakeStats{}, newFakeRegistrar())

	r.executeJob(t.Context(), postgresJob())

	assert.Equal(t, 0, apiClient.reportCalled, "a 410 on the stream is an abort — no report")
}

func Test_ExecuteJob_WhenStreamExhaustsRetries_ReportsFailedWithoutExitCode(t *testing.T) {
	original := backupDownloadBackoffFn
	backupDownloadBackoffFn = func(int) time.Duration { return 0 }
	t.Cleanup(func() { backupDownloadBackoffFn = original })

	apiClient := &fakeAPI{downloadErr: errors.New("connection refused")}
	r := newTestRunner(apiClient, okSpawner(), &fakeRestorer{}, &fakeStats{}, newFakeRegistrar())

	r.executeJob(t.Context(), postgresJob())

	report, ok := apiClient.lastReport()
	require.True(t, ok)
	assert.Equal(t, api.VerificationStatusFailed, report.Status)
	assert.Nil(t, report.PgRestoreExitCode)
}

func Test_ExecuteJob_WhenRestoreFailsAndDiskExhausted_ReportsFailedWithoutExitCode(t *testing.T) {
	apiClient := &fakeAPI{downloadBody: []byte("ARCHIVE")}
	restorer := &fakeRestorer{
		runErr:    fmt.Errorf("restore: %w", restore.ErrRestoreFailed),
		runResult: restore.Result{PgRestoreExitCode: 1, StderrTail: "could not write: No space left on device"},
	}
	r := newTestRunner(apiClient, okSpawner(), restorer, &fakeStats{}, newFakeRegistrar())

	r.executeJob(t.Context(), postgresJob())

	report, ok := apiClient.lastReport()
	require.True(t, ok)
	assert.Equal(t, api.VerificationStatusFailed, report.Status)
	assert.Nil(t, report.PgRestoreExitCode,
		"hitting the estimate-derived disk ceiling is agent infra, not BackupRejected")
}

func Test_ExecuteJob_WhenDiskWatcherTrips_ReportsTerminalDiskLimitExceeded(t *testing.T) {
	originalInterval := diskWatchInterval
	diskWatchInterval = time.Millisecond
	t.Cleanup(func() { diskWatchInterval = originalInterval })

	originalBackoff := backupDownloadBackoffFn
	backupDownloadBackoffFn = func(int) time.Duration { return 0 }
	t.Cleanup(func() { backupDownloadBackoffFn = originalBackoff })

	apiClient := &fakeAPI{downloadBody: []byte("ARCHIVE")}
	jobContainer := fakeContainerWith(testConn())
	jobContainer.diskUsageBytes = 1 << 40
	restorer := &fakeRestorer{runBlocks: true}

	r := newTestRunner(
		apiClient, &fakeSpawner{container: jobContainer}, restorer, &fakeStats{}, newFakeRegistrar())

	r.executeJob(t.Context(), postgresJob())

	report, ok := apiClient.lastReport()
	require.True(t, ok)
	assert.Equal(t, api.VerificationStatusFailed, report.Status)
	require.NotNil(t, report.FailureKind)
	assert.Equal(t, api.FailureKindDiskLimitExceeded, *report.FailureKind)
	assert.Nil(t, report.PgRestoreExitCode, "a disk-limit verdict carries no pg_restore exit code")
	assert.True(t, jobContainer.terminated, "the container must be torn down on disk-limit abort")
}

func Test_ExecuteJob_WhenServerSendsNoDiskBudget_DoesNotFalselyTrip(t *testing.T) {
	apiClient := &fakeAPI{downloadBody: []byte("ARCHIVE")}
	jobContainer := fakeContainerWith(testConn())
	jobContainer.diskUsageBytes = 1 << 40

	job := postgresJob()
	job.MaxContainerDiskMb = 0

	r := newTestRunner(
		apiClient, &fakeSpawner{container: jobContainer}, &fakeRestorer{}, &fakeStats{}, newFakeRegistrar())

	r.executeJob(t.Context(), job)

	report, ok := apiClient.lastReport()
	require.True(t, ok)
	assert.Equal(t, api.VerificationStatusCompleted, report.Status,
		"a zero budget disables the watcher; the job must not be falsely failed")
	assert.Nil(t, report.FailureKind)
}

func Test_ExecuteJob_WhenRestoreFailsNormally_ReportsFailedWithExitCode(t *testing.T) {
	apiClient := &fakeAPI{downloadBody: []byte("ARCHIVE")}
	restorer := &fakeRestorer{
		runErr:    fmt.Errorf("restore: %w", restore.ErrRestoreFailed),
		runResult: restore.Result{PgRestoreExitCode: 1, DurationMs: 42, StderrTail: "syntax error in dump"},
	}
	r := newTestRunner(apiClient, okSpawner(), restorer, &fakeStats{}, newFakeRegistrar())

	r.executeJob(t.Context(), postgresJob())

	report, ok := apiClient.lastReport()
	require.True(t, ok)
	assert.Equal(t, api.VerificationStatusFailed, report.Status)
	require.NotNil(t, report.PgRestoreExitCode)
	assert.Equal(t, 1, *report.PgRestoreExitCode)
}

func Test_ExecuteJob_WhenRestoreFailsWithOnlyMissingExtensions_ReportsCompleted(t *testing.T) {
	apiClient := &fakeAPI{downloadBody: []byte("ARCHIVE")}
	stderr := "pg_restore: error: could not execute query: ERROR:  extension \"set_user\" is not available\n" +
		"Command was: CREATE EXTENSION IF NOT EXISTS set_user WITH SCHEMA public;\n" +
		"pg_restore: error: could not execute query: ERROR:  extension \"set_user\" does not exist\n" +
		"Command was: COMMENT ON EXTENSION set_user IS 'x';\n" +
		"pg_restore: warning: errors ignored on restore: 2\n"
	restorer := &fakeRestorer{
		runErr:    fmt.Errorf("restore: %w", restore.ErrRestoreFailed),
		runResult: restore.Result{PgRestoreExitCode: 1, DurationMs: 10, StderrTail: stderr},
	}
	stats := &fakeStats{stats: verifier.Stats{DBSizeBytes: 9_000_000, TableCount: 3}}
	r := newTestRunner(apiClient, okSpawner(), restorer, stats, newFakeRegistrar())
	r.connAlive = func(context.Context, dbconn.Conn) bool { return true }

	r.executeJob(t.Context(), postgresJob())

	report, ok := apiClient.lastReport()
	require.True(t, ok)
	assert.Equal(t, api.VerificationStatusCompleted, report.Status,
		"a restore whose only failures are missing extensions is restorable")
	require.NotNil(t, report.PgRestoreExitCode)
	assert.Equal(t, 1, *report.PgRestoreExitCode,
		"the real non-zero exit code is preserved on the success report")
	require.NotNil(t, report.DBSizeBytesAfterRestore)
	assert.Equal(t, int64(9_000_000), *report.DBSizeBytesAfterRestore)
}

func Test_ExecuteJob_WhenRestoreExecInfraFails_ReportsFailedWithoutExitCode(t *testing.T) {
	apiClient := &fakeAPI{downloadBody: []byte("ARCHIVE")}
	restorer := &fakeRestorer{runErr: errors.New("exec create failed")}
	r := newTestRunner(apiClient, okSpawner(), restorer, &fakeStats{}, newFakeRegistrar())

	r.executeJob(t.Context(), postgresJob())

	report, ok := apiClient.lastReport()
	require.True(t, ok)
	assert.Equal(t, api.VerificationStatusFailed, report.Status)
	assert.Nil(t, report.PgRestoreExitCode)
}

func Test_ExecuteJob_WhenVerifyFailsAndConnDead_ReportsFailedWithoutExitCode(t *testing.T) {
	apiClient := &fakeAPI{downloadBody: []byte("ARCHIVE")}
	restorer := &fakeRestorer{runResult: restore.Result{PgRestoreExitCode: 0}}
	stats := &fakeStats{err: errors.New("tier 1 failed")}
	r := newTestRunner(apiClient, okSpawner(), restorer, stats, newFakeRegistrar())
	r.connAlive = func(context.Context, dbconn.Conn) bool { return false }

	r.executeJob(t.Context(), postgresJob())

	report, ok := apiClient.lastReport()
	require.True(t, ok)
	assert.Equal(t, api.VerificationStatusFailed, report.Status)
	assert.Nil(t, report.PgRestoreExitCode, "a dead connection is agent infra (retryable)")
}

func Test_ExecuteJob_WhenVerifyFailsAndConnAlive_ReportsFailedWithExitCodeZero(t *testing.T) {
	apiClient := &fakeAPI{downloadBody: []byte("ARCHIVE")}
	restorer := &fakeRestorer{runResult: restore.Result{PgRestoreExitCode: 0}}
	stats := &fakeStats{err: errors.New("tier 1 impossible result")}
	r := newTestRunner(apiClient, okSpawner(), restorer, stats, newFakeRegistrar())
	r.connAlive = func(context.Context, dbconn.Conn) bool { return true }

	r.executeJob(t.Context(), postgresJob())

	report, ok := apiClient.lastReport()
	require.True(t, ok)
	assert.Equal(t, api.VerificationStatusFailed, report.Status)
	require.NotNil(t, report.PgRestoreExitCode)
	assert.Equal(t, 0, *report.PgRestoreExitCode,
		"a live conn with a broken tier-1 result is BackupRejected with exit 0")
}

func Test_ExecuteJob_WhenAllSucceeds_ReportsCompletedWithStats(t *testing.T) {
	apiClient := &fakeAPI{downloadBody: []byte("ARCHIVE")}
	restorer := &fakeRestorer{runResult: restore.Result{PgRestoreExitCode: 0, DurationMs: 1234}}
	stats := &fakeStats{stats: verifier.Stats{
		DBSizeBytes: 9_000_000,
		SchemaCount: 2,
		TableCount:  3,
		TableStats:  []verifier.TableStat{{SchemaName: "public", Name: "t1", RowCount: 10}},
	}}
	r := newTestRunner(apiClient, okSpawner(), restorer, stats, newFakeRegistrar())
	r.connAlive = func(context.Context, dbconn.Conn) bool { return true }

	r.executeJob(t.Context(), postgresJob())

	report, ok := apiClient.lastReport()
	require.True(t, ok)
	assert.Equal(t, api.VerificationStatusCompleted, report.Status)
	require.NotNil(t, report.DBSizeBytesAfterRestore)
	assert.Equal(t, int64(9_000_000), *report.DBSizeBytesAfterRestore)
	require.NotNil(t, report.TableCount)
	assert.Equal(t, 3, *report.TableCount)
	require.Len(t, report.TableStats, 1)
	assert.Equal(t, "public", report.TableStats[0].SchemaName)
}

func timescaleJob() *api.JobAssignment {
	job := postgresJob()
	job.TimescaledbVersion = "2.17.0"

	return job
}

func Test_ExecuteJob_WhenTimescaleJobSucceeds_RunsPreRestoreThenRestoreThenPostRestore(t *testing.T) {
	apiClient := &fakeAPI{downloadBody: []byte("ARCHIVE")}
	restorer := &fakeRestorer{runResult: restore.Result{PgRestoreExitCode: 0}}
	stats := &fakeStats{stats: verifier.Stats{TableCount: 1}}
	r := newTestRunner(apiClient, okSpawner(), restorer, stats, newFakeRegistrar())
	r.connAlive = func(context.Context, dbconn.Conn) bool { return true }

	r.executeJob(t.Context(), timescaleJob())

	report, ok := apiClient.lastReport()
	require.True(t, ok)
	assert.Equal(t, api.VerificationStatusCompleted, report.Status)
	assert.Equal(t, []string{"pre", "restore", "post"}, restorer.calls,
		"timescaledb restore must enter restoring mode, restore, then leave it")
	assert.Equal(t, 1, restorer.runParallelJobs,
		"timescaledb restore must be single-threaded to keep _timescaledb_catalog FK order")
}

func Test_ExecuteJob_WhenNonTimescaleJob_DoesNotRunTimescaleHooks(t *testing.T) {
	apiClient := &fakeAPI{downloadBody: []byte("ARCHIVE")}
	restorer := &fakeRestorer{runResult: restore.Result{PgRestoreExitCode: 0}}
	stats := &fakeStats{stats: verifier.Stats{TableCount: 1}}
	r := newTestRunner(apiClient, okSpawner(), restorer, stats, newFakeRegistrar())
	r.connAlive = func(context.Context, dbconn.Conn) bool { return true }

	r.executeJob(t.Context(), postgresJob())

	assert.Equal(t, []string{"restore"}, restorer.calls,
		"a non-timescaledb restore must not run the timescaledb hooks")
	assert.Equal(t, 2, restorer.runParallelJobs,
		"a non-timescaledb restore keeps parallel jobs (CPUPerJob)")
}

func Test_ExecuteJob_WhenAbortedBeforeFailedReport_DoesNotReport(t *testing.T) {
	apiClient := &fakeAPI{}
	spawner := &fakeSpawner{err: errors.New("spawn failed")}
	reg := newFakeRegistrar()
	reg.aborted = true
	r := newTestRunner(apiClient, spawner, &fakeRestorer{}, &fakeStats{}, reg)

	r.executeJob(t.Context(), postgresJob())

	assert.Equal(t, 0, apiClient.reportCalled,
		"an aborted/cancelled verification must not get a spurious FAILED report")
}

func Test_Run_WhenContextCancelled_DrainsAndStops(t *testing.T) {
	apiClient := &fakeAPI{}
	r := newTestRunner(apiClient, okSpawner(), &fakeRestorer{}, &fakeStats{}, newFakeRegistrar())

	ctx, cancel := context.WithCancel(t.Context())

	done := make(chan struct{})
	go func() {
		r.Run(ctx)
		close(done)
	}()

	cancel()

	select {
	case <-done:
	case <-time.After(5 * time.Second):
		t.Fatal("Run did not return after context cancellation")
	}
}

func Test_Run_WhenCalledTwice_Panics(t *testing.T) {
	r := newTestRunner(&fakeAPI{}, okSpawner(), &fakeRestorer{}, &fakeStats{}, newFakeRegistrar())

	ctx, cancel := context.WithCancel(t.Context())
	cancel()
	r.Run(ctx)

	assert.Panics(t, func() { r.Run(ctx) })
}
