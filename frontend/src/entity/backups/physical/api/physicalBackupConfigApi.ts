import { getApplicationServer } from '../../../../constants';
import RequestOptions from '../../../../shared/api/RequestOptions';
import { apiHelper } from '../../../../shared/api/apiHelper';
import type { TransferDatabaseRequest } from '../../shared';
import type { PhysicalBackupConfig } from '../model/PhysicalBackupConfig';

export const physicalBackupConfigApi = {
  async savePhysicalBackupConfig(config: PhysicalBackupConfig) {
    const requestOptions: RequestOptions = new RequestOptions();
    requestOptions.setBody(JSON.stringify(config));
    return apiHelper.fetchPostJson<PhysicalBackupConfig>(
      `${getApplicationServer()}/api/v1/backup-configs/physical/save`,
      requestOptions,
    );
  },

  async getPhysicalBackupConfigByDbId(databaseId: string) {
    return apiHelper.fetchGetJson<PhysicalBackupConfig>(
      `${getApplicationServer()}/api/v1/backup-configs/physical/database/${databaseId}`,
      undefined,
      true,
    );
  },

  async transferPhysicalDatabase(
    databaseId: string,
    request: TransferDatabaseRequest,
  ): Promise<void> {
    const requestOptions: RequestOptions = new RequestOptions();
    requestOptions.setBody(JSON.stringify(request));
    await apiHelper.fetchPostJson(
      `${getApplicationServer()}/api/v1/backup-configs/physical/database/${databaseId}/transfer`,
      requestOptions,
    );
  },
};
