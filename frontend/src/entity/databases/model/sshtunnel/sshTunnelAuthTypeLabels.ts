import { SshTunnelAuthType } from './SshTunnelAuthType';

export const SSH_TUNNEL_AUTH_TYPE_LABELS: Record<SshTunnelAuthType, string> = {
  [SshTunnelAuthType.PASSWORD]: 'Password',
  [SshTunnelAuthType.PRIVATE_KEY]: 'Private key',
};
