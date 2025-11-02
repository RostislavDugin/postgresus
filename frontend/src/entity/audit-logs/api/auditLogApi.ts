import { getApplicationServer } from '../../../constants';
import RequestOptions from '../../../shared/api/RequestOptions';
import { apiHelper } from '../../../shared/api/apiHelper';
import type { GetAuditLogsRequest } from '../model/GetAuditLogsRequest';
import type { GetAuditLogsResponse } from '../model/GetAuditLogsResponse';

export const auditLogApi = {
  async getGlobalAuditLogs(request?: GetAuditLogsRequest): Promise<GetAuditLogsResponse> {
    const requestOptions: RequestOptions = new RequestOptions();

    let url = `${getApplicationServer()}/api/v1/audit-logs/global`;
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

    if (params.toString()) {
      url += `?${params.toString()}`;
    }

    return apiHelper.fetchGetJson(url, requestOptions);
  },

  async getUserAuditLogs(
    userId: string,
    request?: GetAuditLogsRequest,
  ): Promise<GetAuditLogsResponse> {
    const requestOptions: RequestOptions = new RequestOptions();

    let url = `${getApplicationServer()}/api/v1/audit-logs/users/${userId}`;
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

    if (params.toString()) {
      url += `?${params.toString()}`;
    }

    return apiHelper.fetchGetJson(url, requestOptions);
  },
};
