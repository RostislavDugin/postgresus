package verification_runs

import (
	"math"

	backups_core_logical "databasus-backend/internal/features/backups/backups/core/logical"
)

// diskCostPerJobGapMb is the per-job safe gap added on top of the downloaded
// backup file and the restored database. The restore's peak on-disk footprint
// exceeds backup + final DB size: pg_restore writes WAL into pg_wal (inside
// PGDATA, counted by the agent's disk watcher), parallel index builds spill
// sort/temp space, and the whole-cluster baseline plus FS slack add fixed cost.
// 5 GB of absolute headroom covers this across the realistic DB-size range.
const diskCostPerJobGapMb = 5120

// IsVerificationFitWithinRemainedDiskCapacity reports whether
// running candidateBackup on this agent alongside runningBackups
// stays within the agent's declared disk capacity.
func IsVerificationFitWithinRemainedDiskCapacity(
	capacity AgentCapacity,
	runningBackups []*backups_core_logical.LogicalBackup,
	candidateBackup *backups_core_logical.LogicalBackup,
) bool {
	if capacity.MaxDiskGb <= 0 || candidateBackup == nil {
		return false
	}

	totalBudgetMb := int64(capacity.MaxDiskGb) * 1024
	usedMb := sumEstimatedRequiredDiskMb(runningBackups)
	candidateCostMb := EstimateRequiredForRestoreDiskMb(candidateBackup)

	return usedMb+candidateCostMb <= totalBudgetMb
}

// EstimateRequiredForRestoreDiskMb estimates expected job's on-disk cost.
//
// Includes:
// - Space needed for backup file (if archived - decompressed on the fly while streaming)
// - Space needed for restored database
// - Safe gap (WAL, indexes, sort/temp spills, FS slack)
func EstimateRequiredForRestoreDiskMb(backup *backups_core_logical.LogicalBackup) int64 {
	archiveSizeMb := backup.BackupSizeMb
	if archiveSizeMb < 0 {
		archiveSizeMb = 0
	}

	restoredSizeMb := backup.BackupRawDbSizeMb
	if restoredSizeMb < 0 {
		restoredSizeMb = 0
	}

	return int64(math.Ceil(archiveSizeMb+restoredSizeMb)) + diskCostPerJobGapMb
}

func sumEstimatedRequiredDiskMb(backups []*backups_core_logical.LogicalBackup) int64 {
	var total int64
	for _, backup := range backups {
		total += EstimateRequiredForRestoreDiskMb(backup)
	}

	return total
}
