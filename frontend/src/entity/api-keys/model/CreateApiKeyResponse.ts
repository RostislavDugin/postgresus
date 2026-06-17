import type { ApiKeyRole } from './ApiKeyRole';

export interface CreateApiKeyResponse {
  id: string;
  name: string;
  role: ApiKeyRole;
  token: string;
  tokenPrefix: string;
  workspaceIds: string[];
  expiresAt?: string;
  createdAt: string;
}
