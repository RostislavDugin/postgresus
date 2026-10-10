import { getApplicationServer } from '../../../constants';
import { apiHelper } from '../../../shared/api/apiHelper';
import RequestOptions from '../../../shared/api/RequestOptions';
import type { SendTestEmailResponse } from '../model/SendTestEmailResponse';
import type { UsersSettings } from '../model/UsersSettings';
import type { UsersSettingsResponse } from '../model/UsersSettingsResponse';

export const settingsApi = {
  async getSettings(): Promise<UsersSettingsResponse> {
    const requestOptions: RequestOptions = new RequestOptions();
    return apiHelper.fetchGetJson(
      `${getApplicationServer()}/api/v1/users/settings`,
      requestOptions,
    );
  },

  async updateSettings(settings: UsersSettings): Promise<UsersSettingsResponse> {
    const requestOptions: RequestOptions = new RequestOptions();
    requestOptions.setBody(JSON.stringify(settings));
    return apiHelper.fetchPutJson(
      `${getApplicationServer()}/api/v1/users/settings`,
      requestOptions,
    );
  },

  // No retry: every attempt sends a message and counts against the rate limit.
  async sendTestEmail(): Promise<SendTestEmailResponse> {
    return apiHelper.fetchPostJson(
      `${getApplicationServer()}/api/v1/users/settings/test-email`,
      new RequestOptions(),
    );
  },
};
