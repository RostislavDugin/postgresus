package workspaces_services

import (
	"errors"
	"fmt"
	"time"

	audit_logs "databasus-backend/internal/features/audit_logs"
	users_enums "databasus-backend/internal/features/users/enums"
	users_models "databasus-backend/internal/features/users/models"
	users_services "databasus-backend/internal/features/users/services"
	workspaces_dto "databasus-backend/internal/features/workspaces/dto"
	workspaces_interfaces "databasus-backend/internal/features/workspaces/interfaces"
	workspaces_models "databasus-backend/internal/features/workspaces/models"
	workspaces_repositories "databasus-backend/internal/features/workspaces/repositories"

	"github.com/google/uuid"
)

type WorkspaceService struct {
	workspaceRepository        *workspaces_repositories.WorkspaceRepository
	membershipRepository       *workspaces_repositories.MembershipRepository
	userService                *users_services.UserService
	auditLogService            *audit_logs.AuditLogService
	settingsService            *users_services.SettingsService
	workspaceDeletionListeners []workspaces_interfaces.WorkspaceDeletionListener
}

func (s *WorkspaceService) AddWorkspaceDeletionListener(
	listener workspaces_interfaces.WorkspaceDeletionListener,
) {
	s.workspaceDeletionListeners = append(s.workspaceDeletionListeners, listener)
}

func (s *WorkspaceService) CreateWorkspace(
	request *workspaces_dto.CreateWorkspaceRequestDTO,
	creator *users_models.User,
) (*workspaces_dto.WorkspaceResponseDTO, error) {
	settings, err := s.settingsService.GetSettings()

	if err != nil {
		return nil, fmt.Errorf("failed to get settings: %w", err)
	}

	if !creator.CanCreateWorkspaces(settings) {
		return nil, errors.New("insufficient permissions to create workspaces")
	}

	workspace := &workspaces_models.Workspace{
		ID:        uuid.New(),
		Name:      request.Name,
		CreatedAt: time.Now().UTC(),
	}

	if err := s.workspaceRepository.CreateWorkspace(workspace); err != nil {
		return nil, fmt.Errorf("failed to create workspace: %w", err)
	}

	membership := &workspaces_models.WorkspaceMembership{
		UserID:      creator.ID,
		WorkspaceID: workspace.ID,
		Role:        users_enums.WorkspaceRoleOwner,
		CreatedAt:   time.Now().UTC(),
	}

	if err := s.membershipRepository.CreateMembership(membership); err != nil {
		return nil, fmt.Errorf("failed to create workspace membership: %w", err)
	}

	s.auditLogService.WriteAuditLog(
		fmt.Sprintf("Workspace created: %s", workspace.Name),
		&creator.ID,
		&workspace.ID,
	)

	ownerRole := users_enums.WorkspaceRoleOwner
	return &workspaces_dto.WorkspaceResponseDTO{
		ID:        workspace.ID,
		Name:      workspace.Name,
		CreatedAt: workspace.CreatedAt,
		UserRole:  &ownerRole,
	}, nil
}

func (s *WorkspaceService) GetWorkspace(
	workspaceID uuid.UUID,
	user *users_models.User,
) (*workspaces_models.Workspace, error) {
	canView, _, err := s.CanUserAccessWorkspace(workspaceID, user)
	if err != nil {
		return nil, err
	}
	if !canView {
		return nil, errors.New("insufficient permissions to view workspace")
	}

	return s.workspaceRepository.GetWorkspaceByID(workspaceID)
}

func (s *WorkspaceService) GetUserWorkspaces(
	user *users_models.User,
) (*workspaces_dto.ListWorkspacesResponseDTO, error) {
	workspaces, err := s.membershipRepository.GetWorkspacesWithRolesByUserID(user.Role, user.ID)
	if err != nil {
		return nil, fmt.Errorf("failed to get user workspaces: %w", err)
	}

	return &workspaces_dto.ListWorkspacesResponseDTO{
		Workspaces: workspaces,
	}, nil
}

func (s *WorkspaceService) UpdateWorkspace(
	workspaceID uuid.UUID,
	updateDTO *workspaces_models.Workspace,
	user *users_models.User,
) (*workspaces_models.Workspace, error) {
	canManage, err := s.CanUserManageWorkspace(workspaceID, user)

	if err != nil {
		return nil, err
	}
	if !canManage {
		return nil, errors.New("insufficient permissions to update workspace")
	}

	existingWorkspace, err := s.workspaceRepository.GetWorkspaceByID(workspaceID)
	if err != nil {
		return nil, fmt.Errorf("failed to get workspace: %w", err)
	}

	updateDTO.ID = workspaceID
	updateDTO.CreatedAt = existingWorkspace.CreatedAt

	existingWorkspace.UpdateFromDTO(updateDTO)

	if err := s.workspaceRepository.UpdateWorkspace(existingWorkspace); err != nil {
		return nil, fmt.Errorf("failed to update workspace: %w", err)
	}

	s.auditLogService.WriteAuditLog(
		fmt.Sprintf("Workspace updated: %s", updateDTO.Name),
		&user.ID,
		&workspaceID,
	)

	return existingWorkspace, nil
}

func (s *WorkspaceService) DeleteWorkspace(workspaceID uuid.UUID, user *users_models.User) error {
	if user.Role != users_enums.UserRoleAdmin {
		userWorkspaceRole, err := s.GetUserWorkspaceRole(workspaceID, user.ID)
		if err != nil {
			return fmt.Errorf("failed to get user role: %w", err)
		}

		if userWorkspaceRole == nil || *userWorkspaceRole != users_enums.WorkspaceRoleOwner {
			return errors.New("only workspace owner or admin can delete workspace")
		}
	}

	workspace, err := s.workspaceRepository.GetWorkspaceByID(workspaceID)
	if err != nil {
		return fmt.Errorf("failed to get workspace: %w", err)
	}

	for _, listener := range s.workspaceDeletionListeners {
		if err := listener.OnBeforeWorkspaceDeletion(workspaceID); err != nil {
			return fmt.Errorf("failed to delete workspace: %w", err)
		}
	}

	if err := s.workspaceRepository.DeleteWorkspace(workspaceID); err != nil {
		return fmt.Errorf("failed to delete workspace: %w", err)
	}

	s.auditLogService.WriteAuditLog(
		fmt.Sprintf("Workspace deleted: %s", workspace.Name),
		&user.ID,
		&workspaceID,
	)

	return nil
}

func (s *WorkspaceService) GetUserWorkspaceRole(
	workspaceID uuid.UUID,
	userID uuid.UUID,
) (*users_enums.WorkspaceRole, error) {
	return s.membershipRepository.GetUserWorkspaceRole(workspaceID, userID)
}

func (s *WorkspaceService) CanUserAccessWorkspace(
	workspaceID uuid.UUID,
	user *users_models.User,
) (bool, *users_enums.WorkspaceRole, error) {
	if user.Role == users_enums.UserRoleAdmin {
		adminRole := users_enums.WorkspaceRoleOwner
		return true, &adminRole, nil
	}

	role, err := s.membershipRepository.GetUserWorkspaceRole(workspaceID, user.ID)
	if err != nil {
		return false, nil, nil
	}

	return role != nil, role, nil
}

func (s *WorkspaceService) CanUserManageWorkspace(
	workspaceID uuid.UUID,
	user *users_models.User,
) (bool, error) {
	if user.Role == users_enums.UserRoleAdmin {
		return true, nil
	}

	role, err := s.membershipRepository.GetUserWorkspaceRole(workspaceID, user.ID)
	if err != nil {
		return false, err
	}

	if role == nil {
		return false, nil
	}

	return *role == users_enums.WorkspaceRoleOwner ||
		*role == users_enums.WorkspaceRoleAdmin, nil
}

func (s *WorkspaceService) CanUserManageDBs(
	workspaceID uuid.UUID,
	user *users_models.User,
) (bool, error) {
	if user.Role == users_enums.UserRoleAdmin {
		return true, nil
	}

	role, err := s.membershipRepository.GetUserWorkspaceRole(workspaceID, user.ID)
	if err != nil {
		return false, err
	}

	if role == nil {
		return false, nil
	}

	return *role == users_enums.WorkspaceRoleOwner ||
		*role == users_enums.WorkspaceRoleAdmin || *role == users_enums.WorkspaceRoleMember, nil
}

func (s *WorkspaceService) CanUserManageMembership(
	workspaceID uuid.UUID,
	user *users_models.User,
) (bool, error) {
	if user.Role == users_enums.UserRoleAdmin {
		return true, nil
	}

	role, err := s.membershipRepository.GetUserWorkspaceRole(workspaceID, user.ID)
	if err != nil {
		return false, err
	}

	if role == nil {
		return false, nil
	}

	return *role == users_enums.WorkspaceRoleOwner || *role == users_enums.WorkspaceRoleAdmin, nil
}

func (s *WorkspaceService) CanUserManageAdmins(
	workspaceID uuid.UUID,
	user *users_models.User,
) (bool, error) {
	if user.Role == users_enums.UserRoleAdmin {
		return true, nil
	}

	role, err := s.membershipRepository.GetUserWorkspaceRole(workspaceID, user.ID)
	if err != nil {
		return false, err
	}

	if role == nil {
		return false, nil
	}

	return *role == users_enums.WorkspaceRoleOwner, nil
}

func (s *WorkspaceService) GetWorkspaceAuditLogs(
	workspaceID uuid.UUID,
	user *users_models.User,
	request *audit_logs.GetAuditLogsRequest,
) (*audit_logs.GetAuditLogsResponse, error) {
	canView, _, err := s.CanUserAccessWorkspace(workspaceID, user)
	if err != nil {
		return nil, err
	}
	if !canView {
		return nil, errors.New("insufficient permissions to view workspace audit logs")
	}

	return s.auditLogService.GetWorkspaceAuditLogs(workspaceID, request)
}

func (s *WorkspaceService) GetAllWorkspaces() ([]*workspaces_models.Workspace, error) {
	return s.workspaceRepository.GetAllWorkspaces()
}

func (s *WorkspaceService) GetWorkspaceByID(
	workspaceID uuid.UUID,
) (*workspaces_models.Workspace, error) {
	return s.workspaceRepository.GetWorkspaceByID(workspaceID)
}
