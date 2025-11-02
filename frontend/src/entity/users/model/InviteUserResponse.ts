import type { WorkspaceRole } from './WorkspaceRole';

export interface InviteUserResponse {
  id: string;
  email: string;
  intendedWorkspaceId?: string;
  intendedWorkspaceRole?: WorkspaceRole;
  createdAt: string;
}
