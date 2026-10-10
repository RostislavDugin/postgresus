import { renderToStaticMarkup } from 'react-dom/server';
import { describe, expect, it, vi } from 'vitest';

import {
  type PhysicalBackupListItem,
  PhysicalBackupStatus,
  PhysicalBackupType,
} from '../../../../entity/backups/physical';
import { LocaleProvider } from '../../../../shared/i18n';
import { PhysicalBackupTypeCellComponent } from './PhysicalBackupTypeCellComponent';

// The entity's public API imports its API client, and the client's config
// module reads window at import time. The tests run in node, so they supply
// window. LocaleProvider then reads localStorage, which here reports no saved
// language.
vi.hoisted(() => {
  vi.stubGlobal('window', {});
  vi.stubGlobal('localStorage', { getItem: () => null });
  vi.stubGlobal('navigator', { language: 'en-US', languages: ['en-US'] });
});

const FIRST_SEGMENT = '000000010000000000000010';
const LAST_SEGMENT = '0000000100000000000000FF';

const createWalRange = (change: Partial<PhysicalBackupListItem>): PhysicalBackupListItem => ({
  id: 'range-id',
  type: PhysicalBackupType.WAL,
  status: PhysicalBackupStatus.COMPLETED,
  timelineId: 1,
  startLsn: '0/10000000',
  stopLsn: '1/0',
  fromWalFilename: FIRST_SEGMENT,
  toWalFilename: LAST_SEGMENT,
  walSegmentsCount: 240,
  hasGapBefore: false,
  sizeMb: 3840,
  createdAt: new Date('2026-10-10T03:00:00Z'),
  completedAt: new Date('2026-10-10T09:12:00Z'),
  ...change,
});

const renderCell = (backup: PhysicalBackupListItem) =>
  renderToStaticMarkup(
    <LocaleProvider>
      <PhysicalBackupTypeCellComponent backup={backup} />
    </LocaleProvider>,
  );

describe('PhysicalBackupTypeCellComponent', () => {
  it('shows the segment count and the first and last segment of a WAL range', () => {
    const markup = renderCell(createWalRange({}));

    expect(markup).toContain('Segments: 240');
    expect(markup).toContain(FIRST_SEGMENT);
    expect(markup).toContain(LAST_SEGMENT);
    expect(markup).not.toContain('Gap before this range');
  });

  it('marks a WAL range that follows a gap', () => {
    const markup = renderCell(createWalRange({ hasGapBefore: true }));

    expect(markup).toContain('Gap before this range');
  });

  it('shows one segment name for a single-segment range', () => {
    const markup = renderCell(
      createWalRange({ toWalFilename: FIRST_SEGMENT, walSegmentsCount: 1 }),
    );

    expect(markup.split(FIRST_SEGMENT)).toHaveLength(2);
    expect(markup).toContain('Segments: 1');
  });

  it('shows only the type badge for a backup', () => {
    const markup = renderCell(
      createWalRange({
        type: PhysicalBackupType.FULL,
        fromWalFilename: undefined,
        toWalFilename: undefined,
        walSegmentsCount: undefined,
      }),
    );

    expect(markup).toContain('Full');
    expect(markup).not.toContain('Segments:');
  });
});
