import { getApplicationServer } from '../../../constants';
import RequestOptions from '../../../shared/api/RequestOptions';
import { apiHelper } from '../../../shared/api/apiHelper';
import type { GetBackupsResponse } from '../model/GetBackupsResponse';

export const backupsApi = {
  async getBackups(databaseId: string, limit?: number, offset?: number) {
    const params = new URLSearchParams({ database_id: databaseId });
    if (limit !== undefined) params.append('limit', limit.toString());
    if (offset !== undefined) params.append('offset', offset.toString());

    return apiHelper.fetchGetJson<GetBackupsResponse>(
      `${getApplicationServer()}/api/v1/backups?${params.toString()}`,
      undefined,
      true,
    );
  },

  async makeBackup(databaseId: string) {
    const requestOptions: RequestOptions = new RequestOptions();
    requestOptions.setBody(JSON.stringify({ database_id: databaseId }));
    return apiHelper.fetchPostJson<{ message: string }>(
      `${getApplicationServer()}/api/v1/backups`,
      requestOptions,
    );
  },

  async deleteBackup(id: string) {
    return apiHelper.fetchDeleteRaw(`${getApplicationServer()}/api/v1/backups/${id}`);
  },

  async downloadBackup(id: string): Promise<Blob> {
    return apiHelper.fetchGetBlob(`${getApplicationServer()}/api/v1/backups/${id}/file`);
  },
};
