import type { SshTunnelConfig } from './SshTunnelConfig';

/**
 * Answers from the saved database, never from the edited copy: a database that never had a tunnel
 * would otherwise show the masked placeholder the moment the checkbox is ticked.
 */
export function hasStoredSshTunnelSecrets(
  sshTunnel: SshTunnelConfig | undefined,
  databaseId: string | undefined,
): boolean {
  return !!databaseId && !!sshTunnel?.isEnabled;
}
