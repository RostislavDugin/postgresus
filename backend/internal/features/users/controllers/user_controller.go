package users_controllers

import (
	"net/http"

	"databasus-backend/internal/config"
	user_dto "databasus-backend/internal/features/users/dto"
	user_middleware "databasus-backend/internal/features/users/middleware"
	users_services "databasus-backend/internal/features/users/services"

	"github.com/gin-gonic/gin"
	"golang.org/x/time/rate"
)

type UserController struct {
	userService   *users_services.UserService
	signinLimiter *rate.Limiter
}

func (c *UserController) RegisterRoutes(router *gin.RouterGroup) {
	router.POST("/users/signup", c.SignUp)
	router.POST("/users/signin", c.SignIn)

	// Admin password setup (no auth required)
	router.GET("/users/admin/has-password", c.IsAdminHasPassword)
	router.POST("/users/admin/set-password", c.SetAdminPassword)

	// OAuth callbacks
	router.POST("/auth/github/callback", c.HandleGitHubOAuth)
	router.POST("/auth/google/callback", c.HandleGoogleOAuth)
}

func (c *UserController) RegisterProtectedRoutes(router *gin.RouterGroup) {
	router.GET("/users/me", c.GetCurrentUser)
	router.PUT("/users/me", c.UpdateUserInfo)
	router.PUT("/users/change-password", c.ChangePassword)
	router.POST("/users/invite", c.InviteUser)
}

func (c *UserController) SetSignInLimiter(limiter *rate.Limiter) {
	c.signinLimiter = limiter
}

// SignUp
// @Summary Register a new user
// @Description Register a new user with email and password
// @Tags users
// @Accept json
// @Produce json
// @Param request body users_dto.SignUpRequestDTO true "User signup data"
// @Success 200
// @Failure 400
// @Router /users/signup [post]
func (c *UserController) SignUp(ctx *gin.Context) {
	var request user_dto.SignUpRequestDTO
	if err := ctx.ShouldBindJSON(&request); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request format"})
		return
	}

	err := c.userService.SignUp(&request)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	ctx.JSON(http.StatusOK, gin.H{"message": "User created successfully"})
}

// SignIn
// @Summary Authenticate a user
// @Description Authenticate a user with email and password
// @Tags users
// @Accept json
// @Produce json
// @Param request body users_dto.SignInRequestDTO true "User signin data"
// @Success 200 {object} users_dto.SignInResponseDTO
// @Failure 400
// @Failure 429 {object} map[string]string "Rate limit exceeded"
// @Router /users/signin [post]
func (c *UserController) SignIn(ctx *gin.Context) {
	// We use rate limiter to prevent brute force attacks
	if !c.signinLimiter.Allow() {
		ctx.JSON(
			http.StatusTooManyRequests,
			gin.H{"error": "Rate limit exceeded. Please try again later."},
		)
		return
	}

	var request user_dto.SignInRequestDTO
	if err := ctx.ShouldBindJSON(&request); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request format"})
		return
	}

	response, err := c.userService.SignIn(&request)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	ctx.JSON(http.StatusOK, response)
}

// Admin password endpoints
func (c *UserController) IsAdminHasPassword(ctx *gin.Context) {
	hasPassword, err := c.userService.IsRootAdminHasPassword()
	if err != nil {
		ctx.JSON(
			http.StatusInternalServerError,
			gin.H{"error": err.Error()},
		)
		return
	}

	ctx.JSON(http.StatusOK, user_dto.IsAdminHasPasswordResponseDTO{HasPassword: hasPassword})
}

func (c *UserController) SetAdminPassword(ctx *gin.Context) {
	var request user_dto.SetAdminPasswordRequestDTO
	if err := ctx.ShouldBindJSON(&request); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if err := c.userService.SetRootAdminPassword(request.Password); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	ctx.JSON(http.StatusOK, gin.H{"message": "Admin password set successfully"})
}

// ChangePassword
// @Summary Change user password
// @Description Change the password for the currently authenticated user
// @Tags users
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param request body users_dto.ChangePasswordRequestDTO true "New password data"
// @Success 200 {object} map[string]string
// @Failure 400 {object} map[string]string
// @Failure 401 {object} map[string]string
// @Router /users/change-password [put]
func (c *UserController) ChangePassword(ctx *gin.Context) {
	user, ok := user_middleware.GetUserFromContext(ctx)
	if !ok {
		ctx.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
		return
	}

	var request user_dto.ChangePasswordRequestDTO
	if err := ctx.ShouldBindJSON(&request); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request format"})
		return
	}

	if request.NewPassword == "" {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "New password is required"})
		return
	}

	if len(request.NewPassword) < 8 {
		ctx.JSON(
			http.StatusBadRequest,
			gin.H{"error": "New password must be at least 8 characters long"},
		)
		return
	}

	if err := c.userService.ChangeUserPassword(user.ID, request.NewPassword); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	ctx.JSON(http.StatusOK, gin.H{"message": "Password changed successfully"})
}

// InviteUser
// @Summary Invite a new user
// @Description Invite a new user to the system with optional workspace assignment
// @Tags users
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param request body users_dto.InviteUserRequestDTO true "User invitation data"
// @Success 200 {object} users_dto.InviteUserResponseDTO
// @Failure 400 {object} map[string]string
// @Failure 401 {object} map[string]string
// @Failure 403 {object} map[string]string
// @Router /users/invite [post]
func (c *UserController) InviteUser(ctx *gin.Context) {
	user, ok := user_middleware.GetUserFromContext(ctx)
	if !ok {
		ctx.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
		return
	}

	var request user_dto.InviteUserRequestDTO
	if err := ctx.ShouldBindJSON(&request); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request format"})
		return
	}

	response, err := c.userService.InviteUser(&request, user)
	if err != nil {
		if err.Error() == "insufficient permissions to invite users" {
			ctx.JSON(http.StatusForbidden, gin.H{"error": err.Error()})
			return
		}
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	ctx.JSON(http.StatusOK, response)
}

// GetCurrentUser
// @Summary Get current user profile
// @Description Get the profile information of the currently authenticated user
// @Tags users
// @Accept json
// @Produce json
// @Security BearerAuth
// @Success 200 {object} users_dto.UserProfileResponseDTO
// @Failure 401 {object} map[string]string
// @Router /users/me [get]
func (c *UserController) GetCurrentUser(ctx *gin.Context) {
	user, ok := user_middleware.GetUserFromContext(ctx)
	if !ok {
		ctx.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
		return
	}

	profile := c.userService.GetCurrentUserProfile(user)
	ctx.JSON(http.StatusOK, profile)
}

// UpdateUserInfo
// @Summary Update current user information
// @Description Update name and/or email for the currently authenticated user
// @Tags users
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param request body users_dto.UpdateUserInfoRequestDTO true "User info update data"
// @Success 200 {object} map[string]string
// @Failure 400 {object} map[string]string
// @Failure 401 {object} map[string]string
// @Router /users/me [put]
func (c *UserController) UpdateUserInfo(ctx *gin.Context) {
	user, ok := user_middleware.GetUserFromContext(ctx)
	if !ok {
		ctx.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
		return
	}

	var request user_dto.UpdateUserInfoRequestDTO
	if err := ctx.ShouldBindJSON(&request); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request format"})
		return
	}

	if request.Name == nil && request.Email == nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "No fields to update"})
		return
	}

	if err := c.userService.UpdateUserInfo(user.ID, &request); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	ctx.JSON(http.StatusOK, gin.H{"message": "User info updated successfully"})
}

// HandleGitHubOAuth
// @Summary Handle GitHub OAuth callback
// @Description Exchange GitHub authorization code for JWT token
// @Tags auth
// @Accept json
// @Produce json
// @Param request body users_dto.OAuthCallbackRequestDTO true "OAuth callback data"
// @Success 200 {object} users_dto.OAuthCallbackResponseDTO
// @Failure 400 {object} map[string]string
// @Failure 501 {object} map[string]string
// @Router /auth/github/callback [post]
func (c *UserController) HandleGitHubOAuth(ctx *gin.Context) {
	env := config.GetEnv()
	if env.GitHubClientID == "" || env.GitHubClientSecret == "" {
		ctx.JSON(http.StatusNotImplemented, gin.H{"error": "GitHub OAuth is not configured"})
		return
	}

	var request user_dto.OAuthCallbackRequestDTO
	if err := ctx.ShouldBindJSON(&request); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request format"})
		return
	}

	response, err := c.userService.HandleGitHubOAuth(request.Code, request.RedirectUri)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	ctx.JSON(http.StatusOK, response)
}

// HandleGoogleOAuth
// @Summary Handle Google OAuth callback
// @Description Exchange Google authorization code for JWT token
// @Tags auth
// @Accept json
// @Produce json
// @Param request body users_dto.OAuthCallbackRequestDTO true "OAuth callback data"
// @Success 200 {object} users_dto.OAuthCallbackResponseDTO
// @Failure 400 {object} map[string]string
// @Failure 501 {object} map[string]string
// @Router /auth/google/callback [post]
func (c *UserController) HandleGoogleOAuth(ctx *gin.Context) {
	env := config.GetEnv()
	if env.GoogleClientID == "" || env.GoogleClientSecret == "" {
		ctx.JSON(http.StatusNotImplemented, gin.H{"error": "Google OAuth is not configured"})
		return
	}

	var request user_dto.OAuthCallbackRequestDTO
	if err := ctx.ShouldBindJSON(&request); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request format"})
		return
	}

	response, err := c.userService.HandleGoogleOAuth(request.Code, request.RedirectUri)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	ctx.JSON(http.StatusOK, response)
}
