import { WorkspaceRole } from '../../users/model/WorkspaceRole';

export interface WorkspaceMembership {
  id: string;
  userId: string;
  workspaceId: string;
  role: WorkspaceRole;
  createdAt: Date;
}
