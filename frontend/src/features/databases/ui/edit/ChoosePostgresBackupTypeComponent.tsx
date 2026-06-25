import { Button } from 'antd';
import { useState } from 'react';

import { type Database, DatabaseType } from '../../../../entity/databases';

interface Props {
  database: Database;
  saveButtonText?: string;
  onBack: () => void;
  onSelected: (type: DatabaseType) => void;
}

const backupTypeOptions = [
  {
    type: DatabaseType.POSTGRES_LOGICAL,
    title: 'Logical',
    description: 'Recommended for databases under 50 GB. Simpler to set up.',
  },
  {
    type: DatabaseType.POSTGRES_PHYSICAL,
    title: 'Physical',
    description:
      'For databases over 50 GB. Enables point-in-time recovery and better RPO/RTO, but needs extra setup.',
  },
];

export const ChoosePostgresBackupTypeComponent = ({
  database,
  saveButtonText,
  onBack,
  onSelected,
}: Props) => {
  const [selectedType, setSelectedType] = useState<DatabaseType>(
    database.type === DatabaseType.POSTGRES_PHYSICAL
      ? DatabaseType.POSTGRES_PHYSICAL
      : DatabaseType.POSTGRES_LOGICAL,
  );

  return (
    <div>
      <div className="my-3 text-center text-lg">Choose backup type</div>

      <div className="grid grid-cols-2 gap-3">
        {backupTypeOptions.map((option) => {
          const isSelected = selectedType === option.type;

          return (
            <div
              key={option.type}
              onClick={() => setSelectedType(option.type)}
              className={`flex cursor-pointer flex-col gap-2 rounded-xl border p-3 transition hover:border-blue-400 ${
                isSelected
                  ? 'border-blue-500 bg-blue-50 dark:bg-blue-900/40'
                  : 'border-gray-200 bg-white dark:border-gray-700 dark:bg-gray-800'
              }`}
            >
              <div className="flex items-center gap-2">
                <span className="font-semibold">{option.title}</span>
              </div>

              <span className="text-sm leading-snug text-gray-500 dark:text-gray-400">
                {option.description}
              </span>
            </div>
          );
        })}
      </div>

      <div className="mt-5 flex">
        <Button className="mr-auto" type="primary" ghost onClick={onBack}>
          Back
        </Button>

        <Button type="primary" onClick={() => onSelected(selectedType)}>
          {saveButtonText || 'Continue'}
        </Button>
      </div>
    </div>
  );
};
