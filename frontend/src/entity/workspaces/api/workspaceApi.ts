import { getApplicationServer } from '../../../constants';
import RequestOptions from '../../../shared/api/RequestOptions';
import { apiHelper } from '../../../shared/api/apiHelper';
import type { GetAuditLogsResponse } from '../../audit-logs/model/GetAuditLogsResponse';
import type { CreateWorkspaceRequest } from '../model/CreateWorkspaceRequest';
import type { ListWorkspacesResponse } from '../model/ListWorkspacesResponse';
import type { Workspace } from '../model/Workspace';
import type { WorkspaceResponse } from '../model/WorkspaceResponse';

export const workspaceApi = {
  async createWorkspace(request: CreateWorkspaceRequest): Promise<WorkspaceResponse> {
    const requestOptions: RequestOptions = new RequestOptions();
    requestOptions.setBody(JSON.stringify(request));
    return apiHelper.fetchPostJson(`${getApplicationServer()}/api/v1/workspaces`, requestOptions);
  },

  async getWorkspaces(): Promise<ListWorkspacesResponse> {
    const requestOptions: RequestOptions = new RequestOptions();
    return apiHelper.fetchGetJson(`${getApplicationServer()}/api/v1/workspaces`, requestOptions);
  },

  async getWorkspace(workspaceId: string): Promise<Workspace> {
    const requestOptions: RequestOptions = new RequestOptions();
    return apiHelper.fetchGetJson(
      `${getApplicationServer()}/api/v1/workspaces/${workspaceId}`,
      requestOptions,
    );
  },

  async updateWorkspace(workspaceId: string, workspace: Workspace): Promise<Workspace> {
    const requestOptions: RequestOptions = new RequestOptions();
    requestOptions.setBody(JSON.stringify(workspace));
    return apiHelper.fetchPutJson(
      `${getApplicationServer()}/api/v1/workspaces/${workspaceId}`,
      requestOptions,
    );
  },

  async deleteWorkspace(workspaceId: string): Promise<{ message: string }> {
    const requestOptions: RequestOptions = new RequestOptions();
    return apiHelper.fetchDeleteJson(
      `${getApplicationServer()}/api/v1/workspaces/${workspaceId}`,
      requestOptions,
    );
  },

  async getWorkspaceAuditLogs(
    workspaceId: string,
    params?: {
      limit?: number;
      offset?: number;
      beforeDate?: string;
    },
  ): Promise<GetAuditLogsResponse> {
    const requestOptions: RequestOptions = new RequestOptions();

    let url = `${getApplicationServer()}/api/v1/workspaces/${workspaceId}/audit-logs`;
    const urlParams = new URLSearchParams();

    if (params?.limit) {
      urlParams.append('limit', params.limit.toString());
    }
    if (params?.offset) {
      urlParams.append('offset', params.offset.toString());
    }
    if (params?.beforeDate) {
      urlParams.append('beforeDate', params.beforeDate);
    }

    if (urlParams.toString()) {
      url += `?${urlParams.toString()}`;
    }

    return apiHelper.fetchGetJson(url, requestOptions);
  },
};
