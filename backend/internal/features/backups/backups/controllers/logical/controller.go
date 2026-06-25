package backups_controllers_logical

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	backups_core_logical "databasus-backend/internal/features/backups/backups/core/logical"
	"databasus-backend/internal/features/backups/backups/download/stream_guard"
	backups_dto_logical "databasus-backend/internal/features/backups/backups/dto/logical"
	backups_services "databasus-backend/internal/features/backups/backups/services"
	"databasus-backend/internal/features/databases"
	users_middleware "databasus-backend/internal/features/users/middleware"
	files_utils "databasus-backend/internal/util/files"
)

type BackupController struct {
	backupService *backups_services.LogicalBackupService
}

func (c *BackupController) RegisterRoutes(router *gin.RouterGroup) {
	router.GET("/backups", c.GetBackups)
	router.POST("/backups", c.MakeBackup)
	router.POST("/backups/:id/download-token", c.GenerateDownloadToken)
	router.DELETE("/backups/:id", c.DeleteBackup)
	router.POST("/backups/:id/cancel", c.CancelBackup)
}

// RegisterPublicRoutes registers routes that don't require Bearer authentication
// (they have their own authentication mechanisms like download tokens)
func (c *BackupController) RegisterPublicRoutes(router *gin.RouterGroup) {
	router.GET("/backups/:id/file", c.GetFile)
}

// GetBackups
// @Summary Get backups for a database
// @Description Get paginated backups for the specified database with optional filters
// @Tags backups
// @Produce json
// @Param database_id query string true "Database ID"
// @Param limit query int false "Number of items per page" default(10)
// @Param offset query int false "Offset for pagination" default(0)
// @Param status query []string false "Filter by backup status (can be repeated)" Enums(IN_PROGRESS, COMPLETED, FAILED, CANCELED)
// @Param beforeDate query string false "Filter backups created before this date (RFC3339)" format(date-time)
// @Param pgWalBackupType query string false "Filter by WAL backup type" Enums(PG_FULL_BACKUP, PG_WAL_SEGMENT)
// @Success 200 {object} backups_dto_logical.GetBackupsResponse
// @Failure 400
// @Failure 401
// @Failure 500
// @Router /backups [get]
func (c *BackupController) GetBackups(ctx *gin.Context) {
	user, ok := users_middleware.GetUserFromContext(ctx)
	if !ok {
		ctx.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
		return
	}

	var request backups_dto_logical.GetBackupsRequest
	if err := ctx.ShouldBindQuery(&request); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	databaseID, err := uuid.Parse(request.DatabaseID)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "invalid database_id"})
		return
	}

	filters := c.buildBackupFilters(&request)

	response, err := c.backupService.GetBackups(user, databaseID, request.Limit, request.Offset, filters)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	ctx.JSON(http.StatusOK, response)
}

// MakeBackup
// @Summary Create a backup
// @Description Create a new backup for the specified database
// @Tags backups
// @Accept json
// @Produce json
// @Param request body backups_dto_logical.MakeBackupRequest true "Backup creation data"
// @Success 200 {object} map[string]string
// @Failure 400
// @Failure 401
// @Failure 500
// @Router /backups [post]
func (c *BackupController) MakeBackup(ctx *gin.Context) {
	user, ok := users_middleware.GetUserFromContext(ctx)
	if !ok {
		ctx.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
		return
	}

	var request backups_dto_logical.MakeBackupRequest
	if err := ctx.ShouldBindJSON(&request); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if err := c.backupService.MakeBackupWithAuth(user, request.DatabaseID); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	ctx.JSON(http.StatusOK, gin.H{"message": "backup started successfully"})
}

// DeleteBackup
// @Summary Delete a backup
// @Description Delete an existing backup
// @Tags backups
// @Param id path string true "Backup ID"
// @Success 204
// @Failure 400
// @Failure 401
// @Failure 500
// @Router /backups/{id} [delete]
func (c *BackupController) DeleteBackup(ctx *gin.Context) {
	user, ok := users_middleware.GetUserFromContext(ctx)
	if !ok {
		ctx.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
		return
	}

	id, err := uuid.Parse(ctx.Param("id"))
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "invalid backup ID"})
		return
	}

	if err := c.backupService.DeleteBackup(user, id); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	ctx.Status(http.StatusNoContent)
}

// CancelBackup
// @Summary Cancel an in-progress backup
// @Description Cancel a backup that is currently in progress
// @Tags backups
// @Param id path string true "Backup ID"
// @Success 204
// @Failure 400
// @Failure 401
// @Failure 500
// @Router /backups/{id}/cancel [post]
func (c *BackupController) CancelBackup(ctx *gin.Context) {
	user, ok := users_middleware.GetUserFromContext(ctx)
	if !ok {
		ctx.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
		return
	}

	id, err := uuid.Parse(ctx.Param("id"))
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "invalid backup ID"})
		return
	}

	if err := c.backupService.CancelBackup(user, id); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	ctx.Status(http.StatusNoContent)
}

// GenerateDownloadToken
// @Summary Generate short-lived download token
// @Description Generate a token for downloading a backup file (valid for 5 minutes)
// @Tags backups
// @Param id path string true "Backup ID"
// @Success 200 {object} download_token.GenerateTokenResponse
// @Failure 400
// @Failure 401
// @Failure 409 {object} map[string]string "Download already in progress"
// @Router /backups/{id}/download-token [post]
func (c *BackupController) GenerateDownloadToken(ctx *gin.Context) {
	user, ok := users_middleware.GetUserFromContext(ctx)
	if !ok {
		ctx.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
		return
	}

	id, err := uuid.Parse(ctx.Param("id"))
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "invalid backup ID"})
		return
	}

	response, err := c.backupService.GenerateDownloadToken(user, id)
	if err != nil {
		if errors.Is(err, stream_guard.ErrDownloadAlreadyInProgress) {
			ctx.JSON(
				http.StatusConflict,
				gin.H{
					"error": "Download already in progress for some of backups. Please wait until previous download completed or cancel it",
				},
			)
			return
		}
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	ctx.JSON(http.StatusOK, response)
}

// GetFile
// @Summary Download a backup file
// @Description Download the backup file for the specified backup using a download token.
// @Description
// @Description **Download Concurrency Control:**
// @Description - Only one download per user is allowed at a time
// @Description - If a download is already in progress, returns 409 Conflict
// @Description - Downloads are tracked using cache with 5-second TTL and 3-second heartbeat
// @Description - Browser cancellations automatically release the download lock
// @Description - Server crashes are handled via automatic cache expiry (5 seconds)
// @Tags backups
// @Param id path string true "Backup ID"
// @Param token query string true "Download token"
// @Success 200 {file} file
// @Failure 400 {object} map[string]string
// @Failure 401 {object} map[string]string
// @Failure 409 {object} map[string]string "Download already in progress"
// @Failure 500 {object} map[string]string
// @Router /backups/{id}/file [get]
func (c *BackupController) GetFile(ctx *gin.Context) {
	token := ctx.Query("token")
	if token == "" {
		ctx.JSON(http.StatusUnauthorized, gin.H{"error": "download token is required"})
		return
	}

	backupIDParam := ctx.Param("id")
	backupID, err := uuid.Parse(backupIDParam)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "invalid backup ID"})
		return
	}

	downloadToken, err := c.backupService.ValidateDownloadToken(token)
	if err != nil {
		if errors.Is(err, stream_guard.ErrDownloadAlreadyInProgress) {
			ctx.JSON(
				http.StatusConflict,
				gin.H{
					"error": "download already in progress for this user. Please wait until previous download completed or cancel it",
				},
			)
			return
		}

		ctx.JSON(http.StatusUnauthorized, gin.H{"error": "invalid or expired download token"})
		return
	}

	if downloadToken.BackupID != backupID {
		ctx.JSON(http.StatusUnauthorized, gin.H{"error": "invalid or expired download token"})
		return
	}

	fileReader, backup, database, err := c.backupService.GetBackupFileWithoutAuth(
		downloadToken.BackupID,
	)
	if err != nil {
		c.backupService.ReleaseDownloadLock(downloadToken.UserID)
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	heartbeatCtx, cancelHeartbeat := context.WithCancel(context.Background())
	defer func() {
		cancelHeartbeat()
		c.backupService.ReleaseDownloadLock(downloadToken.UserID)
		if err := fileReader.Close(); err != nil {
			fmt.Printf("Error closing file reader: %v\n", err)
		}
	}()

	go c.startDownloadHeartbeat(heartbeatCtx, downloadToken.UserID)

	filename := c.generateBackupFilename(backup, database)

	if backup.BackupSizeMb > 0 {
		sizeBytes := int64(backup.BackupSizeMb * 1024 * 1024)
		ctx.Header("Content-Length", fmt.Sprintf("%d", sizeBytes))
	}

	ctx.Header("Content-Type", "application/octet-stream")
	ctx.Header(
		"Content-Disposition",
		fmt.Sprintf("attachment; filename=\"%s\"", filename),
	)

	_, err = io.Copy(ctx.Writer, fileReader)
	if err != nil {
		fmt.Printf("Error streaming file: %v\n", err)
	}

	c.backupService.WriteAuditLogForDownload(downloadToken.UserID, backup, database)
}

func (c *BackupController) generateBackupFilename(
	backup *backups_core_logical.LogicalBackup,
	database *databases.Database,
) string {
	// Format timestamp as YYYY-MM-DD_HH-mm-ss
	timestamp := backup.CreatedAt.Format("2006-01-02_15-04-05")

	// Sanitize database name for filename (replace spaces and special chars)
	safeName := files_utils.SanitizeFilename(database.Name)

	// Determine extension based on database type
	extension := c.getBackupExtension(database.Type)

	return fmt.Sprintf("%s_backup_%s%s", safeName, timestamp, extension)
}

func (c *BackupController) getBackupExtension(
	dbType databases.DatabaseType,
) string {
	switch dbType {
	case databases.DatabaseTypeMysql, databases.DatabaseTypeMariadb:
		return ".sql.zst"
	case databases.DatabaseTypePostgresLogical:
		// PostgreSQL custom format
		return ".dump"
	case databases.DatabaseTypeMongodb:
		return ".archive"
	default:
		return ".backup"
	}
}

func (c *BackupController) startDownloadHeartbeat(ctx context.Context, userID uuid.UUID) {
	ticker := time.NewTicker(stream_guard.GetDownloadHeartbeatInterval())
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			c.backupService.RefreshDownloadLock(userID)
		}
	}
}

func (c *BackupController) buildBackupFilters(
	request *backups_dto_logical.GetBackupsRequest,
) *backups_core_logical.BackupFilters {
	isHasFilters := len(request.Statuses) > 0 ||
		request.BeforeDate != nil

	if !isHasFilters {
		return nil
	}

	filters := &backups_core_logical.BackupFilters{}

	if len(request.Statuses) > 0 {
		statuses := make([]backups_core_logical.BackupStatus, 0, len(request.Statuses))
		for _, statusStr := range request.Statuses {
			statuses = append(statuses, backups_core_logical.BackupStatus(statusStr))
		}

		filters.Statuses = statuses
	}

	filters.BeforeDate = request.BeforeDate

	return filters
}
