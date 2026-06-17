package api_keys

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	users_middleware "databasus-backend/internal/features/users/middleware"
)

type ApiKeyController struct {
	apiKeyService *ApiKeyService
}

func (c *ApiKeyController) RegisterRoutes(router *gin.RouterGroup) {
	apiKeyRoutes := router.Group("/api-keys")

	apiKeyRoutes.POST("", c.CreateApiKey)
	apiKeyRoutes.GET("", c.ListApiKeys)
	apiKeyRoutes.DELETE("/:id", c.RevokeApiKey)
}

// CreateApiKey
// @Summary Create an API key (ADMIN only)
// @Description Create a global API key. The raw token is returned once and never again.
// @Tags api-keys
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param request body CreateApiKeyRequestDTO true "API key to create"
// @Success 200 {object} CreateApiKeyResponseDTO
// @Failure 400 {object} map[string]string
// @Failure 401 {object} map[string]string
// @Failure 403 {object} map[string]string
// @Router /api-keys [post]
func (c *ApiKeyController) CreateApiKey(ctx *gin.Context) {
	user, isOk := users_middleware.GetUserFromContext(ctx)
	if !isOk {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "Invalid user context"})
		return
	}

	request := &CreateApiKeyRequestDTO{}
	if err := ctx.ShouldBindJSON(request); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body"})
		return
	}

	response, err := c.apiKeyService.CreateApiKey(user, request)
	if err != nil {
		switch {
		case errors.Is(err, ErrAdminOnly):
			ctx.JSON(http.StatusForbidden, gin.H{"error": err.Error()})
		case errors.Is(err, ErrInvalidRole), errors.Is(err, ErrWorkspacesRequired):
			ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		default:
			ctx.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create API key"})
		}
		return
	}

	ctx.JSON(http.StatusOK, response)
}

// ListApiKeys
// @Summary List API keys (ADMIN only)
// @Description List all active API keys. Tokens are never returned.
// @Tags api-keys
// @Produce json
// @Security BearerAuth
// @Success 200 {object} ListApiKeysResponseDTO
// @Failure 401 {object} map[string]string
// @Failure 403 {object} map[string]string
// @Router /api-keys [get]
func (c *ApiKeyController) ListApiKeys(ctx *gin.Context) {
	user, isOk := users_middleware.GetUserFromContext(ctx)
	if !isOk {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "Invalid user context"})
		return
	}

	response, err := c.apiKeyService.ListApiKeys(user)
	if err != nil {
		if errors.Is(err, ErrAdminOnly) {
			ctx.JSON(http.StatusForbidden, gin.H{"error": err.Error()})
			return
		}

		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to list API keys"})
		return
	}

	ctx.JSON(http.StatusOK, response)
}

// RevokeApiKey
// @Summary Revoke an API key (ADMIN only)
// @Description Soft-revoke an API key so it can no longer authenticate.
// @Tags api-keys
// @Produce json
// @Security BearerAuth
// @Param id path string true "API key ID"
// @Success 200 {object} map[string]string
// @Failure 401 {object} map[string]string
// @Failure 403 {object} map[string]string
// @Failure 404 {object} map[string]string
// @Router /api-keys/{id} [delete]
func (c *ApiKeyController) RevokeApiKey(ctx *gin.Context) {
	user, isOk := users_middleware.GetUserFromContext(ctx)
	if !isOk {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "Invalid user context"})
		return
	}

	apiKeyID, err := uuid.Parse(ctx.Param("id"))
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "Invalid API key ID"})
		return
	}

	if err := c.apiKeyService.RevokeApiKey(user, apiKeyID); err != nil {
		switch {
		case errors.Is(err, ErrAdminOnly):
			ctx.JSON(http.StatusForbidden, gin.H{"error": err.Error()})
		case errors.Is(err, ErrApiKeyNotFound):
			ctx.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		default:
			ctx.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to revoke API key"})
		}
		return
	}

	ctx.JSON(http.StatusOK, gin.H{"message": "API key revoked"})
}
