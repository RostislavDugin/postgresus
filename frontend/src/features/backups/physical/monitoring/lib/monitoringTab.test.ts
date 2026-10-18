import { describe, expect, it, vi } from 'vitest';

import {
  type Database,
  DatabaseType,
  PhysicalDatabaseBackupType,
} from '../../../../../entity/databases';
import { isMonitoringTabShown } from './monitoringTab';

// The entity's public API imports its API client, and the client's config module reads window at
// import time. The tests run in node, so they supply window.
vi.hoisted(() => {
  vi.stubGlobal('window', {});
});

const buildDatabase = (type: DatabaseType, backupType?: PhysicalDatabaseBackupType): Database =>
  ({
    id: 'database-id',
    name: 'database',
    workspaceId: 'workspace-id',
    type,
    notifiers: [],
    postgresqlPhysical: backupType ? { backupType } : undefined,
  }) as Database;

describe('isMonitoringTabShown', () => {
  it('shows the tab for a physical database in WAL streaming mode', () => {
    expect(
      isMonitoringTabShown(
        buildDatabase(
          DatabaseType.POSTGRES_PHYSICAL,
          PhysicalDatabaseBackupType.FULL_INCREMENTAL_WAL_STREAM,
        ),
      ),
    ).toBe(true);
  });

  it('hides the tab for a physical database without WAL streaming', () => {
    expect(
      isMonitoringTabShown(
        buildDatabase(DatabaseType.POSTGRES_PHYSICAL, PhysicalDatabaseBackupType.FULL_INCREMENTAL),
      ),
    ).toBe(false);
  });

  it('hides the tab for a logical database', () => {
    expect(isMonitoringTabShown(buildDatabase(DatabaseType.POSTGRES_LOGICAL))).toBe(false);
  });
});
