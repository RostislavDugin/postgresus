package backups_controllers_physical

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"databasus-backend/internal/features/backups/backups/core/physical/chain_view"
	"databasus-backend/internal/features/backups/backups/download/restore_token"
	"databasus-backend/internal/features/backups/backups/download/stream_guard"
	backups_dto_physical "databasus-backend/internal/features/backups/backups/dto/physical"
	backups_services "databasus-backend/internal/features/backups/backups/services"
	users_middleware "databasus-backend/internal/features/users/middleware"
	users_models "databasus-backend/internal/features/users/models"
)

type PhysicalBackupController struct {
	physicalBackupService *backups_services.PhysicalBackupService
	restoreTokenService   *restore_token.Service
}

func (c *PhysicalBackupController) RegisterRoutes(router *gin.RouterGroup) {
	router.GET("/backups/physical/database/:id/backups", c.GetBackups)
	router.POST("/backups/physical/database/:id/trigger", c.TriggerBackup)
	router.POST("/backups/physical/database/:id/restore-token", c.GenerateRestoreToken)

	router.POST("/backups/physical/backups/:backupId/restore-token", c.GenerateBackupRestoreToken)
	router.POST("/backups/physical/backups/:backupId/cancel", c.CancelBackup)
	router.DELETE("/backups/physical/backups/:backupId", c.DeleteBackup)
}

func (c *PhysicalBackupController) RegisterPublicRoutes(router *gin.RouterGroup) {
	router.GET("/backups/physical/restore-stream", c.GetRestoreStream)
	router.GET("/backups/physical/recovery-script", c.GetRecoveryScript)
}

// GetRecoveryScript
// @Summary Download the physical-restore helper script
// @Description Returns a POSIX sh script that takes a restore-stream URL (with its token) or a local bundle path, decompresses the backup blobs and WAL with the host's zstd up front, runs pg_combinebackup and wires up recovery with a plain-cp restore_command so the started cluster needs no extra tools. Pass --combine-image <postgres-image> to run pg_combinebackup in a container when the host lacks the PostgreSQL tools. It carries no secrets, so it needs no auth.
// @Tags backups-physical
// @Produce plain
// @Success 200 {string} string "shell script"
// @Router /backups/physical/recovery-script [get]
func (c *PhysicalBackupController) GetRecoveryScript(ctx *gin.Context) {
	ctx.Header("Content-Type", "text/x-shellscript; charset=utf-8")
	ctx.Header("Content-Disposition", `inline; filename="databasus-recovery.sh"`)
	ctx.String(http.StatusOK, recoveryScript)
}

// GetBackups
// @Summary List physical backups for a database
// @Description Returns a page of physical backups — FULLs, incrementals and committed WAL segments — as one flat list newest-first, with the database's total on-disk usage and the total count for pagination. The frontend filters by type.
// @Tags backups-physical
// @Produce json
// @Security BearerAuth
// @Param id path string true "Database ID"
// @Param limit query int false "Page size (default 50, max 1000)"
// @Param offset query int false "Offset for pagination" default(0)
// @Param type query []string false "Filter by backup type - repeatable, matches any" Enums(FULL, INCREMENTAL, WAL) collectionFormat(multi)
// @Param status query []string false "Filter by status - repeatable, matches any (e.g. COMPLETED, IN_PROGRESS)" collectionFormat(multi)
// @Param beforeDate query string false "Keep only backups created before this date (RFC3339)" format(date-time)
// @Success 200 {object} backups_dto_physical.GetPhysicalBackupsResponse
// @Failure 400 {object} map[string]string
// @Failure 401 {object} map[string]string
// @Router /backups/physical/database/{id}/backups [get]
func (c *PhysicalBackupController) GetBackups(ctx *gin.Context) {
	user, ok := users_middleware.GetUserFromContext(ctx)
	if !ok {
		ctx.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
		return
	}

	databaseID, err := uuid.Parse(ctx.Param("id"))
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "invalid database ID"})
		return
	}

	var request backups_dto_physical.GetPhysicalBackupsRequest
	if err := ctx.ShouldBindQuery(&request); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	response, err := c.physicalBackupService.GetBackups(user, databaseID, &request)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	ctx.JSON(http.StatusOK, response)
}

// TriggerBackup
// @Summary Trigger a physical backup
// @Description Requests an out-of-cadence backup, honored on the scheduler's next tick. type=full forces a new full; type=incremental extends the current chain (409 if none); type=auto lets the scheduler pick.
// @Tags backups-physical
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path string true "Database ID"
// @Param request body backups_dto_physical.TriggerBackupRequest true "Backup type to trigger"
// @Success 202 {object} map[string]string
// @Failure 400 {object} map[string]string
// @Failure 401 {object} map[string]string
// @Failure 409 {object} map[string]string
// @Router /backups/physical/database/{id}/trigger [post]
func (c *PhysicalBackupController) TriggerBackup(ctx *gin.Context) {
	user, ok := users_middleware.GetUserFromContext(ctx)
	if !ok {
		ctx.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
		return
	}

	databaseID, err := uuid.Parse(ctx.Param("id"))
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "invalid database ID"})
		return
	}

	var request backups_dto_physical.TriggerBackupRequest
	if err := ctx.ShouldBindJSON(&request); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if err := c.requestBackupOfType(user, databaseID, request.Type); err != nil {
		if errors.Is(err, backups_services.ErrNoExtendableChain) {
			ctx.JSON(http.StatusConflict, gin.H{"error": err.Error()})
			return
		}

		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	ctx.JSON(http.StatusAccepted, gin.H{"message": "backup requested"})
}

// GenerateRestoreToken
// @Summary Issue a restore-stream token
// @Description Issues a single-use, short-TTL token that authorizes the agent-less restore stream. Omit targetTime to restore to the latest available point.
// @Tags backups-physical
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path string true "Database ID"
// @Param request body backups_dto_physical.GenerateRestoreTokenRequest true "Restore target"
// @Success 200 {object} backups_dto_physical.GenerateRestoreTokenResponse
// @Failure 400 {object} map[string]string
// @Failure 401 {object} map[string]string
// @Failure 409 {object} map[string]string
// @Router /backups/physical/database/{id}/restore-token [post]
func (c *PhysicalBackupController) GenerateRestoreToken(ctx *gin.Context) {
	user, ok := users_middleware.GetUserFromContext(ctx)
	if !ok {
		ctx.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
		return
	}

	databaseID, err := uuid.Parse(ctx.Param("id"))
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "invalid database ID"})
		return
	}

	var request backups_dto_physical.GenerateRestoreTokenRequest
	if err := ctx.ShouldBindJSON(&request); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	token, err := c.physicalBackupService.GenerateRestoreToken(user, databaseID, &request)
	if err != nil {
		if errors.Is(err, stream_guard.ErrDownloadAlreadyInProgress) {
			ctx.JSON(http.StatusConflict, gin.H{"error": err.Error()})
			return
		}

		// An unreachable target (WAL gap, no chain, target before earliest) is
		// resolved at issue time, so surface it here with the same status the
		// stream endpoint would have returned.
		if status, message, isResolverError := classifyRestoreStreamError(err); isResolverError {
			ctx.JSON(status, gin.H{"error": message})
			return
		}

		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	ctx.JSON(http.StatusOK, backups_dto_physical.GenerateRestoreTokenResponse{
		Token: token,
		URL:   "/api/v1/backups/physical/restore-stream?token=" + token,
	})
}

// GenerateBackupRestoreToken
// @Summary Issue a restore-stream token for a specific backup
// @Description Issues a single-use token authorizing a per-backup restore stream: a FULL restores just itself; an incremental restores its FULL plus its incremental ancestors. No WAL replay is involved.
// @Tags backups-physical
// @Produce json
// @Security BearerAuth
// @Param backupId path string true "Backup ID (FULL or incremental)"
// @Success 200 {object} backups_dto_physical.GenerateRestoreTokenResponse
// @Failure 400 {object} map[string]string
// @Failure 401 {object} map[string]string
// @Failure 404 {object} map[string]string
// @Failure 409 {object} map[string]string
// @Router /backups/physical/backups/{backupId}/restore-token [post]
func (c *PhysicalBackupController) GenerateBackupRestoreToken(ctx *gin.Context) {
	user, ok := users_middleware.GetUserFromContext(ctx)
	if !ok {
		ctx.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
		return
	}

	backupID, err := uuid.Parse(ctx.Param("backupId"))
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "invalid backup ID"})
		return
	}

	token, err := c.physicalBackupService.GenerateBackupRestoreToken(user, backupID)
	if err != nil {
		if errors.Is(err, backups_services.ErrBackupNotFound) {
			ctx.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
			return
		}

		if errors.Is(err, stream_guard.ErrDownloadAlreadyInProgress) {
			ctx.JSON(http.StatusConflict, gin.H{"error": err.Error()})
			return
		}

		// A non-restorable backup (missing chain, not COMPLETED) is resolved up
		// front; surface it with the same status the stream endpoint would.
		if status, message, isResolverError := classifyRestoreStreamError(err); isResolverError {
			ctx.JSON(status, gin.H{"error": message})
			return
		}

		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	ctx.JSON(http.StatusOK, backups_dto_physical.GenerateRestoreTokenResponse{
		Token: token,
		URL:   "/api/v1/backups/physical/restore-stream?token=" + token,
	})
}

// CancelBackup
// @Summary Cancel an in-progress physical backup
// @Description Stops a FULL or incremental backup that is currently running and releases its in-flight claim. Returns 400 if the backup is not in progress.
// @Tags backups-physical
// @Security BearerAuth
// @Param backupId path string true "Backup ID"
// @Success 204
// @Failure 400 {object} map[string]string
// @Failure 401 {object} map[string]string
// @Failure 404 {object} map[string]string
// @Router /backups/physical/backups/{backupId}/cancel [post]
func (c *PhysicalBackupController) CancelBackup(ctx *gin.Context) {
	user, ok := users_middleware.GetUserFromContext(ctx)
	if !ok {
		ctx.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
		return
	}

	backupID, err := uuid.Parse(ctx.Param("backupId"))
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "invalid backup ID"})
		return
	}

	if err := c.physicalBackupService.CancelBackup(user, backupID); err != nil {
		if errors.Is(err, backups_services.ErrBackupNotFound) {
			ctx.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
			return
		}

		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	ctx.Status(http.StatusNoContent)
}

// DeleteBackup
// @Summary Delete a physical backup and its dependent cascade
// @Description Deletes a backup. A FULL removes its whole chain (incrementals + WAL); an incremental removes it and its descendant incrementals (WAL kept); a WAL segment removes it and the later WAL up to the next FULL/incremental. A running backup in the removed set is cancelled first.
// @Tags backups-physical
// @Security BearerAuth
// @Param backupId path string true "Backup ID (FULL, incremental or WAL segment)"
// @Success 204
// @Failure 400 {object} map[string]string
// @Failure 401 {object} map[string]string
// @Failure 404 {object} map[string]string
// @Router /backups/physical/backups/{backupId} [delete]
func (c *PhysicalBackupController) DeleteBackup(ctx *gin.Context) {
	user, ok := users_middleware.GetUserFromContext(ctx)
	if !ok {
		ctx.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
		return
	}

	backupID, err := uuid.Parse(ctx.Param("backupId"))
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "invalid backup ID"})
		return
	}

	if err := c.physicalBackupService.DeleteBackup(user, backupID); err != nil {
		if errors.Is(err, backups_services.ErrBackupNotFound) {
			ctx.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
			return
		}

		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	ctx.Status(http.StatusNoContent)
}

// GetRestoreStream
// @Summary Stream a ready-to-restore directory as a tar
// @Description Resolves the restore set for the token's database + target and streams an artifact-only tar of the stored, still-compressed bytes (full/base.tar.zst, incr-N/base.tar.zst, each with a backup_manifest, plus wal/ and MANIFEST.sha256). curl '<url>' | tar -x, then run the reconstruction command shown in the UI (decompress the blobs, pg_combinebackup + recovery settings). Public: authorized by the ?token= query param, not Bearer.
// @Tags backups-physical
// @Produce application/x-tar
// @Param token query string true "Restore token"
// @Success 200
// @Failure 401 {object} map[string]string
// @Failure 404 {object} map[string]string
// @Failure 409 {object} map[string]string
// @Failure 422 {object} map[string]string
// @Router /backups/physical/restore-stream [get]
func (c *PhysicalBackupController) GetRestoreStream(ctx *gin.Context) {
	token := ctx.Query("token")
	if token == "" {
		ctx.JSON(http.StatusUnauthorized, gin.H{"error": "restore token is required"})
		return
	}

	restoreToken, err := c.restoreTokenService.ValidateAndConsumeRestoreToken(token)
	if err != nil {
		if errors.Is(err, stream_guard.ErrDownloadAlreadyInProgress) {
			ctx.JSON(http.StatusConflict, gin.H{
				"error": "a download or restore is already in progress for this user",
			})
			return
		}

		ctx.JSON(http.StatusUnauthorized, gin.H{"error": "invalid or expired restore token"})
		return
	}

	heartbeatCtx, cancelHeartbeat := context.WithCancel(context.Background())
	defer func() {
		cancelHeartbeat()
		c.restoreTokenService.ReleaseDownloadLock(restoreToken.UserID)
	}()

	go c.startStreamHeartbeat(heartbeatCtx, restoreToken.UserID)

	ctx.Header("Content-Type", "application/x-tar")
	ctx.Header("Content-Disposition",
		fmt.Sprintf(`attachment; filename="restore-%s.tar"`, restoreToken.DatabaseID))

	if err := c.openRestoreStream(restoreToken, ctx.Writer); err != nil {
		// Resolution runs before any bytes are written, so if nothing has been
		// flushed yet we can still answer with a proper status. Once the tar has
		// started, the status is already 200 and we can only log.
		if !ctx.Writer.Written() {
			status, message := mapRestoreStreamError(err)
			ctx.JSON(status, gin.H{"error": message})
		}

		return
	}
}

// mapRestoreStreamError translates resolver failures into HTTP statuses for the
// stream endpoint, where an unrecognized error is an internal failure (500).
func mapRestoreStreamError(err error) (int, string) {
	if status, message, isResolverError := classifyRestoreStreamError(err); isResolverError {
		return status, message
	}

	return http.StatusInternalServerError, err.Error()
}

// classifyRestoreStreamError maps the resolver's typed failures to HTTP statuses
// and reports whether err was one of them. A WAL gap or out-of-range target is
// the caller's mistake to correct (422); no chain is a not-found (404). The bool
// lets callers fall through to their own default for everything else.
func classifyRestoreStreamError(err error) (int, string, bool) {
	var gapErr chain_view.WalGapBeforeTargetError
	if errors.As(err, &gapErr) {
		return http.StatusUnprocessableEntity, gapErr.Error(), true
	}

	if errors.Is(err, chain_view.ErrTargetBeforeEarliest) {
		return http.StatusUnprocessableEntity, err.Error(), true
	}

	if errors.Is(err, chain_view.ErrNoChainForRestore) {
		return http.StatusNotFound, err.Error(), true
	}

	return 0, "", false
}

// openRestoreStream dispatches by what the token carries: a BackupID streams a
// per-backup restore (FULL + incremental ancestors, no WAL); otherwise
// TargetTime drives a point-in-time restore.
func (c *PhysicalBackupController) openRestoreStream(token *restore_token.Token, w io.Writer) error {
	if token.BackupID != nil {
		return c.physicalBackupService.OpenRestoreStreamForBackup(token.DatabaseID, *token.BackupID, w)
	}

	return c.physicalBackupService.OpenRestoreStream(token.DatabaseID, token.TargetTime, w)
}

func (c *PhysicalBackupController) requestBackupOfType(
	user *users_models.User,
	databaseID uuid.UUID,
	backupType backups_dto_physical.TriggerBackupType,
) error {
	switch backupType {
	case backups_dto_physical.TriggerBackupTypeFull:
		return c.physicalBackupService.RequestFullBackup(user, databaseID)

	case backups_dto_physical.TriggerBackupTypeIncremental:
		return c.physicalBackupService.RequestIncrementalBackup(user, databaseID)

	default:
		return c.physicalBackupService.RequestBackup(user, databaseID)
	}
}

func (c *PhysicalBackupController) startStreamHeartbeat(ctx context.Context, userID uuid.UUID) {
	ticker := time.NewTicker(stream_guard.GetDownloadHeartbeatInterval())
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			c.restoreTokenService.RefreshDownloadLock(userID)
		}
	}
}
