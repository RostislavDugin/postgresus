import { getApplicationServer } from '../../../constants';
import RequestOptions from '../../../shared/api/RequestOptions';
import { apiHelper } from '../../../shared/api/apiHelper';
import type { Notifier } from '../models/Notifier';

export const notifierApi = {
  async saveNotifier(notifier: Notifier) {
    const requestOptions: RequestOptions = new RequestOptions();
    requestOptions.setBody(JSON.stringify(notifier));
    return apiHelper.fetchPostJson<Notifier>(
      `${getApplicationServer()}/api/v1/notifiers`,
      requestOptions,
    );
  },

  async getNotifier(id: string) {
    const requestOptions: RequestOptions = new RequestOptions();
    return apiHelper.fetchGetJson<Notifier>(
      `${getApplicationServer()}/api/v1/notifiers/${id}`,
      requestOptions,
      true,
    );
  },

  async getNotifiers(workspaceId: string) {
    const requestOptions: RequestOptions = new RequestOptions();
    return apiHelper.fetchGetJson<Notifier[]>(
      `${getApplicationServer()}/api/v1/notifiers?workspace_id=${workspaceId}`,
      requestOptions,
      true,
    );
  },

  async deleteNotifier(id: string) {
    const requestOptions: RequestOptions = new RequestOptions();
    return apiHelper.fetchDeleteJson(
      `${getApplicationServer()}/api/v1/notifiers/${id}`,
      requestOptions,
    );
  },

  async sendTestNotification(id: string) {
    const requestOptions: RequestOptions = new RequestOptions();
    return apiHelper.fetchPostJson(
      `${getApplicationServer()}/api/v1/notifiers/${id}/test`,
      requestOptions,
    );
  },

  async sendTestNotificationDirect(notifier: Notifier) {
    const requestOptions: RequestOptions = new RequestOptions();
    requestOptions.setBody(JSON.stringify(notifier));
    return apiHelper.fetchPostJson(
      `${getApplicationServer()}/api/v1/notifiers/direct-test`,
      requestOptions,
    );
  },
};
