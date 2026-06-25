import { DatePicker, Select } from 'antd';
import type { Dayjs } from 'dayjs';
import dayjs from 'dayjs';

import {
  PhysicalBackupStatus,
  PhysicalBackupType,
  type PhysicalBackupsFilters,
} from '../../../../entity/backups/physical';
import { PHYSICAL_BACKUP_STATUS_LABELS } from '../model/physicalBackupStatus';

interface Props {
  filters: PhysicalBackupsFilters;
  onFiltersChange: (filters: PhysicalBackupsFilters) => void;
}

const typeOptions = [
  { label: 'Full', value: PhysicalBackupType.FULL },
  { label: 'Incremental', value: PhysicalBackupType.INCREMENTAL },
  { label: 'WAL', value: PhysicalBackupType.WAL },
];

const statusOptions = Object.values(PhysicalBackupStatus).map((status) => ({
  label: PHYSICAL_BACKUP_STATUS_LABELS[status],
  value: status,
}));

export const PhysicalBackupsFiltersPanelComponent = ({ filters, onFiltersChange }: Props) => {
  const handleTypeChange = (types: PhysicalBackupType[]) => {
    onFiltersChange({ ...filters, types: types.length > 0 ? types : undefined });
  };

  const handleStatusChange = (statuses: PhysicalBackupStatus[]) => {
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
        <span className="min-w-[90px] text-sm text-gray-500 dark:text-gray-400">Type</span>
        <Select
          mode="multiple"
          value={filters.types ?? []}
          onChange={handleTypeChange}
          options={typeOptions}
          placeholder="All types"
          size="small"
          variant="filled"
          className="w-[200px] [&_.ant-select-selector]:!rounded-md"
          allowClear
        />
      </div>

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
