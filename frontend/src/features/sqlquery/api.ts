import RequestOptions from '../../shared/api/RequestOptions';
import { apiHelper } from '../../shared/api/apiHelper';

export interface ExecuteRequest {
  database_id: string;
  sql: string;
  max_rows?: number;
  timeout_sec?: number;
}

type SqlValue = string | number | boolean | null;

export interface ExecuteResponse {
  columns: string[];
  rows: SqlValue[][];
  row_count: number;
  truncated: boolean;
  execution_ms: number;
}

export const sqlqueryApi = {
  execute: async (payload: ExecuteRequest): Promise<ExecuteResponse> => {
    const opts = new RequestOptions().setBody(JSON.stringify(payload));
    return apiHelper.fetchPostJson<ExecuteResponse>('/api/v1/sqlquery/execute', opts);
  },
};
