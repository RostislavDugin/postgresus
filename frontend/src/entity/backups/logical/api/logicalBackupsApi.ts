import { getApplicationServer } from '../../../../constants';
import RequestOptions from '../../../../shared/api/RequestOptions';
import { apiHelper } from '../../../../shared/api/apiHelper';
import type { GetLogicalBackupsResponse } from '../model/GetLogicalBackupsResponse';

export interface BackupsFilters {
  statuses?: string[];
  beforeDate?: string;
}

export const logicalBackupsApi = {
  async getBackups(databaseId: string, limit?: number, offset?: number, filters?: BackupsFilters) {
    const params = new URLSearchParams({ database_id: databaseId });
    if (limit !== undefined) params.append('limit', limit.toString());
    if (offset !== undefined) params.append('offset', offset.toString());

    if (filters?.statuses) {
      for (const status of filters.statuses) {
        params.append('status', status);
      }
    }
    if (filters?.beforeDate) {
      params.append('beforeDate', filters.beforeDate);
    }

    return apiHelper.fetchGetJson<GetLogicalBackupsResponse>(
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

  async downloadBackup(id: string): Promise<void> {
    // Generate short-lived download token
    const tokenResponse = await apiHelper.fetchPostJson<{
      token: string;
      filename: string;
      backupId: string;
    }>(`${getApplicationServer()}/api/v1/backups/${id}/download-token`, new RequestOptions());

    // Create direct download link with token
    const downloadUrl = `${getApplicationServer()}/api/v1/backups/${id}/file?token=${tokenResponse.token}`;

    const link = document.createElement('a');
    link.href = downloadUrl;
    link.download = tokenResponse.filename;
    link.style.display = 'none';

    document.body.appendChild(link);
    link.click();
    document.body.removeChild(link);
  },

  async cancelBackup(id: string) {
    return apiHelper.fetchPostRaw(`${getApplicationServer()}/api/v1/backups/${id}/cancel`);
  },
};
