export interface AuditLog {
  id: string;
  userId?: string;
  workspaceId?: string;
  message: string;
  createdAt: string;
  userEmail?: string;
  userName?: string;
  workspaceName?: string;
}
