import { describe, expect, it } from 'vitest';

import type { SshTunnelConfig } from './SshTunnelConfig';
import { createEmptySshTunnelConfig } from './createEmptySshTunnelConfig';
import { isSshTunnelReadyToTest } from './isSshTunnelReadyToTest';

function enabledTunnel(): SshTunnelConfig {
  return {
    ...createEmptySshTunnelConfig(),
    isEnabled: true,
    host: 'bastion.example.com',
    username: 'tunneluser',
    password: 'tunnelpassword',
  };
}

describe('isSshTunnelReadyToTest', () => {
  it('accepts a missing config, since the tunnel is optional', () => {
    expect(isSshTunnelReadyToTest(undefined, false)).toBe(true);
  });

  it('ignores an incomplete config while the tunnel is off', () => {
    expect(isSshTunnelReadyToTest(createEmptySshTunnelConfig(), false)).toBe(true);
  });

  it('accepts a complete config with a password', () => {
    expect(isSshTunnelReadyToTest(enabledTunnel(), false)).toBe(true);
  });

  it('accepts a private key instead of a password', () => {
    expect(
      isSshTunnelReadyToTest({ ...enabledTunnel(), password: '', privateKey: 'key' }, false),
    ).toBe(true);
  });

  it.each(['host', 'username'] as const)('rejects a missing %s', (field) => {
    expect(isSshTunnelReadyToTest({ ...enabledTunnel(), [field]: '' }, false)).toBe(false);
  });

  it('rejects a missing port', () => {
    expect(isSshTunnelReadyToTest({ ...enabledTunnel(), port: 0 }, false)).toBe(false);
  });

  it('rejects a missing secret when creating', () => {
    expect(isSshTunnelReadyToTest({ ...enabledTunnel(), password: '' }, false)).toBe(false);
  });

  it('accepts a missing secret when one is already stored', () => {
    expect(isSshTunnelReadyToTest({ ...enabledTunnel(), password: '' }, true)).toBe(true);
  });
});
