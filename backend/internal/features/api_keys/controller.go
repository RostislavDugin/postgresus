package api_keys

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	backups_core "databasus-backend/internal/features/backups/backups/core"
	backups_services "databasus-backend/internal/features/backups/backups/services"
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

func (c *ApiKeyController) RegisterPublicRoutes(router *gin.RouterGroup) {
	publicRoutes := router.Group("/public")

	publicRoutes.POST("/backups", c.TriggerBackup)
	publicRoutes.GET("/backups/:id", c.GetBackupStatus)
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
// @Failure 400 {object} map[string]string
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

// TriggerBackup
// @Summary Trigger a database backup (API key auth)
// @Description Start a backup and block until it finishes or the configured timeout elapses. Auth via API key in the Authorization header. A 422 with a populated `status` reports a terminal backup outcome whose `status` is one of `FAILED` or `CANCELED` (with `failMessage`), whereas a 422 with `error` set means the backup could not be started (misconfiguration). An optional `timeoutSeconds` in the body caps how long the call blocks (clamped to [1, server max]); when it elapses you get a 202 with a `backupId`, which you poll via `GET /public/backups/{id}`. Agent-managed (Postgres WAL_V1) databases are not supported by this endpoint and return 422, since they back up through their agent.
// @Tags api-keys
// @Accept json
// @Produce json
// @Security ApiKeyAuth
// @Param request body TriggerBackupRequestDTO true "Database to back up"
// @Success 200 {object} TriggerBackupResponseDTO "Backup completed"
// @Success 202 {object} TriggerBackupResponseDTO "Backup still running after timeout"
// @Failure 400 {object} map[string]string
// @Failure 401 {object} map[string]string
// @Failure 403 {object} map[string]string
// @Failure 404 {object} map[string]string
// @Failure 422 {object} TriggerBackupResponseDTO
// @Failure 500 {object} map[string]string
// @Router /public/backups [post]
func (c *ApiKeyController) TriggerBackup(ctx *gin.Context) {
	principal, isOk := GetPrincipalFromContext(ctx)
	if !isOk {
		ctx.JSON(http.StatusUnauthorized, gin.H{"error": "API key authentication required"})
		return
	}

	request := &TriggerBackupRequestDTO{}
	if err := ctx.ShouldBindJSON(request); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body"})
		return
	}

	backup, err := c.apiKeyService.TriggerBackupForPrincipal(
		ctx.Request.Context(),
		principal,
		request.DatabaseID,
		request.TimeoutSeconds,
	)

	switch {
	case errors.Is(err, backups_services.ErrBackupWaitTimeout):
		ctx.JSON(http.StatusAccepted, toTriggerBackupResponse(backup))
		return
	case errors.Is(err, ErrForbidden):
		ctx.JSON(http.StatusForbidden, gin.H{"error": err.Error()})
		return
	case errors.Is(err, ErrDatabaseNotFound):
		ctx.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	case errors.Is(err, ErrDatabaseWithoutWorkspace):
		ctx.JSON(http.StatusUnprocessableEntity, triggerBackupErrorResponse(err.Error()))
		return
	case errors.Is(err, backups_services.ErrAgentManagedBackup):
		ctx.JSON(http.StatusUnprocessableEntity, triggerBackupErrorResponse(err.Error()))
		return
	case errors.Is(err, backups_services.ErrBackupNotStarted):
		ctx.JSON(
			http.StatusUnprocessableEntity,
			triggerBackupErrorResponse(
				"backup could not be started for this database (check that backups and storage are configured)",
			),
		)
		return
	case err != nil:
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to trigger backup"})
		return
	}

	if backup.Status == backups_core.BackupStatusCompleted {
		ctx.JSON(http.StatusOK, toTriggerBackupResponse(backup))
		return
	}

	ctx.JSON(http.StatusUnprocessableEntity, toTriggerBackupResponse(backup))
}

// GetBackupStatus
// @Summary Get a backup's status (API key auth)
// @Description Poll a backup's current status using an API key. Use this after a 202 from the trigger endpoint to learn the backup's final outcome. Auth via API key in the Authorization header.
// @Tags api-keys
// @Produce json
// @Security ApiKeyAuth
// @Param id path string true "Backup ID"
// @Success 200 {object} TriggerBackupResponseDTO
// @Failure 400 {object} map[string]string
// @Failure 401 {object} map[string]string
// @Failure 403 {object} map[string]string
// @Failure 404 {object} map[string]string
// @Failure 422 {object} TriggerBackupResponseDTO
// @Failure 500 {object} map[string]string
// @Router /public/backups/{id} [get]
func (c *ApiKeyController) GetBackupStatus(ctx *gin.Context) {
	principal, isOk := GetPrincipalFromContext(ctx)
	if !isOk {
		ctx.JSON(http.StatusUnauthorized, gin.H{"error": "API key authentication required"})
		return
	}

	backupID, err := uuid.Parse(ctx.Param("id"))
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "Invalid backup ID"})
		return
	}

	backup, err := c.apiKeyService.GetBackupStatusForPrincipal(principal, backupID)

	switch {
	case errors.Is(err, ErrBackupNotFound), errors.Is(err, ErrDatabaseNotFound):
		ctx.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	case errors.Is(err, ErrForbidden):
		ctx.JSON(http.StatusForbidden, gin.H{"error": err.Error()})
		return
	case errors.Is(err, ErrDatabaseWithoutWorkspace):
		ctx.JSON(http.StatusUnprocessableEntity, triggerBackupErrorResponse(err.Error()))
		return
	case err != nil:
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get backup status"})
		return
	}

	ctx.JSON(http.StatusOK, toTriggerBackupResponse(backup))
}

func toTriggerBackupResponse(backup *backups_core.Backup) TriggerBackupResponseDTO {
	return TriggerBackupResponseDTO{
		BackupID:                  backup.ID,
		Status:                    backup.Status,
		FailMessage:               backup.FailMessage,
		RestoreVerificationStatus: backup.RestoreVerificationStatus,
	}
}

func triggerBackupErrorResponse(message string) TriggerBackupResponseDTO {
	return TriggerBackupResponseDTO{Error: &message}
}
