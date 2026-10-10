import { CopyOutlined, InfoCircleOutlined } from '@ant-design/icons';
import { App, Button, Input, InputNumber, Select, Switch, Tooltip } from 'antd';
import { useEffect, useState } from 'react';
import { Trans, useTranslation } from 'react-i18next';

import {
  DATABASE_TYPE_LABEL_KEYS,
  type Database,
  DatabaseType,
  databaseApi,
  hasStoredSshTunnelSecretsForAuthType,
  isSshTunnelReadyToTest,
} from '../../../../entity/databases';
import { MySqlConnectionStringParser } from '../../../../entity/databases/model/mysql/MySqlConnectionStringParser';
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

export const EditMySqlSpecificDataComponent = ({
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
    !!database.mysql?.excludeTables?.length || !!database.mysql?.sshTunnel?.isEnabled;
  const [isShowAdvanced, setShowAdvanced] = useState(hasAdvancedValues);

  const [isShowPasteModal, setIsShowPasteModal] = useState(false);

  const applyConnectionString = (text: string) => {
    const trimmedText = text.trim();

    if (!trimmedText) {
      message.error(t('databases.edit.clipboardEmpty'));
      return;
    }

    const result = MySqlConnectionStringParser.parse(trimmedText);

    if ('error' in result) {
      message.error(translateLocalizedText(result.error, t));
      return;
    }

    if (!editingDatabase?.mysql) return;

    const updatedDatabase: Database = {
      ...editingDatabase,
      mysql: {
        ...editingDatabase.mysql,
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

  const testConnection = async () => {
    if (!editingDatabase?.mysql) return;
    setIsTestingConnection(true);
    setIsConnectionFailed(false);

    const trimmedDatabase = {
      ...editingDatabase,
      mysql: {
        ...editingDatabase.mysql,
        password: editingDatabase.mysql.password?.trim(),
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
    if (!editingDatabase?.mysql) return;

    const trimmedDatabase = {
      ...editingDatabase,
      mysql: {
        ...editingDatabase.mysql,
        password: editingDatabase.mysql.password?.trim(),
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

  const engineName = t(DATABASE_TYPE_LABEL_KEYS[DatabaseType.MYSQL]);

  const hasStoredSshSecrets = hasStoredSshTunnelSecretsForAuthType(
    database.mysql?.sshTunnel,
    editingDatabase.mysql?.sshTunnel?.authType,
    database.id,
  );

  let isAllFieldsFilled = true;
  if (!editingDatabase.mysql?.host) isAllFieldsFilled = false;
  if (!editingDatabase.mysql?.port) isAllFieldsFilled = false;
  if (!editingDatabase.mysql?.username) isAllFieldsFilled = false;
  if (!editingDatabase.id && !editingDatabase.mysql?.password) isAllFieldsFilled = false;
  if (!editingDatabase.mysql?.database) isAllFieldsFilled = false;
  if (!isSshTunnelReadyToTest(editingDatabase.mysql?.sshTunnel, hasStoredSshSecrets))
    isAllFieldsFilled = false;

  // Behind a bastion a loopback address names the database as the bastion sees it, so the hint
  // about reaching a local database would be wrong.
  const isTunnelEnabled = !!editingDatabase.mysql?.sshTunnel?.isEnabled;
  const isLocalhostDb =
    !isTunnelEnabled &&
    (editingDatabase.mysql?.host?.includes('localhost') ||
      editingDatabase.mysql?.host?.includes('127.0.0.1'));

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
          value={editingDatabase.mysql?.host}
          onChange={(e) => {
            if (!editingDatabase.mysql) return;

            setEditingDatabase({
              ...editingDatabase,
              mysql: {
                ...editingDatabase.mysql,
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

      <div className="mb-1 flex w-full items-center">
        <div className="min-w-[150px] pr-2">{t('common.fields.port')}</div>
        <InputNumber
          type="number"
          value={editingDatabase.mysql?.port}
          onChange={(e) => {
            if (!editingDatabase.mysql || e === null) return;

            setEditingDatabase({
              ...editingDatabase,
              mysql: { ...editingDatabase.mysql, port: e },
            });
            setIsConnectionTested(false);
          }}
          size="small"
          className="max-w-[200px] grow"
          placeholder={t('databases.edit.placeholders.port', { engine: engineName })}
        />
      </div>

      <div className="mb-1 flex w-full items-center">
        <div className="min-w-[150px] pr-2">{t('common.fields.username')}</div>
        <Input
          value={editingDatabase.mysql?.username}
          onChange={(e) => {
            if (!editingDatabase.mysql) return;

            setEditingDatabase({
              ...editingDatabase,
              mysql: { ...editingDatabase.mysql, username: e.target.value.trim() },
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
          value={editingDatabase.mysql?.password}
          onChange={(e) => {
            if (!editingDatabase.mysql) return;

            setEditingDatabase({
              ...editingDatabase,
              mysql: { ...editingDatabase.mysql, password: e.target.value },
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
            value={editingDatabase.mysql?.database}
            onChange={(e) => {
              if (!editingDatabase.mysql) return;

              setEditingDatabase({
                ...editingDatabase,
                mysql: { ...editingDatabase.mysql, database: e.target.value.trim() },
              });
              setIsConnectionTested(false);
            }}
            size="small"
            className="max-w-[200px] grow"
            placeholder={t('databases.edit.placeholders.databaseName', { engine: engineName })}
          />
        </div>
      )}

      <div className="mb-3 flex w-full items-center">
        <div className="min-w-[150px] pr-2">{t('databases.fields.useHttps')}</div>
        <Switch
          checked={editingDatabase.mysql?.isHttps}
          onChange={(checked) => {
            if (!editingDatabase.mysql) return;

            setEditingDatabase({
              ...editingDatabase,
              mysql: { ...editingDatabase.mysql, isHttps: checked },
            });
            setIsConnectionTested(false);
          }}
          size="small"
        />
      </div>

      <AdvancedSettingsToggleComponent
        label={t('databases.edit.advancedSettings')}
        isShowAdvanced={isShowAdvanced}
        onToggle={() => setShowAdvanced(!isShowAdvanced)}
      />

      {isShowAdvanced && (
        <>
          <EditSshTunnelComponent
            sshTunnel={editingDatabase.mysql?.sshTunnel}
            hasStoredSecrets={hasStoredSshSecrets}
            onChange={(sshTunnel) => {
              if (!editingDatabase.mysql) return;

              setEditingDatabase({
                ...editingDatabase,
                mysql: { ...editingDatabase.mysql, sshTunnel },
              });
              setIsConnectionTested(false);
            }}
          />

          <div className="mb-1 flex w-full items-center">
            <div className="min-w-[150px] pr-2">{t('databases.fields.excludeTables')}</div>
            <Select
              mode="tags"
              value={editingDatabase.mysql?.excludeTables || []}
              onChange={(values) => {
                if (!editingDatabase.mysql) return;

                setEditingDatabase({
                  ...editingDatabase,
                  mysql: { ...editingDatabase.mysql, excludeTables: normalizeNameList(values) },
                });
              }}
              size="small"
              className="max-w-[200px] grow"
              placeholder={t('databases.edit.excludeTablesPlaceholder')}
              tokenSeparators={NAME_LIST_TOKEN_SEPARATORS}
            />

            <Tooltip className="cursor-pointer" title={t('databases.edit.excludeTablesTooltip')}>
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
