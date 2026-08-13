import type { Database } from './Database';

export const UNKNOWN_INSTANCE_KEY = '';

const DEFAULT_POSTGRES_PORT = 5432;

export interface DatabaseInstance {
  key: string;
  host: string;
  port: number;
  databases: Database[];
}

export function buildInstanceKey(host: string | undefined, port: number | undefined): string {
  const normalizedHost = normalizeHost(host);
  if (!normalizedHost) return UNKNOWN_INSTANCE_KEY;

  return `${normalizedHost}:${port ?? 0}`;
}

export function groupDatabasesByInstance(databases: Database[]): DatabaseInstance[] {
  const byKey = new Map<string, DatabaseInstance>();

  for (const database of databases) {
    const host = normalizeHost(database.postgresql?.host);
    const port = database.postgresql?.port ?? 0;
    const key = buildInstanceKey(database.postgresql?.host, port);

    const existing = byKey.get(key);
    if (existing) {
      existing.databases.push(database);
    } else {
      byKey.set(key, { key, host, port, databases: [database] });
    }
  }

  return [...byKey.values()]
    .map((instance) => ({
      ...instance,
      databases: [...instance.databases].sort((a, b) => a.name.localeCompare(b.name)),
    }))
    .sort(compareInstances);
}

export function filterInstancesBySearch(
  instances: DatabaseInstance[],
  query: string,
): DatabaseInstance[] {
  const normalizedQuery = query.trim().toLowerCase();
  if (!normalizedQuery) return instances;

  return instances
    .map((instance) => ({
      ...instance,
      databases: instance.databases.filter((database) =>
        database.name.toLowerCase().includes(normalizedQuery),
      ),
    }))
    .filter((instance) => instance.databases.length > 0);
}

export function resolveInitialExpandedKeys(
  instances: DatabaseInstance[],
  selectedDatabaseId: string | undefined,
  storedKeys: string[] | null,
): string[] {
  const existingKeys = new Set(instances.map((instance) => instance.key));

  if (storedKeys) {
    return storedKeys.filter((key) => existingKeys.has(key));
  }

  const selectedInstance = instances.find((instance) =>
    instance.databases.some((database) => database.id === selectedDatabaseId),
  );

  if (selectedInstance) return [selectedInstance.key];

  return instances.length > 0 ? [instances[0].key] : [];
}

export function formatInstanceTitle(instance: DatabaseInstance): string {
  if (instance.key === UNKNOWN_INSTANCE_KEY) return 'Unknown host';
  if (instance.port === DEFAULT_POSTGRES_PORT) return instance.host;

  return `${instance.host}:${instance.port}`;
}

function compareInstances(a: DatabaseInstance, b: DatabaseInstance): number {
  if (a.key === UNKNOWN_INSTANCE_KEY) return 1;
  if (b.key === UNKNOWN_INSTANCE_KEY) return -1;

  const byHost = a.host.localeCompare(b.host);
  return byHost !== 0 ? byHost : a.port - b.port;
}

function normalizeHost(host: string | undefined): string {
  return (host ?? '').trim().toLowerCase();
}
