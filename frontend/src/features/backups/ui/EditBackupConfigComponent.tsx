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

import { type BackupConfig, BackupEncryption, backupConfigApi } from '../../../entity/backups';
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

const weekdayOptions = [
  { value: 1, label: 'Mon' },
  { value: 2, label: 'Tue' },
  { value: 3, label: 'Wed' },
  { value: 4, label: 'Thu' },
  { value: 5, label: 'Fri' },
  { value: 6, label: 'Sat' },
  { value: 7, label: 'Sun' },
];

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
  const [backupConfig, setBackupConfig] = useState<BackupConfig>();
  const [isUnsaved, setIsUnsaved] = useState(false);
  const [isSaving, setIsSaving] = useState(false);

  const [storages, setStorages] = useState<Storage[]>([]);
  const [isStoragesLoading, setIsStoragesLoading] = useState(false);
  const [isShowCreateStorage, setShowCreateStorage] = useState(false);

  const [isShowWarn, setIsShowWarn] = useState(false);

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
      const storages = await storageApi.getStorages(database.workspaceId);
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
        storePeriod: Period.THREE_MONTH,
        sendNotificationsOn: [],
        isRetryIfFailed: true,
        maxFailedTriesCount: 3,
        encryption: BackupEncryption.ENCRYPTED,
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
      Boolean(backupConfig.encryption) &&
      Boolean(backupInterval?.interval) &&
      (!backupInterval ||
        ((backupInterval.interval !== IntervalType.WEEKLY || displayedWeekday) &&
          (backupInterval.interval !== IntervalType.MONTHLY || displayedDayOfMonth))));

  return (
    <div>
      <div className="mb-1 flex w-full flex-col items-start sm:flex-row sm:items-center">
        <div className="mb-1 min-w-[150px] sm:mb-0">Backups enabled</div>
        <Switch
          checked={backupConfig.isBackupsEnabled}
          onChange={(checked) => {
            updateBackupConfig({ isBackupsEnabled: checked });
          }}
          size="small"
        />
      </div>

      {backupConfig.isBackupsEnabled && (
        <>
          <div className="mt-4 mb-1 flex w-full flex-col items-start sm:flex-row sm:items-center">
            <div className="mb-1 min-w-[150px] sm:mb-0">Backup interval</div>
            <Select
              value={backupInterval?.interval}
              onChange={(v) => saveInterval({ interval: v })}
              size="small"
              className="w-full max-w-[200px] grow"
              options={[
                { label: 'Hourly', value: IntervalType.HOURLY },
                { label: 'Daily', value: IntervalType.DAILY },
                { label: 'Weekly', value: IntervalType.WEEKLY },
                { label: 'Monthly', value: IntervalType.MONTHLY },
              ]}
            />
          </div>

          {backupInterval?.interval === IntervalType.WEEKLY && (
            <div className="mb-1 flex w-full flex-col items-start sm:flex-row sm:items-center">
              <div className="mb-1 min-w-[150px] sm:mb-0">Backup weekday</div>
              <Select
                value={displayedWeekday}
                onChange={(localWeekday) => {
                  if (!localWeekday) return;
                  const ref = localTime ?? dayjs();
                  saveInterval({ weekday: getUtcWeekday(localWeekday, ref) });
                }}
                size="small"
                className="w-full max-w-[200px] grow"
                options={weekdayOptions}
              />
            </div>
          )}

          {backupInterval?.interval === IntervalType.MONTHLY && (
            <div className="mb-1 flex w-full flex-col items-start sm:flex-row sm:items-center">
              <div className="mb-1 min-w-[150px] sm:mb-0">Backup day of month</div>
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
                className="w-full max-w-[200px] grow"
              />
            </div>
          )}

          {backupInterval?.interval !== IntervalType.HOURLY && (
            <div className="mb-1 flex w-full flex-col items-start sm:flex-row sm:items-center">
              <div className="mb-1 min-w-[150px] sm:mb-0">Backup time of day</div>
              <TimePicker
                value={localTime}
                format={timeFormat.format}
                use12Hours={timeFormat.use12Hours}
                allowClear={false}
                size="small"
                className="w-full max-w-[200px] grow"
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

          <div className="mt-4 mb-1 flex w-full flex-col items-start sm:flex-row sm:items-center">
            <div className="mb-1 min-w-[150px] sm:mb-0">Retry backup if failed</div>
            <div className="flex items-center">
              <Switch
                size="small"
                checked={backupConfig.isRetryIfFailed}
                onChange={(checked) => updateBackupConfig({ isRetryIfFailed: checked })}
              />

              <Tooltip
                className="cursor-pointer"
                title="Automatically retry failed backups. Backups can fail due to network failures, storage issues or temporary database unavailability."
              >
                <InfoCircleOutlined className="ml-2" style={{ color: 'gray' }} />
              </Tooltip>
            </div>
          </div>

          {backupConfig.isRetryIfFailed && (
            <div className="mb-1 flex w-full flex-col items-start sm:flex-row sm:items-center">
              <div className="mb-1 min-w-[150px] sm:mb-0">Max failed tries count</div>
              <div className="flex items-center">
                <InputNumber
                  min={1}
                  max={10}
                  value={backupConfig.maxFailedTriesCount}
                  onChange={(value) => updateBackupConfig({ maxFailedTriesCount: value || 1 })}
                  size="small"
                  className="w-full max-w-[200px] grow"
                />

                <Tooltip
                  className="cursor-pointer"
                  title="Maximum number of retry attempts for failed backups. You will receive a notification when all tries have failed."
                >
                  <InfoCircleOutlined className="ml-2" style={{ color: 'gray' }} />
                </Tooltip>
              </div>
            </div>
          )}

          <div className="mt-5 mb-1 flex w-full flex-col items-start sm:flex-row sm:items-center">
            <div className="mb-1 min-w-[150px] sm:mb-0">CPU count</div>
            <div className="flex items-center">
              <InputNumber
                min={1}
                max={16}
                value={backupConfig.cpuCount}
                onChange={(value) => updateBackupConfig({ cpuCount: value || 1 })}
                size="small"
                className="w-full max-w-[200px] grow"
              />

              <Tooltip
                className="cursor-pointer"
                title="Number of CPU cores to use for restore processing. Higher values may speed up restores, but use more resources."
              >
                <InfoCircleOutlined className="ml-2" style={{ color: 'gray' }} />
              </Tooltip>
            </div>
          </div>

          <div className="mb-1 flex w-full flex-col items-start sm:flex-row sm:items-center">
            <div className="mb-1 min-w-[150px] sm:mb-0">Store period</div>
            <div className="flex items-center">
              <Select
                value={backupConfig.storePeriod}
                onChange={(v) => updateBackupConfig({ storePeriod: v })}
                size="small"
                className="w-full max-w-[200px] grow"
                options={[
                  { label: '1 day', value: Period.DAY },
                  { label: '1 week', value: Period.WEEK },
                  { label: '1 month', value: Period.MONTH },
                  { label: '3 months', value: Period.THREE_MONTH },
                  { label: '6 months', value: Period.SIX_MONTH },
                  { label: '1 year', value: Period.YEAR },
                  { label: '2 years', value: Period.TWO_YEARS },
                  { label: '3 years', value: Period.THREE_YEARS },
                  { label: '4 years', value: Period.FOUR_YEARS },
                  { label: '5 years', value: Period.FIVE_YEARS },
                  { label: 'Forever', value: Period.FOREVER },
                ]}
              />

              <Tooltip
                className="cursor-pointer"
                title="How long to keep the backups? Make sure you have enough storage space."
              >
                <InfoCircleOutlined className="ml-2" style={{ color: 'gray' }} />
              </Tooltip>
            </div>
          </div>

          <div className="mb-3" />
        </>
      )}

      <div className="mt-2 mb-1 flex w-full flex-col items-start sm:flex-row sm:items-center">
        <div className="mb-1 min-w-[150px] sm:mb-0">Storage</div>
        <div className="flex w-full items-center">
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
              { label: 'Create new storage', value: 'create-new-storage' },
            ]}
            placeholder="Select storage"
          />

          {backupConfig.storage?.type && (
            <img
              src={getStorageLogoFromType(backupConfig.storage.type)}
              alt="storageIcon"
              className="ml-1 h-4 w-4"
            />
          )}
        </div>
      </div>

      <div className="mb-1 flex w-full flex-col items-start sm:flex-row sm:items-center">
        <div className="mb-1 min-w-[150px] sm:mb-0">Encryption</div>
        <div className="flex items-center">
          <Select
            value={backupConfig.encryption}
            onChange={(v) => updateBackupConfig({ encryption: v })}
            size="small"
            className="w-full max-w-[200px] grow"
            options={[
              { label: 'None', value: BackupEncryption.NONE },
              { label: 'Encrypt backup files', value: BackupEncryption.ENCRYPTED },
            ]}
          />

          <Tooltip
            className="cursor-pointer"
            title="If backup is encrypted, backup files in your storage (S3, local, etc.) cannot be used directly. You can restore backups through Postgresus or download them unencrypted via the 'Download' button."
          >
            <InfoCircleOutlined className="ml-2" style={{ color: 'gray' }} />
          </Tooltip>
        </div>
      </div>

      {backupConfig.isBackupsEnabled && (
        <>
          <div className="mt-4 mb-1 flex w-full flex-col items-start sm:flex-row sm:items-start">
            <div className="mt-0 mb-1 min-w-[150px] sm:mt-1 sm:mb-0">Notifications</div>
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
                Backup success
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
                Backup failed
              </Checkbox>
            </div>
          </div>
        </>
      )}

      <div className="mt-5 flex">
        {isShowBackButton && (
          <Button className="mr-1" onClick={onBack}>
            Back
          </Button>
        )}

        {isShowCancelButton && (
          <Button danger ghost className="mr-1" onClick={onCancel}>
            Cancel
          </Button>
        )}

        <Button
          type="primary"
          className={`${isShowCancelButton ? 'ml-1' : 'ml-auto'} mr-5`}
          onClick={saveBackupConfig}
          loading={isSaving}
          disabled={!isUnsaved || !isAllFieldsFilled}
        >
          {saveButtonText || 'Save'}
        </Button>
      </div>

      {isShowCreateStorage && (
        <Modal
          title="Add storage"
          footer={<div />}
          open={isShowCreateStorage}
          onCancel={() => setShowCreateStorage(false)}
        >
          <div className="my-3 max-w-[275px] text-gray-500">
            Storage - is a place where backups will be stored (local disk, S3, Google Drive, etc.)
          </div>

          <EditStorageComponent
            workspaceId={database.workspaceId}
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
          description="If you change the storage, all backups in this storage will be deleted."
          actionButtonColor="red"
          actionText="I understand"
          cancelText="Cancel"
          hideCancelButton
        />
      )}
    </div>
  );
};
