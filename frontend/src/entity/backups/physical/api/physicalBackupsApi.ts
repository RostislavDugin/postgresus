import { getApplicationServer } from '../../../../constants';
import RequestOptions from '../../../../shared/api/RequestOptions';
import { apiHelper } from '../../../../shared/api/apiHelper';
import type { GetPhysicalBackupsResponse } from '../model/GetPhysicalBackupsResponse';
import type { PhysicalBackupsFilters } from '../model/PhysicalBackupsFilters';
import type { PhysicalRestoreTokenResponse } from '../model/PhysicalRestoreTokenResponse';

// 'auto' lets the backend pick FULL vs incremental based on the chain state;
// 'full'/'incremental' force a specific kind (incremental 409s with no chain).
export type TriggerPhysicalBackupType = 'auto' | 'full' | 'incremental';

export const physicalBackupsApi = {
  async getPhysicalBackups(
    databaseId: string,
    limit?: number,
    offset?: number,
    filters?: PhysicalBackupsFilters,
  ) {
    const params = new URLSearchParams();
    if (limit !== undefined) params.append('limit', limit.toString());
    if (offset !== undefined) params.append('offset', offset.toString());

    if (filters?.types) for (const type of filters.types) params.append('type', type);
    if (filters?.statuses) for (const status of filters.statuses) params.append('status', status);
    if (filters?.beforeDate) params.append('beforeDate', filters.beforeDate);

    return apiHelper.fetchGetJson<GetPhysicalBackupsResponse>(
      `${getApplicationServer()}/api/v1/backups/physical/database/${databaseId}/backups?${params.toString()}`,
      undefined,
      true,
    );
  },

  async triggerPhysicalBackup(databaseId: string, type: TriggerPhysicalBackupType) {
    const requestOptions: RequestOptions = new RequestOptions();
    requestOptions.setBody(JSON.stringify({ type }));
    return apiHelper.fetchPostJson<{ message: string }>(
      `${getApplicationServer()}/api/v1/backups/physical/database/${databaseId}/trigger`,
      requestOptions,
    );
  },

  async cancelPhysicalBackup(backupId: string) {
    return apiHelper.fetchPostRaw(
      `${getApplicationServer()}/api/v1/backups/physical/backups/${backupId}/cancel`,
    );
  },

  async deletePhysicalBackup(backupId: string) {
    return apiHelper.fetchDeleteRaw(
      `${getApplicationServer()}/api/v1/backups/physical/backups/${backupId}`,
    );
  },

  // Point-in-time restore: omit targetTime to restore to the latest available point.
  async generatePitrRestoreToken(databaseId: string, targetTime?: string) {
    const requestOptions: RequestOptions = new RequestOptions();
    requestOptions.setBody(JSON.stringify({ targetTime }));
    return apiHelper.fetchPostJson<PhysicalRestoreTokenResponse>(
      `${getApplicationServer()}/api/v1/backups/physical/database/${databaseId}/restore-token`,
      requestOptions,
    );
  },

  // Per-backup restore: a FULL restores itself; an incremental restores its FULL +
  // incremental ancestors (no WAL replay).
  async generateBackupRestoreToken(backupId: string) {
    return apiHelper.fetchPostJson<PhysicalRestoreTokenResponse>(
      `${getApplicationServer()}/api/v1/backups/physical/backups/${backupId}/restore-token`,
      new RequestOptions(),
    );
  },
};
