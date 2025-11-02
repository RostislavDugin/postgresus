package users_controllers

import (
	"net/http"
	"testing"

	users_dto "postgresus-backend/internal/features/users/dto"
	users_enums "postgresus-backend/internal/features/users/enums"
	users_middleware "postgresus-backend/internal/features/users/middleware"
	users_services "postgresus-backend/internal/features/users/services"
	users_testing "postgresus-backend/internal/features/users/testing"
	test_utils "postgresus-backend/internal/util/testing"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"golang.org/x/time/rate"
)

func Test_AdminLifecycleE2E_CompletesSuccessfully(t *testing.T) {
	router := createE2ETestRouter()

	users_testing.RecreateInitialAdmin()

	// 1. Set initial admin password
	adminPasswordRequest := users_dto.SetAdminPasswordRequestDTO{
		Password: "adminpassword123",
	}

	test_utils.MakePostRequest(
		t,
		router,
		"/api/v1/users/admin/set-password",
		"",
		adminPasswordRequest,
		http.StatusOK,
	)

	// 2. Admin signs in
	adminSigninRequest := users_dto.SignInRequestDTO{
		Email:    "admin",
		Password: "adminpassword123",
	}

	var adminSigninResponse users_dto.SignInResponseDTO
	test_utils.MakePostRequestAndUnmarshal(
		t,
		router,
		"/api/v1/users/signin",
		"",
		adminSigninRequest,
		http.StatusOK,
		&adminSigninResponse,
	)

	// 3. Admin invites a user
	workspaceID := uuid.New()
	workspaceRole := users_enums.WorkspaceRoleMember
	invitedUserEmail := "invited" + uuid.New().String() + "@example.com"
	inviteRequest := users_dto.InviteUserRequestDTO{
		Email:                 invitedUserEmail,
		IntendedWorkspaceID:   &workspaceID,
		IntendedWorkspaceRole: &workspaceRole,
	}
	test_utils.MakePostRequest(
		t,
		router,
		"/api/v1/users/invite",
		"Bearer "+adminSigninResponse.Token,
		inviteRequest,
		http.StatusOK,
	)

	// 4. Invited user signs up
	userSignupRequest := users_dto.SignUpRequestDTO{
		Email:    invitedUserEmail,
		Password: "userpassword123",
		Name:     "Invited User",
	}
	test_utils.MakePostRequest(
		t,
		router,
		"/api/v1/users/signup",
		"",
		userSignupRequest,
		http.StatusOK,
	)

	// 5. User signs in
	userSigninRequest := users_dto.SignInRequestDTO{
		Email:    invitedUserEmail,
		Password: "userpassword123",
	}

	var userSigninResponse users_dto.SignInResponseDTO
	test_utils.MakePostRequestAndUnmarshal(
		t,
		router,
		"/api/v1/users/signin",
		"",
		userSigninRequest,
		http.StatusOK,
		&userSigninResponse,
	)

	// 6. Admin lists users and sees new user
	var listUsersResponse users_dto.ListUsersResponseDTO
	test_utils.MakeGetRequestAndUnmarshal(
		t,
		router,
		"/api/v1/users",
		"Bearer "+adminSigninResponse.Token,
		http.StatusOK,
		&listUsersResponse,
	)
	assert.GreaterOrEqual(t, len(listUsersResponse.Users), 2) // Admin + new user
}

func Test_UserLifecycleE2E_CompletesSuccessfully(t *testing.T) {
	router := createE2ETestRouter()
	users_testing.ResetSettingsToDefaults()

	// 1. User registers
	userEmail := "testuser" + uuid.New().String() + "@example.com"
	userSignupRequest := users_dto.SignUpRequestDTO{
		Email:    userEmail,
		Password: "userpassword123",
		Name:     "Test User",
	}

	test_utils.MakePostRequest(
		t,
		router,
		"/api/v1/users/signup",
		"",
		userSignupRequest,
		http.StatusOK,
	)

	// 2. User signs in
	userSigninRequest := users_dto.SignInRequestDTO{
		Email:    userEmail,
		Password: "userpassword123",
	}

	var signinResponse users_dto.SignInResponseDTO
	test_utils.MakePostRequestAndUnmarshal(
		t,
		router,
		"/api/v1/users/signin",
		"",
		userSigninRequest,
		http.StatusOK,
		&signinResponse,
	)
	assert.NotEmpty(t, signinResponse.Token)
	assert.NotEqual(t, uuid.Nil, signinResponse.UserID)

	// 3. User gets own profile
	var profileResponse users_dto.UserProfileResponseDTO
	test_utils.MakeGetRequestAndUnmarshal(
		t,
		router,
		"/api/v1/users/"+signinResponse.UserID.String(),
		"Bearer "+signinResponse.Token,
		http.StatusOK,
		&profileResponse,
	)
	assert.Equal(t, signinResponse.UserID, profileResponse.ID)
	assert.Equal(t, userEmail, profileResponse.Email)
	assert.Equal(t, users_enums.UserRoleMember, profileResponse.Role)
	assert.True(t, profileResponse.IsActive)
}

// Test router creation helpers
func createUserTestRouter() *gin.Engine {
	gin.SetMode(gin.TestMode)
	router := gin.New()

	v1 := router.Group("/api/v1")

	// Register public routes
	GetUserController().RegisterRoutes(v1)

	// Register protected routes with auth middleware
	protected := v1.Group("").Use(users_middleware.AuthMiddleware(users_services.GetUserService()))
	GetUserController().RegisterProtectedRoutes(protected.(*gin.RouterGroup))
	GetUserController().SetSignInLimiter(rate.NewLimiter(rate.Limit(100), 100))

	// Setup audit log service
	users_services.GetUserService().SetAuditLogWriter(&AuditLogWriterStub{})

	return router
}

func createSettingsTestRouter() *gin.Engine {
	gin.SetMode(gin.TestMode)
	router := gin.New()

	v1 := router.Group("/api/v1")

	// Register protected routes with auth middleware
	protected := v1.Group("").Use(users_middleware.AuthMiddleware(users_services.GetUserService()))
	GetSettingsController().RegisterRoutes(protected.(*gin.RouterGroup))

	// Setup audit log service
	users_services.GetUserService().SetAuditLogWriter(&AuditLogWriterStub{})
	users_services.GetSettingsService().SetAuditLogWriter(&AuditLogWriterStub{})
	users_services.GetManagementService().SetAuditLogWriter(&AuditLogWriterStub{})

	return router
}

func createManagementTestRouter() *gin.Engine {
	gin.SetMode(gin.TestMode)
	router := gin.New()

	v1 := router.Group("/api/v1")

	// Register protected routes with auth middleware
	protected := v1.Group("").Use(users_middleware.AuthMiddleware(users_services.GetUserService()))
	GetManagementController().RegisterRoutes(protected.(*gin.RouterGroup))

	// Setup audit log service
	users_services.GetUserService().SetAuditLogWriter(&AuditLogWriterStub{})
	users_services.GetSettingsService().SetAuditLogWriter(&AuditLogWriterStub{})
	users_services.GetManagementService().SetAuditLogWriter(&AuditLogWriterStub{})

	return router
}

func createE2ETestRouter() *gin.Engine {
	gin.SetMode(gin.TestMode)
	router := gin.New()

	v1 := router.Group("/api/v1")

	// Register all routes
	GetUserController().RegisterRoutes(v1)

	// Register protected routes with auth middleware
	protected := v1.Group("").Use(users_middleware.AuthMiddleware(users_services.GetUserService()))
	GetUserController().RegisterProtectedRoutes(protected.(*gin.RouterGroup))
	GetSettingsController().RegisterRoutes(protected.(*gin.RouterGroup))
	GetManagementController().RegisterRoutes(protected.(*gin.RouterGroup))

	// Setup audit log service
	users_services.GetUserService().SetAuditLogWriter(&AuditLogWriterStub{})
	users_services.GetSettingsService().SetAuditLogWriter(&AuditLogWriterStub{})
	users_services.GetManagementService().SetAuditLogWriter(&AuditLogWriterStub{})

	return router
}

type AuditLogWriterStub struct{}

func (a *AuditLogWriterStub) WriteAuditLog(
	message string,
	userID *uuid.UUID,
	workspaceID *uuid.UUID,
) {
	// do nothing
}
