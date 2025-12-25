package workspaces_controllers

import (
	workspaces_services "databasus-backend/internal/features/workspaces/services"
)

var workspaceController = &WorkspaceController{
	workspaces_services.GetWorkspaceService(),
}

var membershipController = &MembershipController{
	workspaces_services.GetMembershipService(),
}

func GetWorkspaceController() *WorkspaceController {
	return workspaceController
}

func GetMembershipController() *MembershipController {
	return membershipController
}
