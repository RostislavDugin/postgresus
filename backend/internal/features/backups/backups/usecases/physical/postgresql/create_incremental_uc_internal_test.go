package usecases_physical_postgresql

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/require"

	physical_enums "databasus-backend/internal/features/backups/backups/core/physical/enums"
)

// The stderr fixtures below carry PostgreSQL's own wording. Paraphrasing one would
// let the test pass while the real message went unmatched in production.

func Test_ClassifyIncrStreamError_WhenSummariesMissingForRange_ReturnsChainBrokenSummariesExpired(t *testing.T) {
	stderr := []byte("pg_basebackup: error: could not initiate base backup: ERROR:  WAL summaries are " +
		"required on timeline 1 from 0/1000000 to 0/2000000, but no summaries for that timeline and " +
		"LSN range exist")

	outcome := classifyIncrStreamError(errors.New("exit status 1"), stderr)

	require.Equal(t, physical_enums.PhysicalBackupStatusChainBroken, outcome.Status)
	require.NotNil(t, outcome.ErrorReason)
	require.Equal(t, physical_enums.PhysicalBackupErrorSummariesExpired, *outcome.ErrorReason)
}

func Test_ClassifyIncrStreamError_WhenSummariesIncompleteForRange_ReturnsChainBrokenSummariesExpired(t *testing.T) {
	stderr := []byte("pg_basebackup: error: could not initiate base backup: ERROR:  WAL summaries are " +
		"required on timeline 1 from 0/1000000 to 0/2000000, but the summaries for that timeline and " +
		"LSN range are incomplete")

	outcome := classifyIncrStreamError(errors.New("exit status 1"), stderr)

	require.Equal(t, physical_enums.PhysicalBackupStatusChainBroken, outcome.Status)
	require.NotNil(t, outcome.ErrorReason)
	require.Equal(t, physical_enums.PhysicalBackupErrorSummariesExpired, *outcome.ErrorReason)
}

func Test_ClassifyIncrStreamError_WhenSummarizationNotProgressing_ReturnsErrorSummarizerFallingBehind(t *testing.T) {
	stderr := []byte("pg_basebackup: error: could not initiate base backup: ERROR:  WAL summarization " +
		"is not progressing")

	outcome := classifyIncrStreamError(errors.New("exit status 1"), stderr)

	require.Equal(t, physical_enums.PhysicalBackupStatusError, outcome.Status,
		"a stalled summarizer is transient, so the chain must stay extendable")
	require.NotNil(t, outcome.ErrorReason)
	require.Equal(t, physical_enums.PhysicalBackupErrorSummarizerFallingBehind, *outcome.ErrorReason)
}

func Test_ClassifyIncrStreamError_WhenUnrelatedStderr_ReturnsErrorPgBasebackupFailed(t *testing.T) {
	outcome := classifyIncrStreamError(errors.New("exit status 1"),
		[]byte("pg_basebackup: error: connection to server failed"))

	require.Equal(t, physical_enums.PhysicalBackupStatusError, outcome.Status)
	require.NotNil(t, outcome.ErrorReason)
	require.Equal(t, physical_enums.PhysicalBackupErrorPgBasebackupFailed, *outcome.ErrorReason)
}
