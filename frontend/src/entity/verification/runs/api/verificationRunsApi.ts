import { getApplicationServer } from '../../../../constants';
import { apiHelper } from '../../../../shared/api/apiHelper';
import RequestOptions from '../../../../shared/api/RequestOptions';
import type { GetVerificationsResponse } from '../models/GetVerificationsResponse';
import type { RestoreVerification } from '../models/RestoreVerification';

export const verificationRunsApi = {
  async listByDatabase(databaseId: string, limit?: number, offset?: number) {
    const params = new URLSearchParams();
    if (limit !== undefined) params.append('limit', limit.toString());
    if (offset !== undefined) params.append('offset', offset.toString());

    const query = params.toString();
    return apiHelper.fetchGetJson<GetVerificationsResponse>(
      `${getApplicationServer()}/api/v1/verifications/by-database/${databaseId}${query ? `?${query}` : ''}`,
      undefined,
      true,
    );
  },

  async getById(id: string) {
    const requestOptions: RequestOptions = new RequestOptions();
    return apiHelper.fetchGetJson<RestoreVerification>(
      `${getApplicationServer()}/api/v1/verifications/${id}`,
      requestOptions,
      true,
    );
  },

  async enqueue(backupId: string) {
    const requestOptions: RequestOptions = new RequestOptions();
    requestOptions.setBody(JSON.stringify({ backupId }));
    return apiHelper.fetchPostJson<RestoreVerification>(
      `${getApplicationServer()}/api/v1/verifications/enqueue`,
      requestOptions,
    );
  },

  async cancel(id: string) {
    return apiHelper.fetchPostRaw(`${getApplicationServer()}/api/v1/verifications/${id}/cancel`);
  },
};
