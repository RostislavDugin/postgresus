import { CopyOutlined } from '@ant-design/icons';
import { App, Button, Input, InputNumber, Switch } from 'antd';
import { useEffect, useState } from 'react';

import { type Database, databaseApi } from '../../../../entity/databases';
import { MariadbConnectionStringParser } from '../../../../entity/databases/model/mariadb/MariadbConnectionStringParser';
import { ToastHelper } from '../../../../shared/toast';

interface Props {
  database: Database;

  isShowCancelButton?: boolean;
  onCancel: () => void;

  isShowBackButton: boolean;
  onBack: () => void;

  saveButtonText?: string;
  isSaveToApi: boolean;
  onSaved: (database: Database) => void;

  isShowDbName?: boolean;
}

export const EditMariaDbSpecificDataComponent = ({
  database,

  isShowCancelButton,
  onCancel,

  isShowBackButton,
  onBack,

  saveButtonText,
  isSaveToApi,
  onSaved,
  isShowDbName = true,
}: Props) => {
  const { message } = App.useApp();

  const [editingDatabase, setEditingDatabase] = useState<Database>();
  const [isSaving, setIsSaving] = useState(false);

  const [isConnectionTested, setIsConnectionTested] = useState(false);
  const [isTestingConnection, setIsTestingConnection] = useState(false);
  const [isConnectionFailed, setIsConnectionFailed] = useState(false);

  const parseFromClipboard = async () => {
    try {
      const text = await navigator.clipboard.readText();
      const trimmedText = text.trim();

      if (!trimmedText) {
        message.error('Clipboard is empty');
        return;
      }

      const result = MariadbConnectionStringParser.parse(trimmedText);

      if ('error' in result) {
        message.error(result.error);
        return;
      }

      if (!editingDatabase?.mariadb) return;

      const updatedDatabase: Database = {
        ...editingDatabase,
        mariadb: {
          ...editingDatabase.mariadb,
          host: result.host,
          port: result.port,
          username: result.username,
          password: result.password,
          database: result.database,
          isHttps: result.isHttps,
        },
      };

      setEditingDatabase(updatedDatabase);
      setIsConnectionTested(false);
      message.success('Connection string parsed successfully');
    } catch {
      message.error('Failed to read clipboard. Please check browser permissions.');
    }
  };

  const testConnection = async () => {
    if (!editingDatabase) return;
    setIsTestingConnection(true);
    setIsConnectionFailed(false);

    try {
      await databaseApi.testDatabaseConnectionDirect(editingDatabase);
      setIsConnectionTested(true);
      ToastHelper.showToast({
        title: 'Connection test passed',
        description: 'You can continue with the next step',
      });
    } catch (e) {
      setIsConnectionFailed(true);
      alert((e as Error).message);
    }

    setIsTestingConnection(false);
  };

  const saveDatabase = async () => {
    if (!editingDatabase) return;

    if (isSaveToApi) {
      setIsSaving(true);

      try {
        await databaseApi.updateDatabase(editingDatabase);
      } catch (e) {
        alert((e as Error).message);
      }

      setIsSaving(false);
    }

    onSaved(editingDatabase);
  };

  useEffect(() => {
    setIsSaving(false);
    setIsConnectionTested(false);
    setIsTestingConnection(false);
    setIsConnectionFailed(false);

    setEditingDatabase({ ...database });
  }, [database]);

  if (!editingDatabase) return null;

  let isAllFieldsFilled = true;
  if (!editingDatabase.mariadb?.host) isAllFieldsFilled = false;
  if (!editingDatabase.mariadb?.port) isAllFieldsFilled = false;
  if (!editingDatabase.mariadb?.username) isAllFieldsFilled = false;
  if (!editingDatabase.id && !editingDatabase.mariadb?.password) isAllFieldsFilled = false;
  if (!editingDatabase.mariadb?.database) isAllFieldsFilled = false;

  const isLocalhostDb =
    editingDatabase.mariadb?.host?.includes('localhost') ||
    editingDatabase.mariadb?.host?.includes('127.0.0.1');

  return (
    <div>
      <div className="mb-3 flex">
        <div className="min-w-[150px]" />
        <div
          className="cursor-pointer text-sm text-gray-600 transition-colors hover:text-gray-900 dark:text-gray-400 dark:hover:text-gray-200"
          onClick={parseFromClipboard}
        >
          <CopyOutlined className="mr-1" />
          Parse from clipboard
        </div>
      </div>

      <div className="mb-1 flex w-full items-center">
        <div className="min-w-[150px]">Host</div>
        <Input
          value={editingDatabase.mariadb?.host}
          onChange={(e) => {
            if (!editingDatabase.mariadb) return;

            setEditingDatabase({
              ...editingDatabase,
              mariadb: {
                ...editingDatabase.mariadb,
                host: e.target.value.trim().replace('https://', '').replace('http://', ''),
              },
            });
            setIsConnectionTested(false);
          }}
          size="small"
          className="max-w-[200px] grow"
          placeholder="Enter MariaDB host"
        />
      </div>

      {isLocalhostDb && (
        <div className="mb-1 flex">
          <div className="min-w-[150px]" />
          <div className="max-w-[200px] text-xs text-gray-500 dark:text-gray-400">
            Please{' '}
            <a
              href="https://postgresus.com/faq/localhost"
              target="_blank"
              rel="noreferrer"
              className="!text-blue-600 dark:!text-blue-400"
            >
              read this document
            </a>{' '}
            to study how to backup local database
          </div>
        </div>
      )}

      <div className="mb-1 flex w-full items-center">
        <div className="min-w-[150px]">Port</div>
        <InputNumber
          type="number"
          value={editingDatabase.mariadb?.port}
          onChange={(e) => {
            if (!editingDatabase.mariadb || e === null) return;

            setEditingDatabase({
              ...editingDatabase,
              mariadb: { ...editingDatabase.mariadb, port: e },
            });
            setIsConnectionTested(false);
          }}
          size="small"
          className="max-w-[200px] grow"
          placeholder="Enter MariaDB port"
        />
      </div>

      <div className="mb-1 flex w-full items-center">
        <div className="min-w-[150px]">Username</div>
        <Input
          value={editingDatabase.mariadb?.username}
          onChange={(e) => {
            if (!editingDatabase.mariadb) return;

            setEditingDatabase({
              ...editingDatabase,
              mariadb: { ...editingDatabase.mariadb, username: e.target.value.trim() },
            });
            setIsConnectionTested(false);
          }}
          size="small"
          className="max-w-[200px] grow"
          placeholder="Enter MariaDB username"
        />
      </div>

      <div className="mb-1 flex w-full items-center">
        <div className="min-w-[150px]">Password</div>
        <Input.Password
          value={editingDatabase.mariadb?.password}
          onChange={(e) => {
            if (!editingDatabase.mariadb) return;

            setEditingDatabase({
              ...editingDatabase,
              mariadb: { ...editingDatabase.mariadb, password: e.target.value.trim() },
            });
            setIsConnectionTested(false);
          }}
          size="small"
          className="max-w-[200px] grow"
          placeholder="Enter MariaDB password"
          autoComplete="off"
          data-1p-ignore
          data-lpignore="true"
          data-form-type="other"
        />
      </div>

      {isShowDbName && (
        <div className="mb-1 flex w-full items-center">
          <div className="min-w-[150px]">DB name</div>
          <Input
            value={editingDatabase.mariadb?.database}
            onChange={(e) => {
              if (!editingDatabase.mariadb) return;

              setEditingDatabase({
                ...editingDatabase,
                mariadb: { ...editingDatabase.mariadb, database: e.target.value.trim() },
              });
              setIsConnectionTested(false);
            }}
            size="small"
            className="max-w-[200px] grow"
            placeholder="Enter MariaDB database name"
          />
        </div>
      )}

      <div className="mb-3 flex w-full items-center">
        <div className="min-w-[150px]">Use HTTPS</div>
        <Switch
          checked={editingDatabase.mariadb?.isHttps}
          onChange={(checked) => {
            if (!editingDatabase.mariadb) return;

            setEditingDatabase({
              ...editingDatabase,
              mariadb: { ...editingDatabase.mariadb, isHttps: checked },
            });
            setIsConnectionTested(false);
          }}
          size="small"
        />
      </div>

      <div className="mt-5 flex">
        {isShowCancelButton && (
          <Button className="mr-1" danger ghost onClick={() => onCancel()}>
            Cancel
          </Button>
        )}

        {isShowBackButton && (
          <Button className="mr-auto" type="primary" ghost onClick={() => onBack()}>
            Back
          </Button>
        )}

        {!isConnectionTested && (
          <Button
            type="primary"
            onClick={() => testConnection()}
            loading={isTestingConnection}
            disabled={!isAllFieldsFilled}
            className="mr-5"
          >
            Test connection
          </Button>
        )}

        {isConnectionTested && (
          <Button
            type="primary"
            onClick={() => saveDatabase()}
            loading={isSaving}
            disabled={!isAllFieldsFilled}
            className="mr-5"
          >
            {saveButtonText || 'Save'}
          </Button>
        )}
      </div>

      {isConnectionFailed && (
        <div className="mt-3 text-sm text-gray-500 dark:text-gray-400">
          If your database uses IP whitelist, make sure Postgresus server IP is added to the allowed
          list.
        </div>
      )}
    </div>
  );
};
