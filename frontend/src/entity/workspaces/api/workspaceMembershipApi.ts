import { getApplicationServer } from '../../../constants';
import RequestOptions from '../../../shared/api/RequestOptions';
import { apiHelper } from '../../../shared/api/apiHelper';
import type { AddMemberRequest } from '../model/AddMemberRequest';
import type { AddMemberResponse } from '../model/AddMemberResponse';
import type { ChangeMemberRoleRequest } from '../model/ChangeMemberRoleRequest';
import type { GetMembersResponse } from '../model/GetMembersResponse';
import type { TransferOwnershipRequest } from '../model/TransferOwnershipRequest';

export const workspaceMembershipApi = {
  async getMembers(workspaceId: string): Promise<GetMembersResponse> {
    const requestOptions: RequestOptions = new RequestOptions();
    return apiHelper.fetchGetJson(
      `${getApplicationServer()}/api/v1/workspaces/memberships/${workspaceId}/members`,
      requestOptions,
    );
  },

  async addMember(workspaceId: string, request: AddMemberRequest): Promise<AddMemberResponse> {
    const requestOptions: RequestOptions = new RequestOptions();
    requestOptions.setBody(JSON.stringify(request));
    return apiHelper.fetchPostJson(
      `${getApplicationServer()}/api/v1/workspaces/memberships/${workspaceId}/members`,
      requestOptions,
    );
  },

  async changeMemberRole(
    workspaceId: string,
    userId: string,
    request: ChangeMemberRoleRequest,
  ): Promise<{ message: string }> {
    const requestOptions: RequestOptions = new RequestOptions();
    requestOptions.setBody(JSON.stringify(request));
    return apiHelper.fetchPutJson(
      `${getApplicationServer()}/api/v1/workspaces/memberships/${workspaceId}/members/${userId}/role`,
      requestOptions,
    );
  },

  async removeMember(workspaceId: string, userId: string): Promise<{ message: string }> {
    const requestOptions: RequestOptions = new RequestOptions();
    return apiHelper.fetchDeleteJson(
      `${getApplicationServer()}/api/v1/workspaces/memberships/${workspaceId}/members/${userId}`,
      requestOptions,
    );
  },

  async transferOwnership(
    workspaceId: string,
    request: TransferOwnershipRequest,
  ): Promise<{ message: string }> {
    const requestOptions: RequestOptions = new RequestOptions();
    requestOptions.setBody(JSON.stringify(request));
    return apiHelper.fetchPostJson(
      `${getApplicationServer()}/api/v1/workspaces/memberships/${workspaceId}/transfer-ownership`,
      requestOptions,
    );
  },
};
