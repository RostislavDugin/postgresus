package backups_config_logical

import (
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"

	backups_core_enums "databasus-backend/internal/features/backups/backups/core/enums"
	"databasus-backend/internal/features/intervals"
	"databasus-backend/internal/util/period"
)

func Test_Validate_WhenIntervalTypeIsMissing_ValidationFails(t *testing.T) {
	config := createValidBackupConfig()
	config.BackupInterval = intervals.Interval{}

	err := config.Validate()
	assert.ErrorContains(t, err, "backup interval")
}

func Test_Validate_WhenRetryEnabledButMaxTriesIsZero_ValidationFails(t *testing.T) {
	config := createValidBackupConfig()
	config.IsRetryIfFailed = true
	config.MaxFailedTriesCount = 0

	err := config.Validate()
	assert.EqualError(t, err, "max failed tries count must be greater than 0")
}

func Test_Validate_WhenEncryptionIsInvalid_ValidationFails(t *testing.T) {
	config := createValidBackupConfig()
	config.Encryption = "INVALID"

	err := config.Validate()
	assert.EqualError(t, err, "encryption must be NONE or ENCRYPTED")
}

func Test_Validate_WhenRetentionTimePeriodIsEmpty_ValidationFails(t *testing.T) {
	config := createValidBackupConfig()
	config.RetentionTimePeriod = ""

	err := config.Validate()
	assert.EqualError(t, err, "retention time period is required")
}

func Test_Validate_WhenPolicyTypeIsCount_RequiresPositiveCount(t *testing.T) {
	config := createValidBackupConfig()
	config.RetentionPolicyType = RetentionPolicyTypeCount
	config.RetentionCount = 0

	err := config.Validate()
	assert.EqualError(t, err, "retention count must be greater than 0")
}

func Test_Validate_WhenPolicyTypeIsCount_WithPositiveCount_ValidationPasses(t *testing.T) {
	config := createValidBackupConfig()
	config.RetentionPolicyType = RetentionPolicyTypeCount
	config.RetentionCount = 10

	err := config.Validate()
	assert.NoError(t, err)
}

func Test_Validate_WhenPolicyTypeIsGFS_RequiresAtLeastOneField(t *testing.T) {
	config := createValidBackupConfig()
	config.RetentionPolicyType = RetentionPolicyTypeGFS
	config.RetentionGfsDays = 0
	config.RetentionGfsWeeks = 0
	config.RetentionGfsMonths = 0
	config.RetentionGfsYears = 0

	err := config.Validate()
	assert.EqualError(t, err, "at least one GFS retention field must be greater than 0")
}

func Test_Validate_WhenPolicyTypeIsGFS_WithOnlyHours_ValidationPasses(t *testing.T) {
	config := createValidBackupConfig()
	config.RetentionPolicyType = RetentionPolicyTypeGFS
	config.RetentionGfsHours = 24

	err := config.Validate()
	assert.NoError(t, err)
}

func Test_Validate_WhenPolicyTypeIsGFS_WithOnlyDays_ValidationPasses(t *testing.T) {
	config := createValidBackupConfig()
	config.RetentionPolicyType = RetentionPolicyTypeGFS
	config.RetentionGfsDays = 7

	err := config.Validate()
	assert.NoError(t, err)
}

func Test_Validate_WhenPolicyTypeIsGFS_WithAllFields_ValidationPasses(t *testing.T) {
	config := createValidBackupConfig()
	config.RetentionPolicyType = RetentionPolicyTypeGFS
	config.RetentionGfsHours = 24
	config.RetentionGfsDays = 7
	config.RetentionGfsWeeks = 4
	config.RetentionGfsMonths = 12
	config.RetentionGfsYears = 3

	err := config.Validate()
	assert.NoError(t, err)
}

func Test_Validate_WhenPolicyTypeIsInvalid_ValidationFails(t *testing.T) {
	config := createValidBackupConfig()
	config.RetentionPolicyType = "INVALID"

	err := config.Validate()
	assert.EqualError(t, err, "invalid retention policy type")
}

func Test_Validate_WhenEncryptionIsNotEncrypted_ValidationPasses(t *testing.T) {
	backupConfig := createValidBackupConfig()
	backupConfig.Encryption = backups_core_enums.BackupEncryptionNone

	err := backupConfig.Validate()
	assert.NoError(t, err)
}

func createValidBackupConfig() *LogicalBackupConfig {
	return &LogicalBackupConfig{
		DatabaseID:          uuid.New(),
		IsBackupsEnabled:    true,
		RetentionPolicyType: RetentionPolicyTypeTimePeriod,
		RetentionTimePeriod: period.PeriodMonth,
		BackupInterval:      intervals.Interval{Type: intervals.IntervalHourly},
		SendNotificationsOn: []BackupNotificationType{},
		IsRetryIfFailed:     false,
		MaxFailedTriesCount: 3,
		Encryption:          backups_core_enums.BackupEncryptionNone,
	}
}
