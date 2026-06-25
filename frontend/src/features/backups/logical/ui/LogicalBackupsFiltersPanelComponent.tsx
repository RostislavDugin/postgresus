import { DatePicker, Select } from 'antd';
import type { Dayjs } from 'dayjs';
import dayjs from 'dayjs';

import { LogicalBackupStatus } from '../../../../entity/backups/logical';
import type { BackupsFilters } from '../../../../entity/backups/logical';

interface Props {
  filters: BackupsFilters;
  onFiltersChange: (filters: BackupsFilters) => void;
}

const statusOptions = [
  { label: 'In progress', value: LogicalBackupStatus.IN_PROGRESS },
  { label: 'Successful', value: LogicalBackupStatus.COMPLETED },
  { label: 'Failed', value: LogicalBackupStatus.FAILED },
  { label: 'Canceled', value: LogicalBackupStatus.CANCELED },
];

export const LogicalBackupsFiltersPanelComponent = ({ filters, onFiltersChange }: Props) => {
  const handleStatusChange = (statuses: string[]) => {
    onFiltersChange({ ...filters, statuses: statuses.length > 0 ? statuses : undefined });
  };

  const handleBeforeDateChange = (date: Dayjs | null) => {
    onFiltersChange({
      ...filters,
      beforeDate: date ? date.toISOString() : undefined,
    });
  };

  return (
    <div className="flex flex-col gap-3">
      <div className="flex items-center gap-2">
        <span className="min-w-[90px] text-sm text-gray-500 dark:text-gray-400">Status</span>
        <Select
          mode="multiple"
          value={filters.statuses ?? []}
          onChange={handleStatusChange}
          options={statusOptions}
          placeholder="All statuses"
          size="small"
          variant="filled"
          className="w-[200px] [&_.ant-select-selector]:!rounded-md"
          allowClear
        />
      </div>

      <div className="flex items-center gap-2">
        <span className="min-w-[90px] text-sm text-gray-500 dark:text-gray-400">Before</span>
        <DatePicker
          value={filters.beforeDate ? dayjs(filters.beforeDate) : null}
          onChange={handleBeforeDateChange}
          size="small"
          variant="filled"
          className="w-[200px] !rounded-md"
          allowClear
        />
      </div>
    </div>
  );
};
