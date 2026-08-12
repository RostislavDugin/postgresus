import { describe, expect, it } from 'vitest';

import type { SshTunnelConfig } from './SshTunnelConfig';
import { hasStoredSshTunnelSecrets } from './hasStoredSshTunnelSecrets';

const enabledTunnel = (): SshTunnelConfig => ({
  isEnabled: true,
  host: 'bastion.example.com',
  port: 22,
  username: 'tunneluser',
  password: '',
  privateKey: '',
  privateKeyPassphrase: '',
});

describe('hasStoredSshTunnelSecrets', () => {
  it('is true for a saved database whose tunnel is enabled', () => {
    expect(hasStoredSshTunnelSecrets(enabledTunnel(), 'db-1')).toBe(true);
  });

  it('is false while the database is still being created', () => {
    expect(hasStoredSshTunnelSecrets(enabledTunnel(), undefined)).toBe(false);
  });

  it('is false for a saved database that never had a tunnel', () => {
    expect(hasStoredSshTunnelSecrets(undefined, 'db-1')).toBe(false);
    expect(hasStoredSshTunnelSecrets({ ...enabledTunnel(), isEnabled: false }, 'db-1')).toBe(false);
  });
});
