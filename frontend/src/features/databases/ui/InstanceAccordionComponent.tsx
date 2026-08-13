import { DownOutlined, RightOutlined } from '@ant-design/icons';

import { type DatabaseInstance, formatInstanceTitle } from '../../../entity/databases';
import { HealthStatus } from '../../../entity/databases/model/HealthStatus';
import { DatabaseCardComponent } from './DatabaseCardComponent';

interface Props {
  instance: DatabaseInstance;
  isExpanded: boolean;
  onToggle: (key: string) => void;
  selectedDatabaseId?: string;
  setSelectedDatabaseId: (databaseId: string) => void;
}

export const InstanceAccordionComponent = ({
  instance,
  isExpanded,
  onToggle,
  selectedDatabaseId,
  setSelectedDatabaseId,
}: Props) => {
  const hasProblem = instance.databases.some(
    (database) =>
      database.healthStatus === HealthStatus.UNAVAILABLE || !!database.lastBackupErrorMessage,
  );

  return (
    <div className="mb-2">
      <div
        className="flex cursor-pointer items-center rounded px-2 py-2 hover:bg-gray-100 dark:hover:bg-gray-700"
        onClick={() => onToggle(instance.key)}
        role="button"
        tabIndex={0}
        aria-expanded={isExpanded}
        onKeyDown={(e) => {
          if (e.key === 'Enter') {
            onToggle(instance.key);
          } else if (e.key === ' ') {
            e.preventDefault();
            onToggle(instance.key);
          }
        }}
      >
        {isExpanded ? (
          <DownOutlined style={{ fontSize: '10px' }} className="mr-2" />
        ) : (
          <RightOutlined style={{ fontSize: '10px' }} className="mr-2" />
        )}

        <div className="truncate text-sm font-bold" title={formatInstanceTitle(instance)}>
          {formatInstanceTitle(instance)}
        </div>

        {hasProblem && <div className="ml-2 h-2 w-2 shrink-0 rounded-full bg-red-500" />}

        <div className="ml-auto pl-2 text-xs text-gray-500 dark:text-gray-400">
          {instance.databases.length}
        </div>
      </div>

      {isExpanded && (
        <div className="mt-2 pl-3">
          {instance.databases.map((database) => (
            <DatabaseCardComponent
              key={database.id}
              database={database}
              selectedDatabaseId={selectedDatabaseId}
              setSelectedDatabaseId={setSelectedDatabaseId}
            />
          ))}
        </div>
      )}
    </div>
  );
};
