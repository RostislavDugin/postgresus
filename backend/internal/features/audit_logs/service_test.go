package audit_logs

import (
	"testing"
	"time"

	user_enums "postgresus-backend/internal/features/users/enums"
	users_testing "postgresus-backend/internal/features/users/testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

func Test_AuditLogs_WorkspaceSpecificLogs(t *testing.T) {
	service := GetAuditLogService()
	user1 := users_testing.CreateTestUser(user_enums.UserRoleMember)
	user2 := users_testing.CreateTestUser(user_enums.UserRoleMember)
	workspace1ID, workspace2ID := uuid.New(), uuid.New()

	// Create test logs for workspaces
	createAuditLog(service, "Test workspace1 log first", &user1.UserID, &workspace1ID)
	createAuditLog(service, "Test workspace1 log second", &user2.UserID, &workspace1ID)
	createAuditLog(service, "Test workspace2 log first", &user1.UserID, &workspace2ID)
	createAuditLog(service, "Test workspace2 log second", &user2.UserID, &workspace2ID)
	createAuditLog(service, "Test no workspace log", &user1.UserID, nil)

	request := &GetAuditLogsRequest{Limit: 10, Offset: 0}

	// Test workspace 1 logs
	workspace1Response, err := service.GetWorkspaceAuditLogs(workspace1ID, request)
	assert.NoError(t, err)
	assert.Equal(t, 2, len(workspace1Response.AuditLogs))

	messages := extractMessages(workspace1Response.AuditLogs)
	assert.Contains(t, messages, "Test workspace1 log first")
	assert.Contains(t, messages, "Test workspace1 log second")
	for _, log := range workspace1Response.AuditLogs {
		assert.Equal(t, &workspace1ID, log.WorkspaceID)
	}

	// Test workspace 2 logs
	workspace2Response, err := service.GetWorkspaceAuditLogs(workspace2ID, request)
	assert.NoError(t, err)
	assert.Equal(t, 2, len(workspace2Response.AuditLogs))

	messages2 := extractMessages(workspace2Response.AuditLogs)
	assert.Contains(t, messages2, "Test workspace2 log first")
	assert.Contains(t, messages2, "Test workspace2 log second")

	// Test pagination
	limitedResponse, err := service.GetWorkspaceAuditLogs(workspace1ID,
		&GetAuditLogsRequest{Limit: 1, Offset: 0})
	assert.NoError(t, err)
	assert.Equal(t, 1, len(limitedResponse.AuditLogs))
	assert.Equal(t, 1, limitedResponse.Limit)

	// Test beforeDate filter
	beforeTime := time.Now().UTC().Add(-1 * time.Minute)
	filteredResponse, err := service.GetWorkspaceAuditLogs(workspace1ID,
		&GetAuditLogsRequest{Limit: 10, BeforeDate: &beforeTime})
	assert.NoError(t, err)
	for _, log := range filteredResponse.AuditLogs {
		assert.True(t, log.CreatedAt.Before(beforeTime))
		assert.NotNil(t, log.UserEmail, "User email should be present for logs with user_id")
		assert.NotNil(
			t,
			log.WorkspaceName,
			"Workspace name should be present for logs with workspace_id",
		)
	}
}

func createAuditLog(service *AuditLogService, message string, userID, workspaceID *uuid.UUID) {
	service.WriteAuditLog(message, userID, workspaceID)
}

func extractMessages(logs []*AuditLogDTO) []string {
	messages := make([]string, len(logs))
	for i, log := range logs {
		messages[i] = log.Message
	}
	return messages
}
