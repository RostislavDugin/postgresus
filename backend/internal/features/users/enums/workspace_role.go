package users_enums

type WorkspaceRole string

const (
	WorkspaceRoleOwner  WorkspaceRole = "WORKSPACE_OWNER"
	WorkspaceRoleAdmin  WorkspaceRole = "WORKSPACE_ADMIN"
	WorkspaceRoleMember WorkspaceRole = "WORKSPACE_MEMBER"
	WorkspaceRoleViewer WorkspaceRole = "WORKSPACE_VIEWER"
)

// IsValid validates the WorkspaceRole
func (r WorkspaceRole) IsValid() bool {
	switch r {
	case WorkspaceRoleOwner, WorkspaceRoleAdmin, WorkspaceRoleMember, WorkspaceRoleViewer:
		return true
	default:
		return false
	}
}
