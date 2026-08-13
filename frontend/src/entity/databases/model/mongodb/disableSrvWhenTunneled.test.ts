import { describe, expect, it } from 'vitest';

import { SshTunnelAuthType } from '../sshtunnel/SshTunnelAuthType';
import type { SshTunnelConfig } from '../sshtunnel/SshTunnelConfig';
import type { MongodbDatabase } from './MongodbDatabase';
import { MongodbVersion } from './MongodbVersion';
import { disableSrvWhenTunneled } from './disableSrvWhenTunneled';

const enabledTunnel = (): SshTunnelConfig => ({
  isEnabled: true,
  authType: SshTunnelAuthType.PASSWORD,
  host: 'bastion.example.com',
  port: 22,
  username: 'tunneluser',
  password: 'tunnelpassword',
  privateKey: '',
  privateKeyPassphrase: '',
});

const srvDatabase = (sshTunnel?: SshTunnelConfig): MongodbDatabase => ({
  id: 'db-1',
  version: MongodbVersion.MongodbVersion70,
  host: 'cluster0.example.mongodb.net',
  port: 27017,
  username: 'testuser',
  password: 'testpassword',
  database: 'testdb',
  authDatabase: 'admin',
  isHttps: false,
  isSrv: true,
  isDirectConnection: false,
  cpuCount: 1,
  sshTunnel,
});

describe('disableSrvWhenTunneled', () => {
  it('turns SRV off when the tunnel is enabled', () => {
    expect(disableSrvWhenTunneled(srvDatabase(enabledTunnel())).isSrv).toBe(false);
  });

  it('leaves SRV alone when there is no tunnel', () => {
    expect(disableSrvWhenTunneled(srvDatabase()).isSrv).toBe(true);
    expect(
      disableSrvWhenTunneled(srvDatabase({ ...enabledTunnel(), isEnabled: false })).isSrv,
    ).toBe(true);
  });

  it('does not touch direct connection, which the backend forces per operation', () => {
    expect(disableSrvWhenTunneled(srvDatabase(enabledTunnel())).isDirectConnection).toBe(false);
  });

  it('returns the same object when nothing changes', () => {
    const database = srvDatabase();

    expect(disableSrvWhenTunneled(database)).toBe(database);
  });
});
