import { Button, Checkbox, Input, Spin } from 'antd';
import { useEffect, useState } from 'react';

import { type Database, databaseApi } from '../../../../entity/databases';

interface Props {
  database: Database;
  alreadyAddedNames: string[];
  onBack: () => void;
  onSelected: (databaseNames: string[]) => void;
}

export const SelectServerDatabasesComponent = ({
  database,
  alreadyAddedNames,
  onBack,
  onSelected,
}: Props) => {
  const [isLoading, setIsLoading] = useState(true);
  const [loadError, setLoadError] = useState<string | undefined>();
  const [serverDatabases, setServerDatabases] = useState<string[]>([]);
  const [selectedNames, setSelectedNames] = useState<string[]>([]);
  const [searchQuery, setSearchQuery] = useState('');

  const alreadyAdded = new Set(alreadyAddedNames);

  const loadServerDatabases = () => {
    setIsLoading(true);
    setLoadError(undefined);

    databaseApi
      .listServerDatabases(database)
      .then((response) => setServerDatabases(response.databases))
      .catch((e) => setLoadError((e as Error).message))
      .finally(() => setIsLoading(false));
  };

  useEffect(() => {
    loadServerDatabases();
  }, []);

  if (isLoading) {
    return (
      <div className="flex items-center">
        <Spin />
        <span className="ml-3">Loading databases from the server...</span>
      </div>
    );
  }

  if (loadError) {
    return (
      <div>
        <div className="mb-3 text-red-600 dark:text-red-400">{loadError}</div>

        <div className="flex">
          <Button type="primary" ghost onClick={() => onBack()}>
            Back
          </Button>

          <Button className="ml-auto" type="primary" onClick={loadServerDatabases}>
            Retry
          </Button>
        </div>
      </div>
    );
  }

  const visibleNames = serverDatabases.filter((name) =>
    name.toLowerCase().includes(searchQuery.trim().toLowerCase()),
  );

  const toggleName = (name: string) => {
    setSelectedNames((current) =>
      current.includes(name) ? current.filter((n) => n !== name) : [...current, name],
    );
  };

  return (
    <div>
      <div className="mb-2 text-sm text-gray-500 dark:text-gray-400">
        Only databases the entered user can connect to are listed.
      </div>

      <Input
        placeholder="Search database"
        value={searchQuery}
        onChange={(e) => setSearchQuery(e.target.value)}
        size="small"
        className="mb-2"
      />

      <div className="mb-2 flex">
        <Button
          size="small"
          type="link"
          className="pl-0"
          onClick={() => setSelectedNames(visibleNames.filter((name) => !alreadyAdded.has(name)))}
        >
          Select all
        </Button>

        <Button size="small" type="link" onClick={() => setSelectedNames([])}>
          Clear
        </Button>

        <div className="ml-auto self-center text-xs text-gray-500 dark:text-gray-400">
          {selectedNames.length} selected
        </div>
      </div>

      <div className="mb-3 max-h-[280px] overflow-y-auto rounded border border-gray-200 p-2 dark:border-gray-700">
        {visibleNames.length === 0 && (
          <div className="p-2 text-center text-sm text-gray-500 dark:text-gray-400">
            No databases found
          </div>
        )}

        {visibleNames.map((name) => (
          <div key={name} className="mb-1 flex items-center">
            <Checkbox
              checked={alreadyAdded.has(name) || selectedNames.includes(name)}
              disabled={alreadyAdded.has(name)}
              onChange={() => toggleName(name)}
            >
              {name}
            </Checkbox>

            {alreadyAdded.has(name) && (
              <span className="ml-2 text-xs text-gray-500 dark:text-gray-400">already added</span>
            )}
          </div>
        ))}
      </div>

      <div className="flex">
        <Button type="primary" ghost onClick={() => onBack()}>
          Back
        </Button>

        <Button
          className="ml-auto"
          type="primary"
          disabled={selectedNames.length === 0}
          onClick={() => onSelected(selectedNames)}
        >
          Continue
        </Button>
      </div>
    </div>
  );
};
