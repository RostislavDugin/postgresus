import { InfoCircleOutlined } from '@ant-design/icons';
import {
  Button,
  Checkbox,
  InputNumber,
  Modal,
  Select,
  Spin,
  Switch,
  TimePicker,
  Tooltip,
} from 'antd';
import dayjs, { Dayjs } from 'dayjs';
import { useEffect, useMemo, useState } from 'react';
import { useTranslation } from 'react-i18next';

import { type BackupConfig, backupConfigApi } from '../../../entity/backups';
import { BackupNotificationType } from '../../../entity/backups/model/BackupNotificationType';
import type { Database } from '../../../entity/databases';
import { Period } from '../../../entity/databases/model/Period';
import { type Interval, IntervalType } from '../../../entity/intervals';
import { type Storage, getStorageLogoFromType, storageApi } from '../../../entity/storages';
import {
  getLocalDayOfMonth,
  getLocalWeekday,
  getUserTimeFormat,
  getUtcDayOfMonth,
  getUtcWeekday,
} from '../../../shared/time/utils';
import { ConfirmationComponent } from '../../../shared/ui';
import { EditStorageComponent } from '../../storages/ui/edit/EditStorageComponent';

interface Props {
  database: Database;

  isShowBackButton: boolean;
  onBack: () => void;

  isShowCancelButton?: boolean;
  onCancel: () => void;

  saveButtonText?: string;
  isSaveToApi: boolean;
  onSaved: (backupConfig: BackupConfig) => void;
}

export const EditBackupConfigComponent = ({
  database,

  isShowBackButton,
  onBack,

  isShowCancelButton,
  onCancel,
  saveButtonText,
  isSaveToApi,
  onSaved,
}: Props) => {
  const { t } = useTranslation(['backup', 'common', 'storage']);
  const [backupConfig, setBackupConfig] = useState<BackupConfig>();
  const [isUnsaved, setIsUnsaved] = useState(false);
  const [isSaving, setIsSaving] = useState(false);

  const [storages, setStorages] = useState<Storage[]>([]);
  const [isStoragesLoading, setIsStoragesLoading] = useState(false);
  const [isShowCreateStorage, setShowCreateStorage] = useState(false);

  const [isShowWarn, setIsShowWarn] = useState(false);
  const [isShowBackupDisableConfirm, setIsShowBackupDisableConfirm] = useState(false);

  const weekdayOptions = useMemo(() => [
    { value: 1, label: t('common:time.mon') },
    { value: 2, label: t('common:time.tue') },
    { value: 3, label: t('common:time.wed') },
    { value: 4, label: t('common:time.thu') },
    { value: 5, label: t('common:time.fri') },
    { value: 6, label: t('common:time.sat') },
    { value: 7, label: t('common:time.sun') },
  ], [t]);

  const timeFormat = useMemo(() => {
    const is12 = getUserTimeFormat();
    return { use12Hours: is12, format: is12 ? 'h:mm A' : 'HH:mm' };
  }, []);

  const updateBackupConfig = (patch: Partial<BackupConfig>) => {
    setBackupConfig((prev) => (prev ? { ...prev, ...patch } : prev));
    setIsUnsaved(true);
  };

  const saveInterval = (patch: Partial<Interval>) => {
    setBackupConfig((prev) => {
      if (!prev) return prev;

      const updatedBackupInterval = { ...(prev.backupInterval ?? {}), ...patch };

      if (!updatedBackupInterval.id && prev.backupInterval?.id) {
        updatedBackupInterval.id = prev.backupInterval.id;
      }

      return { ...prev, backupInterval: updatedBackupInterval as Interval };
    });

    setIsUnsaved(true);
  };

  const saveBackupConfig = async () => {
    if (!backupConfig) return;

    if (isSaveToApi) {
      setIsSaving(true);
      try {
        await backupConfigApi.saveBackupConfig(backupConfig);
        setIsUnsaved(false);
      } catch (e) {
        alert((e as Error).message);
      }
      setIsSaving(false);
    }

    onSaved(backupConfig);
  };

  const loadStorages = async () => {
    setIsStoragesLoading(true);

    try {
      const storages = await storageApi.getStorages();
      setStorages(storages);
    } catch (e) {
      alert((e as Error).message);
    }

    setIsStoragesLoading(false);
  };

  useEffect(() => {
    if (database.id) {
      backupConfigApi.getBackupConfigByDbID(database.id).then((res) => {
        setBackupConfig(res);
        setIsUnsaved(false);
        setIsSaving(false);
      });
    } else {
      setBackupConfig({
        databaseId: database.id,
        isBackupsEnabled: true,
        backupInterval: {
          id: undefined as unknown as string,
          interval: IntervalType.DAILY,
          timeOfDay: '00:00',
        },
        storage: undefined,
        cpuCount: 1,
        storePeriod: Period.WEEK,
        sendNotificationsOn: [],
        isRetryIfFailed: true,
        maxFailedTriesCount: 3,
      });
    }
    loadStorages();
  }, [database]);

  if (!backupConfig) return <div />;

  if (isStoragesLoading) {
    return (
      <div className="mb-5 flex items-center">
        <Spin />
      </div>
    );
  }

  const { backupInterval } = backupConfig;

  // UTC → local conversions for display
  const localTime: Dayjs | undefined = backupInterval?.timeOfDay
    ? dayjs.utc(backupInterval.timeOfDay, 'HH:mm').local()
    : undefined;

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

  // mandatory-field check
  const isAllFieldsFilled =
    !backupConfig.isBackupsEnabled ||
    (Boolean(backupConfig.storePeriod) &&
      Boolean(backupConfig.storage?.id) &&
      Boolean(backupConfig.cpuCount) &&
      Boolean(backupInterval?.interval) &&
      (!backupInterval ||
        ((backupInterval.interval !== IntervalType.WEEKLY || displayedWeekday) &&
          (backupInterval.interval !== IntervalType.MONTHLY || displayedDayOfMonth))));

  return (
    <div>
      <div className="mb-1 flex w-full items-center">
        <div className="min-w-[150px]">{t('backup:form.backups_enabled')}</div>
        <Switch
          checked={backupConfig.isBackupsEnabled}
          onChange={(checked) => {
            // If disabling backups on existing database, show confirmation
            if (!checked && database.id && backupConfig.isBackupsEnabled) {
              setIsShowBackupDisableConfirm(true);
            } else {
              updateBackupConfig({ isBackupsEnabled: checked });
            }
          }}
          size="small"
        />
      </div>

      {backupConfig.isBackupsEnabled && (
        <>
          <div className="mt-4 mb-1 flex w-full items-center">
            <div className="min-w-[150px]">{t('backup:form.backup_interval')}</div>
            <Select
              value={backupInterval?.interval}
              onChange={(v) => saveInterval({ interval: v })}
              size="small"
              className="max-w-[200px] grow"
              options={[
                { label: t('backup:interval.hourly'), value: IntervalType.HOURLY },
                { label: t('backup:interval.daily'), value: IntervalType.DAILY },
                { label: t('backup:interval.weekly'), value: IntervalType.WEEKLY },
                { label: t('backup:interval.monthly'), value: IntervalType.MONTHLY },
              ]}
            />
          </div>

          {backupInterval?.interval === IntervalType.WEEKLY && (
            <div className="mb-1 flex w-full items-center">
              <div className="min-w-[150px]">{t('backup:form.backup_weekday')}</div>
              <Select
                value={displayedWeekday}
                onChange={(localWeekday) => {
                  if (!localWeekday) return;
                  const ref = localTime ?? dayjs();
                  saveInterval({ weekday: getUtcWeekday(localWeekday, ref) });
                }}
                size="small"
                className="max-w-[200px] grow"
                options={weekdayOptions}
              />
            </div>
          )}

          {backupInterval?.interval === IntervalType.MONTHLY && (
            <div className="mb-1 flex w-full items-center">
              <div className="min-w-[150px]">{t('backup:form.backup_day_of_month')}</div>
              <InputNumber
                min={1}
                max={31}
                value={displayedDayOfMonth}
                onChange={(localDom) => {
                  if (!localDom) return;
                  const ref = localTime ?? dayjs();
                  saveInterval({ dayOfMonth: getUtcDayOfMonth(localDom, ref) });
                }}
                size="small"
                className="max-w-[200px] grow"
              />
            </div>
          )}

          {backupInterval?.interval !== IntervalType.HOURLY && (
            <div className="mb-1 flex w-full items-center">
              <div className="min-w-[150px]">{t('backup:form.backup_time_of_day')}</div>
              <TimePicker
                value={localTime}
                format={timeFormat.format}
                use12Hours={timeFormat.use12Hours}
                allowClear={false}
                size="small"
                className="max-w-[200px] grow"
                onChange={(t) => {
                  if (!t) return;
                  const patch: Partial<Interval> = { timeOfDay: t.utc().format('HH:mm') };

                  if (backupInterval?.interval === IntervalType.WEEKLY && displayedWeekday) {
                    patch.weekday = getUtcWeekday(displayedWeekday, t);
                  }
                  if (backupInterval?.interval === IntervalType.MONTHLY && displayedDayOfMonth) {
                    patch.dayOfMonth = getUtcDayOfMonth(displayedDayOfMonth, t);
                  }

                  saveInterval(patch);
                }}
              />
            </div>
          )}

          <div className="mt-4 mb-1 flex w-full items-center">
            <div className="min-w-[150px]">{t('backup:form.retry_if_failed')}</div>
            <Switch
              size="small"
              checked={backupConfig.isRetryIfFailed}
              onChange={(checked) => updateBackupConfig({ isRetryIfFailed: checked })}
            />

            <Tooltip
              className="cursor-pointer"
              title={t('backup:form.retry_hint')}
            >
              <InfoCircleOutlined className="ml-2" style={{ color: 'gray' }} />
            </Tooltip>
          </div>

          {backupConfig.isRetryIfFailed && (
            <div className="mb-1 flex w-full items-center">
              <div className="min-w-[150px]">{t('backup:form.max_failed_tries')}</div>
              <InputNumber
                min={1}
                max={10}
                value={backupConfig.maxFailedTriesCount}
                onChange={(value) => updateBackupConfig({ maxFailedTriesCount: value || 1 })}
                size="small"
                className="max-w-[200px] grow"
              />

              <Tooltip
                className="cursor-pointer"
                title={t('backup:form.max_failed_tries_hint')}
              >
                <InfoCircleOutlined className="ml-2" style={{ color: 'gray' }} />
              </Tooltip>
            </div>
          )}

          <div className="mt-5 mb-1 flex w-full items-center">
            <div className="min-w-[150px]">{t('backup:form.cpu_count')}</div>
            <InputNumber
              min={1}
              max={16}
              value={backupConfig.cpuCount}
              onChange={(value) => updateBackupConfig({ cpuCount: value || 1 })}
              size="small"
              className="max-w-[200px] grow"
            />

            <Tooltip
              className="cursor-pointer"
              title={t('backup:form.cpu_count_hint')}
            >
              <InfoCircleOutlined className="ml-2" style={{ color: 'gray' }} />
            </Tooltip>
          </div>

          <div className="mb-1 flex w-full items-center">
            <div className="min-w-[150px]">{t('backup:form.store_period')}</div>
            <Select
              value={backupConfig.storePeriod}
              onChange={(v) => updateBackupConfig({ storePeriod: v })}
              size="small"
              className="max-w-[200px] grow"
              options={[
                { label: t('backup:period.day'), value: Period.DAY },
                { label: t('backup:period.week'), value: Period.WEEK },
                { label: t('backup:period.month'), value: Period.MONTH },
                { label: t('backup:period.three_months'), value: Period.THREE_MONTH },
                { label: t('backup:period.six_months'), value: Period.SIX_MONTH },
                { label: t('backup:period.year'), value: Period.YEAR },
                { label: t('backup:period.two_years'), value: Period.TWO_YEARS },
                { label: t('backup:period.three_years'), value: Period.THREE_YEARS },
                { label: t('backup:period.four_years'), value: Period.FOUR_YEARS },
                { label: t('backup:period.five_years'), value: Period.FIVE_YEARS },
                { label: t('backup:period.forever'), value: Period.FOREVER },
              ]}
            />

            <Tooltip
              className="cursor-pointer"
              title={t('backup:form.store_period_hint')}
            >
              <InfoCircleOutlined className="ml-2" style={{ color: 'gray' }} />
            </Tooltip>
          </div>

          <div className="mt-5 mb-1 flex w-full items-center">
            <div className="min-w-[150px]">{t('backup:form.storage')}</div>
            <Select
              value={backupConfig.storage?.id}
              onChange={(storageId) => {
                if (storageId.includes('create-new-storage')) {
                  setShowCreateStorage(true);
                  return;
                }

                const selectedStorage = storages.find((s) => s.id === storageId);
                updateBackupConfig({ storage: selectedStorage });

                if (backupConfig.storage?.id) {
                  setIsShowWarn(true);
                }
              }}
              size="small"
              className="mr-2 max-w-[200px] grow"
              options={[
                ...storages.map((s) => ({ label: s.name, value: s.id })),
                { label: t('backup:form.create_new_storage'), value: 'create-new-storage' },
              ]}
              placeholder={t('backup:form.storage_placeholder')}
            />

            {backupConfig.storage?.type && (
              <img
                src={getStorageLogoFromType(backupConfig.storage.type)}
                alt="storageIcon"
                className="ml-1 h-4 w-4"
              />
            )}
          </div>

          <div className="mt-4 mb-1 flex w-full items-start">
            <div className="mt-1 min-w-[150px]">{t('backup:form.notifications')}</div>
            <div className="flex flex-col space-y-2">
              <Checkbox
                checked={backupConfig.sendNotificationsOn.includes(
                  BackupNotificationType.BackupSuccess,
                )}
                onChange={(e) => {
                  const notifications = [...backupConfig.sendNotificationsOn];
                  const index = notifications.indexOf(BackupNotificationType.BackupSuccess);
                  if (e.target.checked && index === -1) {
                    notifications.push(BackupNotificationType.BackupSuccess);
                  } else if (!e.target.checked && index > -1) {
                    notifications.splice(index, 1);
                  }
                  updateBackupConfig({ sendNotificationsOn: notifications });
                }}
              >
                {t('backup:notification_type.backup_success')}
              </Checkbox>

              <Checkbox
                checked={backupConfig.sendNotificationsOn.includes(
                  BackupNotificationType.BackupFailed,
                )}
                onChange={(e) => {
                  const notifications = [...backupConfig.sendNotificationsOn];
                  const index = notifications.indexOf(BackupNotificationType.BackupFailed);
                  if (e.target.checked && index === -1) {
                    notifications.push(BackupNotificationType.BackupFailed);
                  } else if (!e.target.checked && index > -1) {
                    notifications.splice(index, 1);
                  }
                  updateBackupConfig({ sendNotificationsOn: notifications });
                }}
              >
                {t('backup:notification_type.backup_failed')}
              </Checkbox>
            </div>
          </div>
        </>
      )}

      <div className="mt-5 flex">
        {isShowBackButton && (
          <Button className="mr-1" onClick={onBack}>
            {t('common:button.back')}
          </Button>
        )}

        {isShowCancelButton && (
          <Button danger ghost className="mr-1" onClick={onCancel}>
            {t('common:button.cancel')}
          </Button>
        )}

        <Button
          type="primary"
          className={`${isShowCancelButton ? 'ml-1' : 'ml-auto'} mr-5`}
          onClick={saveBackupConfig}
          loading={isSaving}
          disabled={!isUnsaved || !isAllFieldsFilled}
        >
          {saveButtonText || t('common:button.save')}
        </Button>
      </div>

      {isShowCreateStorage && (
        <Modal
          title={t('storage:modal.add_title')}
          footer={<div />}
          open={isShowCreateStorage}
          onCancel={() => setShowCreateStorage(false)}
        >
          <div className="my-3 max-w-[275px] text-gray-500">
            {t('storage:modal.description')}
          </div>

          <EditStorageComponent
            isShowName
            isShowClose={false}
            onClose={() => setShowCreateStorage(false)}
            onChanged={() => {
              loadStorages();
              setShowCreateStorage(false);
            }}
          />
        </Modal>
      )}

      {isShowWarn && (
        <ConfirmationComponent
          onConfirm={() => {
            setIsShowWarn(false);
          }}
          onDecline={() => {
            setIsShowWarn(false);
          }}
          description={t('backup:message.storage_change_warning')}
          actionButtonColor="red"
          actionText={t('backup:message.understand')}
          cancelText={t('common:button.cancel')}
          hideCancelButton
        />
      )}

      {isShowBackupDisableConfirm && (
        <ConfirmationComponent
          onConfirm={() => {
            updateBackupConfig({ isBackupsEnabled: false });
            setIsShowBackupDisableConfirm(false);
          }}
          onDecline={() => {
            setIsShowBackupDisableConfirm(false);
          }}
          description={t('backup:message.disable_confirm')}
          actionButtonColor="red"
          actionText={t('backup:message.disable_action')}
          cancelText={t('common:button.cancel')}
        />
      )}
    </div>
  );
};
