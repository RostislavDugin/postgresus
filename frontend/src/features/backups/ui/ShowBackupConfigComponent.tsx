import dayjs from 'dayjs';
import { useMemo } from 'react';
import { useEffect, useState } from 'react';
import { useTranslation } from 'react-i18next';

import { type BackupConfig, backupConfigApi } from '../../../entity/backups';
import { BackupNotificationType } from '../../../entity/backups/model/BackupNotificationType';
import type { Database } from '../../../entity/databases';
import { Period } from '../../../entity/databases/model/Period';
import { IntervalType } from '../../../entity/intervals';
import { getStorageLogoFromType } from '../../../entity/storages/models/getStorageLogoFromType';
import { getLocalDayOfMonth, getLocalWeekday, getUserTimeFormat } from '../../../shared/time/utils';

interface Props {
  database: Database;
}

export const ShowBackupConfigComponent = ({ database }: Props) => {
  const { t } = useTranslation(['backup', 'common']);
  const [backupConfig, setBackupConfig] = useState<BackupConfig>();

  const weekdayLabels = useMemo(() => ({
    1: t('common:time.mon'),
    2: t('common:time.tue'),
    3: t('common:time.wed'),
    4: t('common:time.thu'),
    5: t('common:time.fri'),
    6: t('common:time.sat'),
    7: t('common:time.sun'),
  }), [t]);

  const intervalLabels = useMemo(() => ({
    [IntervalType.HOURLY]: t('backup:interval.hourly'),
    [IntervalType.DAILY]: t('backup:interval.daily'),
    [IntervalType.WEEKLY]: t('backup:interval.weekly'),
    [IntervalType.MONTHLY]: t('backup:interval.monthly'),
  }), [t]);

  const periodLabels = useMemo(() => ({
    [Period.DAY]: t('backup:period.day'),
    [Period.WEEK]: t('backup:period.week'),
    [Period.MONTH]: t('backup:period.month'),
    [Period.THREE_MONTH]: t('backup:period.three_months'),
    [Period.SIX_MONTH]: t('backup:period.six_months'),
    [Period.YEAR]: t('backup:period.year'),
    [Period.TWO_YEARS]: t('backup:period.two_years'),
    [Period.THREE_YEARS]: t('backup:period.three_years'),
    [Period.FOUR_YEARS]: t('backup:period.four_years'),
    [Period.FIVE_YEARS]: t('backup:period.five_years'),
    [Period.FOREVER]: t('backup:period.forever'),
  }), [t]);

  const notificationLabels = useMemo(() => ({
    [BackupNotificationType.BackupFailed]: t('backup:notification_type.backup_failed'),
    [BackupNotificationType.BackupSuccess]: t('backup:notification_type.backup_success'),
  }), [t]);

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
        <div className="min-w-[150px]">{t('backup:form.backups_enabled')}</div>
        <div>{backupConfig.isBackupsEnabled ? t('common:common.yes') : t('common:common.no')}</div>
      </div>

      {backupConfig.isBackupsEnabled ? (
        <>
          <div className="mt-4 mb-1 flex w-full items-center">
            <div className="min-w-[150px]">{t('backup:form.backup_interval')}</div>
            <div>{backupInterval?.interval ? intervalLabels[backupInterval.interval] : ''}</div>
          </div>

          {backupInterval?.interval === IntervalType.WEEKLY && (
            <div className="mb-1 flex w-full items-center">
              <div className="min-w-[150px]">{t('backup:form.backup_weekday')}</div>
              <div>
                {displayedWeekday
                  ? weekdayLabels[displayedWeekday as keyof typeof weekdayLabels]
                  : ''}
              </div>
            </div>
          )}

          {backupInterval?.interval === IntervalType.MONTHLY && (
            <div className="mb-1 flex w-full items-center">
              <div className="min-w-[150px]">{t('backup:form.backup_day_of_month')}</div>
              <div>{displayedDayOfMonth || ''}</div>
            </div>
          )}

          {backupInterval?.interval !== IntervalType.HOURLY && (
            <div className="mb-1 flex w-full items-center">
              <div className="min-w-[150px]">{t('backup:form.backup_time_of_day')}</div>
              <div>{formattedTime}</div>
            </div>
          )}

          <div className="mb-1 flex w-full items-center">
            <div className="min-w-[150px]">{t('backup:form.retry_if_failed')}</div>
            <div>{backupConfig.isRetryIfFailed ? t('common:common.yes') : t('common:common.no')}</div>
          </div>

          {backupConfig.isRetryIfFailed && (
            <div className="mb-1 flex w-full items-center">
              <div className="min-w-[150px]">{t('backup:form.max_failed_tries')}</div>
              <div>{backupConfig.maxFailedTriesCount}</div>
            </div>
          )}

          <div className="mb-1 flex w-full items-center">
            <div className="min-w-[150px]">{t('backup:form.store_period')}</div>
            <div>{backupConfig.storePeriod ? periodLabels[backupConfig.storePeriod] : ''}</div>
          </div>

          <div className="mb-1 flex w-full items-center">
            <div className="min-w-[150px]">{t('backup:form.storage')}</div>
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
            <div className="min-w-[150px]">{t('backup:form.notifications')}</div>
            <div>
              {backupConfig.sendNotificationsOn.length > 0
                ? backupConfig.sendNotificationsOn
                    .map((type) => notificationLabels[type])
                    .join(', ')
                : t('common:common.none')}
            </div>
          </div>
        </>
      ) : (
        <div />
      )}
    </div>
  );
};
