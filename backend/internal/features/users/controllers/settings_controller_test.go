package users_controllers

import (
	"net/http"
	"testing"

	users_enums "databasus-backend/internal/features/users/enums"
	users_models "databasus-backend/internal/features/users/models"
	users_testing "databasus-backend/internal/features/users/testing"
	test_utils "databasus-backend/internal/util/testing"

	"github.com/stretchr/testify/assert"
)

func Test_GetUserSettings_WhenUserIsAdmin_ReturnsSettings(t *testing.T) {
	users_testing.ResetSettingsToDefaults()
	router := createSettingsTestRouter()

	// Create admin user and get token
	testUser := users_testing.CreateTestUser(users_enums.UserRoleAdmin)

	var response users_models.UsersSettings
	test_utils.MakeGetRequestAndUnmarshal(
		t,
		router,
		"/api/v1/users/settings",
		"Bearer "+testUser.Token,
		http.StatusOK,
		&response,
	)

	// Default settings should be true for all
	assert.True(t, response.IsAllowExternalRegistrations)
	assert.True(t, response.IsAllowMemberInvitations)
	assert.True(t, response.IsMemberAllowedToCreateWorkspaces)
}

func Test_GetUserSettings_WhenUserIsMember_ReturnsSettings(t *testing.T) {
	users_testing.ResetSettingsToDefaults()
	router := createSettingsTestRouter()

	// Create member user and get token
	testUser := users_testing.CreateTestUser(users_enums.UserRoleMember)

	_ = test_utils.MakeGetRequest(
		t,
		router,
		"/api/v1/users/settings",
		"Bearer "+testUser.Token,
		http.StatusOK,
	)
}

func Test_GetUserSettings_WithoutAuth_ReturnsUnauthorized(t *testing.T) {
	users_testing.ResetSettingsToDefaults()
	router := createSettingsTestRouter()

	test_utils.MakeGetRequest(t, router, "/api/v1/users/settings", "", http.StatusUnauthorized)
}

func Test_UpdateUserSettings_WhenUserIsAdmin_SettingsUpdated(t *testing.T) {
	users_testing.ResetSettingsToDefaults()
	router := createSettingsTestRouter()

	// Create admin user and get token
	testUser := users_testing.CreateTestUser(users_enums.UserRoleAdmin)

	// Update some settings
	request := users_models.UsersSettings{
		IsAllowExternalRegistrations:      false,
		IsAllowMemberInvitations:          true,
		IsMemberAllowedToCreateWorkspaces: false,
	}

	var response users_models.UsersSettings
	test_utils.MakePutRequestAndUnmarshal(
		t,
		router,
		"/api/v1/users/settings",
		"Bearer "+testUser.Token,
		request,
		http.StatusOK,
		&response,
	)

	// Check that settings were updated
	assert.False(t, response.IsAllowExternalRegistrations)
	assert.True(t, response.IsAllowMemberInvitations)
	assert.False(t, response.IsMemberAllowedToCreateWorkspaces)
}

func Test_UpdateUserSettings_WithPartialData_SettingsUpdated(t *testing.T) {
	users_testing.ResetSettingsToDefaults()
	router := createSettingsTestRouter()

	// Create admin user and get token
	testUser := users_testing.CreateTestUser(users_enums.UserRoleAdmin)

	// Update only one setting
	request := users_models.UsersSettings{
		IsAllowExternalRegistrations: false,
		// Other fields will use default values
		IsAllowMemberInvitations:          true,
		IsMemberAllowedToCreateWorkspaces: true,
	}

	var response users_models.UsersSettings
	test_utils.MakePutRequestAndUnmarshal(
		t,
		router,
		"/api/v1/users/settings",
		"Bearer "+testUser.Token,
		request,
		http.StatusOK,
		&response,
	)

	// Check that only the specified setting was updated
	assert.False(t, response.IsAllowExternalRegistrations)
	// These should remain true (default values)
	assert.True(t, response.IsAllowMemberInvitations)
	assert.True(t, response.IsMemberAllowedToCreateWorkspaces)
}

func Test_UpdateUserSettings_WhenUserIsMember_ReturnsForbidden(t *testing.T) {
	users_testing.ResetSettingsToDefaults()
	router := createSettingsTestRouter()

	// Create member user and get token
	testUser := users_testing.CreateTestUser(users_enums.UserRoleMember)

	request := users_models.UsersSettings{
		IsAllowExternalRegistrations: false,
	}

	resp := test_utils.MakePutRequest(
		t,
		router,
		"/api/v1/users/settings",
		"Bearer "+testUser.Token,
		request,
		http.StatusForbidden,
	)
	assert.Contains(t, string(resp.Body), "Insufficient permissions")
}

func Test_UpdateUserSettings_WithInvalidJSON_ReturnsBadRequest(t *testing.T) {
	users_testing.ResetSettingsToDefaults()
	router := createSettingsTestRouter()

	// Create admin user and get token
	testUser := users_testing.CreateTestUser(users_enums.UserRoleAdmin)

	// Test with invalid JSON structure
	resp := test_utils.MakeRequest(t, router, test_utils.RequestOptions{
		Method:         "PUT",
		URL:            "/api/v1/users/settings",
		Body:           "invalid json",
		AuthToken:      "Bearer " + testUser.Token,
		ExpectedStatus: http.StatusBadRequest,
	})

	assert.Contains(t, string(resp.Body), "Invalid request format")
}

func Test_UpdateUserSettings_WithoutAuth_ReturnsUnauthorized(t *testing.T) {
	users_testing.ResetSettingsToDefaults()
	router := createSettingsTestRouter()

	request := users_models.UsersSettings{
		IsAllowExternalRegistrations: false,
	}

	test_utils.MakePutRequest(
		t,
		router,
		"/api/v1/users/settings",
		"",
		request,
		http.StatusUnauthorized,
	)
}
