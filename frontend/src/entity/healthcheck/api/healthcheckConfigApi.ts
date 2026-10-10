import { getApplicationServer } from '../../../constants';
import { apiHelper } from '../../../shared/api/apiHelper';
import RequestOptions from '../../../shared/api/RequestOptions';
import type { HealthcheckConfig } from '../model/HealthcheckConfig';

export const healthcheckConfigApi = {
  async saveHealthcheckConfig(config: HealthcheckConfig) {
    const requestOptions: RequestOptions = new RequestOptions();
    requestOptions.setBody(JSON.stringify(config));

    return apiHelper.fetchPostJson<{ message: string }>(
      `${getApplicationServer()}/api/v1/healthcheck-config`,
      requestOptions,
    );
  },

  async getHealthcheckConfig(databaseId: string) {
    return apiHelper.fetchGetJson<HealthcheckConfig>(
      `${getApplicationServer()}/api/v1/healthcheck-config/${databaseId}`,
      undefined,
      true,
    );
  },
};
