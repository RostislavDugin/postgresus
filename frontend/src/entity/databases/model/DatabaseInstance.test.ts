import { describe, expect, it } from 'vitest';

import type { Database } from './Database';
import {
  UNKNOWN_INSTANCE_KEY,
  buildInstanceKey,
  filterInstancesBySearch,
  formatInstanceTitle,
  groupDatabasesByInstance,
  resolveInitialExpandedKeys,
} from './DatabaseInstance';
import { DatabaseType } from './DatabaseType';

const makeDatabase = (id: string, name: string, host?: string, port?: number): Database =>
  ({
    id,
    name,
    type: DatabaseType.POSTGRES,
    workspaceId: 'ws-1',
    notifiers: [],
    postgresql: host === undefined ? undefined : ({ host, port } as Database['postgresql']),
  }) as Database;

describe('buildInstanceKey', () => {
  it('normalizes case and surrounding whitespace', () => {
    expect(buildInstanceKey('  DB.Example.COM ', 5432)).toBe('db.example.com:5432');
  });

  it('returns the unknown key for a missing host', () => {
    expect(buildInstanceKey(undefined, 5432)).toBe(UNKNOWN_INSTANCE_KEY);
    expect(buildInstanceKey('   ', 5432)).toBe(UNKNOWN_INSTANCE_KEY);
  });
});

describe('groupDatabasesByInstance', () => {
  it('puts databases from the same host and port into one instance', () => {
    const instances = groupDatabasesByInstance([
      makeDatabase('1', 'metabase', '10.0.0.1', 5432),
      makeDatabase('2', 'n8n', '10.0.0.1', 5432),
    ]);

    expect(instances).toHaveLength(1);
    expect(instances[0].key).toBe('10.0.0.1:5432');
    expect(instances[0].databases.map((db) => db.name)).toEqual(['metabase', 'n8n']);
  });

  it('splits the same host on different ports into separate instances', () => {
    const instances = groupDatabasesByInstance([
      makeDatabase('1', 'a', '10.0.0.1', 5432),
      makeDatabase('2', 'b', '10.0.0.1', 5433),
    ]);

    expect(instances.map((instance) => instance.key)).toEqual(['10.0.0.1:5432', '10.0.0.1:5433']);
  });

  it('treats hosts differing only by case as one instance', () => {
    const instances = groupDatabasesByInstance([
      makeDatabase('1', 'a', 'DB.example.com', 5432),
      makeDatabase('2', 'b', 'db.EXAMPLE.com', 5432),
    ]);

    expect(instances).toHaveLength(1);
    expect(instances[0].databases).toHaveLength(2);
  });

  it('sorts instances by host and databases by name', () => {
    const instances = groupDatabasesByInstance([
      makeDatabase('1', 'zulu', 'b-host', 5432),
      makeDatabase('2', 'alpha', 'b-host', 5432),
      makeDatabase('3', 'solo', 'a-host', 5432),
    ]);

    expect(instances.map((instance) => instance.host)).toEqual(['a-host', 'b-host']);
    expect(instances[1].databases.map((db) => db.name)).toEqual(['alpha', 'zulu']);
  });

  it('returns an empty list for an empty input', () => {
    expect(groupDatabasesByInstance([])).toEqual([]);
  });

  it('collects databases without connection data into the unknown instance, placed last', () => {
    const instances = groupDatabasesByInstance([
      makeDatabase('1', 'orphan'),
      makeDatabase('2', 'normal', 'z-host', 5432),
    ]);

    expect(instances).toHaveLength(2);
    expect(instances[1].key).toBe(UNKNOWN_INSTANCE_KEY);
    expect(instances[1].databases.map((db) => db.name)).toEqual(['orphan']);
  });

  it('keys instances with exactly buildInstanceKey applied to the raw host', () => {
    const instances = groupDatabasesByInstance([makeDatabase('1', 'a', '  DB.Example.COM ', 5433)]);

    expect(instances[0].key).toBe(buildInstanceKey('  DB.Example.COM ', 5433));
    expect(instances[0].host).toBe('db.example.com');
  });
});

describe('filterInstancesBySearch', () => {
  const instances = groupDatabasesByInstance([
    makeDatabase('1', 'metabase', 'host-a', 5432),
    makeDatabase('2', 'n8n', 'host-a', 5432),
    makeDatabase('3', 'billing', 'host-b', 5432),
  ]);

  it('returns everything for an empty query', () => {
    expect(filterInstancesBySearch(instances, '   ')).toEqual(instances);
  });

  it('keeps only matching databases and drops instances left empty', () => {
    const filtered = filterInstancesBySearch(instances, 'META');

    expect(filtered).toHaveLength(1);
    expect(filtered[0].host).toBe('host-a');
    expect(filtered[0].databases.map((db) => db.name)).toEqual(['metabase']);
  });

  it('returns an empty list when nothing matches', () => {
    expect(filterInstancesBySearch(instances, 'nothing-here')).toEqual([]);
  });
});

describe('resolveInitialExpandedKeys', () => {
  const instances = groupDatabasesByInstance([
    makeDatabase('1', 'metabase', 'host-a', 5432),
    makeDatabase('2', 'billing', 'host-b', 5432),
  ]);

  it('expands the instance holding the selected database when nothing is stored', () => {
    expect(resolveInitialExpandedKeys(instances, '2', null)).toEqual(['host-b:5432']);
  });

  it('keeps stored keys and drops the ones that no longer exist', () => {
    expect(resolveInitialExpandedKeys(instances, '2', ['host-a:5432', 'gone:5432'])).toEqual([
      'host-a:5432',
    ]);
  });

  it('falls back to the first instance when there is neither a stored state nor a selected database', () => {
    expect(resolveInitialExpandedKeys(instances, undefined, null)).toEqual(['host-a:5432']);
  });

  it('expands nothing when there are no instances at all', () => {
    expect(resolveInitialExpandedKeys([], undefined, null)).toEqual([]);
  });
});

describe('formatInstanceTitle', () => {
  it('hides the default port', () => {
    expect(
      formatInstanceTitle(groupDatabasesByInstance([makeDatabase('1', 'a', 'host-a', 5432)])[0]),
    ).toBe('host-a');
  });

  it('shows a non-default port', () => {
    expect(
      formatInstanceTitle(groupDatabasesByInstance([makeDatabase('1', 'a', 'host-a', 5433)])[0]),
    ).toBe('host-a:5433');
  });

  it('labels the unknown instance', () => {
    expect(formatInstanceTitle(groupDatabasesByInstance([makeDatabase('1', 'a')])[0])).toBe(
      'Unknown host',
    );
  });
});
