package audit_logs

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	audit_logs_models "databasus-backend/internal/features/audit_logs/models"
	user_enums "databasus-backend/internal/features/users/enums"
	users_testing "databasus-backend/internal/features/users/testing"
	"databasus-backend/internal/util/logger"
)

func Test_AuditLogs_WorkspaceSpecificLogs(t *testing.T) {
	service := GetAuditLogService()
	user1 := users_testing.CreateTestUser(t.Context(), user_enums.UserRoleMember)
	user2 := users_testing.CreateTestUser(t.Context(), user_enums.UserRoleMember)
	workspace1ID, workspace2ID := uuid.New(), uuid.New()

	createAuditLog(t, service, audit_logs_models.AuditEntry{
		Message:     "Test workspace1 log first",
		UserID:      &user1.UserID,
		WorkspaceID: &workspace1ID,
	})
	createAuditLog(t, service, audit_logs_models.AuditEntry{
		Message:     "Test workspace1 log second",
		UserID:      &user2.UserID,
		WorkspaceID: &workspace1ID,
	})
	createAuditLog(t, service, audit_logs_models.AuditEntry{
		Message:     "Test workspace2 log first",
		UserID:      &user1.UserID,
		WorkspaceID: &workspace2ID,
	})
	createAuditLog(t, service, audit_logs_models.AuditEntry{
		Message:     "Test workspace2 log second",
		UserID:      &user2.UserID,
		WorkspaceID: &workspace2ID,
	})
	createAuditLog(t, service, audit_logs_models.AuditEntry{
		Message:     "Test no workspace log",
		UserID:      &user1.UserID,
		WorkspaceID: nil,
	})

	request := &GetAuditLogsRequest{Limit: 10, Offset: 0}

	workspace1Response, err := service.GetWorkspaceAuditLogs(t.Context(), workspace1ID, request)
	assert.NoError(t, err)
	assert.Equal(t, 2, len(workspace1Response.AuditLogs))

	messages := extractMessages(workspace1Response.AuditLogs)
	assert.Contains(t, messages, "Test workspace1 log first")
	assert.Contains(t, messages, "Test workspace1 log second")
	for _, log := range workspace1Response.AuditLogs {
		assert.Equal(t, &workspace1ID, log.WorkspaceID)
	}

	workspace2Response, err := service.GetWorkspaceAuditLogs(t.Context(), workspace2ID, request)
	assert.NoError(t, err)
	assert.Equal(t, 2, len(workspace2Response.AuditLogs))

	messages2 := extractMessages(workspace2Response.AuditLogs)
	assert.Contains(t, messages2, "Test workspace2 log first")
	assert.Contains(t, messages2, "Test workspace2 log second")

	limitedResponse, err := service.GetWorkspaceAuditLogs(t.Context(), workspace1ID,
		&GetAuditLogsRequest{Limit: 1, Offset: 0})
	assert.NoError(t, err)
	assert.Equal(t, 1, len(limitedResponse.AuditLogs))
	assert.Equal(t, 1, limitedResponse.Limit)

	beforeTime := time.Now().UTC().Add(-1 * time.Minute)
	filteredResponse, err := service.GetWorkspaceAuditLogs(t.Context(), workspace1ID,
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

func Test_WriteAuditLog_PersistsRowAndEmitsToLogPipeline(t *testing.T) {
	capturedRecords := &bytes.Buffer{}
	user := users_testing.CreateTestUser(t.Context(), user_enums.UserRoleMember)
	workspaceID := uuid.New()

	service := &AuditLogService{
		GetAuditLogService().auditLogRepository,
		slog.New(slog.NewJSONHandler(io.Discard, nil)),
		slog.New(slog.NewJSONHandler(capturedRecords, nil)),
	}

	service.WriteAuditLog(t.Context(), audit_logs_models.AuditEntry{
		Message:     "Storage created: nightly-backups",
		UserID:      &user.UserID,
		WorkspaceID: &workspaceID,
	})

	var auditRecord map[string]any
	require.NoError(t, json.Unmarshal(capturedRecords.Bytes(), &auditRecord))
	assert.Equal(t, "Storage created: nightly-backups", auditRecord["msg"])
	assert.Equal(t, user.UserID.String(), auditRecord["user_id"])
	assert.Equal(t, workspaceID.String(), auditRecord["workspace_id"])

	persistedLogs, err := service.GetWorkspaceAuditLogs(t.Context(), workspaceID, &GetAuditLogsRequest{Limit: 10})
	require.NoError(t, err)
	assert.Contains(t, extractMessages(persistedLogs.AuditLogs), "Storage created: nightly-backups")
}

// A JSON sink cannot show this: the request id is attached by the fan-out handler from the context
// it is handed, so the capturing handler asserts on the context instead of the output.
func Test_WriteAuditLog_WhenRequestIsInFlight_EmitsUnderThatRequestID(t *testing.T) {
	loggedRequestIDs := &[]string{}
	user := users_testing.CreateTestUser(t.Context(), user_enums.UserRoleMember)
	workspaceID := uuid.New()

	service := &AuditLogService{
		GetAuditLogService().auditLogRepository,
		slog.New(slog.NewJSONHandler(io.Discard, nil)),
		slog.New(requestIDCapturer{loggedRequestIDs}),
	}

	ctx := logger.ContextWithRequestID(t.Context(), "3b0f7a52-6c11-4e39-9f7d-2a5c8e4b1d06")
	service.WriteAuditLog(ctx, audit_logs_models.AuditEntry{
		Message:     "Database deleted: payments",
		UserID:      &user.UserID,
		WorkspaceID: &workspaceID,
	})

	assert.Equal(t, []string{"3b0f7a52-6c11-4e39-9f7d-2a5c8e4b1d06"}, *loggedRequestIDs)
}

type requestIDCapturer struct {
	requestIDs *[]string
}

func (h requestIDCapturer) Enabled(context.Context, slog.Level) bool { return true }

func (h requestIDCapturer) WithAttrs([]slog.Attr) slog.Handler { return h }

func (h requestIDCapturer) WithGroup(string) slog.Handler { return h }

func (h requestIDCapturer) Handle(ctx context.Context, _ slog.Record) error {
	*h.requestIDs = append(*h.requestIDs, logger.GetRequestID(ctx))

	return nil
}

func createAuditLog(t *testing.T, service *AuditLogService, entry audit_logs_models.AuditEntry) {
	t.Helper()

	service.WriteAuditLog(t.Context(), entry)
}

func extractMessages(logs []*AuditLogDTO) []string {
	messages := make([]string, len(logs))
	for i, log := range logs {
		messages[i] = log.Message
	}
	return messages
}
