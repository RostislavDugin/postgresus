import type { ApiKeyRole } from './ApiKeyRole';

export interface ApiKey {
  id: string;
  name: string;
  role: ApiKeyRole;
  tokenPrefix: string;
  workspaceIds: string[];
  lastUsedAt?: string;
  expiresAt?: string;
  createdAt: string;
}
