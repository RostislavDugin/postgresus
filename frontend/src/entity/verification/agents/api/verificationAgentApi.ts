import { getApplicationServer } from '../../../../constants';
import { apiHelper } from '../../../../shared/api/apiHelper';
import RequestOptions from '../../../../shared/api/RequestOptions';
import type {
  CreatedAgentResponse,
  RotateTokenResponse,
  VerificationAgent,
  VerificationAgentAvailability,
} from '../models/VerificationAgent';

export const verificationAgentApi = {
  async listAgents() {
    const requestOptions: RequestOptions = new RequestOptions();
    return apiHelper.fetchGetJson<VerificationAgent[]>(
      `${getApplicationServer()}/api/v1/verification/agents`,
      requestOptions,
      true,
    );
  },

  async getAvailability() {
    const requestOptions: RequestOptions = new RequestOptions();
    return apiHelper.fetchGetJson<VerificationAgentAvailability>(
      `${getApplicationServer()}/api/v1/verification/agents/availability`,
      requestOptions,
      true,
    );
  },

  async createAgent(name: string) {
    const requestOptions: RequestOptions = new RequestOptions();
    requestOptions.setBody(JSON.stringify({ name }));
    return apiHelper.fetchPostJson<CreatedAgentResponse>(
      `${getApplicationServer()}/api/v1/verification/agents`,
      requestOptions,
    );
  },

  async rotateToken(id: string) {
    const requestOptions: RequestOptions = new RequestOptions();
    requestOptions.setBody(JSON.stringify({}));
    return apiHelper.fetchPostJson<RotateTokenResponse>(
      `${getApplicationServer()}/api/v1/verification/agents/${id}/rotate-token`,
      requestOptions,
    );
  },

  async deleteAgent(id: string) {
    const requestOptions: RequestOptions = new RequestOptions();
    return apiHelper.fetchDeleteJson(
      `${getApplicationServer()}/api/v1/verification/agents/${id}`,
      requestOptions,
    );
  },
};
