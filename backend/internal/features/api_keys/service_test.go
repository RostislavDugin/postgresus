package api_keys

import (
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	users_enums "databasus-backend/internal/features/users/enums"
	users_testing "databasus-backend/internal/features/users/testing"
	workspaces_testing "databasus-backend/internal/features/workspaces/testing"
)

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
