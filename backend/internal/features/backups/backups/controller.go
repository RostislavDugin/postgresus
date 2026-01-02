package backups

import (
	"databasus-backend/internal/features/backups/backups/common"
	"databasus-backend/internal/features/databases"
	users_middleware "databasus-backend/internal/features/users/middleware"
	"fmt"
	"io"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type BackupController struct {
	backupService *BackupService
}

func (c *BackupController) RegisterRoutes(router *gin.RouterGroup) {
	router.GET("/backups", c.GetBackups)
	router.POST("/backups", c.MakeBackup)
	router.GET("/backups/:id/file", c.GetFile)
	router.DELETE("/backups/:id", c.DeleteBackup)
	router.POST("/backups/:id/cancel", c.CancelBackup)
}

// GetBackups
// @Summary Get backups for a database
// @Description Get paginated backups for the specified database
// @Tags backups
// @Produce json
// @Param database_id query string true "Database ID"
// @Param limit query int false "Number of items per page" default(10)
// @Param offset query int false "Offset for pagination" default(0)
// @Success 200 {object} GetBackupsResponse
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

	var request GetBackupsRequest
	if err := ctx.ShouldBindQuery(&request); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	databaseID, err := uuid.Parse(request.DatabaseID)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "invalid database_id"})
		return
	}

	response, err := c.backupService.GetBackups(user, databaseID, request.Limit, request.Offset)
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
// @Param request body MakeBackupRequest true "Backup creation data"
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

	var request MakeBackupRequest
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

// GetFile
// @Summary Download a backup file
// @Description Download the backup file for the specified backup
// @Tags backups
// @Param id path string true "Backup ID"
// @Success 200 {file} file
// @Failure 400
// @Failure 401
// @Failure 500
// @Router /backups/{id}/file [get]
func (c *BackupController) GetFile(ctx *gin.Context) {
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

	fileReader, backup, database, err := c.backupService.GetBackupFile(user, id)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	defer func() {
		if err := fileReader.Close(); err != nil {
			fmt.Printf("Error closing file reader: %v\n", err)
		}
	}()

	filename := c.generateBackupFilename(backup, database)

	ctx.Header("Content-Type", "application/octet-stream")
	ctx.Header(
		"Content-Disposition",
		fmt.Sprintf("attachment; filename=\"%s\"", filename),
	)

	_, err = io.Copy(ctx.Writer, fileReader)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "failed to stream file"})
		return
	}
}

type MakeBackupRequest struct {
	DatabaseID uuid.UUID `json:"database_id" binding:"required"`
}

func (c *BackupController) generateBackupFilename(
	backup *Backup,
	database *databases.Database,
) string {
	// Format timestamp as YYYY-MM-DD_HH-mm-ss
	timestamp := backup.CreatedAt.Format("2006-01-02_15-04-05")

	// Sanitize database name for filename (replace spaces and special chars)
	safeName := sanitizeFilename(database.Name)

	// Determine extension based on database type and backup type
	extension := c.getBackupExtension(database.Type, backup.Type)

	return fmt.Sprintf("%s_backup_%s%s", safeName, timestamp, extension)
}

func (c *BackupController) getBackupExtension(
	dbType databases.DatabaseType,
	backupType common.BackupType,
) string {
	switch dbType {
	case databases.DatabaseTypeMysql, databases.DatabaseTypeMariadb:
		return ".sql.zst"
	case databases.DatabaseTypePostgres:
		// For PostgreSQL, use .tar for directory type, .dump for custom type
		if backupType == common.BackupTypeDirectory {
			return ".tar"
		}
		return ".dump"
	case databases.DatabaseTypeMongodb:
		return ".archive"
	default:
		return ".backup"
	}
}

func sanitizeFilename(name string) string {
	// Replace characters that are invalid in filenames
	replacer := map[rune]rune{
		' ':  '_',
		'/':  '-',
		'\\': '-',
		':':  '-',
		'*':  '-',
		'?':  '-',
		'"':  '-',
		'<':  '-',
		'>':  '-',
		'|':  '-',
	}

	result := make([]rune, 0, len(name))
	for _, char := range name {
		if replacement, exists := replacer[char]; exists {
			result = append(result, replacement)
		} else {
			result = append(result, char)
		}
	}

	return string(result)
}
