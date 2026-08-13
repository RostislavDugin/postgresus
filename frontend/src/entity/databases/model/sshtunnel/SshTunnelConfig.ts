import type { SshTunnelAuthType } from './SshTunnelAuthType';

export interface SshTunnelConfig {
  isEnabled: boolean;
  host: string;
  port: number;
  username: string;
  authType: SshTunnelAuthType;
  password: string;
  privateKey: string;
  privateKeyPassphrase: string;
}
