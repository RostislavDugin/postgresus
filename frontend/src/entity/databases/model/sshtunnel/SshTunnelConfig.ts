export interface SshTunnelConfig {
  isEnabled: boolean;
  host: string;
  port: number;
  username: string;
  password: string;
  privateKey: string;
  privateKeyPassphrase: string;
}
