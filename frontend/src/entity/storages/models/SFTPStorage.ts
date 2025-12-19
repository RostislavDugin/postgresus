export interface SFTPStorage {
  host: string;
  port: number;
  username: string;
  password?: string;
  privateKey?: string;
  path?: string;
  skipHostKeyVerify?: boolean;
}
