import { DownOutlined, InfoCircleOutlined, UpOutlined } from '@ant-design/icons';
import {
  Button,
  Checkbox,
  Input,
  InputNumber,
  Modal,
  Select,
  Spin,
  Switch,
  TimePicker,
  Tooltip,
} from 'antd';
import { CronExpressionParser } from 'cron-parser';
import dayjs, { Dayjs } from 'dayjs';
import { useEffect, useMemo, useState } from 'react';

import { IS_CLOUD } from '../../../constants';
import {
  type BackupConfig,
  BackupEncryption,
  type DatabasePlan,
  RetentionPolicyType,
  backupConfigApi,
} from '../../../entity/backups';
import { BackupNotificationType } from '../../../entity/backups/model/BackupNotificationType';
import type { Database } from '../../../entity/databases';
import { Period } from '../../../entity/databases/model/Period';
import { type Interval, IntervalType } from '../../../entity/intervals';
import { type Storage, getStorageLogoFromType, storageApi } from '../../../entity/storages';
import type { UserProfile } from '../../../entity/users';
import { getUserTimeFormat } from '../../../shared/time';
import {
  getUserTimeFormat as getIs12Hour,
  getLocalDayOfMonth,
  getLocalWeekday,
  getUtcDayOfMonth,
  getUtcWeekday,
} from '../../../shared/time/utils';
import { ConfirmationComponent } from '../../../shared/ui';
import { EditStorageComponent } from '../../storages/ui/edit/EditStorageComponent';

interface Props {
  user: UserProfile;
  database: Database;
  initialConfig?: BackupConfig;

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

const retentionPolicyOptions = [
  {
    label: 'GFS (keep last N hourly, daily, weekly, monthly and yearly backups)',
    value: RetentionPolicyType.GFS,
  },
  { label: 'Time period (last N days)', value: RetentionPolicyType.TimePeriod },
  { label: 'Count (N last backups)', value: RetentionPolicyType.Count },
];

export const EditBackupConfigComponent = ({
  user,
  database,
  initialConfig,

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
  const [isShowCreateStorage, setShowCreateStorage] = useState(false);
  const [storageSelectKey, setStorageSelectKey] = useState(0);

  const [isShowWarn, setIsShowWarn] = useState(false);

  const [databasePlan, setDatabasePlan] = useState<DatabasePlan>();
  const [isLoading, setIsLoading] = useState(true);

  const hasAdvancedValues =
    !!backupConfig?.isRetryIfFailed ||
    (backupConfig?.maxBackupSizeMb ?? 0) > 0 ||
    (backupConfig?.maxBackupsTotalSizeMb ?? 0) > 0;
  const [isShowAdvanced, setShowAdvanced] = useState(hasAdvancedValues);
  const [isShowGfsHint, setShowGfsHint] = useState(false);

  const timeFormat = useMemo(() => {
    const is12 = getIs12Hour();
    return { use12Hours: is12, format: is12 ? 'h:mm A' : 'HH:mm' };
  }, []);

  const dateTimeFormat = useMemo(() => getUserTimeFormat(), []);

  const createDefaultPlan = (databaseId: string, isCloud: boolean): DatabasePlan => {
    if (isCloud) {
      return {
        databaseId,
        maxBackupSizeMb: 100,
        maxBackupsTotalSizeMb: 4000,
        maxStoragePeriod: Period.WEEK,
      };
    } else {
      return {
        databaseId,
        maxBackupSizeMb: 0,
        maxBackupsTotalSizeMb: 0,
        maxStoragePeriod: Period.FOREVER,
      };
    }
  };

  const isPeriodAllowed = (period: Period, maxPeriod: Period): boolean => {
    const periodOrder = [
      Period.DAY,
      Period.WEEK,
      Period.MONTH,
      Period.THREE_MONTH,
      Period.SIX_MONTH,
      Period.YEAR,
      Period.TWO_YEARS,
      Period.THREE_YEARS,
      Period.FOUR_YEARS,
      Period.FIVE_YEARS,
      Period.FOREVER,
    ];
    const periodIndex = periodOrder.indexOf(period);
    const maxIndex = periodOrder.indexOf(maxPeriod);
    return periodIndex <= maxIndex;
  };

  const availablePeriods = useMemo(() => {
    const allPeriods = [
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
    ];

    if (!databasePlan) {
      return allPeriods;
    }

    return allPeriods.filter((p) => isPeriodAllowed(p.value, databasePlan.maxStoragePeriod));
  }, [databasePlan]);

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
    try {
      const storages = await storageApi.getStorages(database.workspaceId);
      setStorages(storages);

      if (IS_CLOUD) {
        const systemStorages = storages.filter((s) => s.isSystem);
        if (systemStorages.length > 0) {
          updateBackupConfig({ storage: systemStorages[0] });
        }
      }
    } catch (e) {
      alert((e as Error).message);
    }
  };

  useEffect(() => {
    const run = async () => {
      setIsLoading(true);

      try {
        if (database.id) {
          const config =
            initialConfig !== undefined
              ? initialConfig
              : await backupConfigApi.getBackupConfigByDbID(database.id);
          setBackupConfig(config);
          setIsUnsaved(false);
          setIsSaving(false);

          const plan = await backupConfigApi.getDatabasePlan(database.id);
          setDatabasePlan(plan);
        } else {
          const plan = createDefaultPlan('', IS_CLOUD);
          setDatabasePlan(plan);

          setBackupConfig({
            databaseId: database.id,
            name: 'Default',
            isBackupsEnabled: true,
            backupInterval: {
              id: undefined as unknown as string,
              interval: IntervalType.DAILY,
              timeOfDay: '00:00',
            },
            storage: undefined,
            retentionPolicyType: RetentionPolicyType.GFS,
            retentionTimePeriod:
              plan.maxStoragePeriod === Period.FOREVER ? Period.THREE_MONTH : plan.maxStoragePeriod,
            retentionCount: 100,
            retentionGfsHours: 24,
            retentionGfsDays: 7,
            retentionGfsWeeks: 4,
            retentionGfsMonths: 12,
            retentionGfsYears: 3,
            sendNotificationsOn: [BackupNotificationType.BackupFailed],
            isRetryIfFailed: true,
            maxFailedTriesCount: 3,
            encryption: BackupEncryption.ENCRYPTED,
            maxBackupSizeMb: plan.maxBackupSizeMb,
            maxBackupsTotalSizeMb: plan.maxBackupsTotalSizeMb,
          });
        }

        await loadStorages();
      } catch (e) {
        alert((e as Error).message);
      } finally {
        setIsLoading(false);
      }
    };

    run();
  }, [database]);

  if (isLoading) {
    return (
      <div className="mb-5 flex items-center">
        <Spin />
      </div>
    );
  }

  if (!backupConfig) return <div />;

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

  const retentionPolicyType = backupConfig.retentionPolicyType ?? RetentionPolicyType.TimePeriod;

  const isRetentionValid = (() => {
    switch (retentionPolicyType) {
      case RetentionPolicyType.TimePeriod:
        return Boolean(backupConfig.retentionTimePeriod);
      case RetentionPolicyType.Count:
        return (backupConfig.retentionCount ?? 0) > 0;
      case RetentionPolicyType.GFS:
        return (
          (backupConfig.retentionGfsDays ?? 0) > 0 ||
          (backupConfig.retentionGfsWeeks ?? 0) > 0 ||
          (backupConfig.retentionGfsMonths ?? 0) > 0 ||
          (backupConfig.retentionGfsYears ?? 0) > 0
        );
    }
  })();

  const isAllFieldsFilled =
    Boolean(backupConfig.name) &&
    (!backupConfig.isBackupsEnabled ||
    (isRetentionValid &&
      Boolean(backupConfig.storage?.id) &&
      Boolean(backupConfig.encryption) &&
      Boolean(backupInterval?.interval) &&
      (!backupInterval ||
        ((backupInterval.interval !== IntervalType.WEEKLY || displayedWeekday) &&
          (backupInterval.interval !== IntervalType.MONTHLY || displayedDayOfMonth) &&
          (backupInterval.interval !== IntervalType.CRON || backupInterval.cronExpression)))));

  return (
    <div>
      {database.id && (
        <div className="mb-3 flex w-full flex-col items-start sm:flex-row sm:items-center">
          <div className="mb-1 min-w-[150px] sm:mb-0">Schedule name</div>
          <Input
            value={backupConfig.name}
            onChange={(e) => updateBackupConfig({ name: e.target.value })}
            size="small"
            className="w-full max-w-[200px] grow"
            placeholder="e.g. Daily backup"
          />
        </div>
      )}

      {database.id && (
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
      )}

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
                { label: 'Cron', value: IntervalType.CRON },
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

          {backupInterval?.interval === IntervalType.CRON && (
            <>
              <div className="mb-1 flex w-full flex-col items-start sm:flex-row sm:items-center">
                <div className="mb-1 min-w-[150px] sm:mb-0">Cron expression (UTC)</div>
                <div className="flex items-center">
                  <Input
                    value={backupInterval?.cronExpression || ''}
                    onChange={(e) => saveInterval({ cronExpression: e.target.value })}
                    placeholder="0 2 * * *"
                    size="small"
                    className="w-full max-w-[200px] grow"
                  />
                  <Tooltip
                    className="cursor-pointer"
                    title={
                      <div>
                        <div className="font-bold">
                          Cron format: minute hour day month weekday (UTC)
                        </div>
                        <div className="mt-1">Examples:</div>
                        <div>• 0 2 * * * - Daily at 2:00 AM UTC</div>
                        <div>• 0 */6 * * * - Every 6 hours</div>
                        <div>• 0 3 * * 1 - Every Monday at 3:00 AM UTC</div>
                        <div>• 30 4 1,15 * * - 1st and 15th at 4:30 AM UTC</div>
                      </div>
                    }
                  >
                    <InfoCircleOutlined className="ml-2" style={{ color: 'gray' }} />
                  </Tooltip>
                </div>
              </div>
              {backupInterval?.cronExpression &&
                (() => {
                  try {
                    const interval = CronExpressionParser.parse(backupInterval.cronExpression, {
                      tz: 'UTC',
                    });
                    const nextRun = interval.next().toDate();
                    return (
                      <div className="mb-1 flex w-full flex-col items-start text-xs text-gray-600 sm:flex-row sm:items-center dark:text-gray-400">
                        <div className="mb-1 min-w-[150px] sm:mb-0" />
                        <div className="text-gray-600 dark:text-gray-400">
                          Next run {dayjs(nextRun).local().format(dateTimeFormat.format)}
                          <br />({dayjs(nextRun).fromNow()})
                        </div>
                      </div>
                    );
                  } catch {
                    return (
                      <div className="mb-1 flex w-full flex-col items-start text-red-500 sm:flex-row sm:items-center">
                        <div className="mb-1 min-w-[150px] sm:mb-0" />
                        <div className="text-red-500">Invalid cron expression</div>
                      </div>
                    );
                  }
                })()}
            </>
          )}

          {backupInterval?.interval !== IntervalType.HOURLY &&
            backupInterval?.interval !== IntervalType.CRON && (
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

          <div className="mb-3" />
        </>
      )}

      <div className="mt-5 mb-1 flex w-full flex-col items-start sm:flex-row sm:items-center">
        <div className="mb-1 min-w-[150px] sm:mb-0">Storage</div>
        <div className="flex w-full items-center">
          <Select
            key={storageSelectKey}
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

      {!IS_CLOUD && (
        <div className="mb-1 flex w-full flex-col items-start sm:flex-row sm:items-center">
          <div className="mb-1 min-w-[150px] sm:mb-0">Encryption</div>
          <div className="flex items-center">
            <Select
              value={backupConfig.encryption}
              onChange={(v) => updateBackupConfig({ encryption: v })}
              size="small"
              className="w-[200px]"
              options={[
                { label: 'None', value: BackupEncryption.NONE },
                { label: 'Encrypt backup files', value: BackupEncryption.ENCRYPTED },
              ]}
            />

            <Tooltip
              className="cursor-pointer"
              title="If backup is encrypted, backup files in your storage (S3, local, etc.) cannot be used directly. You can restore backups through Databasus or download them unencrypted via the 'Download' button."
            >
              <InfoCircleOutlined className="ml-2" style={{ color: 'gray' }} />
            </Tooltip>
          </div>
        </div>
      )}

      <div className="mt-5 mb-1 flex w-full flex-col items-start sm:flex-row sm:items-start">
        <div className="mt-1 mb-1 min-w-[150px] sm:mb-0">Retention policy</div>
        <div className="flex flex-col gap-1">
          <Select
            value={retentionPolicyType}
            options={retentionPolicyOptions}
            size="small"
            className="w-[200px]"
            popupMatchSelectWidth={false}
            onChange={(v) => {
              const type = v as RetentionPolicyType;
              const updates: Partial<typeof backupConfig> = { retentionPolicyType: type };

              if (type === RetentionPolicyType.GFS) {
                updates.retentionGfsHours = 24;
                updates.retentionGfsDays = 7;
                updates.retentionGfsWeeks = 4;
                updates.retentionGfsMonths = 12;
                updates.retentionGfsYears = 3;
              } else if (type === RetentionPolicyType.Count) {
                updates.retentionCount = 100;
              }

              updateBackupConfig(updates);
            }}
          />

          {retentionPolicyType === RetentionPolicyType.TimePeriod && (
            <div className="flex items-center">
              <Select
                value={backupConfig.retentionTimePeriod}
                onChange={(v) => updateBackupConfig({ retentionTimePeriod: v })}
                size="small"
                className="w-[200px]"
                options={availablePeriods}
              />

              <Tooltip
                className="cursor-pointer"
                title="How long to keep the backups. Backups older than this period are automatically deleted."
              >
                <InfoCircleOutlined className="ml-2" style={{ color: 'gray' }} />
              </Tooltip>
            </div>
          )}

          {retentionPolicyType === RetentionPolicyType.Count && (
            <div className="flex items-center">
              <InputNumber
                min={1}
                value={backupConfig.retentionCount}
                onChange={(v) => updateBackupConfig({ retentionCount: v ?? 1 })}
                size="small"
                className="w-[80px]"
              />
              <span className="ml-2 text-sm text-gray-600 dark:text-gray-400">
                most recent backups
              </span>

              <Tooltip
                className="cursor-pointer"
                title="Keep only the specified number of most recent backups. Older backups beyond this count are automatically deleted."
              >
                <InfoCircleOutlined className="ml-2" style={{ color: 'gray' }} />
              </Tooltip>
            </div>
          )}

          {retentionPolicyType === RetentionPolicyType.GFS && (
            <>
              <div>
                <span
                  className="cursor-pointer text-xs text-blue-600 hover:text-blue-800"
                  onClick={() => setShowGfsHint(!isShowGfsHint)}
                >
                  {isShowGfsHint ? 'Hide' : 'What is GFS (Grandfather-Father-Son)?'}
                </span>

                {isShowGfsHint && (
                  <div className="mt-1 max-w-[280px] text-xs text-gray-600 dark:text-gray-400">
                    GFS (Grandfather-Father-Son) rotation: keep the last N hourly, daily, weekly,
                    monthly and yearly backups. This allows keeping backups over long periods of
                    time within a reasonable storage space.
                  </div>
                )}
              </div>

              <div className="flex flex-col gap-1">
                <div className="flex items-center gap-2">
                  <span className="w-[110px] text-sm text-gray-600 dark:text-gray-400">
                    Hourly backups
                  </span>
                  <InputNumber
                    min={0}
                    value={backupConfig.retentionGfsHours}
                    onChange={(v) => updateBackupConfig({ retentionGfsHours: v ?? 0 })}
                    size="small"
                    className="w-[80px]"
                  />
                </div>

                <div className="flex items-center gap-2">
                  <span className="w-[110px] text-sm text-gray-600 dark:text-gray-400">
                    Daily backups
                  </span>
                  <InputNumber
                    min={0}
                    value={backupConfig.retentionGfsDays}
                    onChange={(v) => updateBackupConfig({ retentionGfsDays: v ?? 0 })}
                    size="small"
                    className="w-[80px]"
                  />
                </div>

                <div className="flex items-center gap-2">
                  <span className="w-[110px] text-sm text-gray-600 dark:text-gray-400">
                    Weekly backups
                  </span>
                  <InputNumber
                    min={0}
                    value={backupConfig.retentionGfsWeeks}
                    onChange={(v) => updateBackupConfig({ retentionGfsWeeks: v ?? 0 })}
                    size="small"
                    className="w-[80px]"
                  />
                </div>

                <div className="flex items-center gap-2">
                  <span className="w-[110px] text-sm text-gray-600 dark:text-gray-400">
                    Monthly backups
                  </span>
                  <InputNumber
                    min={0}
                    value={backupConfig.retentionGfsMonths}
                    onChange={(v) => updateBackupConfig({ retentionGfsMonths: v ?? 0 })}
                    size="small"
                    className="w-[80px]"
                  />
                </div>
                <div className="flex items-center gap-2">
                  <span className="w-[110px] text-sm text-gray-600 dark:text-gray-400">
                    Yearly backups
                  </span>
                  <InputNumber
                    min={0}
                    value={backupConfig.retentionGfsYears}
                    onChange={(v) => updateBackupConfig({ retentionGfsYears: v ?? 0 })}
                    size="small"
                    className="w-[80px]"
                  />
                </div>
              </div>
            </>
          )}
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

      <div className="mt-4 mb-1 flex items-center">
        <div
          className="flex cursor-pointer items-center text-sm text-blue-600 hover:text-blue-800"
          onClick={() => setShowAdvanced(!isShowAdvanced)}
        >
          <span className="mr-2">Advanced settings</span>

          {isShowAdvanced ? (
            <UpOutlined style={{ fontSize: '12px' }} />
          ) : (
            <DownOutlined style={{ fontSize: '12px' }} />
          )}
        </div>
      </div>

      {isShowAdvanced && backupConfig.isBackupsEnabled && (
        <>
          <div className="mb-1 flex w-full flex-col items-start sm:flex-row sm:items-center">
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
                  className="w-full max-w-[75px] grow"
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
            <div className="mb-1 min-w-[150px] sm:mb-0">Max backup size limit</div>
            <div className="flex items-center">
              <Switch
                size="small"
                checked={backupConfig.maxBackupSizeMb > 0}
                disabled={IS_CLOUD}
                onChange={(checked) => {
                  updateBackupConfig({
                    maxBackupSizeMb: checked ? backupConfig.maxBackupSizeMb || 1000 : 0,
                  });
                }}
              />

              <Tooltip
                className="cursor-pointer"
                title="Limits the size of each individual backup. Note that backups are typically 15× smaller than the database size. For example, a 100 MB backup represents approximately 1.5 GB database."
              >
                <InfoCircleOutlined className="ml-2" style={{ color: 'gray' }} />
              </Tooltip>
            </div>
          </div>

          {backupConfig.maxBackupSizeMb > 0 && (
            <div className="mb-5 flex w-full flex-col items-start sm:flex-row sm:items-center">
              <div className="mb-1 min-w-[150px] sm:mb-0">Max file size (MB)</div>

              <InputNumber
                min={1}
                max={
                  databasePlan?.maxBackupSizeMb && databasePlan.maxBackupSizeMb > 0
                    ? databasePlan.maxBackupSizeMb
                    : undefined
                }
                value={backupConfig.maxBackupSizeMb}
                onChange={(value) => {
                  const newValue = value || 1;
                  if (databasePlan?.maxBackupSizeMb && databasePlan.maxBackupSizeMb > 0) {
                    updateBackupConfig({
                      maxBackupSizeMb: Math.min(newValue, databasePlan.maxBackupSizeMb),
                    });
                  } else {
                    updateBackupConfig({ maxBackupSizeMb: newValue });
                  }
                }}
                size="small"
                className="w-full max-w-[75px] grow"
              />

              <div className="ml-2 text-xs text-gray-600 dark:text-gray-400">
                ~{((backupConfig.maxBackupSizeMb / 1024) * 15).toFixed(2)} GB DB size
              </div>
            </div>
          )}

          <div className="mb-1 flex w-full flex-col items-start sm:flex-row sm:items-center">
            <div className="mb-1 min-w-[150px] sm:mb-0">Limit total backups size</div>
            <div className="flex items-center">
              <Switch
                size="small"
                checked={backupConfig.maxBackupsTotalSizeMb > 0}
                disabled={IS_CLOUD}
                onChange={(checked) => {
                  updateBackupConfig({
                    maxBackupsTotalSizeMb: checked
                      ? backupConfig.maxBackupsTotalSizeMb || 1_000_000
                      : 0,
                  });
                }}
              />

              <Tooltip
                className="cursor-pointer"
                title="Limits the total size of all backups in storage (like S3, local, etc.). Once this limit is exceeded, the oldest backups are automatically removed until the total size is within the limit again."
              >
                <InfoCircleOutlined className="ml-2" style={{ color: 'gray' }} />
              </Tooltip>
            </div>
          </div>

          {backupConfig.maxBackupsTotalSizeMb > 0 && (
            <div className="mb-1 flex w-full flex-col items-start sm:flex-row sm:items-center">
              <div className="mb-1 min-w-[150px] sm:mb-0">Backups files size (MB)</div>
              <InputNumber
                min={1}
                max={
                  databasePlan?.maxBackupsTotalSizeMb && databasePlan.maxBackupsTotalSizeMb > 0
                    ? databasePlan.maxBackupsTotalSizeMb
                    : undefined
                }
                value={backupConfig.maxBackupsTotalSizeMb}
                onChange={(value) => {
                  const newValue = value || 1;
                  if (
                    databasePlan?.maxBackupsTotalSizeMb &&
                    databasePlan.maxBackupsTotalSizeMb > 0
                  ) {
                    updateBackupConfig({
                      maxBackupsTotalSizeMb: Math.min(newValue, databasePlan.maxBackupsTotalSizeMb),
                    });
                  } else {
                    updateBackupConfig({ maxBackupsTotalSizeMb: newValue });
                  }
                }}
                size="small"
                className="w-full max-w-[75px] grow"
              />

              <div className="ml-2 text-xs text-gray-600 dark:text-gray-400">
                {(backupConfig.maxBackupsTotalSizeMb / 1024).toFixed(2)} GB (~
                {backupConfig.maxBackupsTotalSizeMb / backupConfig.maxBackupSizeMb} backups)
              </div>
            </div>
          )}
        </>
      )}

      <div className="mt-5 flex">
        {isShowBackButton && (
          <Button className="mr-1" type="primary" ghost onClick={onBack}>
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
          onCancel={() => {
            setShowCreateStorage(false);
            setStorageSelectKey((prev) => prev + 1);
          }}
          maskClosable={false}
        >
          <div className="my-3 max-w-[275px] text-gray-500 dark:text-gray-400">
            Storage - is a place where backups will be stored (local disk, S3, Google Drive, etc.)
          </div>

          <EditStorageComponent
            user={user}
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
