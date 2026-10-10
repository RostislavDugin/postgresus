import { CopyOutlined, InfoCircleOutlined } from '@ant-design/icons';
import { App, Button, Input, InputNumber, Select, Switch, Tooltip } from 'antd';
import { useEffect, useState } from 'react';
import { Trans, useTranslation } from 'react-i18next';

import {
  DATABASE_TYPE_LABEL_KEYS,
  type Database,
  DatabaseType,
  databaseApi,
  disableSrvWhenTunneled,
  hasStoredSshTunnelSecretsForAuthType,
  isSshTunnelReadyToTest,
} from '../../../../entity/databases';
import { MongodbConnectionStringParser } from '../../../../entity/databases/model/mongodb/MongodbConnectionStringParser';
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
}

export const EditMongoDbSpecificDataComponent = ({
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
  const { t } = useTranslation();
  const { locale } = useLocale();
  const { message } = App.useApp();

  const [editingDatabase, setEditingDatabase] = useState<Database>();
  const [isSaving, setIsSaving] = useState(false);

  const [isConnectionTested, setIsConnectionTested] = useState(false);
  const [isTestingConnection, setIsTestingConnection] = useState(false);
  const [isConnectionFailed, setIsConnectionFailed] = useState(false);

  const hasAdvancedValues =
    !!database.mongodb?.authDatabase ||
    !!database.mongodb?.isSrv ||
    !!database.mongodb?.isDirectConnection ||
    !!database.mongodb?.excludeCollections?.length ||
    !!database.mongodb?.sshTunnel?.isEnabled;
  const [isShowAdvanced, setShowAdvanced] = useState(hasAdvancedValues);

  const [isShowPasteModal, setIsShowPasteModal] = useState(false);

  const applyConnectionString = (text: string) => {
    const trimmedText = text.trim();

    if (!trimmedText) {
      message.error(t('databases.edit.clipboardEmpty'));
      return;
    }

    const result = MongodbConnectionStringParser.parse(trimmedText);

    if ('error' in result) {
      message.error(translateLocalizedText(result.error, t));
      return;
    }

    if (!editingDatabase?.mongodb) return;

    const updatedDatabase: Database = {
      ...editingDatabase,
      mongodb: disableSrvWhenTunneled({
        ...editingDatabase.mongodb,
        host: result.host,
        port: result.port,
        username: result.username,
        password: result.password || '',
        database: result.database,
        authDatabase: result.authDatabase,
        isHttps: result.useTls,
        isSrv: result.isSrv,
        isDirectConnection: result.isDirectConnection,
        cpuCount: 1,
      }),
    };

    if (result.isSrv || result.isDirectConnection) {
      setShowAdvanced(true);
    }

    setEditingDatabase(updatedDatabase);
    setIsConnectionTested(false);

    if (!result.password) {
      message.warning(t('databases.edit.connectionStringParsedWithoutPassword'));
    } else {
      message.success(t('databases.edit.connectionStringParsed'));
    }
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

  const testConnection = async () => {
    if (!editingDatabase?.mongodb) return;
    setIsTestingConnection(true);
    setIsConnectionFailed(false);

    const trimmedDatabase = {
      ...editingDatabase,
      mongodb: {
        ...editingDatabase.mongodb,
        password: editingDatabase.mongodb.password?.trim(),
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
    if (!editingDatabase?.mongodb) return;

    const trimmedDatabase = {
      ...editingDatabase,
      mongodb: {
        ...editingDatabase.mongodb,
        password: editingDatabase.mongodb.password?.trim(),
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

  useEffect(() => {
    setIsSaving(false);
    setIsConnectionTested(false);
    setIsTestingConnection(false);
    setIsConnectionFailed(false);

    setEditingDatabase({ ...database });
  }, [database]);

  if (!editingDatabase) return null;

  const engineName = t(DATABASE_TYPE_LABEL_KEYS[DatabaseType.MONGODB]);

  const isSrvConnection = editingDatabase.mongodb?.isSrv || false;

  const hasStoredSshSecrets = hasStoredSshTunnelSecretsForAuthType(
    database.mongodb?.sshTunnel,
    editingDatabase.mongodb?.sshTunnel?.authType,
    database.id,
  );

  let isAllFieldsFilled = true;
  if (!editingDatabase.mongodb?.host) isAllFieldsFilled = false;
  if (!isSrvConnection && !editingDatabase.mongodb?.port) isAllFieldsFilled = false;
  if (!editingDatabase.mongodb?.username) isAllFieldsFilled = false;
  if (!editingDatabase.id && !editingDatabase.mongodb?.password) isAllFieldsFilled = false;
  if (!editingDatabase.mongodb?.database) isAllFieldsFilled = false;

  if (!isSshTunnelReadyToTest(editingDatabase.mongodb?.sshTunnel, hasStoredSshSecrets))
    isAllFieldsFilled = false;

  // Behind a bastion a loopback address names the database as the bastion sees it, so the hint
  // about reaching a local database would be wrong.
  const isTunnelEnabled = !!editingDatabase.mongodb?.sshTunnel?.isEnabled;
  const isLocalhostDb =
    !isTunnelEnabled &&
    (editingDatabase.mongodb?.host?.includes('localhost') ||
      editingDatabase.mongodb?.host?.includes('127.0.0.1'));

  return (
    <div>
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
          value={editingDatabase.mongodb?.host}
          onChange={(e) => {
            if (!editingDatabase.mongodb) return;

            setEditingDatabase({
              ...editingDatabase,
              mongodb: {
                ...editingDatabase.mongodb,
                host: e.target.value.trim().replace('https://', '').replace('http://', ''),
              },
            });
            setIsConnectionTested(false);
          }}
          size="small"
          className="max-w-[200px] grow"
          placeholder={t('databases.edit.placeholders.host', { engine: engineName })}
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

      {!isSrvConnection && (
        <div className="mb-1 flex w-full items-center">
          <div className="min-w-[150px] pr-2">{t('common.fields.port')}</div>
          <InputNumber
            type="number"
            value={editingDatabase.mongodb?.port}
            onChange={(e) => {
              if (!editingDatabase.mongodb || e === null) return;

              setEditingDatabase({
                ...editingDatabase,
                mongodb: { ...editingDatabase.mongodb, port: e },
              });
              setIsConnectionTested(false);
            }}
            size="small"
            className="max-w-[200px] grow"
            placeholder="27017"
          />
        </div>
      )}

      <div className="mb-1 flex w-full items-center">
        <div className="min-w-[150px] pr-2">{t('common.fields.username')}</div>
        <Input
          value={editingDatabase.mongodb?.username}
          onChange={(e) => {
            if (!editingDatabase.mongodb) return;

            setEditingDatabase({
              ...editingDatabase,
              mongodb: { ...editingDatabase.mongodb, username: e.target.value.trim() },
            });
            setIsConnectionTested(false);
          }}
          size="small"
          className="max-w-[200px] grow"
          placeholder={t('databases.edit.placeholders.username', { engine: engineName })}
        />
      </div>

      <div className="mb-1 flex w-full items-center">
        <div className="min-w-[150px] pr-2">{t('common.fields.password')}</div>
        <Input.Password
          value={editingDatabase.mongodb?.password}
          onChange={(e) => {
            if (!editingDatabase.mongodb) return;

            setEditingDatabase({
              ...editingDatabase,
              mongodb: { ...editingDatabase.mongodb, password: e.target.value },
            });
            setIsConnectionTested(false);
          }}
          size="small"
          className="max-w-[200px] grow"
          placeholder={t('databases.edit.placeholders.password', { engine: engineName })}
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
            value={editingDatabase.mongodb?.database}
            onChange={(e) => {
              if (!editingDatabase.mongodb) return;

              setEditingDatabase({
                ...editingDatabase,
                mongodb: { ...editingDatabase.mongodb, database: e.target.value.trim() },
              });
              setIsConnectionTested(false);
            }}
            size="small"
            className="max-w-[200px] grow"
            placeholder={t('databases.edit.placeholders.databaseName', { engine: engineName })}
          />
        </div>
      )}

      <div className="mb-1 flex w-full items-center">
        <div className="min-w-[150px] pr-2">{t('databases.fields.useHttps')}</div>
        <Switch
          checked={editingDatabase.mongodb?.isHttps}
          onChange={(checked) => {
            if (!editingDatabase.mongodb) return;

            setEditingDatabase({
              ...editingDatabase,
              mongodb: { ...editingDatabase.mongodb, isHttps: checked },
            });
            setIsConnectionTested(false);
          }}
          size="small"
        />
      </div>

      <div className="mb-5 flex w-full items-center">
        <div className="min-w-[150px] pr-2">{t('databases.fields.cpuCount')}</div>
        <div className="flex items-center">
          <InputNumber
            min={1}
            max={16}
            value={editingDatabase.mongodb?.cpuCount}
            onChange={(value) => {
              if (!editingDatabase.mongodb) return;

              setEditingDatabase({
                ...editingDatabase,
                mongodb: { ...editingDatabase.mongodb, cpuCount: value || 1 },
              });
              setIsConnectionTested(false);
            }}
            size="small"
            className="max-w-[200px] grow"
          />

          <Tooltip className="cursor-pointer" title={t('databases.edit.cpuCountTooltip')}>
            <InfoCircleOutlined className="ml-2" style={{ color: 'gray' }} />
          </Tooltip>
        </div>
      </div>

      <AdvancedSettingsToggleComponent
        label={t('databases.edit.advancedSettings')}
        isShowAdvanced={isShowAdvanced}
        onToggle={() => setShowAdvanced(!isShowAdvanced)}
      />

      {isShowAdvanced && (
        <>
          <EditSshTunnelComponent
            sshTunnel={editingDatabase.mongodb?.sshTunnel}
            hasStoredSecrets={hasStoredSshSecrets}
            onChange={(sshTunnel) => {
              if (!editingDatabase.mongodb) return;

              setEditingDatabase({
                ...editingDatabase,
                mongodb: disableSrvWhenTunneled({ ...editingDatabase.mongodb, sshTunnel }),
              });
              setIsConnectionTested(false);
            }}
          />

          <div className="mb-1 flex w-full items-center">
            <div className="min-w-[150px] pr-2">{t('databases.fields.useSrvConnection')}</div>
            <div className="flex items-center">
              <Switch
                checked={editingDatabase.mongodb?.isSrv || false}
                disabled={isTunnelEnabled}
                onChange={(checked) => {
                  if (!editingDatabase.mongodb) return;

                  setEditingDatabase({
                    ...editingDatabase,
                    mongodb: { ...editingDatabase.mongodb, isSrv: checked },
                  });
                  setIsConnectionTested(false);
                }}
                size="small"
              />
              <Tooltip
                className="cursor-pointer"
                title={
                  isTunnelEnabled
                    ? t('databases.edit.mongodb.srvOverTunnelTooltip')
                    : t('databases.edit.mongodb.srvTooltip')
                }
              >
                <InfoCircleOutlined className="ml-2" style={{ color: 'gray' }} />
              </Tooltip>
            </div>
          </div>

          <div className="mb-1 flex w-full items-center">
            <div className="min-w-[150px] pr-2">{t('databases.fields.directConnection')}</div>
            <div className="flex items-center">
              {/* Shown as on but never written: the backend forces it per connection, so the
                  stored value still applies once the tunnel is switched off. */}
              <Switch
                checked={isTunnelEnabled || editingDatabase.mongodb?.isDirectConnection || false}
                disabled={isTunnelEnabled}
                onChange={(checked) => {
                  if (!editingDatabase.mongodb) return;

                  setEditingDatabase({
                    ...editingDatabase,
                    mongodb: { ...editingDatabase.mongodb, isDirectConnection: checked },
                  });
                  setIsConnectionTested(false);
                }}
                size="small"
              />
              <Tooltip
                className="cursor-pointer"
                title={
                  isTunnelEnabled
                    ? t('databases.edit.mongodb.directConnectionOverTunnelTooltip')
                    : t('databases.edit.mongodb.directConnectionTooltip')
                }
              >
                <InfoCircleOutlined className="ml-2" style={{ color: 'gray' }} />
              </Tooltip>
            </div>
          </div>

          <div className="mb-1 flex w-full items-center">
            <div className="min-w-[150px] pr-2">{t('databases.fields.authDatabase')}</div>
            <Input
              value={editingDatabase.mongodb?.authDatabase}
              onChange={(e) => {
                if (!editingDatabase.mongodb) return;

                setEditingDatabase({
                  ...editingDatabase,
                  mongodb: { ...editingDatabase.mongodb, authDatabase: e.target.value.trim() },
                });
                setIsConnectionTested(false);
              }}
              size="small"
              className="max-w-[200px] grow"
              // eslint-disable-next-line i18next/no-literal-string -- MongoDB's default authentication database
              placeholder="admin"
            />
          </div>

          <div className="mb-1 flex w-full items-center">
            <div className="min-w-[150px] pr-2">{t('databases.fields.excludeCollections')}</div>
            <Select
              mode="tags"
              value={editingDatabase.mongodb?.excludeCollections || []}
              onChange={(values) => {
                if (!editingDatabase.mongodb) return;

                setEditingDatabase({
                  ...editingDatabase,
                  mongodb: {
                    ...editingDatabase.mongodb,
                    excludeCollections: normalizeNameList(values),
                  },
                });
              }}
              size="small"
              className="max-w-[200px] grow"
              placeholder={t('databases.edit.mongodb.excludeCollectionsPlaceholder')}
              tokenSeparators={NAME_LIST_TOKEN_SEPARATORS}
            />

            <Tooltip
              className="cursor-pointer"
              title={t('databases.edit.mongodb.excludeCollectionsTooltip')}
            >
              <InfoCircleOutlined className="ml-2" style={{ color: 'gray' }} />
            </Tooltip>
          </div>
        </>
      )}

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
      </div>

      {isConnectionFailed && (
        <div className="mt-3 text-sm text-gray-500 dark:text-gray-400">
          {t('databases.edit.ipWhitelistHint')}
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
