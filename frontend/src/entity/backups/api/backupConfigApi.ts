import { getApplicationServer } from '../../../constants';
import RequestOptions from '../../../shared/api/RequestOptions';
import { apiHelper } from '../../../shared/api/apiHelper';
import type { BackupConfig } from '../model/BackupConfig';
import type { TransferDatabaseRequest } from '../model/TransferDatabaseRequest';

export const backupConfigApi = {
  async saveBackupConfig(config: BackupConfig) {
    const requestOptions: RequestOptions = new RequestOptions();
    requestOptions.setBody(JSON.stringify(config));
    return apiHelper.fetchPostJson<BackupConfig>(
      `${getApplicationServer()}/api/v1/backup-configs/save`,
      requestOptions,
    );
  },

  async getBackupConfigByDbID(databaseId: string) {
    return apiHelper.fetchGetJson<BackupConfig>(
      `${getApplicationServer()}/api/v1/backup-configs/database/${databaseId}`,
      undefined,
      true,
    );
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
};
