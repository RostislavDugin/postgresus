package api_keys

import (
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	backups_core "databasus-backend/internal/features/backups/backups/core"
	backups_services "databasus-backend/internal/features/backups/backups/services"
	"databasus-backend/internal/features/databases"
	users_enums "databasus-backend/internal/features/users/enums"
	users_testing "databasus-backend/internal/features/users/testing"
	workspaces_testing "databasus-backend/internal/features/workspaces/testing"
)

func Test_BuildTriggerAuditMessage_AlwaysAttributesApiKeyAndReflectsOutcome(t *testing.T) {
	principal := &Principal{
		ApiKeyID: uuid.New(),
		Name:     "ci-key",
		Role:     users_enums.UserRoleAdmin,
	}
	database := &databases.Database{Name: "prod-db"}

	completedBackup := &backups_core.Backup{ID: uuid.New(), Status: backups_core.BackupStatusCompleted}
	failedBackup := &backups_core.Backup{ID: uuid.New(), Status: backups_core.BackupStatusFailed}
	canceledBackup := &backups_core.Backup{ID: uuid.New(), Status: backups_core.BackupStatusCanceled}
	inProgressBackup := &backups_core.Backup{ID: uuid.New(), Status: backups_core.BackupStatusInProgress}

	t.Run("completed reports success with backup id", func(t *testing.T) {
		message := buildTriggerAuditMessage(principal, database, completedBackup, nil)

		assert.Contains(t, message, principal.ApiKeyID.String())
		assert.Contains(t, message, "completed")
		assert.Contains(t, message, completedBackup.ID.String())
	})

	t.Run("timeout is not reported as success", func(t *testing.T) {
		message := buildTriggerAuditMessage(principal, database, inProgressBackup, backups_services.ErrBackupWaitTimeout)

		assert.Contains(t, message, principal.ApiKeyID.String())
		assert.Contains(t, message, "timeout")
		assert.NotContains(t, message, "completed")
	})

	t.Run("generic error is not reported as success", func(t *testing.T) {
		message := buildTriggerAuditMessage(principal, database, nil, errors.New("connection refused"))

		assert.Contains(t, message, principal.ApiKeyID.String())
		assert.Contains(t, message, "failed")
		assert.Contains(t, message, "connection refused")
		assert.NotContains(t, message, "completed")
	})

	t.Run("terminal failed is not reported as success", func(t *testing.T) {
		message := buildTriggerAuditMessage(principal, database, failedBackup, nil)

		assert.Contains(t, message, principal.ApiKeyID.String())
		assert.Contains(t, message, string(backups_core.BackupStatusFailed))
		assert.NotContains(t, message, "completed")
	})

	t.Run("terminal canceled is not reported as success", func(t *testing.T) {
		message := buildTriggerAuditMessage(principal, database, canceledBackup, nil)

		assert.Contains(t, message, principal.ApiKeyID.String())
		assert.Contains(t, message, string(backups_core.BackupStatusCanceled))
		assert.NotContains(t, message, "completed")
	})
}

func Test_GenerateToken_ProducesPrefixedTokenAndMatchingHash(t *testing.T) {
	plain, hashed, prefix, err := generateToken()

	require.NoError(t, err)
	assert.True(t, strings.HasPrefix(plain, TokenPrefix))
	assert.Equal(t, hashToken(plain), hashed)
	assert.True(t, strings.HasPrefix(plain, prefix))
	assert.NotEqual(t, plain, hashed)
}

func Test_AuthenticateToken_WhenTokenValid_ReturnsPrincipalWithGrants(t *testing.T) {
	plain, hashed, prefix, err := generateToken()
	require.NoError(t, err)

	creator := users_testing.CreateTestUser(users_enums.UserRoleAdmin)
	workspace, err := workspaces_testing.CreateTestWorkspaceDirect("API Key Auth WS", creator.UserID)
	require.NoError(t, err)
	t.Cleanup(func() { _ = workspaces_testing.RemoveTestWorkspaceDirect(workspace.ID) })

	repo := &ApiKeyRepository{}
	apiKey := &ApiKey{
		Name:            "Auth Test",
		HashedToken:     hashed,
		TokenPrefix:     prefix,
		Role:            users_enums.UserRoleMember,
		CreatedByUserID: creator.UserID,
		CreatedAt:       time.Now().UTC(),
	}
	require.NoError(t, repo.Create(apiKey, []uuid.UUID{workspace.ID}))
	t.Cleanup(func() { _ = repo.DeleteByID(apiKey.ID) })

	principal, err := GetApiKeyService().AuthenticateToken(plain)

	require.NoError(t, err)
	assert.Equal(t, users_enums.UserRoleMember, principal.Role)
	assert.Equal(t, []uuid.UUID{workspace.ID}, principal.WorkspaceIDs)
	assert.True(t, principal.CanAccessWorkspace(workspace.ID))
}

func Test_AuthenticateToken_WhenTokenRevoked_ReturnsInvalid(t *testing.T) {
	plain, hashed, prefix, err := generateToken()
	require.NoError(t, err)

	creator := users_testing.CreateTestUser(users_enums.UserRoleAdmin)

	repo := &ApiKeyRepository{}
	apiKey := &ApiKey{
		Name:            "Revoked Test",
		HashedToken:     hashed,
		TokenPrefix:     prefix,
		Role:            users_enums.UserRoleAdmin,
		CreatedByUserID: creator.UserID,
		CreatedAt:       time.Now().UTC(),
	}
	require.NoError(t, repo.Create(apiKey, nil))
	t.Cleanup(func() { _ = repo.DeleteByID(apiKey.ID) })
	require.NoError(t, repo.Revoke(apiKey.ID, time.Now().UTC()))

	_, err = GetApiKeyService().AuthenticateToken(plain)

	assert.ErrorIs(t, err, ErrInvalidApiKey)
}
