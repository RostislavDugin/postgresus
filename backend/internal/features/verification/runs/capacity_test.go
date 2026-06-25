package verification_runs

import (
	"testing"

	"github.com/stretchr/testify/assert"

	backups_core_logical "databasus-backend/internal/features/backups/backups/core/logical"
)

func backupWithSize(archiveMb, restoredMb float64) *backups_core_logical.LogicalBackup {
	return &backups_core_logical.LogicalBackup{
		BackupSizeMb:      archiveMb,
		BackupRawDbSizeMb: restoredMb,
	}
}

func Test_DoesVerificationFit_AcrossScenarios_RespectsAgentBudget(t *testing.T) {
	cases := []struct {
		name           string
		capacity       AgentCapacity
		runningBackups []*backups_core_logical.LogicalBackup
		candidate      *backups_core_logical.LogicalBackup
		wantFits       bool
	}{
		{
			name:           "empty agent fits small candidate",
			capacity:       AgentCapacity{MaxDiskGb: 10},
			runningBackups: nil,
			candidate:      backupWithSize(50, 100),
			wantFits:       true,
		},
		{
			name:           "empty agent fits candidate at exact budget boundary",
			capacity:       AgentCapacity{MaxDiskGb: 10},
			runningBackups: nil,
			// archive + restored + per-job gap = 10240 (full 10 GB budget)
			candidate: backupWithSize(4000, 1120),
			wantFits:  true,
		},
		{
			name:           "empty agent rejects candidate one MB over budget",
			capacity:       AgentCapacity{MaxDiskGb: 10},
			runningBackups: nil,
			candidate:      backupWithSize(4000, 1121),
			wantFits:       false,
		},
		{
			name:     "one running, room for another small",
			capacity: AgentCapacity{MaxDiskGb: 12},
			runningBackups: []*backups_core_logical.LogicalBackup{
				backupWithSize(50, 200),
			},
			candidate: backupWithSize(50, 200),
			wantFits:  true,
		},
		{
			name:     "one running, restored size of candidate blows budget",
			capacity: AgentCapacity{MaxDiskGb: 10},
			runningBackups: []*backups_core_logical.LogicalBackup{
				backupWithSize(500, 4000),
			},
			candidate: backupWithSize(500, 4000),
			wantFits:  false,
		},
		{
			name:     "three concurrent small jobs fit in 25 GB",
			capacity: AgentCapacity{MaxDiskGb: 25},
			runningBackups: []*backups_core_logical.LogicalBackup{
				backupWithSize(100, 500),
				backupWithSize(100, 500),
				backupWithSize(100, 500),
			},
			candidate: backupWithSize(100, 500),
			wantFits:  true,
		},
		{
			name:     "per-job gap saturation: three RUNNING already over a 5 GB agent",
			capacity: AgentCapacity{MaxDiskGb: 5},
			runningBackups: []*backups_core_logical.LogicalBackup{
				backupWithSize(100, 400),
				backupWithSize(100, 400),
				backupWithSize(100, 400),
			},
			candidate: backupWithSize(10, 10),
			wantFits:  false,
		},
		{
			name:           "zero-size candidate still consumes the per-job gap and fits",
			capacity:       AgentCapacity{MaxDiskGb: 10},
			runningBackups: nil,
			candidate:      backupWithSize(0, 0),
			wantFits:       true,
		},
		{
			name:           "negative sizes clamp to zero and still fit",
			capacity:       AgentCapacity{MaxDiskGb: 10},
			runningBackups: nil,
			candidate:      backupWithSize(-50, -100),
			wantFits:       true,
		},
		{
			name:           "zero-capacity agent rejects everything",
			capacity:       AgentCapacity{MaxDiskGb: 0},
			runningBackups: nil,
			candidate:      backupWithSize(1, 1),
			wantFits:       false,
		},
		{
			name:           "nil candidate is rejected",
			capacity:       AgentCapacity{MaxDiskGb: 10},
			runningBackups: nil,
			candidate:      nil,
			wantFits:       false,
		},
		{
			name:           "raw DB size dominates archive size in the cost",
			capacity:       AgentCapacity{MaxDiskGb: 2},
			runningBackups: nil,
			candidate:      backupWithSize(10, 2000),
			wantFits:       false,
		},
	}

	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			fits := IsVerificationFitWithinRemainedDiskCapacity(
				testCase.capacity,
				testCase.runningBackups,
				testCase.candidate,
			)

			assert.Equal(t, testCase.wantFits, fits)
		})
	}
}
