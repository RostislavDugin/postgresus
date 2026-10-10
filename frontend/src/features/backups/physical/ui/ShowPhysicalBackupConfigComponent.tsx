import { InfoCircleOutlined } from '@ant-design/icons';
import { Tooltip } from 'antd';
import dayjs from 'dayjs';
import { type JSX, useEffect, useMemo, useState } from 'react';
import { useTranslation } from 'react-i18next';

import {
  type FullBackupsRetention,
  type NextScheduledBackupsResponse,
  PHYSICAL_BACKUP_NOTIFICATION_TYPE_LABEL_KEYS,
  PHYSICAL_RETENTION_LABEL_KEYS,
  type PhysicalBackupConfig,
  PhysicalFullBackupsPolicy,
  PhysicalRetention,
  physicalBackupConfigApi,
  physicalBackupsApi,
} from '../../../../entity/backups/physical';
import {
  BACKUP_ENCRYPTION_LABEL_KEYS,
  NextScheduledBackupComponent,
  type ScheduledBackupTime,
} from '../../../../entity/backups/shared';
import type { Database } from '../../../../entity/databases';
import {
  INTERVAL_TYPE_LABEL_KEYS,
  type Interval,
  IntervalType,
  WEEKDAY_LABEL_KEYS,
  type Weekday,
} from '../../../../entity/intervals';
import { getStorageLogoFromType } from '../../../../entity/storages';
import { type LocalizedText, translateLocalizedText } from '../../../../shared/i18n';
import {
  getUserTimeFormat as getIs12Hour,
  getLocalDayOfMonth,
  getLocalWeekday,
} from '../../../../shared/time/utils';

interface Props {
  database: Database;
}

const getGfsRetentionTexts = (retention: FullBackupsRetention): LocalizedText[] => {
  const texts: LocalizedText[] = [];

  if (retention.gfsHours > 0) {
    texts.push({ key: 'backups.config.gfs.hourly', params: { count: retention.gfsHours } });
  }
  if (retention.gfsDays > 0) {
    texts.push({ key: 'backups.config.gfs.daily', params: { count: retention.gfsDays } });
  }
  if (retention.gfsWeeks > 0) {
    texts.push({ key: 'backups.config.gfs.weekly', params: { count: retention.gfsWeeks } });
  }
  if (retention.gfsMonths > 0) {
    texts.push({ key: 'backups.config.gfs.monthly', params: { count: retention.gfsMonths } });
  }
  if (retention.gfsYears > 0) {
    texts.push({ key: 'backups.config.gfs.yearly', params: { count: retention.gfsYears } });
  }

  return texts;
};

export const ShowPhysicalBackupConfigComponent = ({ database }: Props): JSX.Element => {
  const { t } = useTranslation();
  const listSeparator = t('common.listSeparator');
  const [backupConfig, setBackupConfig] = useState<PhysicalBackupConfig>();
  const [nextScheduledBackups, setNextScheduledBackups] =
    useState<NextScheduledBackupsResponse | null>(null);

  const describeGfsRetention = (retention: FullBackupsRetention): string => {
    const texts = getGfsRetentionTexts(retention);
    if (texts.length === 0) return t('backups.config.gfs.notConfigured');

    return texts.map((text) => translateLocalizedText(text, t)).join(listSeparator);
  };

  const timeFormat = useMemo(() => {
    const is12Hour = getIs12Hour();
    return { use12Hours: is12Hour, format: is12Hour ? 'h:mm A' : 'HH:mm' };
  }, []);

  // Time of day, weekday and day of month are stored in UTC and shown in local time.
  const renderInterval = (
    label: string,
    interval: Interval | undefined,
    nextScheduledBackup: ScheduledBackupTime | null | undefined,
  ): JSX.Element | null => {
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

    return (
      <>
        <div className="mt-4 mb-1 flex w-full items-center">
          <div className="max-w-[150px] min-w-[150px] pr-2">{label}</div>
          <div>{t(INTERVAL_TYPE_LABEL_KEYS[interval.type])}</div>
        </div>

        {interval.type === IntervalType.WEEKLY && (
          <div className="mb-1 flex w-full items-center">
            <div className="min-w-[180px] pr-2">{t('backups.config.schedule.weekday')}</div>
            <div>{displayedWeekday ? t(WEEKDAY_LABEL_KEYS[displayedWeekday as Weekday]) : ''}</div>
          </div>
        )}

        {interval.type === IntervalType.MONTHLY && (
          <div className="mb-1 flex w-full items-center">
            <div className="min-w-[180px] pr-2">{t('backups.config.schedule.dayOfMonth')}</div>
            <div>{displayedDayOfMonth || ''}</div>
          </div>
        )}

        {interval.type === IntervalType.CRON && (
          <div className="mb-1 flex w-full items-center">
            <div className="min-w-[180px] pr-2">{t('backups.config.schedule.cronExpression')}</div>
            <code className="rounded bg-gray-100 px-2 py-0.5 text-sm dark:bg-gray-700">
              {interval.cronExpression || ''}
            </code>
          </div>
        )}

        {interval.type !== IntervalType.HOURLY && interval.type !== IntervalType.CRON && (
          <div className="mb-1 flex w-full items-center">
            <div className="min-w-[180px] pr-2">{t('backups.config.schedule.timeOfDay')}</div>
            <div>{formattedTime}</div>
          </div>
        )}

        {nextScheduledBackup && (
          <NextScheduledBackupComponent
            scheduledBackupTime={nextScheduledBackup}
            labelWidthClassName="min-w-[180px]"
          />
        )}
      </>
    );
  };

  useEffect(() => {
    if (database.id) {
      physicalBackupConfigApi.getPhysicalBackupConfigByDbId(database.id).then((config) => {
        setBackupConfig(config);
      });
      physicalBackupsApi.getNextScheduledBackups(database.id).then(setNextScheduledBackups);
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

  const retentionLabelKey = PHYSICAL_RETENTION_LABEL_KEYS[backupConfig.retention];

  return (
    <div>
      <div className="mb-1 flex w-full items-center">
        <div className="min-w-[180px] pr-2">{t('backups.config.backupsEnabled')}</div>
        <div className={backupConfig.isBackupsEnabled ? '' : 'font-bold text-red-600'}>
          {backupConfig.isBackupsEnabled ? t('common.answers.yes') : t('common.answers.no')}
        </div>
      </div>

      {backupConfig.isBackupsEnabled && (
        <>
          {renderInterval(
            t('backups.physical.config.fullBackupCadence'),
            backupConfig.fullBackupInterval,
            nextScheduledBackups?.nextFullBackup,
          )}
          {renderInterval(
            t('backups.physical.config.incrementalBackupCadence'),
            backupConfig.incrementalBackupInterval,
            nextScheduledBackups?.nextIncrementalBackup,
          )}

          <div className="mt-4 mb-1 flex w-full items-center">
            <div className="min-w-[180px] pr-2">{t('backups.physical.config.retention')}</div>
            <div>{retentionLabelKey ? t(retentionLabelKey) : '-'}</div>
          </div>

          {isShowChainsCount && (
            <div className="mb-1 flex w-full items-center">
              <div className="min-w-[180px] pr-2">{t('backups.physical.config.chainsKept')}</div>
              <div>{backupConfig.chainsRetention?.count ?? '-'}</div>
            </div>
          )}

          {isShowFullBackups && (
            <div className="mb-1 flex w-full items-center">
              <div className="min-w-[180px] pr-2">
                {t('backups.physical.config.fullBackupsKept')}
              </div>
              <div className="flex items-center gap-1">
                {fullBackupsRetention.policy === PhysicalFullBackupsPolicy.LAST_N ? (
                  <span>
                    {t('backups.physical.config.fullBackupsCountValue', {
                      count: fullBackupsRetention.count,
                    })}
                  </span>
                ) : (
                  <span className="flex items-center gap-1">
                    {describeGfsRetention(fullBackupsRetention)}
                    <Tooltip title={t('backups.physical.config.gfsTooltip')}>
                      <InfoCircleOutlined style={{ color: 'gray' }} />
                    </Tooltip>
                  </span>
                )}
              </div>
            </div>
          )}

          <div className="mb-1 flex w-full items-center">
            <div className="min-w-[180px] pr-2">{t('backups.config.storage')}</div>
            <div className="flex items-center">
              <div>{backupConfig.storage?.name || '-'}</div>
              {backupConfig.storage?.type && (
                <img
                  src={getStorageLogoFromType(backupConfig.storage.type)}
                  alt=""
                  className="ml-1 h-4 w-4"
                />
              )}
            </div>
          </div>

          <div className="mb-1 flex w-full items-center">
            <div className="min-w-[180px] pr-2">{t('backups.config.encryption')}</div>
            <div>{t(BACKUP_ENCRYPTION_LABEL_KEYS[backupConfig.encryption])}</div>
          </div>

          <div className="mb-1 flex w-full items-center">
            <div className="min-w-[180px] pr-2">{t('backups.config.notifications')}</div>
            <div>
              {backupConfig.sendNotificationsOn.length > 0
                ? backupConfig.sendNotificationsOn
                    .map((type) => t(PHYSICAL_BACKUP_NOTIFICATION_TYPE_LABEL_KEYS[type]))
                    .join(listSeparator)
                : t('backups.config.noNotifications')}
            </div>
          </div>
        </>
      )}
    </div>
  );
};
