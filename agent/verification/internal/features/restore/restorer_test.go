package restore

import (
	"context"
	"errors"
	"io"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"databasus-verification-agent/internal/features/dbconn"
	"databasus-verification-agent/internal/testutil"
)

type recordedExec struct {
	cmd   []string
	env   []string
	stdin []byte
}

type fakeExecRunner struct {
	recorded []recordedExec
	result   ExecResult
	err      error
}

func (f *fakeExecRunner) Exec(
	_ context.Context, cmd []string, stdin io.Reader, env []string,
) (ExecResult, error) {
	rec := recordedExec{cmd: cmd, env: env}
	if stdin != nil {
		rec.stdin, _ = io.ReadAll(stdin)
	}

	f.recorded = append(f.recorded, rec)

	return f.result, f.err
}

func testConn() dbconn.Conn {
	return dbconn.Conn{
		Host: "127.0.0.1", Port: 5432,
		User: "postgres", Password: "deadbeef", Database: "verifydb",
	}
}

func testSpec(parallelJobs int) PgRestoreSpec {
	return PgRestoreSpec{
		ArchivePath:  "/restore/x.dump",
		Conn:         testConn(),
		ParallelJobs: parallelJobs,
	}
}

func Test_StageBackupViaExec_WhenDdSucceeds_StreamsBodyToDest(t *testing.T) {
	exec := &fakeExecRunner{result: ExecResult{ExitCode: 0}}
	r := NewRestorer(testutil.DiscardLogger())

	err := r.StageBackupViaExec(
		t.Context(), exec, strings.NewReader("ARCHIVE BYTES"), "/restore/x.dump")

	require.NoError(t, err)
	require.Len(t, exec.recorded, 1)
	assert.Equal(t, []string{"dd", "of=/restore/x.dump", "bs=4M"}, exec.recorded[0].cmd)
	assert.Equal(t, "ARCHIVE BYTES", string(exec.recorded[0].stdin))
}

func Test_StageBackupViaExec_WhenDdExitsNonZero_ReturnsError(t *testing.T) {
	exec := &fakeExecRunner{result: ExecResult{ExitCode: 1, Stderr: "No space left on device"}}
	r := NewRestorer(testutil.DiscardLogger())

	err := r.StageBackupViaExec(t.Context(), exec, strings.NewReader("x"), "/restore/x.dump")

	require.Error(t, err)
	assert.Contains(t, err.Error(), "stage dd exited 1")
	assert.Contains(t, err.Error(), "No space left")
}

func Test_StageBackupViaExec_WhenExecInfraFails_ReturnsError(t *testing.T) {
	exec := &fakeExecRunner{err: errors.New("docker daemon unreachable")}
	r := NewRestorer(testutil.DiscardLogger())

	err := r.StageBackupViaExec(t.Context(), exec, strings.NewReader("x"), "/restore/x.dump")

	require.Error(t, err)
	assert.Contains(t, err.Error(), "stage exec")
}

func Test_RunPgRestore_WhenPgRestoreSucceeds_ReturnsResultWithoutError(t *testing.T) {
	exec := &fakeExecRunner{result: ExecResult{ExitCode: 0}}
	r := NewRestorer(testutil.DiscardLogger())

	result, err := r.RunPgRestore(t.Context(), exec, testSpec(4))

	require.NoError(t, err)
	assert.Equal(t, 0, result.PgRestoreExitCode)

	require.Len(t, exec.recorded, 1)
	cmd := exec.recorded[0].cmd
	assert.Equal(t, "pg_restore", cmd[0])
	assert.Contains(t, cmd, "-Fc")
	assert.Contains(t, cmd, "--no-password")
	assert.Contains(t, cmd, "--no-owner")
	assert.Contains(t, cmd, "--no-acl")
	assert.Contains(t, cmd, "-j")
	assert.Contains(t, cmd, "4")
	assert.Contains(t, cmd, "verifydb")
	assert.Equal(t, "/restore/x.dump", cmd[len(cmd)-1])
	assert.Equal(t, []string{"PGPASSWORD=deadbeef"}, exec.recorded[0].env)
}

func Test_RunPgRestore_WhenNotTimescaledb_PassesCleanAndIfExists(t *testing.T) {
	exec := &fakeExecRunner{result: ExecResult{ExitCode: 0}}
	r := NewRestorer(testutil.DiscardLogger())

	_, err := r.RunPgRestore(t.Context(), exec, testSpec(1))

	require.NoError(t, err)
	require.Len(t, exec.recorded, 1)
	assert.Contains(t, exec.recorded[0].cmd, "--clean")
	assert.Contains(t, exec.recorded[0].cmd, "--if-exists")
}

func Test_RunPgRestore_WhenTimescaledb_OmitsCleanAndIfExists(t *testing.T) {
	exec := &fakeExecRunner{result: ExecResult{ExitCode: 0}}
	r := NewRestorer(testutil.DiscardLogger())

	spec := testSpec(1)
	spec.IsTimescaledb = true

	_, err := r.RunPgRestore(t.Context(), exec, spec)

	require.NoError(t, err)
	require.Len(t, exec.recorded, 1)
	assert.NotContains(t, exec.recorded[0].cmd, "--clean",
		"--clean would DROP EXTENSION timescaledb before pre_restore needs its catalog")
	assert.NotContains(t, exec.recorded[0].cmd, "--if-exists")
}

func Test_RunPgRestore_WhenPgRestoreExitsNonZero_ReturnsErrRestoreFailedWithPopulatedResult(t *testing.T) {
	exec := &fakeExecRunner{result: ExecResult{ExitCode: 1, Stderr: "could not execute query"}}
	r := NewRestorer(testutil.DiscardLogger())

	result, err := r.RunPgRestore(t.Context(), exec, testSpec(2))

	require.Error(t, err)
	assert.ErrorIs(t, err, ErrRestoreFailed)
	assert.Equal(t, 1, result.PgRestoreExitCode)
	assert.Contains(t, result.StderrTail, "could not execute query")
}

func Test_RunPgRestore_WhenExecInfraFails_ReturnsNonRestoreError(t *testing.T) {
	exec := &fakeExecRunner{err: errors.New("exec create failed")}
	r := NewRestorer(testutil.DiscardLogger())

	_, err := r.RunPgRestore(t.Context(), exec, testSpec(1))

	require.Error(t, err)
	assert.NotErrorIs(t, err, ErrRestoreFailed,
		"an exec-infrastructure failure must not look like a non-zero pg_restore exit")
}

func Test_RunPgRestore_WhenStderrHuge_TailIsTruncatedSuffix(t *testing.T) {
	huge := strings.Repeat("A", 9000) + "TAIL-MARKER"
	exec := &fakeExecRunner{result: ExecResult{ExitCode: 0, Stderr: huge}}
	r := NewRestorer(testutil.DiscardLogger())

	result, err := r.RunPgRestore(t.Context(), exec, testSpec(1))

	require.NoError(t, err)
	assert.Len(t, result.StderrTail, 8192)
	assert.True(t, strings.HasSuffix(result.StderrTail, "TAIL-MARKER"))
}
