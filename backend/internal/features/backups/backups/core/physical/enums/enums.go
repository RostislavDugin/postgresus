package physical_enums

type PhysicalBackupStatus string

const (
	// pg_basebackup spawned (or about to spawn), not yet finished.
	PhysicalBackupStatusInProgress PhysicalBackupStatus = "IN_PROGRESS"

	// Artifact in storage, all metadata populated. Chain is extendable
	// from this row.
	PhysicalBackupStatusCompleted PhysicalBackupStatus = "COMPLETED"

	// Transient failure (network blip, pg_basebackup spurious error,
	// storage upload retry exhausted, byte-stall watcher tripped). Chain is
	// still valid for this attempt's parent; scheduler retries the SAME kind
	// (FULL or INCR) on next tick per retry policy.
	PhysicalBackupStatusError PhysicalBackupStatus = "ERROR"

	// Chain cannot be extended further from this point. Next attempt MUST
	// escalate to a fresh FULL (new chain identity). Rare for FULL but
	// possible (corrupted manifest mid-stream, sys_id mismatch caught
	// partway, start_lsn outside any known timeline range). Common for INCR
	// (summaries expired, summarizer off, parent manifest missing).
	PhysicalBackupStatusChainBroken PhysicalBackupStatus = "CHAIN_BROKEN"

	// User- or system-initiated cancel. See PhysicalBackupErrorReason for
	// the cancel variants.
	PhysicalBackupStatusCanceled PhysicalBackupStatus = "CANCELED"
)

type PhysicalBackupErrorReason string

const (
	PhysicalBackupErrorPgBasebackupFailed       PhysicalBackupErrorReason = "PG_BASEBACKUP_FAILED"
	PhysicalBackupErrorStorageUploadFailed      PhysicalBackupErrorReason = "STORAGE_UPLOAD_FAILED"
	PhysicalBackupErrorNetworkFailure           PhysicalBackupErrorReason = "NETWORK_FAILURE"
	PhysicalBackupErrorNetworkStallTimeout      PhysicalBackupErrorReason = "NETWORK_STALL_TIMEOUT"
	PhysicalBackupErrorApplicationRestart       PhysicalBackupErrorReason = "APPLICATION_RESTART"
	PhysicalBackupErrorSystemIdentifierMismatch PhysicalBackupErrorReason = "SYSTEM_IDENTIFIER_MISMATCH"
	PhysicalBackupErrorTimelineRegression       PhysicalBackupErrorReason = "TIMELINE_REGRESSION"
	PhysicalBackupErrorManifestCorrupted        PhysicalBackupErrorReason = "MANIFEST_CORRUPTED"
	PhysicalBackupErrorStartLsnOutsideTimeline  PhysicalBackupErrorReason = "START_LSN_OUTSIDE_TIMELINE_RANGE"

	// Covers both an explicit user cancel on an in-flight backup AND
	// in-flight backups cancelled by OnBackupConfigChanged when the user
	// disables backups or demotes BackupType.
	PhysicalBackupErrorCanceledByUser PhysicalBackupErrorReason = "CANCELED_BY_USER"

	// In-flight backup killed because the parent DB was removed via
	// OnBeforeDatabaseRemove. The backup row is cascade-deleted moments
	// later — so this value is rarely observed in steady-state queries, but
	// it appears in archived audit logs and in tests that snapshot status
	// between the CANCEL and the DROP.
	PhysicalBackupErrorCanceledByDbRemoval PhysicalBackupErrorReason = "CANCELED_BY_DB_REMOVAL"

	// INCR-specific (all chain-killing, status = CHAIN_BROKEN).
	PhysicalBackupErrorSummariesExpired      PhysicalBackupErrorReason = "SUMMARIES_EXPIRED"
	PhysicalBackupErrorSummarizerOff         PhysicalBackupErrorReason = "SUMMARIZER_OFF"
	PhysicalBackupErrorParentManifestMissing PhysicalBackupErrorReason = "PARENT_MANIFEST_MISSING"

	// The summarizer is on and covers the parent, but trails current WAL by
	// more than the lag threshold (or stayed behind for the whole bounded
	// wait): pushing an INCR would race a moving target, so the chain is closed
	// and the next tick opens a fresh FULL.
	PhysicalBackupErrorSummarizerFallingBehind PhysicalBackupErrorReason = "SUMMARIZER_FALLING_BEHIND"
)

type PhysicalWalStreamerStatus string

const (
	PhysicalWalStreamerStatusRunning PhysicalWalStreamerStatus = "RUNNING"
	PhysicalWalStreamerStatusFailed  PhysicalWalStreamerStatus = "FAILED"
)

type PhysicalBackupCompression string

const (
	PhysicalBackupCompressionZstd PhysicalBackupCompression = "ZSTD"
	PhysicalBackupCompressionGzip PhysicalBackupCompression = "GZIP"
	PhysicalBackupCompressionNone PhysicalBackupCompression = "NONE"
)

type PhysicalBackupType string

const (
	PhysicalBackupTypeFull        PhysicalBackupType = "FULL"
	PhysicalBackupTypeIncremental PhysicalBackupType = "INCREMENTAL"

	// PhysicalBackupTypeWal tags a committed WAL segment in the flat backup
	// list. WAL has no dedicated backup table-row identity beyond the segment,
	// so this value exists only as a list/UI discriminator, never as a backup
	// the scheduler creates.
	PhysicalBackupTypeWal PhysicalBackupType = "WAL"
)
