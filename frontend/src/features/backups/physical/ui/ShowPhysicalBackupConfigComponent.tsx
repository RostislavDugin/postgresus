import { InfoCircleOutlined } from '@ant-design/icons';
import { Tooltip } from 'antd';
import { CronExpressionParser } from 'cron-parser';
import dayjs from 'dayjs';
import { type JSX, useEffect, useMemo, useState } from 'react';

import {
  type FullBackupsRetention,
  type PhysicalBackupConfig,
  PhysicalBackupNotificationType,
  PhysicalFullBackupsPolicy,
  PhysicalRetention,
  physicalBackupConfigApi,
} from '../../../../entity/backups/physical';
import { BackupEncryption } from '../../../../entity/backups/shared';
import type { Database } from '../../../../entity/databases';
import { type Interval, IntervalType } from '../../../../entity/intervals';
import { getStorageLogoFromType } from '../../../../entity/storages';
import { getUserTimeFormat } from '../../../../shared/time';
import {
  getUserTimeFormat as getIs12Hour,
  getLocalDayOfMonth,
  getLocalWeekday,
} from '../../../../shared/time/utils';

interface Props {
  database: Database;
}

const weekdayLabels: Record<number, string> = {
  1: 'Mon',
  2: 'Tue',
  3: 'Wed',
  4: 'Thu',
  5: 'Fri',
  6: 'Sat',
  7: 'Sun',
};

const intervalLabels: Record<IntervalType, string> = {
  [IntervalType.HOURLY]: 'Hourly',
  [IntervalType.DAILY]: 'Daily',
  [IntervalType.WEEKLY]: 'Weekly',
  [IntervalType.MONTHLY]: 'Monthly',
  [IntervalType.CRON]: 'Cron',
};

const notificationLabels: Record<PhysicalBackupNotificationType, string> = {
  [PhysicalBackupNotificationType.BACKUP_SUCCESS]: 'Backup success',
  [PhysicalBackupNotificationType.BACKUP_FAILED]: 'Backup failed',
  [PhysicalBackupNotificationType.CHAIN_BROKEN]: 'Chain broken',
  [PhysicalBackupNotificationType.WAL_GAP]: 'WAL gap',
};

const retentionLabels: Record<PhysicalRetention, string> = {
  [PhysicalRetention.CHAINS]: 'Chains',
  [PhysicalRetention.FULL_BACKUPS]: 'Full backups',
  [PhysicalRetention.CHAINS_AND_FULL_BACKUPS]: 'Chains and full backups',
};

const formatGfsRetention = (retention: FullBackupsRetention): string => {
  const parts: string[] = [];

  if (retention.gfsHours > 0) parts.push(`${retention.gfsHours} hourly`);
  if (retention.gfsDays > 0) parts.push(`${retention.gfsDays} daily`);
  if (retention.gfsWeeks > 0) parts.push(`${retention.gfsWeeks} weekly`);
  if (retention.gfsMonths > 0) parts.push(`${retention.gfsMonths} monthly`);
  if (retention.gfsYears > 0) parts.push(`${retention.gfsYears} yearly`);

  return parts.length > 0 ? parts.join(', ') : 'Not configured';
};

export const ShowPhysicalBackupConfigComponent = ({ database }: Props): JSX.Element => {
  const [backupConfig, setBackupConfig] = useState<PhysicalBackupConfig>();

  const timeFormat = useMemo(() => {
    const is12Hour = getIs12Hour();
    return { use12Hours: is12Hour, format: is12Hour ? 'h:mm A' : 'HH:mm' };
  }, []);

  const dateTimeFormat = useMemo(() => getUserTimeFormat(), []);

  // Read-only mirror of an interval: type, plus local time / weekday / day-of-month
  // converted from UTC, or the next cron run.
  const renderInterval = (label: string, interval?: Interval): JSX.Element | null => {
    if (!interval?.type) return null;

    const localTime = interval.timeOfDay
      ? dayjs.utc(interval.timeOfDay, 'HH:mm').local()
      : undefined;
    const formattedTime = localTime ? localTime.format(timeFormat.format) : '';

    const displayedWeekday =
      interval.type === IntervalType.WEEKLY && interval.weekday && interval.timeOfDay
        ? getLocalWeekday(interval.weekday, interval.timeOfDay)
        : interval.weekday;

    const displayedDayOfMonth =
      interval.type === IntervalType.MONTHLY && interval.dayOfMonth && interval.timeOfDay
        ? getLocalDayOfMonth(interval.dayOfMonth, interval.timeOfDay)
        : interval.dayOfMonth;

    const renderCronNextRun = (): JSX.Element | null => {
      if (!interval.cronExpression) return null;
      try {
        const parsed = CronExpressionParser.parse(interval.cronExpression, { tz: 'UTC' });
        const nextRun = parsed.next().toDate();
        return (
          <div className="mb-1 flex w-full items-center text-xs text-gray-600 dark:text-gray-400">
            <div className="min-w-[180px]" />
            <div>
              Next run {dayjs(nextRun).local().format(dateTimeFormat.format)}
              <br />({dayjs(nextRun).fromNow()})
            </div>
          </div>
        );
      } catch {
        return null;
      }
    };

    return (
      <>
        <div className="mt-4 mb-1 flex w-full items-center">
          <div className="max-w-[150px] min-w-[150px]">{label}</div>
          <div>{intervalLabels[interval.type]}</div>
        </div>

        {interval.type === IntervalType.WEEKLY && (
          <div className="mb-1 flex w-full items-center">
            <div className="min-w-[180px]">Weekday</div>
            <div>{displayedWeekday ? weekdayLabels[displayedWeekday] : ''}</div>
          </div>
        )}

        {interval.type === IntervalType.MONTHLY && (
          <div className="mb-1 flex w-full items-center">
            <div className="min-w-[180px]">Day of month</div>
            <div>{displayedDayOfMonth || ''}</div>
          </div>
        )}

        {interval.type === IntervalType.CRON && (
          <>
            <div className="mb-1 flex w-full items-center">
              <div className="min-w-[180px]">Cron expression (UTC)</div>
              <code className="rounded bg-gray-100 px-2 py-0.5 text-sm dark:bg-gray-700">
                {interval.cronExpression || ''}
              </code>
            </div>
            {renderCronNextRun()}
          </>
        )}

        {interval.type !== IntervalType.HOURLY && interval.type !== IntervalType.CRON && (
          <div className="mb-1 flex w-full items-center">
            <div className="min-w-[180px]">Time of day</div>
            <div>{formattedTime}</div>
          </div>
        )}
      </>
    );
  };

  useEffect(() => {
    if (database.id) {
      physicalBackupConfigApi.getPhysicalBackupConfigByDbId(database.id).then((config) => {
        setBackupConfig(config);
      });
    }
  }, [database]);

  if (!backupConfig) return <div />;

  const fullBackupsRetention = backupConfig.fullBackupsRetention;

  const isShowChainsCount =
    backupConfig.retention === PhysicalRetention.CHAINS ||
    backupConfig.retention === PhysicalRetention.CHAINS_AND_FULL_BACKUPS;

  const isShowFullBackups =
    backupConfig.retention === PhysicalRetention.FULL_BACKUPS ||
    backupConfig.retention === PhysicalRetention.CHAINS_AND_FULL_BACKUPS;

  return (
    <div>
      <div className="mb-1 flex w-full items-center">
        <div className="min-w-[180px]">Backups enabled</div>
        <div className={backupConfig.isBackupsEnabled ? '' : 'font-bold text-red-600'}>
          {backupConfig.isBackupsEnabled ? 'Yes' : 'No'}
        </div>
      </div>

      {backupConfig.isBackupsEnabled && (
        <>
          {renderInterval('Full backup cadence', backupConfig.fullBackupInterval)}
          {renderInterval('Incremental backup cadence', backupConfig.incrementalBackupInterval)}

          <div className="mt-4 mb-1 flex w-full items-center">
            <div className="min-w-[180px]">Retention</div>
            <div>{retentionLabels[backupConfig.retention] ?? '-'}</div>
          </div>

          {isShowChainsCount && (
            <div className="mb-1 flex w-full items-center">
              <div className="min-w-[180px]">Chains kept</div>
              <div>{backupConfig.chainsRetention?.count ?? '-'}</div>
            </div>
          )}

          {isShowFullBackups && (
            <div className="mb-1 flex w-full items-center">
              <div className="min-w-[180px]">Full backups kept</div>
              <div className="flex items-center gap-1">
                {fullBackupsRetention.policy === PhysicalFullBackupsPolicy.LAST_N ? (
                  <span>Last {fullBackupsRetention.count} full backups</span>
                ) : (
                  <span className="flex items-center gap-1">
                    {formatGfsRetention(fullBackupsRetention)}
                    <Tooltip title="Grandfather-Father-Son rotation: keep the last N hourly, daily, weekly, monthly and yearly full backups.">
                      <InfoCircleOutlined style={{ color: 'gray' }} />
                    </Tooltip>
                  </span>
                )}
              </div>
            </div>
          )}

          <div className="mb-1 flex w-full items-center">
            <div className="min-w-[180px]">Storage</div>
            <div className="flex items-center">
              <div>{backupConfig.storage?.name || '-'}</div>
              {backupConfig.storage?.type && (
                <img
                  src={getStorageLogoFromType(backupConfig.storage.type)}
                  alt="storageIcon"
                  className="ml-1 h-4 w-4"
                />
              )}
            </div>
          </div>

          <div className="mb-1 flex w-full items-center">
            <div className="min-w-[180px]">Encryption</div>
            <div>{backupConfig.encryption === BackupEncryption.ENCRYPTED ? 'Enabled' : 'None'}</div>
          </div>

          <div className="mb-1 flex w-full items-center">
            <div className="min-w-[180px]">Notifications</div>
            <div>
              {backupConfig.sendNotificationsOn.length > 0
                ? backupConfig.sendNotificationsOn
                    .map((type) => notificationLabels[type])
                    .join(', ')
                : 'None'}
            </div>
          </div>
        </>
      )}
    </div>
  );
};
