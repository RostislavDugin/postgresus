import { CopyOutlined, DownOutlined, InfoCircleOutlined, UpOutlined } from '@ant-design/icons';
import { App, Button, Checkbox, Input, InputNumber, Select, Switch, Tooltip } from 'antd';
import { useEffect, useState } from 'react';

import { type Database, DatabaseType, databaseApi } from '../../../../entity/databases';
import { ConnectionStringParser } from '../../../../entity/databases/model/postgresql/ConnectionStringParser';
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
  isRestoreMode?: boolean;
}

export const EditDatabaseSpecificDataComponent = ({
  database,

  isShowCancelButton,
  onCancel,

  isShowBackButton,
  onBack,

  saveButtonText,
  isSaveToApi,
  onSaved,
  isShowDbName = true,
  isRestoreMode = false,
}: Props) => {
  const { message } = App.useApp();

  const [editingDatabase, setEditingDatabase] = useState<Database>();
  const [isSaving, setIsSaving] = useState(false);

  const [isConnectionTested, setIsConnectionTested] = useState(false);
  const [isTestingConnection, setIsTestingConnection] = useState(false);
  const [isConnectionFailed, setIsConnectionFailed] = useState(false);

  const hasAdvancedValues =
    !!database.postgresql?.includeSchemas?.length || !!database.postgresql?.isExcludeExtensions;
  const [isShowAdvanced, setShowAdvanced] = useState(hasAdvancedValues);

  const [hasAutoAddedPublicSchema, setHasAutoAddedPublicSchema] = useState(false);

  const parseFromClipboard = async () => {
    try {
      const text = await navigator.clipboard.readText();
      const trimmedText = text.trim();

      if (!trimmedText) {
        message.error('Clipboard is empty');
        return;
      }

      const result = ConnectionStringParser.parse(trimmedText);

      if ('error' in result) {
        message.error(result.error);
        return;
      }

      if (!editingDatabase?.postgresql) return;

      const updatedDatabase: Database = {
        ...editingDatabase,
        postgresql: {
          ...editingDatabase.postgresql,
          host: result.host,
          port: result.port,
          username: result.username,
          password: result.password,
          database: result.database,
          isHttps: result.isHttps,
        },
      };

      setEditingDatabase(autoAddPublicSchemaForSupabase(updatedDatabase));
      setIsConnectionTested(false);
      message.success('Connection string parsed successfully');
    } catch {
      message.error('Failed to read clipboard. Please check browser permissions.');
    }
  };

  const autoAddPublicSchemaForSupabase = (updatedDatabase: Database): Database => {
    if (hasAutoAddedPublicSchema) return updatedDatabase;

    const host = updatedDatabase.postgresql?.host || '';
    const username = updatedDatabase.postgresql?.username || '';
    const isSupabase = host.includes('supabase') || username.includes('supabase');

    if (isSupabase && updatedDatabase.postgresql) {
      setHasAutoAddedPublicSchema(true);

      const currentSchemas = updatedDatabase.postgresql.includeSchemas || [];
      if (!currentSchemas.includes('public')) {
        return {
          ...updatedDatabase,
          postgresql: {
            ...updatedDatabase.postgresql,
            includeSchemas: ['public', ...currentSchemas],
          },
        };
      }
    }

    return updatedDatabase;
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
  if (!editingDatabase.postgresql?.host) isAllFieldsFilled = false;
  if (!editingDatabase.postgresql?.port) isAllFieldsFilled = false;
  if (!editingDatabase.postgresql?.username) isAllFieldsFilled = false;
  if (!editingDatabase.id && !editingDatabase.postgresql?.password) isAllFieldsFilled = false;
  if (!editingDatabase.postgresql?.database) isAllFieldsFilled = false;

  const isLocalhostDb =
    editingDatabase.postgresql?.host?.includes('localhost') ||
    editingDatabase.postgresql?.host?.includes('127.0.0.1');

  const isSupabaseDb =
    editingDatabase.postgresql?.host?.includes('supabase') ||
    editingDatabase.postgresql?.username?.includes('supabase');

  return (
    <div>
      {editingDatabase.type === DatabaseType.POSTGRES && (
        <>
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
              value={editingDatabase.postgresql?.host}
              onChange={(e) => {
                if (!editingDatabase.postgresql) return;

                const updatedDatabase = {
                  ...editingDatabase,
                  postgresql: {
                    ...editingDatabase.postgresql,
                    host: e.target.value.trim().replace('https://', '').replace('http://', ''),
                  },
                };
                setEditingDatabase(autoAddPublicSchemaForSupabase(updatedDatabase));
                setIsConnectionTested(false);
              }}
              size="small"
              className="max-w-[200px] grow"
              placeholder="Enter PG host"
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

          {isSupabaseDb && (
            <div className="mb-1 flex">
              <div className="min-w-[150px]" />
              <div className="max-w-[200px] text-xs text-gray-500 dark:text-gray-400">
                Please{' '}
                <a
                  href="https://postgresus.com/faq/supabase"
                  target="_blank"
                  rel="noreferrer"
                  className="!text-blue-600 dark:!text-blue-400"
                >
                  read this document
                </a>{' '}
                to study how to backup Supabase database
              </div>
            </div>
          )}

          <div className="mb-1 flex w-full items-center">
            <div className="min-w-[150px]">Port</div>
            <InputNumber
              type="number"
              value={editingDatabase.postgresql?.port}
              onChange={(e) => {
                if (!editingDatabase.postgresql || e === null) return;

                setEditingDatabase({
                  ...editingDatabase,
                  postgresql: { ...editingDatabase.postgresql, port: e },
                });
                setIsConnectionTested(false);
              }}
              size="small"
              className="max-w-[200px] grow"
              placeholder="Enter PG port"
            />
          </div>

          <div className="mb-1 flex w-full items-center">
            <div className="min-w-[150px]">Username</div>
            <Input
              value={editingDatabase.postgresql?.username}
              onChange={(e) => {
                if (!editingDatabase.postgresql) return;

                const updatedDatabase = {
                  ...editingDatabase,
                  postgresql: { ...editingDatabase.postgresql, username: e.target.value.trim() },
                };
                setEditingDatabase(autoAddPublicSchemaForSupabase(updatedDatabase));
                setIsConnectionTested(false);
              }}
              size="small"
              className="max-w-[200px] grow"
              placeholder="Enter PG username"
            />
          </div>

          <div className="mb-1 flex w-full items-center">
            <div className="min-w-[150px]">Password</div>
            <Input.Password
              value={editingDatabase.postgresql?.password}
              onChange={(e) => {
                if (!editingDatabase.postgresql) return;

                setEditingDatabase({
                  ...editingDatabase,
                  postgresql: { ...editingDatabase.postgresql, password: e.target.value.trim() },
                });
                setIsConnectionTested(false);
              }}
              size="small"
              className="max-w-[200px] grow"
              placeholder="Enter PG password"
            />
          </div>

          {isShowDbName && (
            <div className="mb-1 flex w-full items-center">
              <div className="min-w-[150px]">DB name</div>
              <Input
                value={editingDatabase.postgresql?.database}
                onChange={(e) => {
                  if (!editingDatabase.postgresql) return;

                  setEditingDatabase({
                    ...editingDatabase,
                    postgresql: { ...editingDatabase.postgresql, database: e.target.value.trim() },
                  });
                  setIsConnectionTested(false);
                }}
                size="small"
                className="max-w-[200px] grow"
                placeholder="Enter PG database name (optional)"
              />
            </div>
          )}

          <div className="mb-3 flex w-full items-center">
            <div className="min-w-[150px]">Use HTTPS</div>
            <Switch
              checked={editingDatabase.postgresql?.isHttps}
              onChange={(checked) => {
                if (!editingDatabase.postgresql) return;

                setEditingDatabase({
                  ...editingDatabase,
                  postgresql: { ...editingDatabase.postgresql, isHttps: checked },
                });
                setIsConnectionTested(false);
              }}
              size="small"
            />
          </div>

          <div className="mt-4 mb-3 flex items-center">
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
              {!isRestoreMode && (
                <div className="mb-1 flex w-full items-center">
                  <div className="min-w-[150px]">Include schemas</div>
                  <Select
                    mode="tags"
                    value={editingDatabase.postgresql?.includeSchemas || []}
                    onChange={(values) => {
                      if (!editingDatabase.postgresql) return;

                      setEditingDatabase({
                        ...editingDatabase,
                        postgresql: { ...editingDatabase.postgresql, includeSchemas: values },
                      });
                    }}
                    size="small"
                    className="max-w-[200px] grow"
                    placeholder="All schemas (default)"
                    tokenSeparators={[',']}
                  />
                </div>
              )}

              {isRestoreMode && (
                <div className="mb-1 flex w-full items-center">
                  <div className="min-w-[150px]">Exclude extensions</div>
                  <div className="flex items-center">
                    <Checkbox
                      checked={editingDatabase.postgresql?.isExcludeExtensions || false}
                      onChange={(e) => {
                        if (!editingDatabase.postgresql) return;

                        setEditingDatabase({
                          ...editingDatabase,
                          postgresql: {
                            ...editingDatabase.postgresql,
                            isExcludeExtensions: e.target.checked,
                          },
                        });
                      }}
                    >
                      Skip extensions
                    </Checkbox>

                    <Tooltip
                      className="cursor-pointer"
                      title="Skip restoring extension definitions (CREATE EXTENSION statements). Enable this if you're restoring to a managed PostgreSQL service where extensions are managed by the provider."
                    >
                      <InfoCircleOutlined className="ml-2" style={{ color: 'gray' }} />
                    </Tooltip>
                  </div>
                </div>
              )}
            </>
          )}
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
          If your database uses IP whitelist, make sure Postgresus server IP is added to the allowed
          list.
        </div>
      )}
    </div>
  );
};
