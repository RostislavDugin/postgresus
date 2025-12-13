export interface FTPStorage {
  host: string;
  port: number;
  username: string;
  password: string;
  useSsl: boolean;
  skipTlsVerify?: boolean;
  path?: string;
}
