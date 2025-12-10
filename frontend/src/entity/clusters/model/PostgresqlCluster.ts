export interface PostgresqlCluster {
  id?: string;
  clusterId?: string;
  version: string; // '12' | '13' | ...
  host: string;
  port: number;
  username: string;
  password: string;
  isHttps: boolean;
}
