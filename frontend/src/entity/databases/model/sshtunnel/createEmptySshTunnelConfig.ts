import type { SshTunnelConfig } from './SshTunnelConfig';

export const DEFAULT_SSH_PORT = 22;

export function createEmptySshTunnelConfig(): SshTunnelConfig {
  return {
    isEnabled: false,
    host: '',
    port: DEFAULT_SSH_PORT,
    username: '',
    password: '',
    privateKey: '',
    privateKeyPassphrase: '',
  };
}
