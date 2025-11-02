import type { WorkspaceRole } from './WorkspaceRole';

export interface InviteUserRequest {
  email: string;
  intendedWorkspaceId?: string;
  intendedWorkspaceRole?: WorkspaceRole;
}
