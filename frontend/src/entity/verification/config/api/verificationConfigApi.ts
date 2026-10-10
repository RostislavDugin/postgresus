import { getApplicationServer } from '../../../../constants';
import { apiHelper } from '../../../../shared/api/apiHelper';
import RequestOptions from '../../../../shared/api/RequestOptions';
import type { BackupVerificationConfig } from '../models/BackupVerificationConfig';

export const verificationConfigApi = {
  async getByDatabaseId(databaseId: string) {
    const requestOptions: RequestOptions = new RequestOptions();
    return apiHelper.fetchGetJson<BackupVerificationConfig>(
      `${getApplicationServer()}/api/v1/verification-config/${databaseId}`,
      requestOptions,
      true,
    );
  },

  async save(databaseId: string, config: BackupVerificationConfig) {
    const requestOptions: RequestOptions = new RequestOptions();
    requestOptions.setBody(JSON.stringify(config));
    return apiHelper.fetchPutJson<BackupVerificationConfig>(
      `${getApplicationServer()}/api/v1/verification-config/${databaseId}`,
      requestOptions,
    );
  },
};
