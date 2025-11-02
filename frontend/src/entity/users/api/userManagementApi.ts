import { getApplicationServer } from '../../../constants';
import RequestOptions from '../../../shared/api/RequestOptions';
import { apiHelper } from '../../../shared/api/apiHelper';
import type { ChangeUserRoleRequest } from '../model/ChangeUserRoleRequest';
import type { ListUsersRequest } from '../model/ListUsersRequest';
import type { ListUsersResponse } from '../model/ListUsersResponse';
import type { UserProfile } from '../model/UserProfile';

export const userManagementApi = {
  async getUsers(request?: ListUsersRequest): Promise<ListUsersResponse> {
    const requestOptions: RequestOptions = new RequestOptions();

    let url = `${getApplicationServer()}/api/v1/users`;
    const params = new URLSearchParams();

    if (request?.limit) {
      params.append('limit', request.limit.toString());
    }
    if (request?.offset) {
      params.append('offset', request.offset.toString());
    }
    if (request?.beforeDate) {
      params.append('beforeDate', request.beforeDate);
    }
    if (request?.query) {
      params.append('query', request.query);
    }

    if (params.toString()) {
      url += `?${params.toString()}`;
    }

    return apiHelper.fetchGetJson(url, requestOptions);
  },

  async getUserProfile(userId: string): Promise<UserProfile> {
    const requestOptions: RequestOptions = new RequestOptions();
    return apiHelper.fetchGetJson(
      `${getApplicationServer()}/api/v1/users/${userId}`,
      requestOptions,
    );
  },

  async deactivateUser(userId: string): Promise<{ message: string }> {
    const requestOptions: RequestOptions = new RequestOptions();
    return apiHelper.fetchPostJson(
      `${getApplicationServer()}/api/v1/users/${userId}/deactivate`,
      requestOptions,
    );
  },

  async activateUser(userId: string): Promise<{ message: string }> {
    const requestOptions: RequestOptions = new RequestOptions();
    return apiHelper.fetchPostJson(
      `${getApplicationServer()}/api/v1/users/${userId}/activate`,
      requestOptions,
    );
  },

  async changeUserRole(
    userId: string,
    roleRequest: ChangeUserRoleRequest,
  ): Promise<{ message: string }> {
    const requestOptions: RequestOptions = new RequestOptions();
    requestOptions.setBody(JSON.stringify(roleRequest));
    return apiHelper.fetchPutJson(
      `${getApplicationServer()}/api/v1/users/${userId}/role`,
      requestOptions,
    );
  },
};
