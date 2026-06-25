package backups_config_physical

import (
	"testing"

	"github.com/stretchr/testify/assert"

	backups_core_enums "databasus-backend/internal/features/backups/backups/core/enums"
	postgresql_physical "databasus-backend/internal/features/databases/databases/postgresql/physical"
	"databasus-backend/internal/features/intervals"
)

func dailyAt(hhmm string) intervals.Interval {
	return intervals.Interval{
		Type:      intervals.IntervalDaily,
		TimeOfDay: new(hhmm),
	}
}

func hourly() intervals.Interval {
	return intervals.Interval{Type: intervals.IntervalHourly}
}

func weeklyMondayAt(hhmm string) intervals.Interval {
	return intervals.Interval{
		Type:      intervals.IntervalWeekly,
		Weekday:   new(1),
		TimeOfDay: new(hhmm),
	}
}

func validFullOnlyConfig() *PhysicalBackupConfig {
	return &PhysicalBackupConfig{
		IsBackupsEnabled:   true,
		FullBackupInterval: dailyAt("02:00"),
		Retention:          RetentionFullBackups,
		FullBackupsRetention: FullBackupsRetention{
			Policy:  FullBackupsRetentionPolicyGfs,
			GfsDays: 7,
		},
		Encryption: backups_core_enums.BackupEncryptionNone,
		PostgresqlPhysical: &postgresql_physical.PostgresqlPhysicalDatabase{
			BackupType: postgresql_physical.BackupTypeFullOnly,
		},
	}
}

func validFullIncrementalConfig() *PhysicalBackupConfig {
	return &PhysicalBackupConfig{
		IsBackupsEnabled:          true,
		FullBackupInterval:        weeklyMondayAt("02:00"),
		IncrementalBackupInterval: dailyAt("02:00"),
		Retention:                 RetentionChains,
		ChainsRetention:           ChainsRetention{Count: 3},
		Encryption:                backups_core_enums.BackupEncryptionNone,
		PostgresqlPhysical: &postgresql_physical.PostgresqlPhysicalDatabase{
			BackupType: postgresql_physical.BackupTypeFullAndIncremental,
		},
	}
}

func validFullIncrementalWalStreamConfig() *PhysicalBackupConfig {
	return &PhysicalBackupConfig{
		IsBackupsEnabled:          true,
		FullBackupInterval:        weeklyMondayAt("02:00"),
		IncrementalBackupInterval: dailyAt("02:00"),
		Retention:                 RetentionChains,
		ChainsRetention:           ChainsRetention{Count: 3},
		WalLagThresholdBytes:      16 * 1024 * 1024,
		Encryption:                backups_core_enums.BackupEncryptionNone,
		PostgresqlPhysical: &postgresql_physical.PostgresqlPhysicalDatabase{
			BackupType: postgresql_physical.BackupTypeFullIncrementalAndWalStream,
		},
	}
}

func Test_Validate_RejectsBlankFullBackupInterval(t *testing.T) {
	c := validFullOnlyConfig()
	c.FullBackupInterval = intervals.Interval{}

	assert.Error(t, c.Validate())
}

func Test_Validate_RejectsUnknownEncryption(t *testing.T) {
	c := validFullOnlyConfig()
	c.Encryption = "GIBBERISH"

	assert.Error(t, c.Validate())
}

func Test_Validate_RejectsNegativeWalLagThreshold(t *testing.T) {
	c := validFullIncrementalWalStreamConfig()
	c.WalLagThresholdBytes = -1

	assert.Error(t, c.Validate())
}

func Test_Validate_FullOnly_RejectsIncrementalIntervalSet(t *testing.T) {
	c := validFullOnlyConfig()
	c.IncrementalBackupInterval = hourly()

	err := c.Validate()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "incremental")
}

func Test_Validate_FullOnly_AcceptsFullBackupsLastN(t *testing.T) {
	c := validFullOnlyConfig()
	c.FullBackupsRetention = FullBackupsRetention{
		Policy: FullBackupsRetentionPolicyLastN,
		Count:  5,
	}

	assert.NoError(t, c.Validate())
}

func Test_Validate_FullOnly_AcceptsFullBackupsGfs(t *testing.T) {
	c := validFullOnlyConfig()
	c.FullBackupsRetention = FullBackupsRetention{
		Policy:  FullBackupsRetentionPolicyGfs,
		GfsDays: 7,
	}

	assert.NoError(t, c.Validate())
}

func Test_Validate_FullOnly_RejectsChainsRetention(t *testing.T) {
	c := validFullOnlyConfig()
	c.Retention = RetentionChains
	c.ChainsRetention = ChainsRetention{Count: 3}
	c.FullBackupsRetention = FullBackupsRetention{}

	err := c.Validate()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "FULL_ONLY")
}

func Test_Validate_FullOnly_RejectsChainsAndFullBackupsRetention(t *testing.T) {
	c := validFullOnlyConfig()
	c.Retention = RetentionChainsAndFullBackups
	c.ChainsRetention = ChainsRetention{Count: 3}

	err := c.Validate()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "FULL_ONLY")
}

func Test_Validate_FullOnly_RejectsWalLagThresholdSet(t *testing.T) {
	c := validFullOnlyConfig()
	c.WalLagThresholdBytes = 1024

	assert.Error(t, c.Validate())
}

func Test_Validate_FullIncremental_RejectsBlankIncrementalInterval(t *testing.T) {
	c := validFullIncrementalConfig()
	c.IncrementalBackupInterval = intervals.Interval{}

	assert.Error(t, c.Validate())
}

func Test_Validate_FullIncremental_RejectsIncrementalCadenceEqualToFull(t *testing.T) {
	c := validFullIncrementalConfig()
	c.FullBackupInterval = dailyAt("02:00")
	c.IncrementalBackupInterval = dailyAt("12:00")

	err := c.Validate()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "more frequent")
}

func Test_Validate_FullIncremental_RejectsIncrementalCadenceLargerThanFull(t *testing.T) {
	c := validFullIncrementalConfig()
	c.FullBackupInterval = hourly()
	c.IncrementalBackupInterval = dailyAt("02:00")

	err := c.Validate()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "more frequent")
}

func Test_Validate_FullIncremental_AcceptsChainsRetention(t *testing.T) {
	c := validFullIncrementalConfig()
	c.Retention = RetentionChains
	c.ChainsRetention = ChainsRetention{Count: 3}
	c.FullBackupsRetention = FullBackupsRetention{}

	assert.NoError(t, c.Validate())
}

func Test_Validate_FullIncremental_AcceptsChainsAndFullBackupsRetention(t *testing.T) {
	c := validFullIncrementalConfig()
	c.Retention = RetentionChainsAndFullBackups
	c.ChainsRetention = ChainsRetention{Count: 3}
	c.FullBackupsRetention = FullBackupsRetention{
		Policy:   FullBackupsRetentionPolicyGfs,
		GfsWeeks: 4,
	}

	assert.NoError(t, c.Validate())
}

func Test_Validate_FullIncremental_RejectsFullBackupsOnlyRetention(t *testing.T) {
	c := validFullIncrementalConfig()
	c.Retention = RetentionFullBackups
	c.ChainsRetention = ChainsRetention{}
	c.FullBackupsRetention = FullBackupsRetention{
		Policy: FullBackupsRetentionPolicyLastN,
		Count:  5,
	}

	err := c.Validate()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "CHAINS")
}

func Test_Validate_FullIncremental_RejectsWalLagThresholdSet(t *testing.T) {
	c := validFullIncrementalConfig()
	c.WalLagThresholdBytes = 1024

	assert.Error(t, c.Validate())
}

func Test_Validate_RejectsChainsRetentionWithZeroCount(t *testing.T) {
	c := validFullIncrementalConfig()
	c.Retention = RetentionChains
	c.ChainsRetention = ChainsRetention{Count: 0}

	err := c.Validate()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "chains_retention.count")
}

func Test_Validate_RejectsChainsRetentionWithFullBackupsFieldsSet(t *testing.T) {
	c := validFullIncrementalConfig()
	c.Retention = RetentionChains
	c.ChainsRetention = ChainsRetention{Count: 3}
	c.FullBackupsRetention = FullBackupsRetention{
		Policy: FullBackupsRetentionPolicyLastN,
		Count:  5,
	}

	err := c.Validate()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "full_backups_retention")
}

func Test_Validate_RejectsFullBackupsLastNWithZeroCount(t *testing.T) {
	c := validFullOnlyConfig()
	c.FullBackupsRetention = FullBackupsRetention{
		Policy: FullBackupsRetentionPolicyLastN,
		Count:  0,
	}

	err := c.Validate()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "count > 0")
}

func Test_Validate_RejectsFullBackupsLastNWithGfsBucketsSet(t *testing.T) {
	c := validFullOnlyConfig()
	c.FullBackupsRetention = FullBackupsRetention{
		Policy:  FullBackupsRetentionPolicyLastN,
		Count:   5,
		GfsDays: 3,
	}

	err := c.Validate()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "GFS buckets")
}

func Test_Validate_RejectsFullBackupsGfsWithAllZeroBuckets(t *testing.T) {
	c := validFullOnlyConfig()
	c.FullBackupsRetention = FullBackupsRetention{
		Policy: FullBackupsRetentionPolicyGfs,
	}

	err := c.Validate()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "bucket")
}

func Test_Validate_RejectsFullBackupsGfsWithCountSet(t *testing.T) {
	c := validFullOnlyConfig()
	c.FullBackupsRetention = FullBackupsRetention{
		Policy:  FullBackupsRetentionPolicyGfs,
		Count:   3,
		GfsDays: 7,
	}

	err := c.Validate()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "count")
}

func Test_Validate_RejectsFullBackupsRetentionWithChainsCountSet(t *testing.T) {
	c := validFullOnlyConfig()
	c.ChainsRetention = ChainsRetention{Count: 3}

	err := c.Validate()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "chains_retention")
}

func Test_Validate_RejectsUnknownRetention(t *testing.T) {
	c := validFullOnlyConfig()
	c.Retention = "GIBBERISH"

	assert.Error(t, c.Validate())
}

func Test_Validate_RejectsUnknownFullBackupsPolicy(t *testing.T) {
	c := validFullOnlyConfig()
	c.FullBackupsRetention = FullBackupsRetention{
		Policy:  "GIBBERISH",
		GfsDays: 7,
	}

	assert.Error(t, c.Validate())
}

func Test_Validate_FullIncrementalWalStream_RejectsZeroWalLagThreshold(t *testing.T) {
	c := validFullIncrementalWalStreamConfig()
	c.WalLagThresholdBytes = 0

	err := c.Validate()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "wal lag threshold")
}

func Test_Validate_FullIncrementalWalStream_AcceptsValidConfig(t *testing.T) {
	assert.NoError(t, validFullIncrementalWalStreamConfig().Validate())
}

func Test_Validate_RejectsUnknownBackupType(t *testing.T) {
	c := validFullOnlyConfig()
	c.PostgresqlPhysical.BackupType = "GIBBERISH"

	assert.Error(t, c.Validate())
}

func Test_Validate_RejectsMissingPostgresqlPhysicalPreload(t *testing.T) {
	c := validFullOnlyConfig()
	c.PostgresqlPhysical = nil

	err := c.Validate()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "preloaded")
}
