import type { MongodbDatabase } from './MongodbDatabase';

/**
 * SRV resolves its own host list from DNS and never carries a port, so it can never travel through
 * a forwarded local port. The backend rejects the combination, so the form must not be able to
 * produce it - including via a pasted mongodb+srv:// connection string.
 */
export function disableSrvWhenTunneled(mongodb: MongodbDatabase): MongodbDatabase {
  if (!mongodb.sshTunnel?.isEnabled) return mongodb;

  return { ...mongodb, isSrv: false };
}
