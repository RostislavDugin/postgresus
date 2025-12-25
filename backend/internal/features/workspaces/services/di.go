package workspaces_services

import (
	"databasus-backend/internal/features/audit_logs"
	users_services "databasus-backend/internal/features/users/services"
	workspaces_interfaces "databasus-backend/internal/features/workspaces/interfaces"
	workspaces_repositories "databasus-backend/internal/features/workspaces/repositories"
)

var workspaceRepository = &workspaces_repositories.WorkspaceRepository{}
var membershipRepository = &workspaces_repositories.MembershipRepository{}

var workspaceService = &WorkspaceService{
	workspaceRepository,
	membershipRepository,
	users_services.GetUserService(),
	audit_logs.GetAuditLogService(),
	users_services.GetSettingsService(),
	[]workspaces_interfaces.WorkspaceDeletionListener{},
}

var membershipService = &MembershipService{
	membershipRepository,
	workspaceRepository,
	users_services.GetUserService(),
	audit_logs.GetAuditLogService(),
	workspaceService,
	users_services.GetSettingsService(),
}

func GetWorkspaceService() *WorkspaceService {
	return workspaceService
}

func GetMembershipService() *MembershipService {
	return membershipService
}
