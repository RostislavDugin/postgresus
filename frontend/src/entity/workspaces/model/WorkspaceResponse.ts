import { WorkspaceRole } from '../../users/model/WorkspaceRole';

export interface WorkspaceResponse {
  id: string;
  name: string;
  createdAt: Date;
  userRole?: WorkspaceRole;
}
