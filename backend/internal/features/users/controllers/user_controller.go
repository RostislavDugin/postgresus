package users_controllers

import (
	"errors"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"

	"databasus-backend/internal/config"
	user_dto "databasus-backend/internal/features/users/dto"
	users_errors "databasus-backend/internal/features/users/errors"
	user_middleware "databasus-backend/internal/features/users/middleware"
	users_services "databasus-backend/internal/features/users/services"
	cache_utils "databasus-backend/internal/util/cache"
	cloudflare_turnstile "databasus-backend/internal/util/cloudflare_turnstile"
)

type UserController struct {
	userService *users_services.UserService
	rateLimiter *cache_utils.RateLimiter
}

func (c *UserController) RegisterRoutes(router *gin.RouterGroup) {
	router.POST("/users/signup", c.SignUp)
	router.POST("/users/signin", c.SignIn)

	// Admin password setup (no auth required)
	router.GET("/users/admin/has-password", c.IsAdminHasPassword)
	router.POST("/users/admin/set-password", c.SetAdminPassword)

	// Password reset (no auth required)
	router.POST("/users/send-reset-password-code", c.SendResetPasswordCode)
	router.POST("/users/reset-password", c.ResetPassword)

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

// SignUp
// @Summary Register a new user
// @Description Register a new user with email and password
// @Tags users
// @Accept json
// @Produce json
// @Param request body users_dto.SignUpRequestDTO true "User signup data"
// @Success 200 {object} users_dto.SignInResponseDTO
// @Failure 400
// @Router /users/signup [post]
func (c *UserController) SignUp(ctx *gin.Context) {
	var request user_dto.SignUpRequestDTO
	if err := ctx.ShouldBindJSON(&request); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request format"})
		return
	}

	// Verify Cloudflare Turnstile if enabled
	turnstileService := cloudflare_turnstile.GetCloudflareTurnstileService()
	if turnstileService.IsEnabled() {
		if request.CloudflareTurnstileToken == nil || *request.CloudflareTurnstileToken == "" {
			ctx.JSON(
				http.StatusBadRequest,
				gin.H{"error": "Cloudflare Turnstile verification required"},
			)
			return
		}

		clientIP := ctx.ClientIP()
		isValid, err := turnstileService.VerifyToken(*request.CloudflareTurnstileToken, clientIP)
		if err != nil || !isValid {
			ctx.JSON(
				http.StatusBadRequest,
				gin.H{"error": "Cloudflare Turnstile verification failed"},
			)
			return
		}
	}

	user, err := c.userService.SignUp(ctx.Request.Context(), &request)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	response, err := c.userService.GenerateAccessToken(ctx.Request.Context(), user)
	if err != nil {
		_ = ctx.Error(err)

		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to generate token"})
		return
	}

	ctx.JSON(http.StatusOK, response)
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
	var request user_dto.SignInRequestDTO
	if err := ctx.ShouldBindJSON(&request); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request format"})
		return
	}

	// Verify Cloudflare Turnstile if enabled
	turnstileService := cloudflare_turnstile.GetCloudflareTurnstileService()
	if turnstileService.IsEnabled() {
		if request.CloudflareTurnstileToken == nil || *request.CloudflareTurnstileToken == "" {
			ctx.JSON(
				http.StatusBadRequest,
				gin.H{"error": "Cloudflare Turnstile verification required"},
			)
			return
		}

		clientIP := ctx.ClientIP()
		isValid, err := turnstileService.VerifyToken(*request.CloudflareTurnstileToken, clientIP)
		if err != nil || !isValid {
			ctx.JSON(
				http.StatusBadRequest,
				gin.H{"error": "Cloudflare Turnstile verification failed"},
			)
			return
		}
	}

	allowed, _ := c.rateLimiter.CheckLimit(request.Email, "signin", 10, 1*time.Minute)
	if !allowed {
		ctx.JSON(
			http.StatusTooManyRequests,
			gin.H{"error": "Rate limit exceeded. Please try again later."},
		)
		return
	}

	response, err := c.userService.SignIn(ctx.Request.Context(), &request)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	ctx.JSON(http.StatusOK, response)
}

// Admin password endpoints
func (c *UserController) IsAdminHasPassword(ctx *gin.Context) {
	hasPassword, err := c.userService.IsRootAdminHasPassword(ctx.Request.Context())
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

	if err := c.userService.SetRootAdminPassword(ctx.Request.Context(), request.Password); err != nil {
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

	if err := c.userService.ChangeUserPassword(ctx.Request.Context(), user.ID, request.NewPassword); err != nil {
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

	response, err := c.userService.InviteUser(ctx.Request.Context(), &request, user)
	if err != nil {
		if errors.Is(err, users_errors.ErrInsufficientPermissionsToInviteUsers) {
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

	profile := c.userService.GetCurrentUserProfile(ctx.Request.Context(), user)
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

	if err := c.userService.UpdateUserInfo(ctx.Request.Context(), user.ID, &request); err != nil {
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

	response, err := c.userService.HandleGitHubOAuth(ctx.Request.Context(), request.Code, request.RedirectUri)
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

	response, err := c.userService.HandleGoogleOAuth(ctx.Request.Context(), request.Code, request.RedirectUri)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	ctx.JSON(http.StatusOK, response)
}

// SendResetPasswordCode
// @Summary Send password reset code
// @Description Send a password reset code to the user's email
// @Tags users
// @Accept json
// @Produce json
// @Param request body users_dto.SendResetPasswordCodeRequestDTO true "Email address"
// @Success 200 {object} map[string]string
// @Failure 400 {object} map[string]string
// @Failure 429 {object} map[string]string
// @Router /users/send-reset-password-code [post]
func (c *UserController) SendResetPasswordCode(ctx *gin.Context) {
	var request user_dto.SendResetPasswordCodeRequestDTO
	if err := ctx.ShouldBindJSON(&request); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request format"})
		return
	}

	// Verify Cloudflare Turnstile if enabled
	turnstileService := cloudflare_turnstile.GetCloudflareTurnstileService()
	if turnstileService.IsEnabled() {
		if request.CloudflareTurnstileToken == nil || *request.CloudflareTurnstileToken == "" {
			ctx.JSON(
				http.StatusBadRequest,
				gin.H{"error": "Cloudflare Turnstile verification required"},
			)
			return
		}

		clientIP := ctx.ClientIP()
		isValid, err := turnstileService.VerifyToken(*request.CloudflareTurnstileToken, clientIP)
		if err != nil || !isValid {
			ctx.JSON(
				http.StatusBadRequest,
				gin.H{"error": "Cloudflare Turnstile verification failed"},
			)
			return
		}
	}

	allowed, _ := c.rateLimiter.CheckLimit(
		request.Email,
		"reset-password",
		3,
		1*time.Hour,
	)
	if !allowed {
		ctx.JSON(
			http.StatusTooManyRequests,
			gin.H{"error": "Rate limit exceeded. Please try again later."},
		)
		return
	}

	err := c.userService.SendResetPasswordCode(ctx.Request.Context(), request.Email)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	ctx.JSON(http.StatusOK, gin.H{"message": "If the email exists, a reset code has been sent"})
}

// ResetPassword
// @Summary Reset password with code
// @Description Reset user password using the code sent via email
// @Tags users
// @Accept json
// @Produce json
// @Param request body users_dto.ResetPasswordRequestDTO true "Reset password data"
// @Success 200 {object} map[string]string
// @Failure 400 {object} map[string]string
// @Router /users/reset-password [post]
func (c *UserController) ResetPassword(ctx *gin.Context) {
	var request user_dto.ResetPasswordRequestDTO
	if err := ctx.ShouldBindJSON(&request); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request format"})
		return
	}

	err := c.userService.ResetPassword(ctx.Request.Context(), request.Email, request.Code, request.NewPassword)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	ctx.JSON(http.StatusOK, gin.H{"message": "Password reset successfully"})
}
