import { InfoCircleOutlined } from '@ant-design/icons';
import { Tooltip } from 'antd';
import { CronExpressionParser } from 'cron-parser';
import dayjs from 'dayjs';
import { useMemo } from 'react';
import { useEffect, useState } from 'react';

import {
  type LogicalBackupConfig,
  LogicalBackupNotificationType,
  LogicalRetentionPolicyType,
  logicalBackupConfigApi,
} from '../../../../entity/backups/logical';
import { BackupEncryption } from '../../../../entity/backups/shared';
import type { Database } from '../../../../entity/databases';
import { Period } from '../../../../entity/databases/model/Period';
import { IntervalType } from '../../../../entity/intervals';
import { getStorageLogoFromType } from '../../../../entity/storages/models/getStorageLogoFromType';
import { getUserTimeFormat } from '../../../../shared/time';
import {
  getUserTimeFormat as getIs12Hour,
  getLocalDayOfMonth,
  getLocalWeekday,
} from '../../../../shared/time/utils';

interface Props {
  database: Database;
}

const weekdayLabels = {
  1: 'Mon',
  2: 'Tue',
  3: 'Wed',
  4: 'Thu',
  5: 'Fri',
  6: 'Sat',
  7: 'Sun',
};

const intervalLabels = {
  [IntervalType.HOURLY]: 'Hourly',
  [IntervalType.DAILY]: 'Daily',
  [IntervalType.WEEKLY]: 'Weekly',
  [IntervalType.MONTHLY]: 'Monthly',
  [IntervalType.CRON]: 'Cron',
};

const periodLabels = {
  [Period.DAY]: '1 day',
  [Period.WEEK]: '1 week',
  [Period.MONTH]: '1 month',
  [Period.THREE_MONTH]: '3 months',
  [Period.SIX_MONTH]: '6 months',
  [Period.YEAR]: '1 year',
  [Period.TWO_YEARS]: '2 years',
  [Period.THREE_YEARS]: '3 years',
  [Period.FOUR_YEARS]: '4 years',
  [Period.FIVE_YEARS]: '5 years',
  [Period.FOREVER]: 'Forever',
};

const notificationLabels = {
  [LogicalBackupNotificationType.BackupFailed]: 'Backup failed',
  [LogicalBackupNotificationType.BackupSuccess]: 'Backup success',
};

const formatGfsRetention = (config: LogicalBackupConfig): string => {
  const parts: string[] = [];

  if (config.retentionGfsHours > 0) parts.push(`${config.retentionGfsHours} hourly`);
  if (config.retentionGfsDays > 0) parts.push(`${config.retentionGfsDays} daily`);
  if (config.retentionGfsWeeks > 0) parts.push(`${config.retentionGfsWeeks} weekly`);
  if (config.retentionGfsMonths > 0) parts.push(`${config.retentionGfsMonths} monthly`);
  if (config.retentionGfsYears > 0) parts.push(`${config.retentionGfsYears} yearly`);

  return parts.length > 0 ? parts.join(', ') : 'Not configured';
};

export const ShowLogicalBackupConfigComponent = ({ database }: Props) => {
  const [backupConfig, setBackupConfig] = useState<LogicalBackupConfig>();

  const timeFormat = useMemo(() => {
    const is12Hour = getIs12Hour();
    return {
      use12Hours: is12Hour,
      format: is12Hour ? 'h:mm A' : 'HH:mm',
    };
  }, []);

  const dateTimeFormat = useMemo(() => getUserTimeFormat(), []);

  useEffect(() => {
    if (database.id) {
      logicalBackupConfigApi.getBackupConfigByDbID(database.id).then((res) => {
        setBackupConfig(res);
      });
    }
  }, [database]);

  if (!backupConfig) return <div />;

  const { backupInterval } = backupConfig;

  const localTime = backupInterval?.timeOfDay
    ? dayjs.utc(backupInterval.timeOfDay, 'HH:mm').local()
    : undefined;

  const formattedTime = localTime ? localTime.format(timeFormat.format) : '';

  const displayedWeekday: number | undefined =
    backupInterval?.type === IntervalType.WEEKLY &&
    backupInterval.weekday &&
    backupInterval.timeOfDay
      ? getLocalWeekday(backupInterval.weekday, backupInterval.timeOfDay)
      : backupInterval?.weekday;

  const displayedDayOfMonth: number | undefined =
    backupInterval?.type === IntervalType.MONTHLY &&
    backupInterval.dayOfMonth &&
    backupInterval.timeOfDay
      ? getLocalDayOfMonth(backupInterval.dayOfMonth, backupInterval.timeOfDay)
      : backupInterval?.dayOfMonth;

  const retentionPolicyType =
    backupConfig.retentionPolicyType ?? LogicalRetentionPolicyType.TimePeriod;

  return (
    <div>
      <div className="mb-1 flex w-full items-center">
        <div className="min-w-[150px]">Backups enabled</div>
        <div className={backupConfig.isBackupsEnabled ? '' : 'font-bold text-red-600'}>
          {backupConfig.isBackupsEnabled ? 'Yes' : 'No'}
        </div>
      </div>

      {backupConfig.isBackupsEnabled ? (
        <>
          <div className="mt-4 mb-1 flex w-full items-center">
            <div className="min-w-[150px]">Backup interval</div>
            <div>{backupInterval?.type ? intervalLabels[backupInterval.type] : ''}</div>
          </div>

          {backupInterval?.type === IntervalType.WEEKLY && (
            <div className="mb-1 flex w-full items-center">
              <div className="min-w-[150px]">Backup weekday</div>
              <div>
                {displayedWeekday
                  ? weekdayLabels[displayedWeekday as keyof typeof weekdayLabels]
                  : ''}
              </div>
            </div>
          )}

          {backupInterval?.type === IntervalType.MONTHLY && (
            <div className="mb-1 flex w-full items-center">
              <div className="min-w-[150px]">Backup day of month</div>
              <div>{displayedDayOfMonth || ''}</div>
            </div>
          )}

          {backupInterval?.type === IntervalType.CRON && (
            <>
              <div className="mb-1 flex w-full items-center">
                <div className="min-w-[150px]">Cron expression (UTC)</div>
                <code className="rounded bg-gray-100 px-2 py-0.5 text-sm dark:bg-gray-700">
                  {backupInterval?.cronExpression || ''}
                </code>
              </div>
              {backupInterval?.cronExpression &&
                (() => {
                  try {
                    const interval = CronExpressionParser.parse(backupInterval.cronExpression, {
                      tz: 'UTC',
                    });
                    const nextRun = interval.next().toDate();
                    return (
                      <div className="mb-1 flex w-full items-center text-xs text-gray-600 dark:text-gray-400">
                        <div className="min-w-[150px]" />
                        <div>
                          Next run {dayjs(nextRun).local().format(dateTimeFormat.format)}
                          <br />({dayjs(nextRun).fromNow()})
                        </div>
                      </div>
                    );
                  } catch {
                    return null;
                  }
                })()}
            </>
          )}

          {backupInterval?.type !== IntervalType.HOURLY &&
            backupInterval?.type !== IntervalType.CRON && (
              <div className="mb-1 flex w-full items-center">
                <div className="min-w-[150px]">Backup time of day</div>
                <div>{formattedTime}</div>
              </div>
            )}

          <div className="mb-1 flex w-full items-center">
            <div className="min-w-[150px]">Retry if failed</div>
            <div>{backupConfig.isRetryIfFailed ? 'Yes' : 'No'}</div>
          </div>

          {backupConfig.isRetryIfFailed && (
            <div className="mb-1 flex w-full items-center">
              <div className="min-w-[150px]">Max failed tries count</div>
              <div>{backupConfig.maxFailedTriesCount}</div>
            </div>
          )}

          <div className="mb-1 flex w-full items-center">
            <div className="min-w-[150px]">Retention policy</div>
            <div className="flex items-center gap-1">
              {retentionPolicyType === LogicalRetentionPolicyType.TimePeriod && (
                <span>
                  {backupConfig.retentionTimePeriod
                    ? periodLabels[backupConfig.retentionTimePeriod]
                    : ''}
                </span>
              )}
              {retentionPolicyType === LogicalRetentionPolicyType.Count && (
                <span>Keep last {backupConfig.retentionCount} backups</span>
              )}
              {retentionPolicyType === LogicalRetentionPolicyType.GFS && (
                <span className="flex items-center gap-1">
                  {formatGfsRetention(backupConfig)}
                  <Tooltip title="Grandfather-Father-Son rotation: keep the last N hourly, daily, weekly, monthly and yearly backups.">
                    <InfoCircleOutlined style={{ color: 'gray' }} />
                  </Tooltip>
                </span>
              )}
            </div>
          </div>

          <div className="mb-1 flex w-full items-center">
            <div className="min-w-[150px]">Storage</div>
            <div className="flex items-center">
              <div>{backupConfig.storage?.name || ''}</div>
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
            <div className="min-w-[150px]">Encryption</div>
            <div>{backupConfig.encryption === BackupEncryption.ENCRYPTED ? 'Enabled' : 'None'}</div>

            <Tooltip
              className="cursor-pointer"
              title="If backup is encrypted, backup files in your storage (S3, local, etc.) cannot be used directly. You can restore backups through Databasus or download them unencrypted via the 'Download' button."
            >
              <InfoCircleOutlined className="ml-2" style={{ color: 'gray' }} />
            </Tooltip>
          </div>

          <div className="mb-1 flex w-full items-center">
            <div className="min-w-[150px]">Notifications</div>
            <div>
              {backupConfig.sendNotificationsOn.length > 0
                ? backupConfig.sendNotificationsOn
                    .map((type) => notificationLabels[type])
                    .join(', ')
                : 'None'}
            </div>
          </div>
        </>
      ) : (
        <div />
      )}
    </div>
  );
};
