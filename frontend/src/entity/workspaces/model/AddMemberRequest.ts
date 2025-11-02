import { WorkspaceRole } from '../../users/model/WorkspaceRole';

export interface AddMemberRequest {
  email: string;
  role: WorkspaceRole;
}
