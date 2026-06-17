package api_keys

import (
	"net/http"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	users_enums "databasus-backend/internal/features/users/enums"
	users_testing "databasus-backend/internal/features/users/testing"
	test_utils "databasus-backend/internal/util/testing"
)

func Test_CreateApiKey_WhenUserIsAdmin_ReturnsTokenOnce(t *testing.T) {
	router := CreateApiKeyTestRouter()
	admin := users_testing.CreateTestUser(users_enums.UserRoleAdmin)

	request := CreateApiKeyRequestDTO{Name: "CI Key", Role: users_enums.UserRoleAdmin}
	var response CreateApiKeyResponseDTO
	test_utils.MakePostRequestAndUnmarshal(
		t, router, "/api/v1/api-keys", "Bearer "+admin.Token, request, http.StatusOK, &response,
	)
	t.Cleanup(func() { _ = (&ApiKeyRepository{}).DeleteByID(response.ID) })

	assert.Equal(t, "CI Key", response.Name)
	assert.NotEmpty(t, response.Token)
	assert.Contains(t, response.Token, TokenPrefix)
}

func Test_CreateApiKey_WhenUserIsMember_ReturnsForbidden(t *testing.T) {
	router := CreateApiKeyTestRouter()
	member := users_testing.CreateTestUser(users_enums.UserRoleMember)

	request := CreateApiKeyRequestDTO{Name: "Nope", Role: users_enums.UserRoleAdmin}
	var errorResponse map[string]string
	test_utils.MakePostRequestAndUnmarshal(
		t, router, "/api/v1/api-keys", "Bearer "+member.Token, request, http.StatusForbidden, &errorResponse,
	)

	assert.Equal(t, ErrAdminOnly.Error(), errorResponse["error"])
}

func Test_CreateApiKey_WhenMemberRoleWithoutWorkspaces_ReturnsBadRequest(t *testing.T) {
	router := CreateApiKeyTestRouter()
	admin := users_testing.CreateTestUser(users_enums.UserRoleAdmin)

	request := CreateApiKeyRequestDTO{Name: "No grants", Role: users_enums.UserRoleMember}
	var errorResponse map[string]string
	test_utils.MakePostRequestAndUnmarshal(
		t, router, "/api/v1/api-keys", "Bearer "+admin.Token, request, http.StatusBadRequest, &errorResponse,
	)

	assert.Equal(t, ErrWorkspacesRequired.Error(), errorResponse["error"])
}

func Test_ListAndRevokeApiKey_WhenUserIsAdmin_Works(t *testing.T) {
	router := CreateApiKeyTestRouter()
	admin := users_testing.CreateTestUser(users_enums.UserRoleAdmin)

	createRequest := CreateApiKeyRequestDTO{Name: "Listable", Role: users_enums.UserRoleAdmin}
	var created CreateApiKeyResponseDTO
	test_utils.MakePostRequestAndUnmarshal(
		t, router, "/api/v1/api-keys", "Bearer "+admin.Token, createRequest, http.StatusOK, &created,
	)
	t.Cleanup(func() { _ = (&ApiKeyRepository{}).DeleteByID(created.ID) })

	var list ListApiKeysResponseDTO
	test_utils.MakeGetRequestAndUnmarshal(
		t, router, "/api/v1/api-keys", "Bearer "+admin.Token, http.StatusOK, &list,
	)
	require.NotEmpty(t, list.ApiKeys)

	test_utils.MakeDeleteRequest(
		t, router, "/api/v1/api-keys/"+created.ID.String(), "Bearer "+admin.Token, http.StatusOK,
	)

	revoked, err := (&ApiKeyRepository{}).FindByHashedToken(hashToken(created.Token))
	require.NoError(t, err)
	assert.NotNil(t, revoked.RevokedAt)
}

func Test_RevokeApiKey_WhenIdUnknown_ReturnsNotFound(t *testing.T) {
	router := CreateApiKeyTestRouter()
	admin := users_testing.CreateTestUser(users_enums.UserRoleAdmin)

	test_utils.MakeDeleteRequest(
		t, router, "/api/v1/api-keys/"+uuid.New().String(), "Bearer "+admin.Token, http.StatusNotFound,
	)
}
