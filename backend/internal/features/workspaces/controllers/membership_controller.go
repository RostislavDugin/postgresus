package workspaces_controllers

import (
	"errors"
	"net/http"

	users_middleware "databasus-backend/internal/features/users/middleware"
	workspaces_dto "databasus-backend/internal/features/workspaces/dto"
	workspaces_errors "databasus-backend/internal/features/workspaces/errors"
	workspaces_services "databasus-backend/internal/features/workspaces/services"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type MembershipController struct {
	membershipService *workspaces_services.MembershipService
}

func (c *MembershipController) RegisterRoutes(router *gin.RouterGroup) {
	workspaceRoutes := router.Group("/workspaces/memberships/:id")

	workspaceRoutes.GET("/members", c.ListMembers)
	workspaceRoutes.POST("/members", c.AddMember)
	workspaceRoutes.PUT("/members/:userId/role", c.ChangeMemberRole)
	workspaceRoutes.DELETE("/members/:userId", c.RemoveMember)
	workspaceRoutes.POST("/transfer-ownership", c.TransferOwnership)
}

// ListMembers
// @Summary List workspace members
// @Description Get list of all workspace members
// @Tags workspace-membership
// @Produce json
// @Security BearerAuth
// @Param id path string true "Workspace ID"
// @Success 200 {object} workspaces_dto.GetMembersResponseDTO
// @Failure 400 {object} map[string]string
// @Failure 401 {object} map[string]string
// @Failure 403 {object} map[string]string
// @Router /workspaces/memberships/{id}/members [get]
func (c *MembershipController) ListMembers(ctx *gin.Context) {
	user, ok := users_middleware.GetUserFromContext(ctx)
	if !ok {
		ctx.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
		return
	}

	workspaceIDStr := ctx.Param("id")
	workspaceID, err := uuid.Parse(workspaceIDStr)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "Invalid workspace ID"})
		return
	}

	response, err := c.membershipService.GetMembers(workspaceID, user)
	if err != nil {
		if errors.Is(err, workspaces_errors.ErrInsufficientPermissionsToViewMembers) {
			ctx.JSON(http.StatusForbidden, gin.H{"error": err.Error()})
			return
		}
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	ctx.JSON(http.StatusOK, response)
}

// AddMember
// @Summary Add member to workspace (supports both existing and new users)
// @Description Add an existing user to the workspace or invite a new user if they don't exist
// @Tags workspace-membership
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path string true "Workspace ID"
// @Param request body workspaces_dto.AddMemberRequestDTO true "Member addition data"
// @Success 200 {object} workspaces_dto.AddMemberResponseDTO
// @Failure 400 {object} map[string]string
// @Failure 401 {object} map[string]string
// @Failure 403 {object} map[string]string
// @Router /workspaces/memberships/{id}/members [post]
func (c *MembershipController) AddMember(ctx *gin.Context) {
	user, ok := users_middleware.GetUserFromContext(ctx)
	if !ok {
		ctx.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
		return
	}

	workspaceIDStr := ctx.Param("id")
	workspaceID, err := uuid.Parse(workspaceIDStr)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "Invalid workspace ID"})
		return
	}

	var request workspaces_dto.AddMemberRequestDTO
	if err := ctx.ShouldBindJSON(&request); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request format"})
		return
	}

	if !request.Role.IsValid() {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "Invalid role"})
		return
	}

	response, err := c.membershipService.AddMember(workspaceID, &request, user)
	if err != nil {
		if errors.Is(err, workspaces_errors.ErrInsufficientPermissionsToManageMembers) ||
			errors.Is(err, workspaces_errors.ErrOnlyOwnerCanAddManageAdmins) {
			ctx.JSON(http.StatusForbidden, gin.H{"error": err.Error()})
			return
		}
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	ctx.JSON(http.StatusOK, response)
}

// ChangeMemberRole
// @Summary Change member role
// @Description Change the role of an existing workspace member
// @Tags workspace-membership
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path string true "Workspace ID"
// @Param userId path string true "User ID"
// @Param request body workspaces_dto.ChangeMemberRoleRequestDTO true "Role change data"
// @Success 200 {object} map[string]string
// @Failure 400 {object} map[string]string
// @Failure 401 {object} map[string]string
// @Failure 403 {object} map[string]string
// @Router /workspaces/memberships/{id}/members/{userId}/role [put]
func (c *MembershipController) ChangeMemberRole(ctx *gin.Context) {
	user, ok := users_middleware.GetUserFromContext(ctx)
	if !ok {
		ctx.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
		return
	}

	workspaceIDStr := ctx.Param("id")
	workspaceID, err := uuid.Parse(workspaceIDStr)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "Invalid workspace ID"})
		return
	}

	userIDStr := ctx.Param("userId")
	userID, err := uuid.Parse(userIDStr)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "Invalid user ID"})
		return
	}

	var request workspaces_dto.ChangeMemberRoleRequestDTO
	if err := ctx.ShouldBindJSON(&request); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request format"})
		return
	}

	if err := c.membershipService.ChangeMemberRole(workspaceID, userID, &request, user); err != nil {
		if errors.Is(err, workspaces_errors.ErrInsufficientPermissionsToManageMembers) ||
			errors.Is(err, workspaces_errors.ErrOnlyOwnerCanAddManageAdmins) {
			ctx.JSON(http.StatusForbidden, gin.H{"error": err.Error()})
			return
		}
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	ctx.JSON(http.StatusOK, gin.H{"message": "Member role changed successfully"})
}

// RemoveMember
// @Summary Remove member from workspace
// @Description Remove a member from the workspace
// @Tags workspace-membership
// @Security BearerAuth
// @Param id path string true "Workspace ID"
// @Param userId path string true "User ID"
// @Success 200 {object} map[string]string
// @Failure 400 {object} map[string]string
// @Failure 401 {object} map[string]string
// @Failure 403 {object} map[string]string
// @Router /workspaces/memberships/{id}/members/{userId} [delete]
func (c *MembershipController) RemoveMember(ctx *gin.Context) {
	user, ok := users_middleware.GetUserFromContext(ctx)
	if !ok {
		ctx.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
		return
	}

	workspaceIDStr := ctx.Param("id")
	workspaceID, err := uuid.Parse(workspaceIDStr)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "Invalid workspace ID"})
		return
	}

	userIDStr := ctx.Param("userId")
	userID, err := uuid.Parse(userIDStr)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "Invalid user ID"})
		return
	}

	if err := c.membershipService.RemoveMember(workspaceID, userID, user); err != nil {
		if errors.Is(err, workspaces_errors.ErrInsufficientPermissionsToRemoveMembers) ||
			errors.Is(err, workspaces_errors.ErrOnlyOwnerCanRemoveAdmins) {
			ctx.JSON(http.StatusForbidden, gin.H{"error": err.Error()})
			return
		}
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	ctx.JSON(http.StatusOK, gin.H{"message": "Member removed successfully"})
}

// TransferOwnership
// @Summary Transfer workspace ownership
// @Description Transfer workspace ownership to another workspace admin
// @Tags workspace-membership
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path string true "Workspace ID"
// @Param request body workspaces_dto.TransferOwnershipRequestDTO true "Ownership transfer data"
// @Success 200 {object} map[string]string
// @Failure 400 {object} map[string]string
// @Failure 401 {object} map[string]string
// @Failure 403 {object} map[string]string
// @Router /workspaces/memberships/{id}/transfer-ownership [post]
func (c *MembershipController) TransferOwnership(ctx *gin.Context) {
	user, ok := users_middleware.GetUserFromContext(ctx)
	if !ok {
		ctx.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
		return
	}

	workspaceIDStr := ctx.Param("id")
	workspaceID, err := uuid.Parse(workspaceIDStr)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "Invalid workspace ID"})
		return
	}

	var request workspaces_dto.TransferOwnershipRequestDTO
	if err := ctx.ShouldBindJSON(&request); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request format"})
		return
	}

	if err := c.membershipService.TransferOwnership(workspaceID, &request, user); err != nil {
		if errors.Is(err, workspaces_errors.ErrOnlyOwnerOrAdminCanTransferOwnership) {
			ctx.JSON(http.StatusForbidden, gin.H{"error": err.Error()})
			return
		}
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	ctx.JSON(http.StatusOK, gin.H{"message": "Ownership transferred successfully"})
}
