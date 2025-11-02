import { getApplicationServer } from '../../../constants';
import RequestOptions from '../../../shared/api/RequestOptions';
import { apiHelper } from '../../../shared/api/apiHelper';
import type { Storage } from '../models/Storage';

export const storageApi = {
  async saveStorage(storage: Storage) {
    const requestOptions: RequestOptions = new RequestOptions();
    requestOptions.setBody(JSON.stringify(storage));
    return apiHelper.fetchPostJson<Storage>(
      `${getApplicationServer()}/api/v1/storages`,
      requestOptions,
    );
  },

  async getStorage(id: string) {
    const requestOptions: RequestOptions = new RequestOptions();
    return apiHelper.fetchGetJson<Storage>(
      `${getApplicationServer()}/api/v1/storages/${id}`,
      requestOptions,
      true,
    );
  },

  async getStorages(workspaceId: string) {
    const requestOptions: RequestOptions = new RequestOptions();
    return apiHelper.fetchGetJson<Storage[]>(
      `${getApplicationServer()}/api/v1/storages?workspace_id=${workspaceId}`,
      requestOptions,
      true,
    );
  },

  async deleteStorage(id: string) {
    const requestOptions: RequestOptions = new RequestOptions();
    return apiHelper.fetchDeleteJson(
      `${getApplicationServer()}/api/v1/storages/${id}`,
      requestOptions,
    );
  },

  async testStorageConnection(id: string) {
    const requestOptions: RequestOptions = new RequestOptions();
    return apiHelper.fetchPostJson(
      `${getApplicationServer()}/api/v1/storages/${id}/test`,
      requestOptions,
    );
  },

  async testStorageConnectionDirect(storage: Storage) {
    const requestOptions: RequestOptions = new RequestOptions();
    requestOptions.setBody(JSON.stringify(storage));
    return apiHelper.fetchPostJson(
      `${getApplicationServer()}/api/v1/storages/direct-test`,
      requestOptions,
    );
  },
};
