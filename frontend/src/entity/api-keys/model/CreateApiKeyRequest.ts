import type { ApiKeyRole } from './ApiKeyRole';

export interface CreateApiKeyRequest {
  name: string;
  role: ApiKeyRole;
  expiresAt?: string;
  workspaceIds?: string[];
}
