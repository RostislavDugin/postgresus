import { InfoCircleOutlined } from '@ant-design/icons';
import { Tooltip } from 'antd';
import dayjs from 'dayjs';
import { useMemo } from 'react';
import { useEffect, useState } from 'react';
import { useTranslation } from 'react-i18next';

import {
  LOGICAL_BACKUP_NOTIFICATION_TYPE_LABEL_KEYS,
  type LogicalBackupConfig,
  LogicalRetentionPolicyType,
  logicalBackupConfigApi,
} from '../../../../entity/backups/logical';
import { BACKUP_ENCRYPTION_LABEL_KEYS } from '../../../../entity/backups/shared';
import { type Database, PERIOD_LABEL_KEYS } from '../../../../entity/databases';
import {
  INTERVAL_TYPE_LABEL_KEYS,
  IntervalType,
  WEEKDAY_LABEL_KEYS,
  type Weekday,
} from '../../../../entity/intervals';
import { getStorageLogoFromType } from '../../../../entity/storages/models/getStorageLogoFromType';
import { type LocalizedText, translateLocalizedText } from '../../../../shared/i18n';
import {
  getUserTimeFormat as getIs12Hour,
  getLocalDayOfMonth,
  getLocalWeekday,
} from '../../../../shared/time/utils';

interface Props {
  database: Database;
}

const getGfsRetentionTexts = (config: LogicalBackupConfig): LocalizedText[] => {
  const texts: LocalizedText[] = [];

  if (config.retentionGfsHours > 0) {
    texts.push({ key: 'backups.config.gfs.hourly', params: { count: config.retentionGfsHours } });
  }
  if (config.retentionGfsDays > 0) {
    texts.push({ key: 'backups.config.gfs.daily', params: { count: config.retentionGfsDays } });
  }
  if (config.retentionGfsWeeks > 0) {
    texts.push({ key: 'backups.config.gfs.weekly', params: { count: config.retentionGfsWeeks } });
  }
  if (config.retentionGfsMonths > 0) {
    texts.push({ key: 'backups.config.gfs.monthly', params: { count: config.retentionGfsMonths } });
  }
  if (config.retentionGfsYears > 0) {
    texts.push({ key: 'backups.config.gfs.yearly', params: { count: config.retentionGfsYears } });
  }

  return texts;
};

export const ShowLogicalBackupConfigComponent = ({ database }: Props) => {
  const { t } = useTranslation();
  const listSeparator = t('common.listSeparator');
  const [backupConfig, setBackupConfig] = useState<LogicalBackupConfig>();

  const describeGfsRetention = (config: LogicalBackupConfig): string => {
    const texts = getGfsRetentionTexts(config);
    if (texts.length === 0) return t('backups.config.gfs.notConfigured');

    return texts.map((text) => translateLocalizedText(text, t)).join(listSeparator);
  };

  const timeFormat = useMemo(() => {
    const is12Hour = getIs12Hour();
    return {
      use12Hours: is12Hour,
      format: is12Hour ? 'h:mm A' : 'HH:mm',
    };
  }, []);

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
        <div className="min-w-[150px] pr-2">{t('backups.config.backupsEnabled')}</div>
        <div className={backupConfig.isBackupsEnabled ? '' : 'font-bold text-red-600'}>
          {backupConfig.isBackupsEnabled ? t('common.answers.yes') : t('common.answers.no')}
        </div>
      </div>

      {backupConfig.isBackupsEnabled ? (
        <>
          <div className="mt-4 mb-1 flex w-full items-center">
            <div className="min-w-[150px] pr-2">{t('backups.logical.config.interval')}</div>
            <div>
              {backupInterval?.type ? t(INTERVAL_TYPE_LABEL_KEYS[backupInterval.type]) : ''}
            </div>
          </div>

          {backupInterval?.type === IntervalType.WEEKLY && (
            <div className="mb-1 flex w-full items-center">
              <div className="min-w-[150px] pr-2">{t('backups.logical.config.weekday')}</div>
              <div>
                {displayedWeekday ? t(WEEKDAY_LABEL_KEYS[displayedWeekday as Weekday]) : ''}
              </div>
            </div>
          )}

          {backupInterval?.type === IntervalType.MONTHLY && (
            <div className="mb-1 flex w-full items-center">
              <div className="min-w-[150px] pr-2">{t('backups.logical.config.dayOfMonth')}</div>
              <div>{displayedDayOfMonth || ''}</div>
            </div>
          )}

          {backupInterval?.type === IntervalType.CRON && (
            <div className="mb-1 flex w-full items-center">
              <div className="min-w-[150px] pr-2">
                {t('backups.config.schedule.cronExpression')}
              </div>
              <code className="rounded bg-gray-100 px-2 py-0.5 text-sm dark:bg-gray-700">
                {backupInterval?.cronExpression || ''}
              </code>
            </div>
          )}

          {backupInterval?.type !== IntervalType.HOURLY &&
            backupInterval?.type !== IntervalType.CRON && (
              <div className="mb-1 flex w-full items-center">
                <div className="min-w-[150px] pr-2">{t('backups.logical.config.timeOfDay')}</div>
                <div>{formattedTime}</div>
              </div>
            )}

          <div className="mb-1 flex w-full items-center">
            <div className="min-w-[150px] pr-2">
              {t('backups.logical.config.retryIfFailedShort')}
            </div>
            <div>
              {backupConfig.isRetryIfFailed ? t('common.answers.yes') : t('common.answers.no')}
            </div>
          </div>

          {backupConfig.isRetryIfFailed && (
            <div className="mb-1 flex w-full items-center">
              <div className="min-w-[150px] pr-2">{t('backups.logical.config.maxFailedTries')}</div>
              <div>{backupConfig.maxFailedTriesCount}</div>
            </div>
          )}

          <div className="mb-1 flex w-full items-center">
            <div className="min-w-[150px] pr-2">{t('backups.logical.config.retentionPolicy')}</div>
            <div className="flex items-center gap-1">
              {retentionPolicyType === LogicalRetentionPolicyType.TimePeriod && (
                <span>
                  {backupConfig.retentionTimePeriod
                    ? t(PERIOD_LABEL_KEYS[backupConfig.retentionTimePeriod])
                    : ''}
                </span>
              )}
              {retentionPolicyType === LogicalRetentionPolicyType.Count && (
                <span>
                  {t('backups.logical.config.retention.countValue', {
                    count: backupConfig.retentionCount,
                  })}
                </span>
              )}
              {retentionPolicyType === LogicalRetentionPolicyType.GFS && (
                <span className="flex items-center gap-1">
                  {describeGfsRetention(backupConfig)}
                  <Tooltip title={t('backups.logical.config.retention.gfsTooltip')}>
                    <InfoCircleOutlined style={{ color: 'gray' }} />
                  </Tooltip>
                </span>
              )}
            </div>
          </div>

          <div className="mb-1 flex w-full items-center">
            <div className="min-w-[150px] pr-2">{t('backups.config.storage')}</div>
            <div className="flex items-center">
              <div>{backupConfig.storage?.name || ''}</div>
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
            <div className="min-w-[150px] pr-2">{t('backups.config.encryption')}</div>
            <div>{t(BACKUP_ENCRYPTION_LABEL_KEYS[backupConfig.encryption])}</div>

            <Tooltip
              className="cursor-pointer"
              title={t('backups.logical.config.encryptionTooltip')}
            >
              <InfoCircleOutlined className="ml-2" style={{ color: 'gray' }} />
            </Tooltip>
          </div>

          <div className="mb-1 flex w-full items-center">
            <div className="min-w-[150px] pr-2">{t('backups.config.notifications')}</div>
            <div>
              {backupConfig.sendNotificationsOn.length > 0
                ? backupConfig.sendNotificationsOn
                    .map((type) => t(LOGICAL_BACKUP_NOTIFICATION_TYPE_LABEL_KEYS[type]))
                    .join(listSeparator)
                : t('backups.config.noNotifications')}
            </div>
          </div>
        </>
      ) : (
        <div />
      )}
    </div>
  );
};
