import type { SshTunnelConfig } from './SshTunnelConfig';

/**
 * Secrets are only required when creating: an existing database keeps the ones already stored, and
 * the edit form never receives them back from the API.
 */
export function isSshTunnelReadyToTest(
  sshTunnel: SshTunnelConfig | undefined,
  hasStoredSecrets: boolean,
): boolean {
  if (!sshTunnel?.isEnabled) return true;

  if (!sshTunnel.host) return false;
  if (!sshTunnel.port) return false;
  if (!sshTunnel.username) return false;

  return hasStoredSecrets || !!sshTunnel.password || !!sshTunnel.privateKey;
}
