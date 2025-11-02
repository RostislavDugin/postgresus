import { WorkspaceRole } from '../../users/model/WorkspaceRole';

export interface WorkspaceMemberResponse {
  id: string;
  userId: string;
  email: string;
  name: string;
  role: WorkspaceRole;
  createdAt: Date;
}
