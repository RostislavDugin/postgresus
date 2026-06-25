import { CopyOutlined, DownOutlined, InfoCircleOutlined, UpOutlined } from '@ant-design/icons';
import { App, Button, Checkbox, Input, InputNumber, Select, Switch, Tooltip } from 'antd';
import { useEffect, useState } from 'react';

import { type Database, databaseApi } from '../../../../entity/databases';
import { MariadbConnectionStringParser } from '../../../../entity/databases/model/mariadb/MariadbConnectionStringParser';
import { ClipboardHelper } from '../../../../shared/lib/ClipboardHelper';
import { ToastHelper } from '../../../../shared/toast';
import { ClipboardPasteModalComponent } from '../../../../shared/ui';

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

  const hasAdvancedValues =
    !!database.mariadb?.isExcludeEvents ||
    !!database.mariadb?.isUseExtendedInsert ||
    !!database.mariadb?.isSkipGaleraDisable ||
    !!database.mariadb?.excludeTables?.length;
  const [isShowAdvanced, setShowAdvanced] = useState(hasAdvancedValues);

  const [isShowPasteModal, setIsShowPasteModal] = useState(false);

  const applyConnectionString = (text: string) => {
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
  };

  const parseFromClipboard = async () => {
    if (!ClipboardHelper.isClipboardApiAvailable()) {
      setIsShowPasteModal(true);
      return;
    }

    try {
      const text = await ClipboardHelper.readFromClipboard();
      applyConnectionString(text);
    } catch {
      message.error('Failed to read clipboard. Please check browser permissions.');
    }
  };

  const testConnection = async () => {
    if (!editingDatabase?.mariadb) return;
    setIsTestingConnection(true);
    setIsConnectionFailed(false);

    const trimmedDatabase = {
      ...editingDatabase,
      mariadb: {
        ...editingDatabase.mariadb,
        password: editingDatabase.mariadb.password?.trim(),
      },
    };

    try {
      await databaseApi.testDatabaseConnectionDirect(trimmedDatabase);
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
    if (!editingDatabase?.mariadb) return;

    const trimmedDatabase = {
      ...editingDatabase,
      mariadb: {
        ...editingDatabase.mariadb,
        password: editingDatabase.mariadb.password?.trim(),
      },
    };

    if (isSaveToApi) {
      setIsSaving(true);

      try {
        await databaseApi.updateDatabase(trimmedDatabase);
      } catch (e) {
        alert((e as Error).message);
      }

      setIsSaving(false);
    }

    onSaved(trimmedDatabase);
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
              href="https://databasus.com/faq/localhost"
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
              mariadb: { ...editingDatabase.mariadb, password: e.target.value },
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

      <div className="mb-1 flex w-full items-center">
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

      {isShowAdvanced && (
        <>
          <div className="mb-1 flex w-full items-center">
            <div className="min-w-[150px]">Exclude events</div>
            <div className="flex items-center">
              <Checkbox
                checked={editingDatabase.mariadb?.isExcludeEvents || false}
                onChange={(e) => {
                  if (!editingDatabase.mariadb) return;

                  setEditingDatabase({
                    ...editingDatabase,
                    mariadb: {
                      ...editingDatabase.mariadb,
                      isExcludeEvents: e.target.checked,
                    },
                  });
                }}
              >
                Skip events
              </Checkbox>

              <Tooltip
                className="cursor-pointer"
                title="Skip backing up database events. Enable this if the event scheduler is disabled on your MariaDB server."
              >
                <InfoCircleOutlined className="ml-2" style={{ color: 'gray' }} />
              </Tooltip>
            </div>
          </div>

          <div className="mb-1 flex w-full items-center">
            <div className="min-w-[150px]">Use extended inserts</div>
            <div className="flex items-center">
              <Checkbox
                checked={editingDatabase.mariadb?.isUseExtendedInsert || false}
                onChange={(e) => {
                  if (!editingDatabase.mariadb) return;

                  setEditingDatabase({
                    ...editingDatabase,
                    mariadb: {
                      ...editingDatabase.mariadb,
                      isUseExtendedInsert: e.target.checked,
                    },
                  });
                }}
              >
                Enable extended inserts
              </Checkbox>

              <Tooltip
                className="cursor-pointer"
                title="Batch multiple rows per INSERT for much faster restores. Off by default because it uses more memory during backup - enable only if restores are slow and your server has enough memory."
              >
                <InfoCircleOutlined className="ml-2" style={{ color: 'gray' }} />
              </Tooltip>
            </div>
          </div>

          <div className="mb-1 flex w-full items-center">
            <div className="min-w-[150px]">Galera replication</div>
            <div className="flex items-center">
              <Checkbox
                checked={editingDatabase.mariadb?.isSkipGaleraDisable || false}
                onChange={(e) => {
                  if (!editingDatabase.mariadb) return;

                  setEditingDatabase({
                    ...editingDatabase,
                    mariadb: {
                      ...editingDatabase.mariadb,
                      isSkipGaleraDisable: e.target.checked,
                    },
                  });
                }}
              >
                Skip disabling on restore
              </Checkbox>

              <Tooltip
                className="cursor-pointer"
                title="By default Databasus runs SET SESSION wsrep_on=OFF during restore to avoid Galera writeset-size errors. That requires the SUPER privilege. Enable this to skip it if your managed provider denies SUPER - large restores may then hit Galera writeset limits."
              >
                <InfoCircleOutlined className="ml-2" style={{ color: 'gray' }} />
              </Tooltip>
            </div>
          </div>

          <div className="mb-1 flex w-full items-center">
            <div className="min-w-[150px]">Exclude tables</div>
            <Select
              mode="tags"
              value={editingDatabase.mariadb?.excludeTables || []}
              onChange={(values) => {
                if (!editingDatabase.mariadb) return;

                setEditingDatabase({
                  ...editingDatabase,
                  mariadb: { ...editingDatabase.mariadb, excludeTables: values },
                });
              }}
              size="small"
              className="max-w-[200px] grow"
              placeholder="No tables excluded"
              tokenSeparators={[',']}
            />

            <Tooltip className="cursor-pointer" title="Table names to exclude from the backup.">
              <InfoCircleOutlined className="ml-2" style={{ color: 'gray' }} />
            </Tooltip>
          </div>
        </>
      )}

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
          If your database uses IP whitelist, make sure Databasus server IP is added to the allowed
          list.
        </div>
      )}

      <ClipboardPasteModalComponent
        open={isShowPasteModal}
        onSubmit={(text) => {
          setIsShowPasteModal(false);
          applyConnectionString(text);
        }}
        onCancel={() => setIsShowPasteModal(false)}
      />
    </div>
  );
};
