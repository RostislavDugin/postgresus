import { InfoCircleOutlined } from '@ant-design/icons';
import { Input, InputNumber, Select, TimePicker, Tooltip } from 'antd';
import { CronExpressionParser } from 'cron-parser';
import dayjs, { Dayjs } from 'dayjs';
import { type JSX, useMemo } from 'react';

import { type Interval, IntervalType } from '../../../../entity/intervals';
import { getUserTimeFormat } from '../../../../shared/time';
import {
  getUserTimeFormat as getIs12Hour,
  getLocalDayOfMonth,
  getLocalWeekday,
  getUtcDayOfMonth,
  getUtcWeekday,
} from '../../../../shared/time/utils';

interface Props {
  label: string;
  interval?: Interval;
  onChange: (patch: Partial<Interval>) => void;
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

// Reusable interval sub-form for a single backup cadence. All times are stored in
// UTC (timeOfDay 'HH:mm', weekday/dayOfMonth as UTC values) and displayed in the
// user's local timezone - the getUtc*/getLocal* helpers translate across the date
// boundary so e.g. "Sunday 23:00 local" can map to "Monday 04:00 UTC".
export const PhysicalIntervalEditor = ({ label, interval, onChange }: Props): JSX.Element => {
  const timeFormat = useMemo(() => {
    const is12 = getIs12Hour();
    return { use12Hours: is12, format: is12 ? 'h:mm A' : 'HH:mm' };
  }, []);

  const dateTimeFormat = useMemo(() => getUserTimeFormat(), []);

  // UTC -> local conversions for display
  const localTime: Dayjs | undefined = interval?.timeOfDay
    ? dayjs.utc(interval.timeOfDay, 'HH:mm').local()
    : undefined;

  const displayedWeekday: number | undefined =
    interval?.type === IntervalType.WEEKLY && interval.weekday && interval.timeOfDay
      ? getLocalWeekday(interval.weekday, interval.timeOfDay)
      : interval?.weekday;

  const displayedDayOfMonth: number | undefined =
    interval?.type === IntervalType.MONTHLY && interval.dayOfMonth && interval.timeOfDay
      ? getLocalDayOfMonth(interval.dayOfMonth, interval.timeOfDay)
      : interval?.dayOfMonth;

  return (
    <>
      <div className="mt-4 mb-1 flex w-full flex-col items-start sm:flex-row sm:items-center">
        <div className="mb-1 max-w-[150px] min-w-[150px] leading-4 sm:mb-0">{label}</div>
        <Select
          value={interval?.type}
          onChange={(v) => onChange({ type: v })}
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

      {interval?.type === IntervalType.WEEKLY && (
        <div className="mb-1 flex w-full flex-col items-start sm:flex-row sm:items-center">
          <div className="mb-1 min-w-[150px] sm:mb-0">Weekday</div>
          <Select
            value={displayedWeekday}
            onChange={(localWeekday) => {
              if (!localWeekday) return;
              const ref = localTime ?? dayjs();
              onChange({ weekday: getUtcWeekday(localWeekday, ref) });
            }}
            size="small"
            className="w-full max-w-[200px] grow"
            options={weekdayOptions}
          />
        </div>
      )}

      {interval?.type === IntervalType.MONTHLY && (
        <div className="mb-1 flex w-full flex-col items-start sm:flex-row sm:items-center">
          <div className="mb-1 min-w-[150px] sm:mb-0">Day of month</div>
          <InputNumber
            min={1}
            max={31}
            value={displayedDayOfMonth}
            onChange={(localDom) => {
              if (!localDom) return;
              const ref = localTime ?? dayjs();
              onChange({ dayOfMonth: getUtcDayOfMonth(localDom, ref) });
            }}
            size="small"
            className="w-full max-w-[200px] grow"
          />
        </div>
      )}

      {interval?.type === IntervalType.CRON && (
        <>
          <div className="mb-1 flex w-full flex-col items-start sm:flex-row sm:items-center">
            <div className="mb-1 min-w-[150px] sm:mb-0">Cron expression (UTC)</div>
            <div className="flex items-center">
              <Input
                value={interval?.cronExpression || ''}
                onChange={(e) => onChange({ cronExpression: e.target.value })}
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
                    <div>0 2 * * * - Daily at 2:00 AM UTC</div>
                    <div>0 */6 * * * - Every 6 hours</div>
                    <div>0 3 * * 1 - Every Monday at 3:00 AM UTC</div>
                    <div>30 4 1,15 * * - 1st and 15th at 4:30 AM UTC</div>
                  </div>
                }
              >
                <InfoCircleOutlined className="ml-2" style={{ color: 'gray' }} />
              </Tooltip>
            </div>
          </div>
          {interval?.cronExpression &&
            (() => {
              try {
                const parsed = CronExpressionParser.parse(interval.cronExpression, {
                  tz: 'UTC',
                });
                const nextRun = parsed.next().toDate();
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

      {interval?.type !== IntervalType.HOURLY && interval?.type !== IntervalType.CRON && (
        <div className="mb-1 flex w-full flex-col items-start sm:flex-row sm:items-center">
          <div className="mb-1 min-w-[150px] sm:mb-0">Time of day</div>
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

              if (interval?.type === IntervalType.WEEKLY && displayedWeekday) {
                patch.weekday = getUtcWeekday(displayedWeekday, t);
              }
              if (interval?.type === IntervalType.MONTHLY && displayedDayOfMonth) {
                patch.dayOfMonth = getUtcDayOfMonth(displayedDayOfMonth, t);
              }

              onChange(patch);
            }}
          />
        </div>
      )}
    </>
  );
};
