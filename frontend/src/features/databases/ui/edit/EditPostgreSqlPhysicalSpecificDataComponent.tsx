import { CopyOutlined, InfoCircleOutlined } from '@ant-design/icons';
import { Alert, App, Button, Input, InputNumber, Radio, Select, Space, Tooltip } from 'antd';
import { useEffect, useState } from 'react';

import {
  ConnectionErrorCode,
  type Database,
  type PhysicalConnectionErrorTextRun,
  PhysicalDatabaseBackupType,
  PostgresSslMode,
  type PostgresqlPhysicalDatabase,
  databaseApi,
  physicalConnectionErrorContent,
} from '../../../../entity/databases';
import { ConnectionStringParser } from '../../../../entity/databases/model/postgresql/ConnectionStringParser';
import { ApiError } from '../../../../shared/api';
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

  onConnectionErrorChange?: (hasConnectionError: boolean) => void;
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
  postgresqlPhysical: PostgresqlPhysicalDatabase,
  sslMode: PostgresSslMode,
): PostgresqlPhysicalDatabase => {
  if (sslMode === PostgresSslMode.Disable) {
    return {
      ...postgresqlPhysical,
      sslMode,
      sslClientCert: '',
      sslClientKey: '',
      sslRootCert: '',
    };
  }

  return { ...postgresqlPhysical, sslMode };
};

export const EditPostgreSqlPhysicalSpecificDataComponent = ({
  database,

  isShowCancelButton,
  onCancel,

  isShowBackButton,
  onBack,

  saveButtonText,
  isSaveToApi,
  onSaved,
  onConnectionErrorChange,
}: Props) => {
  const { message } = App.useApp();

  const [editingDatabase, setEditingDatabase] = useState<Database>();
  const [isSaving, setIsSaving] = useState(false);

  const [isConnectionTested, setIsConnectionTested] = useState(false);
  const [isTestingConnection, setIsTestingConnection] = useState(false);

  const [hasUserChosenSslMode, setHasUserChosenSslMode] = useState(!!database.id);
  const [isReplacingCerts, setIsReplacingCerts] = useState(false);

  const [isShowPasteModal, setIsShowPasteModal] = useState(false);

  const [connectionErrorCode, setConnectionErrorCode] = useState<ConnectionErrorCode | null>(null);

  const invalidateConnectionTest = () => {
    setIsConnectionTested(false);
    setConnectionErrorCode(null);
  };

  const copyCommand = async (command: string) => {
    try {
      await ClipboardHelper.copyToClipboard(command);
      message.success('Copied to clipboard');
    } catch {
      message.error('Failed to copy');
    }
  };

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

    if (!editingDatabase?.postgresqlPhysical) return;

    const updatedDatabase: Database = {
      ...editingDatabase,
      postgresqlPhysical: {
        ...editingDatabase.postgresqlPhysical,
        host: result.host,
        port: result.port,
        username: result.username,
        password: result.password,
        sslMode: result.sslMode,
      },
    };

    setHasUserChosenSslMode(true);
    setEditingDatabase(updatedDatabase);
    invalidateConnectionTest();
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

  const updateBackupType = (backupType: PhysicalDatabaseBackupType) => {
    if (!editingDatabase?.postgresqlPhysical) return;

    setEditingDatabase({
      ...editingDatabase,
      postgresqlPhysical: { ...editingDatabase.postgresqlPhysical, backupType },
    });
    invalidateConnectionTest();
  };

  const testConnection = async () => {
    if (!editingDatabase?.postgresqlPhysical) return;
    setIsTestingConnection(true);
    setConnectionErrorCode(null);

    const trimmedDatabase = {
      ...editingDatabase,
      postgresqlPhysical: {
        ...editingDatabase.postgresqlPhysical,
        password: editingDatabase.postgresqlPhysical.password?.trim(),
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
      if (e instanceof ApiError && e.code && e.code in physicalConnectionErrorContent) {
        setConnectionErrorCode(e.code as ConnectionErrorCode);
      } else {
        message.error((e as Error).message);
      }
    }

    setIsTestingConnection(false);
  };

  const saveDatabase = async () => {
    if (!editingDatabase?.postgresqlPhysical) return;

    const trimmedDatabase = {
      ...editingDatabase,
      postgresqlPhysical: {
        ...editingDatabase.postgresqlPhysical,
        password: editingDatabase.postgresqlPhysical.password?.trim(),
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
    if (!editingDatabase?.postgresqlPhysical) return;

    setEditingDatabase({
      ...editingDatabase,
      postgresqlPhysical: { ...editingDatabase.postgresqlPhysical, [field]: value },
    });
    invalidateConnectionTest();
  };

  const startReplacingCerts = () => {
    if (!editingDatabase?.postgresqlPhysical) return;

    setIsReplacingCerts(true);
    setEditingDatabase({
      ...editingDatabase,
      postgresqlPhysical: {
        ...editingDatabase.postgresqlPhysical,
        sslClientCert: '',
        sslClientKey: '',
        sslRootCert: '',
      },
    });
    invalidateConnectionTest();
  };

  useEffect(() => {
    setIsSaving(false);
    invalidateConnectionTest();
    setIsTestingConnection(false);
    setIsReplacingCerts(false);
    setHasUserChosenSslMode(!!database.id);

    setEditingDatabase({ ...database });
  }, [database]);

  useEffect(() => {
    onConnectionErrorChange?.(connectionErrorCode !== null);
  }, [connectionErrorCode, onConnectionErrorChange]);

  if (!editingDatabase) return null;

  const backupTypeOptions = [
    {
      label: 'Full backups only',
      value: PhysicalDatabaseBackupType.FULL,
      tooltip: 'Periodic standalone full backups. Each backup is self-contained.',
    },
    {
      label: 'Full + incremental',
      value: PhysicalDatabaseBackupType.FULL_INCREMENTAL,
      tooltip:
        'Full backups plus incremental ones that store only the changes since the previous backup. Smaller and faster.',
    },
    {
      label: 'Full + incremental + WAL',
      value: PhysicalDatabaseBackupType.FULL_INCREMENTAL_WAL_STREAM,
      tooltip:
        'Adds continuous WAL streaming on top of full and incremental backups. Only this option enables point-in-time recovery (PITR), but it requires more space and slower in restore.',
    },
  ];

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
    const sslMode = editingDatabase.postgresqlPhysical?.sslMode ?? PostgresSslMode.Disable;
    if (sslMode === PostgresSslMode.Disable) return null;

    const hadSslCert = !!database.postgresqlPhysical?.sslClientCert;
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
            value={editingDatabase.postgresqlPhysical?.sslClientCert || ''}
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
            value={editingDatabase.postgresqlPhysical?.sslClientKey || ''}
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
            value={editingDatabase.postgresqlPhysical?.sslRootCert || ''}
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

  const renderCopyableCommand = (command: string) => (
    <div className="relative mt-2">
      <pre className="rounded-md bg-gray-900 p-3 pr-10 font-mono text-xs break-all whitespace-pre-wrap text-gray-100">
        {command}
      </pre>
      <Tooltip title="Copy">
        <button
          type="button"
          className="absolute top-2 right-2 cursor-pointer rounded p-1 text-gray-400 hover:text-white"
          onClick={() => copyCommand(command)}
        >
          <CopyOutlined />
        </button>
      </Tooltip>
    </div>
  );

  const renderNoteRuns = (runs: PhysicalConnectionErrorTextRun[]) =>
    runs.map((run, runIndex) =>
      run.isBold ? (
        <strong key={runIndex}>{run.text}</strong>
      ) : (
        <span key={runIndex}>{run.text}</span>
      ),
    );

  const renderConnectionError = () => {
    if (!connectionErrorCode) return null;

    const content = physicalConnectionErrorContent[connectionErrorCode];
    const commandContext = { username: editingDatabase.postgresqlPhysical?.username ?? '' };
    const steps = content.buildSteps?.(commandContext) ?? [];
    const managedNote =
      typeof content.managedNote === 'function'
        ? content.managedNote(commandContext)
        : content.managedNote;

    return (
      <Alert
        type="error"
        className="mt-3"
        message={<span className="text-sm font-bold">{content.title}</span>}
        description={
          <div>
            <div>{content.summary}</div>
            {steps.map((step, index) =>
              step.type === 'command' ? (
                <div key={index}>{renderCopyableCommand(step.command)}</div>
              ) : (
                <p key={index} className="mt-2 text-sm">
                  {renderNoteRuns(step.runs)}
                </p>
              ),
            )}
            {managedNote && <div className="mt-2 text-sm">{managedNote}</div>}
          </div>
        }
      />
    );
  };

  const renderForm = () => {
    let isAllFieldsFilled = true;
    if (!editingDatabase.postgresqlPhysical?.host) isAllFieldsFilled = false;
    if (!editingDatabase.postgresqlPhysical?.port) isAllFieldsFilled = false;
    if (!editingDatabase.postgresqlPhysical?.username) isAllFieldsFilled = false;
    if (!editingDatabase.id && !editingDatabase.postgresqlPhysical?.password)
      isAllFieldsFilled = false;

    return (
      <>
        <div className="mb-3 flex w-full items-start">
          <div className="min-w-[150px]">Backup type</div>
          <Radio.Group
            value={
              editingDatabase.postgresqlPhysical?.backupType ?? PhysicalDatabaseBackupType.FULL
            }
            onChange={(e) => updateBackupType(e.target.value)}
          >
            <Space direction="vertical" size={0}>
              {backupTypeOptions.map((option) => (
                <Radio key={option.value} value={option.value} className="my-1! leading-[17px]!">
                  {option.label}
                  <Tooltip className="cursor-pointer" title={option.tooltip}>
                    <InfoCircleOutlined className="ml-2" style={{ color: 'gray' }} />
                  </Tooltip>
                </Radio>
              ))}
            </Space>
          </Radio.Group>
        </div>

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
            value={editingDatabase.postgresqlPhysical?.host}
            onChange={(e) => {
              if (!editingDatabase.postgresqlPhysical) return;

              const rawHost = e.target.value;
              const basePostgresql = {
                ...editingDatabase.postgresqlPhysical,
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

              setEditingDatabase({
                ...editingDatabase,
                postgresqlPhysical:
                  derivedSslMode !== null
                    ? applySslMode(basePostgresql, derivedSslMode)
                    : basePostgresql,
              });
              invalidateConnectionTest();
            }}
            size="small"
            className="max-w-[200px] grow"
            placeholder="Enter PG host"
          />
        </div>

        <div className="mb-1 flex w-full items-center">
          <div className="min-w-[150px]">Port</div>
          <InputNumber
            type="number"
            value={editingDatabase.postgresqlPhysical?.port}
            onChange={(e) => {
              if (!editingDatabase.postgresqlPhysical || e === null) return;

              setEditingDatabase({
                ...editingDatabase,
                postgresqlPhysical: { ...editingDatabase.postgresqlPhysical, port: e },
              });
              invalidateConnectionTest();
            }}
            size="small"
            className="max-w-[200px] grow"
            placeholder="Enter PG port"
          />
        </div>

        <div className="mb-1 flex w-full items-center">
          <div className="min-w-[150px]">Username</div>
          <Input
            value={editingDatabase.postgresqlPhysical?.username}
            onChange={(e) => {
              if (!editingDatabase.postgresqlPhysical) return;

              setEditingDatabase({
                ...editingDatabase,
                postgresqlPhysical: {
                  ...editingDatabase.postgresqlPhysical,
                  username: e.target.value.trim(),
                },
              });
              invalidateConnectionTest();
            }}
            size="small"
            className="max-w-[200px] grow"
            placeholder="Enter PG username"
          />
        </div>

        <div className="mb-1 flex w-full items-center">
          <div className="min-w-[150px]">Password</div>
          <Input.Password
            value={editingDatabase.postgresqlPhysical?.password}
            onChange={(e) => {
              if (!editingDatabase.postgresqlPhysical) return;

              setEditingDatabase({
                ...editingDatabase,
                postgresqlPhysical: {
                  ...editingDatabase.postgresqlPhysical,
                  password: e.target.value,
                },
              });
              invalidateConnectionTest();
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

        <div className="mb-1 flex w-full items-center">
          <div className="min-w-[150px]">SSL mode</div>
          <Select
            value={editingDatabase.postgresqlPhysical?.sslMode ?? PostgresSslMode.Disable}
            onChange={(value: PostgresSslMode) => {
              if (!editingDatabase.postgresqlPhysical) return;

              setHasUserChosenSslMode(true);
              setEditingDatabase({
                ...editingDatabase,
                postgresqlPhysical: applySslMode(editingDatabase.postgresqlPhysical, value),
              });
              invalidateConnectionTest();
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

        {renderSslCertSection()}

        {renderConnectionError()}

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
      </>
    );
  };

  return (
    <div>
      {renderForm()}

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
