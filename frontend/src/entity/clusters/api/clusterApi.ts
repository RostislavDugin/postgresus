import { getApplicationServer } from '../../../constants';
import RequestOptions from '../../../shared/api/RequestOptions';
import { apiHelper } from '../../../shared/api/apiHelper';
import type { Cluster } from '../model/Cluster';

export type PropagationChange = {
  databaseId: string;
  name: string;
  changeStorage: boolean;
  changeSchedule: boolean;
};

export type ApplyPropagationRequest = {
  applyStorage: boolean;
  applySchedule: boolean;
  applyEnableBackups: boolean;
  respectExclusions: boolean;
};

export const clusterApi = {
  async createCluster(cluster: Cluster) {
    const requestOptions: RequestOptions = new RequestOptions();
    requestOptions.setBody(JSON.stringify(cluster));
    return apiHelper.fetchPostJson<Cluster>(
      `${getApplicationServer()}/api/v1/clusters`,
      requestOptions,
    );
  },

  async getClusters(workspaceId: string) {
    const requestOptions: RequestOptions = new RequestOptions();
    return apiHelper.fetchGetJson<Cluster[]>(
      `${getApplicationServer()}/api/v1/clusters?workspace_id=${workspaceId}`,
      requestOptions,
      true,
    );
  },

  async runClusterBackup(clusterId: string) {
    const requestOptions: RequestOptions = new RequestOptions();
    return apiHelper.fetchPostJson(
      `${getApplicationServer()}/api/v1/clusters/${clusterId}/run-backup`,
      requestOptions,
    );
  },
  
  async updateCluster(clusterId: string, cluster: Cluster) {
    const requestOptions: RequestOptions = new RequestOptions();
    requestOptions.setBody(JSON.stringify(cluster));
    return apiHelper.fetchPutJson<Cluster>(
      `${getApplicationServer()}/api/v1/clusters/${clusterId}`,
      requestOptions,
    );
  },

  async getClusterDatabases(clusterId: string) {
    const requestOptions: RequestOptions = new RequestOptions();
    return apiHelper
      .fetchGetJson<{ databases: string[] }>(
        `${getApplicationServer()}/api/v1/clusters/${clusterId}/databases`,
        requestOptions,
        true,
      )
      .then((res) => res.databases);
  },

  async previewPropagation(
    clusterId: string,
    opts: ApplyPropagationRequest,
  ) {
    const requestOptions: RequestOptions = new RequestOptions();
    const qs = new URLSearchParams({
      applyStorage: String(!!opts.applyStorage),
      applySchedule: String(!!opts.applySchedule),
      applyEnableBackups: String(!!opts.applyEnableBackups),
      respectExclusions: String(!!opts.respectExclusions),
    }).toString();
    return apiHelper.fetchGetJson<PropagationChange[]>(
      `${getApplicationServer()}/api/v1/clusters/${clusterId}/propagation/preview?${qs}`,
      requestOptions,
      true,
    );
  },

  async applyPropagation(
    clusterId: string,
    body: ApplyPropagationRequest,
  ) {
    const requestOptions: RequestOptions = new RequestOptions();
    requestOptions.setBody(JSON.stringify(body));
    return apiHelper.fetchPostJson<PropagationChange[]>(
      `${getApplicationServer()}/api/v1/clusters/${clusterId}/propagation/apply`,
      requestOptions,
    );
  },
};
