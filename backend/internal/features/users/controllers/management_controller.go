package users_controllers

import (
	"fmt"
	"net/http"

	user_dto "databasus-backend/internal/features/users/dto"
	user_enums "databasus-backend/internal/features/users/enums"
	user_middleware "databasus-backend/internal/features/users/middleware"
	users_services "databasus-backend/internal/features/users/services"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type ManagementController struct {
	managementService *users_services.UserManagementService
}

func (c *ManagementController) RegisterRoutes(router *gin.RouterGroup) {
	router.GET("/users", user_middleware.RequireRole(user_enums.UserRoleAdmin), c.GetUsers)
	router.GET("/users/:id", c.GetUserProfile)
	router.POST(
		"/users/:id/deactivate",
		user_middleware.RequireRole(user_enums.UserRoleAdmin),
		c.DeactivateUser,
	)
	router.POST(
		"/users/:id/activate",
		user_middleware.RequireRole(user_enums.UserRoleAdmin),
		c.ActivateUser,
	)
	router.PUT(
		"/users/:id/role",
		user_middleware.RequireRole(user_enums.UserRoleAdmin),
		c.ChangeUserRole,
	)
}

// ListUsers
// @Summary List users
// @Description Get list of users (admin only)
// @Tags user-management
// @Produce json
// @Security BearerAuth
// @Param limit query int false "Number of items per page" default(20)
// @Param offset query int false "Page offset" default(0)
// @Param beforeDate query string false "Filter users created before this date (RFC3339 format)" format(date-time)
// @Param query query string false "Search by email or name (case-insensitive)"
// @Success 200 {object} users_dto.ListUsersResponseDTO
// @Failure 401 {object} map[string]string "Unauthorized"
// @Failure 403 {object} map[string]string "Forbidden"
// @Router /users [get]
func (c *ManagementController) GetUsers(ctx *gin.Context) {
	fmt.Println("GetUsers")

	user, ok := user_middleware.GetUserFromContext(ctx)
	if !ok {
		ctx.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
		return
	}

	request := &user_dto.ListUsersRequestDTO{}
	if err := ctx.ShouldBindQuery(request); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "Invalid query parameters"})
		return
	}

	// Set defaults if not provided
	if request.Limit <= 0 || request.Limit > 100 {
		request.Limit = 20
	}
	if request.Offset < 0 {
		request.Offset = 0
	}

	users, total, err := c.managementService.GetUsers(
		user,
		request.Limit,
		request.Offset,
		request.BeforeDate,
		request.Query,
	)
	if err != nil {
		ctx.JSON(http.StatusForbidden, gin.H{"error": err.Error()})
		return
	}

	// Convert to response format
	userProfiles := make([]user_dto.UserProfileResponseDTO, len(users))
	for i, u := range users {
		userProfiles[i] = user_dto.UserProfileResponseDTO{
			ID:        u.ID,
			Email:     u.Email,
			Name:      u.Name,
			Role:      u.Role,
			IsActive:  u.IsActiveUser(),
			CreatedAt: u.CreatedAt,
		}
	}

	response := user_dto.ListUsersResponseDTO{
		Users: userProfiles,
		Total: total,
	}

	ctx.JSON(http.StatusOK, response)
}

// GetUserProfile
// @Summary Get user profile
// @Description Get user profile information (users can view own profile, admins can view any)
// @Tags user-management
// @Produce json
// @Security BearerAuth
// @Param id path string true "User ID"
// @Success 200 {object} users_dto.UserProfileResponseDTO
// @Failure 400 {object} map[string]string "Bad request"
// @Failure 401 {object} map[string]string "Unauthorized"
// @Failure 403 {object} map[string]string "Forbidden"
// @Router /users/{id} [get]
func (c *ManagementController) GetUserProfile(ctx *gin.Context) {
	currentUser, ok := user_middleware.GetUserFromContext(ctx)
	if !ok {
		ctx.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
		return
	}

	userIDStr := ctx.Param("id")
	userID, err := uuid.Parse(userIDStr)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "Invalid user ID"})
		return
	}

	user, err := c.managementService.GetUserProfile(userID, currentUser)
	if err != nil {
		ctx.JSON(http.StatusForbidden, gin.H{"error": err.Error()})
		return
	}

	profile := user_dto.UserProfileResponseDTO{
		ID:        user.ID,
		Email:     user.Email,
		Name:      user.Name,
		Role:      user.Role,
		IsActive:  user.IsActiveUser(),
		CreatedAt: user.CreatedAt,
	}

	ctx.JSON(http.StatusOK, profile)
}

// DeactivateUser
// @Summary Deactivate user
// @Description Deactivate a user account (admin only)
// @Tags user-management
// @Security BearerAuth
// @Param id path string true "User ID"
// @Success 200 {object} map[string]string
// @Failure 400 {object} map[string]string "Bad request"
// @Failure 401 {object} map[string]string "Unauthorized"
// @Failure 403 {object} map[string]string "Forbidden"
// @Router /users/{id}/deactivate [post]
func (c *ManagementController) DeactivateUser(ctx *gin.Context) {
	currentUser, ok := user_middleware.GetUserFromContext(ctx)
	if !ok {
		ctx.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
		return
	}

	userIDStr := ctx.Param("id")
	userID, err := uuid.Parse(userIDStr)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "Invalid user ID"})
		return
	}

	if err := c.managementService.DeactivateUser(userID, currentUser); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	ctx.JSON(http.StatusOK, gin.H{"message": "User deactivated successfully"})
}

// ActivateUser
// @Summary Activate user
// @Description Activate a user account (admin only)
// @Tags user-management
// @Security BearerAuth
// @Param id path string true "User ID"
// @Success 200 {object} map[string]string
// @Failure 400 {object} map[string]string "Bad request"
// @Failure 401 {object} map[string]string "Unauthorized"
// @Failure 403 {object} map[string]string "Forbidden"
// @Router /users/{id}/activate [post]
func (c *ManagementController) ActivateUser(ctx *gin.Context) {
	currentUser, ok := user_middleware.GetUserFromContext(ctx)
	if !ok {
		ctx.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
		return
	}

	userIDStr := ctx.Param("id")
	userID, err := uuid.Parse(userIDStr)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "Invalid user ID"})
		return
	}

	if err := c.managementService.ActivateUser(userID, currentUser); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	ctx.JSON(http.StatusOK, gin.H{"message": "User activated successfully"})
}

// ChangeUserRole
// @Summary Change user role
// @Description Change a user's role (admin only)
// @Tags user-management
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path string true "User ID"
// @Param request body users_dto.ChangeUserRoleRequestDTO true "Role change data"
// @Success 200 {object} map[string]string
// @Failure 400 {object} map[string]string "Bad request"
// @Failure 401 {object} map[string]string "Unauthorized"
// @Failure 403 {object} map[string]string "Forbidden"
// @Router /users/{id}/role [put]
func (c *ManagementController) ChangeUserRole(ctx *gin.Context) {
	currentUser, ok := user_middleware.GetUserFromContext(ctx)
	if !ok {
		ctx.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
		return
	}

	userIDStr := ctx.Param("id")
	userID, err := uuid.Parse(userIDStr)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "Invalid user ID"})
		return
	}

	var request user_dto.ChangeUserRoleRequestDTO
	if err := ctx.ShouldBindJSON(&request); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request format"})
		return
	}

	if err := c.managementService.ChangeUserRole(userID, request.Role, currentUser); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	ctx.JSON(http.StatusOK, gin.H{"message": "User role changed successfully"})
}
