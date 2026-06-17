import { getApplicationServer } from '../../../constants';
import RequestOptions from '../../../shared/api/RequestOptions';
import { apiHelper } from '../../../shared/api/apiHelper';
import type { ApiKey } from '../model/ApiKey';
import type { CreateApiKeyRequest } from '../model/CreateApiKeyRequest';
import type { CreateApiKeyResponse } from '../model/CreateApiKeyResponse';

interface ListApiKeysResponse {
  apiKeys: ApiKey[];
}

export const apiKeysApi = {
  async listApiKeys(): Promise<ListApiKeysResponse> {
    const requestOptions: RequestOptions = new RequestOptions();
    return apiHelper.fetchGetJson<ListApiKeysResponse>(
      `${getApplicationServer()}/api/v1/api-keys`,
      requestOptions,
      true,
    );
  },

  async createApiKey(request: CreateApiKeyRequest): Promise<CreateApiKeyResponse> {
    const requestOptions: RequestOptions = new RequestOptions();
    requestOptions.setBody(JSON.stringify(request));
    return apiHelper.fetchPostJson<CreateApiKeyResponse>(
      `${getApplicationServer()}/api/v1/api-keys`,
      requestOptions,
    );
  },

  async revokeApiKey(id: string): Promise<void> {
    const requestOptions: RequestOptions = new RequestOptions();
    return apiHelper.fetchDeleteJson<void>(
      `${getApplicationServer()}/api/v1/api-keys/${id}`,
      requestOptions,
    );
  },
};
