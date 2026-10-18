import { CopyOutlined, InfoCircleOutlined } from '@ant-design/icons';
import { App, Button, Checkbox, Input, InputNumber, Select, Tooltip } from 'antd';
import { useEffect, useState } from 'react';
import { Trans, useTranslation } from 'react-i18next';

import {
  type Database,
  POSTGRESQL_SHORT_NAME,
  POSTGRES_SSL_MODE_LABEL_KEYS,
  PostgresSslMode,
  type PostgresqlLogicalDatabase,
  databaseApi,
  hasStoredSshTunnelSecretsForAuthType,
  isSshTunnelReadyToTest,
} from '../../../../entity/databases';
import { ConnectionStringParser } from '../../../../entity/databases/model/postgresql/ConnectionStringParser';
import {
  getWebsitePageUrl,
  translateApiError,
  translateLocalizedText,
  useLocale,
} from '../../../../shared/i18n';
import { NAME_LIST_TOKEN_SEPARATORS, normalizeNameList } from '../../../../shared/lib';
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
  const { t } = useTranslation();
  const { locale } = useLocale();
  const { message } = App.useApp();

  const [editingDatabase, setEditingDatabase] = useState<Database>({ ...database });
  const [isSaving, setIsSaving] = useState(false);

  const [isConnectionTested, setIsConnectionTested] = useState(false);
  const [isTestingConnection, setIsTestingConnection] = useState(false);
  const [isConnectionFailed, setIsConnectionFailed] = useState(false);

  const hasAdvancedValues =
    !!database.postgresqlLogical?.sslClientCert ||
    !!database.postgresqlLogical?.sslRootCert ||
    !!database.postgresqlLogical?.sshTunnel?.isEnabled ||
    (isRestoreMode
      ? !!database.postgresqlLogical?.isExcludeExtensions ||
        !!database.postgresqlLogical?.isRestoreOwnership ||
        !!database.postgresqlLogical?.isRestorePrivileges
      : !!database.postgresqlLogical?.includeSchemas?.length ||
        !!database.postgresqlLogical?.excludeTables?.length ||
        !!database.postgresqlLogical?.excludeTableData?.length ||
        !!database.postgresqlLogical?.isSkipUserMappings);
  const [isShowAdvanced, setShowAdvanced] = useState(hasAdvancedValues);

  const [hasAutoAddedPublicSchema, setHasAutoAddedPublicSchema] = useState(false);
  const [hasUserChosenSslMode, setHasUserChosenSslMode] = useState(!!database.id);
  const [isReplacingCerts, setIsReplacingCerts] = useState(false);

  const [isShowPasteModal, setIsShowPasteModal] = useState(false);

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
            // eslint-disable-next-line i18next/no-literal-string -- PostgreSQL schema name
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
        title: t('databases.edit.connectionTestPassed.title'),
        description: t('databases.edit.connectionTestPassed.description'),
      });
    } catch (e) {
      setIsConnectionFailed(true);
      alert(translateApiError(e, t));
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
    const sslMode = editingDatabase.postgresqlLogical?.sslMode ?? PostgresSslMode.Disable;
    if (sslMode === PostgresSslMode.Disable) return null;

    const hadSslCert = !!database.postgresqlLogical?.sslClientCert;
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
            value={editingDatabase.postgresqlLogical?.sslClientCert || ''}
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
            value={editingDatabase.postgresqlLogical?.sslClientKey || ''}
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
            value={editingDatabase.postgresqlLogical?.sslRootCert || ''}
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

  const renderPgDumpForm = () => {
    let isAllFieldsFilled = true;
    if (!editingDatabase.postgresqlLogical?.host) isAllFieldsFilled = false;
    if (!editingDatabase.postgresqlLogical?.port) isAllFieldsFilled = false;
    if (!editingDatabase.postgresqlLogical?.username) isAllFieldsFilled = false;
    if (!editingDatabase.id && !editingDatabase.postgresqlLogical?.password)
      isAllFieldsFilled = false;
    if (!editingDatabase.postgresqlLogical?.database) isAllFieldsFilled = false;
    if (!isSshTunnelReadyToTest(editingDatabase.postgresqlLogical?.sshTunnel, hasStoredSshSecrets))
      isAllFieldsFilled = false;

    const isTunnelEnabled = !!editingDatabase.postgresqlLogical?.sshTunnel?.isEnabled;

    // Behind a tunnel a loopback address means "on the bastion", which is the documented setup
    // rather than the mistake this hint warns about.
    const isLocalhostDb =
      !isTunnelEnabled &&
      (editingDatabase.postgresqlLogical?.host?.includes('localhost') ||
        editingDatabase.postgresqlLogical?.host?.includes('127.0.0.1'));

    const isSupabaseDb =
      editingDatabase.postgresqlLogical?.host?.includes('supabase') ||
      editingDatabase.postgresqlLogical?.username?.includes('supabase');

    return (
      <>
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
            placeholder={t('databases.edit.placeholders.host', { engine: POSTGRESQL_SHORT_NAME })}
          />
        </div>

        {isLocalhostDb && (
          <div className="mb-1 flex">
            <div className="min-w-[150px] pr-2" />
            <div className="max-w-[200px] text-xs text-gray-500 dark:text-gray-400">
              <Trans
                i18nKey="databases.edit.localhostHint"
                components={{
                  docsLink: (
                    <a
                      href={getWebsitePageUrl('faqLocalhost', locale)}
                      target="_blank"
                      rel="noreferrer"
                      className="!text-blue-600 dark:!text-blue-400"
                    />
                  ),
                }}
              />
            </div>
          </div>
        )}

        {isSupabaseDb && (
          <div className="mb-1 flex">
            <div className="min-w-[150px] pr-2" />
            <div className="max-w-[200px] text-xs text-gray-500 dark:text-gray-400">
              <Trans
                i18nKey="databases.edit.postgresql.supabaseHint"
                components={{
                  docsLink: (
                    <a
                      href={getWebsitePageUrl('faqSupabase', locale)}
                      target="_blank"
                      rel="noreferrer"
                      className="!text-blue-600 dark:!text-blue-400"
                    />
                  ),
                }}
              />
            </div>
          </div>
        )}

        <div className="mb-1 flex w-full items-center">
          <div className="min-w-[150px] pr-2">{t('common.fields.port')}</div>
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
            placeholder={t('databases.edit.placeholders.port', { engine: POSTGRESQL_SHORT_NAME })}
          />
        </div>

        <div className="mb-1 flex w-full items-center">
          <div className="min-w-[150px] pr-2">{t('common.fields.username')}</div>
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
            placeholder={t('databases.edit.placeholders.username', {
              engine: POSTGRESQL_SHORT_NAME,
            })}
          />
        </div>

        <div className="mb-1 flex w-full items-center">
          <div className="min-w-[150px] pr-2">{t('common.fields.password')}</div>
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
            placeholder={t('databases.edit.placeholders.password', {
              engine: POSTGRESQL_SHORT_NAME,
            })}
            autoComplete="off"
            data-1p-ignore
            data-lpignore="true"
            data-form-type="other"
          />
        </div>

        {isShowDbName && (
          <div className="mb-1 flex w-full items-center">
            <div className="min-w-[150px] pr-2">{t('databases.fields.databaseName')}</div>
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
              placeholder={t('databases.edit.placeholders.databaseName', {
                engine: POSTGRESQL_SHORT_NAME,
              })}
            />
          </div>
        )}

        <div className="mb-1 flex w-full items-center">
          <div className="min-w-[150px] pr-2">{t('databases.fields.sslMode')}</div>
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
            options={Object.values(PostgresSslMode).map((sslMode) => ({
              label: t(POSTGRES_SSL_MODE_LABEL_KEYS[sslMode]),
              value: sslMode,
            }))}
            size="small"
            className="max-w-[200px] grow"
          />
        </div>

        {isRestoreMode && (
          <div className="mb-5 flex w-full items-center">
            <div className="min-w-[150px] pr-2">{t('databases.fields.cpuCount')}</div>
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

              <Tooltip className="cursor-pointer" title={t('databases.edit.cpuCountTooltip')}>
                <InfoCircleOutlined className="ml-2" style={{ color: 'gray' }} />
              </Tooltip>
            </div>
          </div>
        )}

        <AdvancedSettingsToggleComponent
          label={t('databases.edit.advancedSettings')}
          isShowAdvanced={isShowAdvanced}
          onToggle={() => setShowAdvanced(!isShowAdvanced)}
        />

        {isShowAdvanced && (
          <>
            <EditSshTunnelComponent
              sshTunnel={editingDatabase.postgresqlLogical?.sshTunnel}
              hasStoredSecrets={hasStoredSshSecrets}
              onChange={(sshTunnel) => {
                if (!editingDatabase.postgresqlLogical) return;

                setEditingDatabase({
                  ...editingDatabase,
                  postgresqlLogical: { ...editingDatabase.postgresqlLogical, sshTunnel },
                });
                setIsConnectionTested(false);
              }}
            />

            {!isRestoreMode && (
              <div className="mb-1 flex w-full items-center">
                <div className="min-w-[150px] pr-2">{t('databases.fields.includeSchemas')}</div>
                <Select
                  mode="tags"
                  value={editingDatabase.postgresqlLogical?.includeSchemas || []}
                  onChange={(values) => {
                    if (!editingDatabase.postgresqlLogical) return;

                    setEditingDatabase({
                      ...editingDatabase,
                      postgresqlLogical: {
                        ...editingDatabase.postgresqlLogical,
                        includeSchemas: normalizeNameList(values),
                      },
                    });
                  }}
                  size="small"
                  className="max-w-[200px] grow"
                  placeholder={t('databases.edit.postgresql.includeSchemasPlaceholder')}
                  tokenSeparators={NAME_LIST_TOKEN_SEPARATORS}
                />
              </div>
            )}

            {!isRestoreMode && (
              <div className="mb-1 flex w-full items-center">
                <div className="min-w-[150px] pr-2">{t('databases.fields.excludeWholeTables')}</div>
                <Select
                  mode="tags"
                  value={editingDatabase.postgresqlLogical?.excludeTables || []}
                  onChange={(values) => {
                    if (!editingDatabase.postgresqlLogical) return;

                    setEditingDatabase({
                      ...editingDatabase,
                      postgresqlLogical: {
                        ...editingDatabase.postgresqlLogical,
                        excludeTables: normalizeNameList(values),
                      },
                    });
                  }}
                  size="small"
                  className="max-w-[200px] grow"
                  placeholder={t('databases.edit.excludeTablesPlaceholder')}
                  tokenSeparators={NAME_LIST_TOKEN_SEPARATORS}
                />

                <Tooltip
                  className="cursor-pointer"
                  title={t('databases.edit.postgresql.excludeTablesTooltip')}
                >
                  <InfoCircleOutlined className="ml-2" style={{ color: 'gray' }} />
                </Tooltip>
              </div>
            )}

            {!isRestoreMode && (
              <div className="mb-1 flex w-full items-center">
                <div className="min-w-[150px] pr-2">{t('databases.fields.excludeTableData')}</div>
                <Select
                  mode="tags"
                  value={editingDatabase.postgresqlLogical?.excludeTableData || []}
                  onChange={(values) => {
                    if (!editingDatabase.postgresqlLogical) return;

                    setEditingDatabase({
                      ...editingDatabase,
                      postgresqlLogical: {
                        ...editingDatabase.postgresqlLogical,
                        excludeTableData: normalizeNameList(values),
                      },
                    });
                  }}
                  size="small"
                  className="max-w-[200px] grow"
                  placeholder={t('databases.edit.postgresql.excludeTableDataPlaceholder')}
                  tokenSeparators={NAME_LIST_TOKEN_SEPARATORS}
                />

                <Tooltip
                  className="cursor-pointer"
                  title={t('databases.edit.postgresql.excludeTableDataTooltip')}
                >
                  <InfoCircleOutlined className="ml-2" style={{ color: 'gray' }} />
                </Tooltip>
              </div>
            )}

            {!isRestoreMode && (
              <div className="mb-1 flex w-full items-center">
                <div className="flex min-w-[150px] items-center pr-2">
                  <span>{t('databases.fields.skipUserMappings')}</span>
                  <Tooltip
                    className="cursor-pointer"
                    title={t('databases.edit.postgresql.skipUserMappingsTooltip')}
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
                <div className="flex min-w-[150px] items-center pr-2">
                  <span>{t('databases.fields.excludeExtensions')}</span>
                  <Tooltip
                    className="cursor-pointer"
                    title={t('databases.edit.postgresql.excludeExtensionsTooltip')}
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
                <div className="flex min-w-[150px] items-center pr-2">
                  <span>{t('databases.fields.restoreOwnership')}</span>
                  <Tooltip
                    className="cursor-pointer"
                    title={t('databases.edit.postgresql.restoreOwnershipTooltip')}
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
                <div className="flex min-w-[150px] items-center pr-2">
                  <span>{t('databases.fields.restorePrivileges')}</span>
                  <Tooltip
                    className="cursor-pointer"
                    title={t('databases.edit.postgresql.restorePrivilegesTooltip')}
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

        {isConnectionFailed && (
          <div className="mt-3 text-sm text-gray-500 dark:text-gray-400">
            {t('databases.edit.ipWhitelistHint')}
          </div>
        )}
      </>
    );
  };

  const hasStoredSshSecrets = hasStoredSshTunnelSecretsForAuthType(
    database.postgresqlLogical?.sshTunnel,
    editingDatabase.postgresqlLogical?.sshTunnel?.authType,
    database.id,
  );

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
