package audit_logs

import (
	"databasus-backend/internal/storage"
	"fmt"
	"testing"
	"time"

	user_enums "databasus-backend/internal/features/users/enums"
	users_testing "databasus-backend/internal/features/users/testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"gorm.io/gorm"
)

func Test_CleanOldAuditLogs_DeletesLogsOlderThanOneYear(t *testing.T) {
	service := GetAuditLogService()
	user := users_testing.CreateTestUser(user_enums.UserRoleMember)
	db := storage.GetDb()
	baseTime := time.Now().UTC()

	// Create old logs (more than 1 year old)
	createTimedAuditLog(db, &user.UserID, "Old log 1", baseTime.Add(-400*24*time.Hour))
	createTimedAuditLog(db, &user.UserID, "Old log 2", baseTime.Add(-370*24*time.Hour))

	// Create recent logs (less than 1 year old)
	createAuditLog(service, "Recent log 1", &user.UserID, nil)
	createAuditLog(service, "Recent log 2", &user.UserID, nil)

	// Run cleanup
	err := service.CleanOldAuditLogs()
	assert.NoError(t, err)

	// Verify old logs were deleted
	oneYearAgo := baseTime.Add(-365 * 24 * time.Hour)
	var oldLogs []*AuditLog
	db.Where("created_at < ?", oneYearAgo).Find(&oldLogs)
	assert.Equal(t, 0, len(oldLogs), "All logs older than 1 year should be deleted")

	// Verify recent logs still exist
	var recentLogs []*AuditLog
	db.Where("created_at >= ?", oneYearAgo).Find(&recentLogs)
	assert.GreaterOrEqual(t, len(recentLogs), 2, "Recent logs should not be deleted")
}

func Test_CleanOldAuditLogs_PreservesLogsNewerThanOneYear(t *testing.T) {
	service := GetAuditLogService()
	user := users_testing.CreateTestUser(user_enums.UserRoleMember)
	db := storage.GetDb()
	baseTime := time.Now().UTC()

	// Create logs exactly at boundary (1 year old)
	boundaryTime := baseTime.Add(-365 * 24 * time.Hour)
	createTimedAuditLog(db, &user.UserID, "Boundary log", boundaryTime)

	// Create recent logs
	createTimedAuditLog(db, &user.UserID, "Recent log 1", baseTime.Add(-364*24*time.Hour))
	createTimedAuditLog(db, &user.UserID, "Recent log 2", baseTime.Add(-100*24*time.Hour))
	createAuditLog(service, "Current log", &user.UserID, nil)

	// Get count before cleanup
	var countBefore int64
	db.Model(&AuditLog{}).Count(&countBefore)

	// Run cleanup
	err := service.CleanOldAuditLogs()
	assert.NoError(t, err)

	// Get count after cleanup
	var countAfter int64
	db.Model(&AuditLog{}).Count(&countAfter)

	// Verify logs newer than 1 year are preserved
	oneYearAgo := baseTime.Add(-365 * 24 * time.Hour)
	var recentLogs []*AuditLog
	db.Where("created_at >= ?", oneYearAgo).Find(&recentLogs)

	messages := make([]string, len(recentLogs))
	for i, log := range recentLogs {
		messages[i] = log.Message
	}

	assert.Contains(t, messages, "Recent log 1")
	assert.Contains(t, messages, "Recent log 2")
	assert.Contains(t, messages, "Current log")
}

func Test_CleanOldAuditLogs_HandlesEmptyDatabase(t *testing.T) {
	service := GetAuditLogService()

	// Run cleanup on database that may have no old logs
	err := service.CleanOldAuditLogs()
	assert.NoError(t, err)
}

func Test_CleanOldAuditLogs_DeletesMultipleOldLogs(t *testing.T) {
	service := GetAuditLogService()
	user := users_testing.CreateTestUser(user_enums.UserRoleMember)
	db := storage.GetDb()
	baseTime := time.Now().UTC()

	// Create many old logs with specific UUIDs to track them
	testLogIDs := make([]uuid.UUID, 5)
	for i := 0; i < 5; i++ {
		testLogIDs[i] = uuid.New()
		daysAgo := 400 + (i * 10)
		log := &AuditLog{
			ID:        testLogIDs[i],
			UserID:    &user.UserID,
			Message:   fmt.Sprintf("Test old log %d", i),
			CreatedAt: baseTime.Add(-time.Duration(daysAgo) * 24 * time.Hour),
		}
		result := db.Create(log)
		assert.NoError(t, result.Error)
	}

	// Verify logs exist before cleanup
	var logsBeforeCleanup []*AuditLog
	db.Where("id IN ?", testLogIDs).Find(&logsBeforeCleanup)
	assert.Equal(t, 5, len(logsBeforeCleanup), "All test logs should exist before cleanup")

	// Run cleanup
	err := service.CleanOldAuditLogs()
	assert.NoError(t, err)

	// Verify test logs were deleted
	var logsAfterCleanup []*AuditLog
	db.Where("id IN ?", testLogIDs).Find(&logsAfterCleanup)
	assert.Equal(t, 0, len(logsAfterCleanup), "All old test logs should be deleted")
}

func createTimedAuditLog(db *gorm.DB, userID *uuid.UUID, message string, createdAt time.Time) {
	log := &AuditLog{
		ID:        uuid.New(),
		UserID:    userID,
		Message:   message,
		CreatedAt: createdAt,
	}
	db.Create(log)
}
