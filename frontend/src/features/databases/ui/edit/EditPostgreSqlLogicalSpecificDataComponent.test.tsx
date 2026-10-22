import { renderToStaticMarkup } from 'react-dom/server';
import { describe, expect, it, vi } from 'vitest';

import {
  type Database,
  DatabaseType,
  type PostgresqlLogicalDatabase,
  PostgresSslMode,
  PostgresqlVersion,
} from '../../../../entity/databases';
import { LocaleProvider } from '../../../../shared/i18n';
import { EditPostgreSqlLogicalSpecificDataComponent } from './EditPostgreSqlLogicalSpecificDataComponent';

// The entity's public API imports its API client, and the client's config
// module reads window at import time. The tests run in node, so they supply
// window. LocaleProvider then reads localStorage, which here reports no saved
// language.
vi.hoisted(() => {
  vi.stubGlobal('window', {});
  vi.stubGlobal('localStorage', { getItem: () => null });
  vi.stubGlobal('navigator', { language: 'en-US', languages: ['en-US'] });
});

const createPostgresqlDatabase = (change: Partial<PostgresqlLogicalDatabase>): Database => ({
  id: 'database-id',
  name: 'Orders',
  workspaceId: 'workspace-id',
  type: DatabaseType.POSTGRES_LOGICAL,
  notifiers: [],
  postgresqlLogical: {
    id: 'postgresql-id',
    version: PostgresqlVersion.PostgresqlVersion16,
    host: 'db.example.com',
    port: 5432,
    username: 'backup',
    password: '',
    database: 'orders',
    sslMode: PostgresSslMode.Require,
    cpuCount: 1,
    ...change,
  },
});

const renderPostgresqlForm = (database: Database) =>
  renderToStaticMarkup(
    <LocaleProvider>
      <EditPostgreSqlLogicalSpecificDataComponent
        database={database}
        onCancel={() => {}}
        isShowBackButton={false}
        onBack={() => {}}
        isSaveToApi={false}
        onSaved={() => {}}
      />
    </LocaleProvider>,
  );

describe('EditPostgreSqlLogicalSpecificDataComponent', () => {
  it('opens the advanced settings with both exclusion fields when only table data is excluded', () => {
    const markup = renderPostgresqlForm(createPostgresqlDatabase({ excludeTableData: ['logs'] }));

    const wholeTablesLabelIndex = markup.indexOf('Exclude whole tables');
    const tableDataLabelIndex = markup.indexOf('Exclude table data');

    expect(wholeTablesLabelIndex).toBeGreaterThan(-1);
    expect(tableDataLabelIndex).toBeGreaterThan(wholeTablesLabelIndex);
    expect(markup.indexOf('logs', tableDataLabelIndex)).toBeGreaterThan(tableDataLabelIndex);
  });

  it('keeps the advanced settings collapsed when no advanced value is set', () => {
    const markup = renderPostgresqlForm(createPostgresqlDatabase({}));

    expect(markup).toContain('Advanced settings');
    expect(markup).not.toContain('Exclude whole tables');
    expect(markup).not.toContain('Exclude table data');
  });
});
