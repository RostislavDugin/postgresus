import { InfoCircleOutlined } from '@ant-design/icons';
import { Tooltip } from 'antd';
import dayjs from 'dayjs';
import { useMemo } from 'react';
import { useEffect, useState } from 'react';

import { type BackupConfig, BackupEncryption, backupConfigApi } from '../../../entity/backups';
import { BackupNotificationType } from '../../../entity/backups/model/BackupNotificationType';
import type { Database } from '../../../entity/databases';
import { Period } from '../../../entity/databases/model/Period';
import { IntervalType } from '../../../entity/intervals';
import { getStorageLogoFromType } from '../../../entity/storages/models/getStorageLogoFromType';
import { getLocalDayOfMonth, getLocalWeekday, getUserTimeFormat } from '../../../shared/time/utils';

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
  [BackupNotificationType.BackupFailed]: 'Backup failed',
  [BackupNotificationType.BackupSuccess]: 'Backup success',
};

export const ShowBackupConfigComponent = ({ database }: Props) => {
  const [backupConfig, setBackupConfig] = useState<BackupConfig>();

  // Detect user's preferred time format (12-hour vs 24-hour)
  const timeFormat = useMemo(() => {
    const is12Hour = getUserTimeFormat();
    return {
      use12Hours: is12Hour,
      format: is12Hour ? 'h:mm A' : 'HH:mm',
    };
  }, []);

  useEffect(() => {
    if (database.id) {
      backupConfigApi.getBackupConfigByDbID(database.id).then((res) => {
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

  // Convert UTC weekday/day-of-month to local equivalents for display
  const displayedWeekday: number | undefined =
    backupInterval?.interval === IntervalType.WEEKLY &&
    backupInterval.weekday &&
    backupInterval.timeOfDay
      ? getLocalWeekday(backupInterval.weekday, backupInterval.timeOfDay)
      : backupInterval?.weekday;

  const displayedDayOfMonth: number | undefined =
    backupInterval?.interval === IntervalType.MONTHLY &&
    backupInterval.dayOfMonth &&
    backupInterval.timeOfDay
      ? getLocalDayOfMonth(backupInterval.dayOfMonth, backupInterval.timeOfDay)
      : backupInterval?.dayOfMonth;

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
            <div>{backupInterval?.interval ? intervalLabels[backupInterval.interval] : ''}</div>
          </div>

          {backupInterval?.interval === IntervalType.WEEKLY && (
            <div className="mb-1 flex w-full items-center">
              <div className="min-w-[150px]">Backup weekday</div>
              <div>
                {displayedWeekday
                  ? weekdayLabels[displayedWeekday as keyof typeof weekdayLabels]
                  : ''}
              </div>
            </div>
          )}

          {backupInterval?.interval === IntervalType.MONTHLY && (
            <div className="mb-1 flex w-full items-center">
              <div className="min-w-[150px]">Backup day of month</div>
              <div>{displayedDayOfMonth || ''}</div>
            </div>
          )}

          {backupInterval?.interval !== IntervalType.HOURLY && (
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
            <div className="min-w-[150px]">Store period</div>
            <div>{backupConfig.storePeriod ? periodLabels[backupConfig.storePeriod] : ''}</div>
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
              title="If backup is encrypted, backup files in your storage (S3, local, etc.) cannot be used directly. You can restore backups through Postgresus or download them unencrypted via the 'Download' button."
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
