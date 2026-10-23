package users_controllers

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"

	user_dto "databasus-backend/internal/features/users/dto"
	users_oidc "databasus-backend/internal/features/users/oidc"
	"databasus-backend/internal/util/ratelimiter"
)

const (
	oidcStartRateLimitScope    = "oidc-start"
	oidcCallbackRateLimitScope = "oidc-callback"
)

// StartOIDCSignIn
// @Summary Start an OpenID Connect sign-in
// @Description Create a sign-in attempt and return the provider's authorization URL
// @Tags auth
// @Accept json
// @Produce json
// @Param request body users_dto.OIDCStartRequestDTO true "Redirect address the provider returns to"
// @Success 200 {object} users_dto.OIDCStartResponseDTO
// @Failure 400 {object} map[string]string
// @Failure 429 {object} map[string]string
// @Failure 501 {object} map[string]string
// @Router /auth/oidc/start [post]
func (c *UserController) StartOIDCSignIn(ctx *gin.Context) {
	if !users_oidc.GetProviderConfig().IsConfigured() {
		ctx.JSON(
			http.StatusNotImplemented,
			gin.H{"error": "OpenID Connect sign-in is not configured"},
		)

		return
	}

	var request user_dto.OIDCStartRequestDTO
	if err := ctx.ShouldBindJSON(&request); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request format"})
		return
	}

	if !checkRateLimitOrRespond(ctx, c.rateLimiter, c.logger, ratelimiter.Attempt{
		Scope:      oidcStartRateLimitScope,
		Identifier: ctx.ClientIP(),
		Limit:      20,
		Window:     time.Minute,
	}) {
		return
	}

	response, err := c.userService.StartOIDCSignIn(ctx.Request.Context(), request.RedirectUri)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	ctx.JSON(http.StatusOK, response)
}

// CompleteOIDCSignIn
// @Summary Complete an OpenID Connect sign-in
// @Description Verify the provider's authorization code against the attempt and issue a session token
// @Tags auth
// @Accept json
// @Produce json
// @Param request body users_dto.OIDCCallbackRequestDTO true "Authorization code and state"
// @Success 200 {object} users_dto.OAuthCallbackResponseDTO
// @Failure 400 {object} map[string]string
// @Failure 429 {object} map[string]string
// @Failure 501 {object} map[string]string
// @Router /auth/oidc/callback [post]
func (c *UserController) CompleteOIDCSignIn(ctx *gin.Context) {
	if !users_oidc.GetProviderConfig().IsConfigured() {
		ctx.JSON(
			http.StatusNotImplemented,
			gin.H{"error": "OpenID Connect sign-in is not configured"},
		)

		return
	}

	var request user_dto.OIDCCallbackRequestDTO
	if err := ctx.ShouldBindJSON(&request); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request format"})
		return
	}

	if !checkRateLimitOrRespond(ctx, c.rateLimiter, c.logger, ratelimiter.Attempt{
		Scope:      oidcCallbackRateLimitScope,
		Identifier: ctx.ClientIP(),
		Limit:      20,
		Window:     time.Minute,
	}) {
		return
	}

	response, err := c.userService.CompleteOIDCSignIn(ctx.Request.Context(), &request)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	ctx.JSON(http.StatusOK, response)
}
