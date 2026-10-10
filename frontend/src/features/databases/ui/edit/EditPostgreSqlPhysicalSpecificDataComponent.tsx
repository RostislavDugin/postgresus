import { CopyOutlined, InfoCircleOutlined } from '@ant-design/icons';
import { Alert, App, Button, Input, InputNumber, Radio, Select, Space, Tooltip } from 'antd';
import { useEffect, useState } from 'react';
import { useTranslation } from 'react-i18next';

import {
  ConnectionErrorCode,
  type Database,
  PHYSICAL_DATABASE_BACKUP_TYPE_LABEL_KEYS,
  POSTGRESQL_SHORT_NAME,
  POSTGRES_SSL_MODE_LABEL_KEYS,
  PhysicalDatabaseBackupType,
  PostgresSslMode,
  type PostgresqlPhysicalDatabase,
  databaseApi,
  hasStoredSshTunnelSecretsForAuthType,
  isSshTunnelReadyToTest,
  physicalConnectionErrorContent,
} from '../../../../entity/databases';
import { ConnectionStringParser } from '../../../../entity/databases/model/postgresql/ConnectionStringParser';
import { ApiError } from '../../../../shared/api';
import {
  TransByKey,
  type TranslationKey,
  translateApiError,
  translateLocalizedText,
} from '../../../../shared/i18n';
import { ClipboardHelper } from '../../../../shared/lib/ClipboardHelper';
import { ToastHelper } from '../../../../shared/toast';
import {
  AdvancedSettingsToggleComponent,
  ClipboardPasteModalComponent,
} from '../../../../shared/ui';
import { EditSshTunnelComponent } from './EditSshTunnelComponent';

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

const BACKUP_TYPE_TOOLTIP_KEYS: Record<PhysicalDatabaseBackupType, TranslationKey> = {
  [PhysicalDatabaseBackupType.FULL]: 'databases.edit.postgresql.backupTypeTooltips.full',
  [PhysicalDatabaseBackupType.FULL_INCREMENTAL]:
    'databases.edit.postgresql.backupTypeTooltips.fullIncremental',
  [PhysicalDatabaseBackupType.FULL_INCREMENTAL_WAL_STREAM]:
    'databases.edit.postgresql.backupTypeTooltips.fullIncrementalWalStream',
};

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
  const { t } = useTranslation();
  const { message } = App.useApp();

  const [editingDatabase, setEditingDatabase] = useState<Database>();
  const [isSaving, setIsSaving] = useState(false);

  const [isConnectionTested, setIsConnectionTested] = useState(false);
  const [isTestingConnection, setIsTestingConnection] = useState(false);

  const [hasUserChosenSslMode, setHasUserChosenSslMode] = useState(!!database.id);
  const [isReplacingCerts, setIsReplacingCerts] = useState(false);

  const [isShowAdvanced, setShowAdvanced] = useState(
    !!database.postgresqlPhysical?.sshTunnel?.isEnabled,
  );

  const [isShowPasteModal, setIsShowPasteModal] = useState(false);

  const [connectionErrorCode, setConnectionErrorCode] = useState<ConnectionErrorCode | null>(null);

  const invalidateConnectionTest = () => {
    setIsConnectionTested(false);
    setConnectionErrorCode(null);
  };

  const copyCommand = async (command: string) => {
    try {
      await ClipboardHelper.copyToClipboard(command);
      message.success(t('common.messages.copiedToClipboard'));
    } catch {
      message.error(t('databases.edit.copyFailed'));
    }
  };

  const applyConnectionString = (text: string) => {
    const trimmedText = text.trim();

    if (!trimmedText) {
      message.error(t('databases.edit.clipboardEmpty'));
      return;
    }

    const result = ConnectionStringParser.parse(trimmedText);

    if ('error' in result) {
      message.error(translateLocalizedText(result.error, t));
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
    message.success(t('databases.edit.connectionStringParsed'));
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
      message.error(t('databases.edit.clipboardReadFailed'));
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
        title: t('databases.edit.connectionTestPassed.title'),
        description: t('databases.edit.connectionTestPassed.description'),
      });
    } catch (e) {
      if (e instanceof ApiError && e.code && e.code in physicalConnectionErrorContent) {
        setConnectionErrorCode(e.code as ConnectionErrorCode);
      } else {
        message.error(translateApiError(e, t));
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
        alert(translateApiError(e, t));
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

  const renderFooter = (footerContent?: React.ReactNode) => (
    <div className="mt-5 flex">
      {isShowCancelButton && (
        <Button className="mr-1" danger ghost onClick={() => onCancel()}>
          {t('common.actions.cancel')}
        </Button>
      )}

      {isShowBackButton && (
        <Button className="mr-auto" type="primary" ghost onClick={() => onBack()}>
          {t('common.actions.back')}
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
          <div className="min-w-[150px] pr-2">{t('databases.fields.clientCertificate')}</div>
          <div className="flex items-center">
            <span className="mr-3">*************</span>
            <Button size="small" onClick={startReplacingCerts}>
              {t('databases.actions.replace')}
            </Button>
          </div>
        </div>
      );
    }

    return (
      <>
        <div className="mb-1 flex w-full items-start">
          <div className="min-w-[150px] pr-2">{t('databases.fields.clientCertificate')}</div>
          <Input.TextArea
            value={editingDatabase.postgresqlPhysical?.sslClientCert || ''}
            onChange={(e) => updatePostgresqlCert('sslClientCert', e.target.value)}
            size="small"
            className="max-w-[300px] grow"
            // eslint-disable-next-line i18next/no-literal-string -- PEM format example, not copy
            placeholder="-----BEGIN CERTIFICATE-----"
            autoSize={{ minRows: 2, maxRows: 5 }}
          />
        </div>

        <div className="mb-1 flex w-full items-start">
          <div className="min-w-[150px] pr-2">{t('databases.fields.clientKey')}</div>
          <Input.TextArea
            value={editingDatabase.postgresqlPhysical?.sslClientKey || ''}
            onChange={(e) => updatePostgresqlCert('sslClientKey', e.target.value)}
            size="small"
            className="max-w-[300px] grow"
            // eslint-disable-next-line i18next/no-literal-string -- PEM format example, not copy
            placeholder="-----BEGIN PRIVATE KEY-----"
            autoSize={{ minRows: 2, maxRows: 5 }}
          />
        </div>

        <div className="mb-1 flex w-full items-start">
          <div className="flex min-w-[150px] items-center pr-2">
            <span>{t('databases.fields.serverCaCertificate')}</span>
            <Tooltip
              className="cursor-pointer"
              title={t('databases.edit.serverCaCertificateTooltip')}
            >
              <InfoCircleOutlined className="ml-2" style={{ color: 'gray' }} />
            </Tooltip>
          </div>
          <Input.TextArea
            value={editingDatabase.postgresqlPhysical?.sslRootCert || ''}
            onChange={(e) => updatePostgresqlCert('sslRootCert', e.target.value)}
            size="small"
            className="max-w-[300px] grow"
            // eslint-disable-next-line i18next/no-literal-string -- PEM format example, not copy
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
      <Tooltip title={t('common.actions.copy')}>
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

  const renderConnectionError = () => {
    if (!connectionErrorCode) return null;

    const content = physicalConnectionErrorContent[connectionErrorCode];
    const commandContext = { username: editingDatabase.postgresqlPhysical?.username ?? '' };
    const steps = content.buildSteps?.(commandContext) ?? [];
    const managedNote = content.buildManagedNote?.(commandContext);

    return (
      <Alert
        type="error"
        className="mt-3"
        message={<span className="text-sm font-bold">{t(content.titleKey)}</span>}
        description={
          <div>
            <div>{t(content.summaryKey)}</div>
            {steps.map((step, index) =>
              step.type === 'command' ? (
                <div key={index}>{renderCopyableCommand(step.command)}</div>
              ) : (
                <p key={index} className="mt-2 text-sm">
                  <TransByKey i18nKey={step.key} components={{ bold: <strong /> }} />
                </p>
              ),
            )}
            {managedNote && (
              <div className="mt-2 text-sm">{translateLocalizedText(managedNote, t)}</div>
            )}
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
    if (!isSshTunnelReadyToTest(editingDatabase.postgresqlPhysical?.sshTunnel, hasStoredSshSecrets))
      isAllFieldsFilled = false;

    return (
      <>
        <div className="mb-3 flex w-full items-start">
          <div className="min-w-[150px] pr-2">{t('databases.fields.backupType')}</div>
          <Radio.Group
            value={
              editingDatabase.postgresqlPhysical?.backupType ?? PhysicalDatabaseBackupType.FULL
            }
            onChange={(e) => updateBackupType(e.target.value)}
          >
            <Space direction="vertical" size={0}>
              {Object.values(PhysicalDatabaseBackupType).map((backupType) => (
                <Radio key={backupType} value={backupType} className="my-1! leading-[17px]!">
                  {t(PHYSICAL_DATABASE_BACKUP_TYPE_LABEL_KEYS[backupType])}
                  <Tooltip
                    className="cursor-pointer"
                    title={t(BACKUP_TYPE_TOOLTIP_KEYS[backupType])}
                  >
                    <InfoCircleOutlined className="ml-2" style={{ color: 'gray' }} />
                  </Tooltip>
                </Radio>
              ))}
            </Space>
          </Radio.Group>
        </div>

        <div className="mb-3 flex">
          <div className="min-w-[150px] pr-2" />
          <div
            className="cursor-pointer text-sm text-gray-600 transition-colors hover:text-gray-900 dark:text-gray-400 dark:hover:text-gray-200"
            onClick={parseFromClipboard}
          >
            <CopyOutlined className="mr-1" />
            {t('databases.edit.parseFromClipboard')}
          </div>
        </div>

        <div className="mb-1 flex w-full items-center">
          <div className="min-w-[150px] pr-2">{t('common.fields.host')}</div>
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
            placeholder={t('databases.edit.placeholders.host', { engine: POSTGRESQL_SHORT_NAME })}
          />
        </div>

        <div className="mb-1 flex w-full items-center">
          <div className="min-w-[150px] pr-2">{t('common.fields.port')}</div>
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
            placeholder={t('databases.edit.placeholders.port', { engine: POSTGRESQL_SHORT_NAME })}
          />
        </div>

        <div className="mb-1 flex w-full items-center">
          <div className="min-w-[150px] pr-2">{t('common.fields.username')}</div>
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
            placeholder={t('databases.edit.placeholders.username', {
              engine: POSTGRESQL_SHORT_NAME,
            })}
          />
        </div>

        <div className="mb-1 flex w-full items-center">
          <div className="min-w-[150px] pr-2">{t('common.fields.password')}</div>
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
            placeholder={t('databases.edit.placeholders.password', {
              engine: POSTGRESQL_SHORT_NAME,
            })}
            autoComplete="off"
            data-1p-ignore
            data-lpignore="true"
            data-form-type="other"
          />
        </div>

        <div className="mb-1 flex w-full items-center">
          <div className="min-w-[150px] pr-2">{t('databases.fields.sslMode')}</div>
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
            options={Object.values(PostgresSslMode).map((sslMode) => ({
              label: t(POSTGRES_SSL_MODE_LABEL_KEYS[sslMode]),
              value: sslMode,
            }))}
            size="small"
            className="max-w-[200px] grow"
          />
        </div>

        {renderSslCertSection()}

        <AdvancedSettingsToggleComponent
          label={t('databases.edit.advancedSettings')}
          isShowAdvanced={isShowAdvanced}
          onToggle={() => setShowAdvanced(!isShowAdvanced)}
        />

        {isShowAdvanced && (
          <EditSshTunnelComponent
            sshTunnel={editingDatabase.postgresqlPhysical?.sshTunnel}
            hasStoredSecrets={hasStoredSshSecrets}
            onChange={(sshTunnel) => {
              if (!editingDatabase.postgresqlPhysical) return;

              setEditingDatabase({
                ...editingDatabase,
                postgresqlPhysical: { ...editingDatabase.postgresqlPhysical, sshTunnel },
              });
              setIsConnectionTested(false);
            }}
          />
        )}

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
                {t('databases.actions.testConnection')}
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
                {saveButtonText || t('common.actions.save')}
              </Button>
            )}
          </>,
        )}
      </>
    );
  };

  const hasStoredSshSecrets = hasStoredSshTunnelSecretsForAuthType(
    database.postgresqlPhysical?.sshTunnel,
    editingDatabase.postgresqlPhysical?.sshTunnel?.authType,
    database.id,
  );

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
