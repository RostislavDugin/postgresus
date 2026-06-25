import { CopyOutlined, DownOutlined, InfoCircleOutlined, UpOutlined } from '@ant-design/icons';
import { App, Button, Checkbox, Input, InputNumber, Select, Tooltip } from 'antd';
import { useEffect, useState } from 'react';

import {
  type Database,
  PostgresSslMode,
  type PostgresqlLogicalDatabase,
  databaseApi,
} from '../../../../entity/databases';
import { ConnectionStringParser } from '../../../../entity/databases/model/postgresql/ConnectionStringParser';
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
  isRestoreMode?: boolean;
}

const IPV4_PATTERN = /^\d{1,3}(\.\d{1,3}){3}$/;

const deriveSslModeFromHost = (rawHost: string): PostgresSslMode | null => {
  const trimmed = rawHost.trim().toLowerCase();

  if (trimmed.startsWith('https://')) return PostgresSslMode.Require;
  if (trimmed.startsWith('http://')) return PostgresSslMode.Disable;

  const bareHost = trimmed.split(':')[0];
  if (bareHost === 'localhost' || IPV4_PATTERN.test(bareHost)) {
    return PostgresSslMode.Disable;
  }

  return null;
};

const applySslMode = (
  postgresqlLogical: PostgresqlLogicalDatabase,
  sslMode: PostgresSslMode,
): PostgresqlLogicalDatabase => {
  if (sslMode === PostgresSslMode.Disable) {
    return { ...postgresqlLogical, sslMode, sslClientCert: '', sslClientKey: '', sslRootCert: '' };
  }

  return { ...postgresqlLogical, sslMode };
};

export const EditPostgreSqlLogicalSpecificDataComponent = ({
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
    !!database.postgresqlLogical?.sslClientCert ||
    !!database.postgresqlLogical?.sslRootCert ||
    (isRestoreMode
      ? !!database.postgresqlLogical?.isExcludeExtensions ||
        !!database.postgresqlLogical?.isRestoreOwnership ||
        !!database.postgresqlLogical?.isRestorePrivileges
      : !!database.postgresqlLogical?.includeSchemas?.length ||
        !!database.postgresqlLogical?.excludeTables?.length ||
        !!database.postgresqlLogical?.isSkipUserMappings);
  const [isShowAdvanced, setShowAdvanced] = useState(hasAdvancedValues);

  const [hasAutoAddedPublicSchema, setHasAutoAddedPublicSchema] = useState(false);
  const [hasUserChosenSslMode, setHasUserChosenSslMode] = useState(!!database.id);
  const [isReplacingCerts, setIsReplacingCerts] = useState(false);

  const [isShowPasteModal, setIsShowPasteModal] = useState(false);

  const applyConnectionString = (text: string) => {
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

    if (!editingDatabase?.postgresqlLogical) return;

    const updatedDatabase: Database = {
      ...editingDatabase,
      postgresqlLogical: {
        ...editingDatabase.postgresqlLogical,
        host: result.host,
        port: result.port,
        username: result.username,
        password: result.password,
        database: result.database,
        sslMode: result.sslMode,
        cpuCount: 1,
      },
    };

    setHasUserChosenSslMode(true);
    setEditingDatabase(autoAddPublicSchemaForSupabase(updatedDatabase));
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

  const autoAddPublicSchemaForSupabase = (updatedDatabase: Database): Database => {
    if (hasAutoAddedPublicSchema) return updatedDatabase;

    const host = updatedDatabase.postgresqlLogical?.host || '';
    const username = updatedDatabase.postgresqlLogical?.username || '';
    const isSupabase = host.includes('supabase') || username.includes('supabase');

    if (isSupabase && updatedDatabase.postgresqlLogical) {
      setHasAutoAddedPublicSchema(true);

      const currentSchemas = updatedDatabase.postgresqlLogical.includeSchemas || [];
      if (!currentSchemas.includes('public')) {
        return {
          ...updatedDatabase,
          postgresqlLogical: {
            ...updatedDatabase.postgresqlLogical,
            includeSchemas: ['public', ...currentSchemas],
          },
        };
      }
    }

    return updatedDatabase;
  };

  const testConnection = async () => {
    if (!editingDatabase?.postgresqlLogical) return;
    setIsTestingConnection(true);
    setIsConnectionFailed(false);

    const trimmedDatabase = {
      ...editingDatabase,
      postgresqlLogical: {
        ...editingDatabase.postgresqlLogical,
        password: editingDatabase.postgresqlLogical.password?.trim(),
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
    if (!editingDatabase?.postgresqlLogical) return;

    const trimmedDatabase = {
      ...editingDatabase,
      postgresqlLogical: {
        ...editingDatabase.postgresqlLogical,
        password: editingDatabase.postgresqlLogical.password?.trim(),
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

  const updatePostgresqlCert = (
    field: 'sslClientCert' | 'sslClientKey' | 'sslRootCert',
    value: string,
  ) => {
    if (!editingDatabase?.postgresqlLogical) return;

    setEditingDatabase({
      ...editingDatabase,
      postgresqlLogical: { ...editingDatabase.postgresqlLogical, [field]: value },
    });
    setIsConnectionTested(false);
  };

  const startReplacingCerts = () => {
    if (!editingDatabase?.postgresqlLogical) return;

    setIsReplacingCerts(true);
    setEditingDatabase({
      ...editingDatabase,
      postgresqlLogical: {
        ...editingDatabase.postgresqlLogical,
        sslClientCert: '',
        sslClientKey: '',
        sslRootCert: '',
      },
    });
    setIsConnectionTested(false);
  };

  useEffect(() => {
    setIsSaving(false);
    setIsConnectionTested(false);
    setIsTestingConnection(false);
    setIsConnectionFailed(false);
    setIsReplacingCerts(false);
    setHasUserChosenSslMode(!!database.id);

    setEditingDatabase({ ...database });
  }, [database]);

  if (!editingDatabase) return null;

  const renderFooter = (footerContent?: React.ReactNode) => (
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

      {footerContent}
    </div>
  );

  const renderSslCertSection = () => {
    const sslMode = editingDatabase.postgresqlLogical?.sslMode ?? PostgresSslMode.Disable;
    if (sslMode === PostgresSslMode.Disable) return null;

    const hadSslCert = !!database.postgresqlLogical?.sslClientCert;
    if (hadSslCert && !isReplacingCerts) {
      return (
        <div className="mb-1 flex w-full items-center">
          <div className="min-w-[150px]">Client certificate</div>
          <div className="flex items-center">
            <span className="mr-3">*************</span>
            <Button size="small" onClick={startReplacingCerts}>
              Replace
            </Button>
          </div>
        </div>
      );
    }

    return (
      <>
        <div className="mb-1 flex w-full items-start">
          <div className="min-w-[150px]">Client certificate</div>
          <Input.TextArea
            value={editingDatabase.postgresqlLogical?.sslClientCert || ''}
            onChange={(e) => updatePostgresqlCert('sslClientCert', e.target.value)}
            size="small"
            className="max-w-[300px] grow"
            placeholder="-----BEGIN CERTIFICATE-----"
            autoSize={{ minRows: 2, maxRows: 5 }}
          />
        </div>

        <div className="mb-1 flex w-full items-start">
          <div className="min-w-[150px]">Client key</div>
          <Input.TextArea
            value={editingDatabase.postgresqlLogical?.sslClientKey || ''}
            onChange={(e) => updatePostgresqlCert('sslClientKey', e.target.value)}
            size="small"
            className="max-w-[300px] grow"
            placeholder="-----BEGIN PRIVATE KEY-----"
            autoSize={{ minRows: 2, maxRows: 5 }}
          />
        </div>

        <div className="mb-1 flex w-full items-start">
          <div className="flex min-w-[150px] items-center">
            <span>Server CA certificate</span>
            <Tooltip
              className="cursor-pointer"
              title="Optional. When provided, the server certificate is verified against this CA (verify-ca / verify-full)."
            >
              <InfoCircleOutlined className="ml-2" style={{ color: 'gray' }} />
            </Tooltip>
          </div>
          <Input.TextArea
            value={editingDatabase.postgresqlLogical?.sslRootCert || ''}
            onChange={(e) => updatePostgresqlCert('sslRootCert', e.target.value)}
            size="small"
            className="max-w-[300px] grow"
            placeholder="-----BEGIN CERTIFICATE-----"
            autoSize={{ minRows: 2, maxRows: 5 }}
          />
        </div>
      </>
    );
  };

  const renderPgDumpForm = () => {
    let isAllFieldsFilled = true;
    if (!editingDatabase.postgresqlLogical?.host) isAllFieldsFilled = false;
    if (!editingDatabase.postgresqlLogical?.port) isAllFieldsFilled = false;
    if (!editingDatabase.postgresqlLogical?.username) isAllFieldsFilled = false;
    if (!editingDatabase.id && !editingDatabase.postgresqlLogical?.password)
      isAllFieldsFilled = false;
    if (!editingDatabase.postgresqlLogical?.database) isAllFieldsFilled = false;

    const isLocalhostDb =
      editingDatabase.postgresqlLogical?.host?.includes('localhost') ||
      editingDatabase.postgresqlLogical?.host?.includes('127.0.0.1');

    const isSupabaseDb =
      editingDatabase.postgresqlLogical?.host?.includes('supabase') ||
      editingDatabase.postgresqlLogical?.username?.includes('supabase');

    return (
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
            value={editingDatabase.postgresqlLogical?.host}
            onChange={(e) => {
              if (!editingDatabase.postgresqlLogical) return;

              const rawHost = e.target.value;
              const basePostgresql = {
                ...editingDatabase.postgresqlLogical,
                host: rawHost.trim().replace('https://', '').replace('http://', ''),
              };
              const isHttpsHost = rawHost.trim().toLowerCase().startsWith('https://');
              const currentSslMode = basePostgresql.sslMode ?? PostgresSslMode.Disable;

              let derivedSslMode: PostgresSslMode | null = null;
              if (hasUserChosenSslMode) {
                if (isHttpsHost && currentSslMode === PostgresSslMode.Disable) {
                  derivedSslMode = PostgresSslMode.Require;
                }
              } else {
                derivedSslMode = deriveSslModeFromHost(rawHost);
              }

              const updatedDatabase = {
                ...editingDatabase,
                postgresqlLogical:
                  derivedSslMode !== null
                    ? applySslMode(basePostgresql, derivedSslMode)
                    : basePostgresql,
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

        {isSupabaseDb && (
          <div className="mb-1 flex">
            <div className="min-w-[150px]" />
            <div className="max-w-[200px] text-xs text-gray-500 dark:text-gray-400">
              Please{' '}
              <a
                href="https://databasus.com/faq/supabase"
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
            value={editingDatabase.postgresqlLogical?.port}
            onChange={(e) => {
              if (!editingDatabase.postgresqlLogical || e === null) return;

              setEditingDatabase({
                ...editingDatabase,
                postgresqlLogical: { ...editingDatabase.postgresqlLogical, port: e },
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
            value={editingDatabase.postgresqlLogical?.username}
            onChange={(e) => {
              if (!editingDatabase.postgresqlLogical) return;

              const updatedDatabase = {
                ...editingDatabase,
                postgresqlLogical: {
                  ...editingDatabase.postgresqlLogical,
                  username: e.target.value.trim(),
                },
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
            value={editingDatabase.postgresqlLogical?.password}
            onChange={(e) => {
              if (!editingDatabase.postgresqlLogical) return;

              setEditingDatabase({
                ...editingDatabase,
                postgresqlLogical: {
                  ...editingDatabase.postgresqlLogical,
                  password: e.target.value,
                },
              });
              setIsConnectionTested(false);
            }}
            size="small"
            className="max-w-[200px] grow"
            placeholder="Enter PG password"
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
              value={editingDatabase.postgresqlLogical?.database}
              onChange={(e) => {
                if (!editingDatabase.postgresqlLogical) return;

                setEditingDatabase({
                  ...editingDatabase,
                  postgresqlLogical: {
                    ...editingDatabase.postgresqlLogical,
                    database: e.target.value.trim(),
                  },
                });
                setIsConnectionTested(false);
              }}
              size="small"
              className="max-w-[200px] grow"
              placeholder="Enter PG database name"
            />
          </div>
        )}

        <div className="mb-1 flex w-full items-center">
          <div className="min-w-[150px]">SSL mode</div>
          <Select
            value={editingDatabase.postgresqlLogical?.sslMode ?? PostgresSslMode.Disable}
            onChange={(value: PostgresSslMode) => {
              if (!editingDatabase.postgresqlLogical) return;

              setHasUserChosenSslMode(true);
              setEditingDatabase({
                ...editingDatabase,
                postgresqlLogical: applySslMode(editingDatabase.postgresqlLogical, value),
              });
              setIsConnectionTested(false);
            }}
            options={[
              { label: 'Disable', value: PostgresSslMode.Disable },
              { label: 'Require', value: PostgresSslMode.Require },
              { label: 'Verify CA', value: PostgresSslMode.VerifyCa },
              { label: 'Verify full', value: PostgresSslMode.VerifyFull },
            ]}
            size="small"
            className="max-w-[200px] grow"
          />
        </div>

        {isRestoreMode && (
          <div className="mb-5 flex w-full items-center">
            <div className="min-w-[150px]">CPU count</div>
            <div className="flex items-center">
              <InputNumber
                min={1}
                max={128}
                value={editingDatabase.postgresqlLogical?.cpuCount}
                onChange={(value) => {
                  if (!editingDatabase.postgresqlLogical) return;

                  setEditingDatabase({
                    ...editingDatabase,
                    postgresqlLogical: {
                      ...editingDatabase.postgresqlLogical,
                      cpuCount: value || 1,
                    },
                  });
                  setIsConnectionTested(false);
                }}
                size="small"
                className="max-w-[75px] grow"
              />

              <Tooltip
                className="cursor-pointer"
                title="Number of CPU cores to use for backup and restore operations. Higher values may speed up operations but use more resources."
              >
                <InfoCircleOutlined className="ml-2" style={{ color: 'gray' }} />
              </Tooltip>
            </div>
          </div>
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

        {isShowAdvanced && (
          <>
            {!isRestoreMode && (
              <div className="mb-1 flex w-full items-center">
                <div className="min-w-[150px]">Include schemas</div>
                <Select
                  mode="tags"
                  value={editingDatabase.postgresqlLogical?.includeSchemas || []}
                  onChange={(values) => {
                    if (!editingDatabase.postgresqlLogical) return;

                    setEditingDatabase({
                      ...editingDatabase,
                      postgresqlLogical: {
                        ...editingDatabase.postgresqlLogical,
                        includeSchemas: values,
                      },
                    });
                  }}
                  size="small"
                  className="max-w-[200px] grow"
                  placeholder="All schemas (default)"
                  tokenSeparators={[',']}
                />
              </div>
            )}

            {!isRestoreMode && (
              <div className="mb-1 flex w-full items-center">
                <div className="min-w-[150px]">Exclude tables</div>
                <Select
                  mode="tags"
                  value={editingDatabase.postgresqlLogical?.excludeTables || []}
                  onChange={(values) => {
                    if (!editingDatabase.postgresqlLogical) return;

                    setEditingDatabase({
                      ...editingDatabase,
                      postgresqlLogical: {
                        ...editingDatabase.postgresqlLogical,
                        excludeTables: values,
                      },
                    });
                  }}
                  size="small"
                  className="max-w-[200px] grow"
                  placeholder="No tables excluded"
                  tokenSeparators={[',']}
                />

                <Tooltip
                  className="cursor-pointer"
                  title="Tables to exclude from the backup. Use 'tablename' or 'schema.tablename'. Glob patterns are supported (e.g. 'logs_*')."
                >
                  <InfoCircleOutlined className="ml-2" style={{ color: 'gray' }} />
                </Tooltip>
              </div>
            )}

            {!isRestoreMode && (
              <div className="mb-1 flex w-full items-center">
                <div className="flex min-w-[150px] items-center">
                  <span>Skip user mappings</span>
                  <Tooltip
                    className="cursor-pointer"
                    title="Skip restoring user mappings (CREATE USER MAPPING statements). Enable this when the backup role cannot read the mapping credentials - otherwise they are dumped without options and break restore for FDWs like oracle_fdw."
                  >
                    <InfoCircleOutlined className="ml-2" style={{ color: 'gray' }} />
                  </Tooltip>
                </div>
                <Checkbox
                  checked={editingDatabase.postgresqlLogical?.isSkipUserMappings || false}
                  onChange={(e) => {
                    if (!editingDatabase.postgresqlLogical) return;

                    setEditingDatabase({
                      ...editingDatabase,
                      postgresqlLogical: {
                        ...editingDatabase.postgresqlLogical,
                        isSkipUserMappings: e.target.checked,
                      },
                    });
                  }}
                />
              </div>
            )}

            {isRestoreMode && (
              <div className="mb-1 flex w-full items-center">
                <div className="flex min-w-[150px] items-center">
                  <span>Exclude extensions</span>
                  <Tooltip
                    className="cursor-pointer"
                    title="Skip restoring extension definitions (CREATE EXTENSION statements). Enable this if you're restoring to a managed PostgreSQL service where extensions are managed by the provider."
                  >
                    <InfoCircleOutlined className="ml-2" style={{ color: 'gray' }} />
                  </Tooltip>
                </div>
                <Checkbox
                  checked={editingDatabase.postgresqlLogical?.isExcludeExtensions || false}
                  onChange={(e) => {
                    if (!editingDatabase.postgresqlLogical) return;

                    setEditingDatabase({
                      ...editingDatabase,
                      postgresqlLogical: {
                        ...editingDatabase.postgresqlLogical,
                        isExcludeExtensions: e.target.checked,
                      },
                    });
                  }}
                />
              </div>
            )}

            {isRestoreMode && (
              <div className="mb-1 flex w-full items-center">
                <div className="flex min-w-[150px] items-center">
                  <span>Restore ownership</span>
                  <Tooltip
                    className="cursor-pointer"
                    title="Apply ALTER OWNER statements from the dump so restored objects keep their original owner. The connection user must be able to assign these roles - typically a superuser."
                  >
                    <InfoCircleOutlined className="ml-2" style={{ color: 'gray' }} />
                  </Tooltip>
                </div>
                <Checkbox
                  checked={editingDatabase.postgresqlLogical?.isRestoreOwnership || false}
                  onChange={(e) => {
                    if (!editingDatabase.postgresqlLogical) return;

                    setEditingDatabase({
                      ...editingDatabase,
                      postgresqlLogical: {
                        ...editingDatabase.postgresqlLogical,
                        isRestoreOwnership: e.target.checked,
                      },
                    });
                  }}
                />
              </div>
            )}

            {isRestoreMode && (
              <div className="mb-1 flex w-full items-center">
                <div className="flex min-w-[150px] items-center">
                  <span>Restore privileges</span>
                  <Tooltip
                    className="cursor-pointer"
                    title="Apply GRANT and REVOKE statements from the dump so restored objects keep their original ACLs. The connection user must be able to grant to the referenced roles - typically a superuser."
                  >
                    <InfoCircleOutlined className="ml-2" style={{ color: 'gray' }} />
                  </Tooltip>
                </div>
                <Checkbox
                  checked={editingDatabase.postgresqlLogical?.isRestorePrivileges || false}
                  onChange={(e) => {
                    if (!editingDatabase.postgresqlLogical) return;

                    setEditingDatabase({
                      ...editingDatabase,
                      postgresqlLogical: {
                        ...editingDatabase.postgresqlLogical,
                        isRestorePrivileges: e.target.checked,
                      },
                    });
                  }}
                />
              </div>
            )}

            {renderSslCertSection()}
          </>
        )}

        {renderFooter(
          <>
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
          </>,
        )}

        {isConnectionFailed && (
          <div className="mt-3 text-sm text-gray-500 dark:text-gray-400">
            If your database uses IP whitelist, make sure Databasus server IP is added to the
            allowed list.
          </div>
        )}
      </>
    );
  };

  return (
    <div>
      {renderPgDumpForm()}

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
