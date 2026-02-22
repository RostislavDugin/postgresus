import { getApplicationServer } from '../../../constants';
import RequestOptions from '../../../shared/api/RequestOptions';
import { apiHelper } from '../../../shared/api/apiHelper';
import type { DatabasePlan } from '../../plan';
import type { BackupConfig } from '../model/BackupConfig';
import type { TransferDatabaseRequest } from '../model/TransferDatabaseRequest';

export const backupConfigApi = {
  async saveBackupConfig(config: BackupConfig) {
    const requestOptions: RequestOptions = new RequestOptions();
    requestOptions.setBody(JSON.stringify(config));
    if (config.id) {
      return apiHelper.fetchPutJson<BackupConfig>(
        `${getApplicationServer()}/api/v1/backup-configs/${config.id}`,
        requestOptions,
      );
    }
    return apiHelper.fetchPostJson<BackupConfig>(
      `${getApplicationServer()}/api/v1/backup-configs`,
      requestOptions,
    );
  },

  async getBackupConfigByDbID(databaseId: string) {
    const configs = await apiHelper.fetchGetJson<BackupConfig[]>(
      `${getApplicationServer()}/api/v1/backup-configs/database/${databaseId}`,
      undefined,
      true,
    );
    return configs[0] ?? null;
  },

  async isStorageUsing(storageId: string): Promise<boolean> {
    return await apiHelper
      .fetchGetJson<{
        isUsing: boolean;
      }>(
        `${getApplicationServer()}/api/v1/backup-configs/storage/${storageId}/is-using`,
        undefined,
        true,
      )
      .then((res) => res.isUsing);
  },

  async getDatabasesCountForStorage(storageId: string): Promise<number> {
    return await apiHelper
      .fetchGetJson<{
        count: number;
      }>(
        `${getApplicationServer()}/api/v1/backup-configs/storage/${storageId}/databases-count`,
        undefined,
        true,
      )
      .then((res) => res.count);
  },

  async transferDatabase(databaseId: string, request: TransferDatabaseRequest): Promise<void> {
    const requestOptions: RequestOptions = new RequestOptions();
    requestOptions.setBody(JSON.stringify(request));
    await apiHelper.fetchPostJson(
      `${getApplicationServer()}/api/v1/backup-configs/database/${databaseId}/transfer`,
      requestOptions,
    );
  },

  async getDatabasePlan(databaseId: string) {
    return apiHelper.fetchGetJson<DatabasePlan>(
      `${getApplicationServer()}/api/v1/backup-configs/database/${databaseId}/plan`,
      undefined,
      true,
    );
  },

  async getBackupConfigsByDbID(databaseId: string) {
    return apiHelper.fetchGetJson<BackupConfig[]>(
      `${getApplicationServer()}/api/v1/backup-configs/database/${databaseId}`,
      undefined,
      true,
    );
  },

  async deleteBackupConfig(id: string): Promise<void> {
    await apiHelper.fetchDeleteRaw(
      `${getApplicationServer()}/api/v1/backup-configs/${id}`,
    );
  },
};
