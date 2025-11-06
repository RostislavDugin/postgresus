import { InfoCircleOutlined } from '@ant-design/icons';
import { Button, Input, InputNumber, Select, Switch, Tooltip } from 'antd';
import { useEffect, useState } from 'react';
import { useTranslation } from 'react-i18next';

import {
  type Database,
  DatabaseType,
  type PostgresqlDatabase,
  PostgresqlVersion,
  databaseApi,
} from '../../../../entity/databases';
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

  isShowDbVersionHint?: boolean;
  isShowDbName?: boolean;
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

  isShowDbVersionHint = true,
  isShowDbName = true,
}: Props) => {
  const { t } = useTranslation(['database', 'common']);
  const [editingDatabase, setEditingDatabase] = useState<Database>();
  const [isSaving, setIsSaving] = useState(false);

  const [isConnectionTested, setIsConnectionTested] = useState(false);
  const [isTestingConnection, setIsTestingConnection] = useState(false);

  const testConnection = async () => {
    if (!editingDatabase) return;
    setIsTestingConnection(true);

    try {
      await databaseApi.testDatabaseConnectionDirect(editingDatabase);
      setIsConnectionTested(true);
      ToastHelper.showToast({
        title: t('database:form.connection_test_passed'),
        description: t('database:form.connection_test_description'),
      });
    } catch (e) {
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

    setEditingDatabase({ ...database });
  }, [database]);

  if (!editingDatabase) return null;

  let isAllFieldsFilled = true;
  if (!editingDatabase.postgresql?.version) isAllFieldsFilled = false;
  if (!editingDatabase.postgresql?.host) isAllFieldsFilled = false;
  if (!editingDatabase.postgresql?.port) isAllFieldsFilled = false;
  if (!editingDatabase.postgresql?.username) isAllFieldsFilled = false;
  if (!editingDatabase.postgresql?.password) isAllFieldsFilled = false;
  if (!editingDatabase.postgresql?.database) isAllFieldsFilled = false;

  return (
    <div>
      <div className="mb-5 flex w-full items-center">
        <div className="min-w-[150px]">{t('database:form.type_label')}</div>
        <Select
          value={database.type}
          onChange={(v) => {
            setEditingDatabase({
              ...editingDatabase,
              type: v,
              postgresql: {} as unknown as PostgresqlDatabase,
            } as Database);

            setIsConnectionTested(false);
          }}
          disabled={!!editingDatabase.id}
          size="small"
          className="max-w-[200px] grow"
          options={[{ label: 'PostgreSQL', value: DatabaseType.POSTGRES }]}
          placeholder={t('database:form.type_placeholder')}
        />
      </div>

      {editingDatabase.type === DatabaseType.POSTGRES && (
        <>
          <div className="mb-1 flex w-full items-center">
            <div className="min-w-[150px]">{t('database:form.version_label')}</div>

            <Select
              value={editingDatabase.postgresql?.version}
              onChange={(v) => {
                if (!editingDatabase.postgresql) return;

                setEditingDatabase({
                  ...editingDatabase,
                  postgresql: {
                    ...editingDatabase.postgresql,
                    version: v as PostgresqlVersion,
                  },
                });
                setIsConnectionTested(false);
              }}
              size="small"
              className="max-w-[200px] grow"
              placeholder={t('database:form.version_placeholder')}
              options={[
                { label: '13', value: PostgresqlVersion.PostgresqlVersion13 },
                { label: '14', value: PostgresqlVersion.PostgresqlVersion14 },
                { label: '15', value: PostgresqlVersion.PostgresqlVersion15 },
                { label: '16', value: PostgresqlVersion.PostgresqlVersion16 },
                { label: '17', value: PostgresqlVersion.PostgresqlVersion17 },
                { label: '18', value: PostgresqlVersion.PostgresqlVersion18 },
              ]}
            />

            {isShowDbVersionHint && (
              <Tooltip
                className="cursor-pointer"
                title={t('database:form.version_hint')}
              >
                <InfoCircleOutlined className="ml-2" style={{ color: 'gray' }} />
              </Tooltip>
            )}
          </div>

          <div className="mb-1 flex w-full items-center">
            <div className="min-w-[150px]">{t('database:form.host_label')}</div>
            <Input
              value={editingDatabase.postgresql?.host}
              onChange={(e) => {
                if (!editingDatabase.postgresql) return;

                setEditingDatabase({
                  ...editingDatabase,
                  postgresql: {
                    ...editingDatabase.postgresql,
                    host: e.target.value.trim().replace('https://', '').replace('http://', ''),
                  },
                });
              }}
              size="small"
              className="max-w-[200px] grow"
              placeholder={t('database:form.host_placeholder')}
            />
          </div>

          <div className="mb-1 flex w-full items-center">
            <div className="min-w-[150px]">{t('database:form.port_label')}</div>
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
              placeholder={t('database:form.port_placeholder')}
            />
          </div>

          <div className="mb-1 flex w-full items-center">
            <div className="min-w-[150px]">{t('database:form.username_label')}</div>
            <Input
              value={editingDatabase.postgresql?.username}
              onChange={(e) => {
                if (!editingDatabase.postgresql) return;

                setEditingDatabase({
                  ...editingDatabase,
                  postgresql: { ...editingDatabase.postgresql, username: e.target.value.trim() },
                });
              }}
              size="small"
              className="max-w-[200px] grow"
              placeholder={t('database:form.username_placeholder')}
            />
          </div>

          <div className="mb-1 flex w-full items-center">
            <div className="min-w-[150px]">{t('database:form.password_label')}</div>
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
              placeholder={t('database:form.password_placeholder')}
            />
          </div>

          {isShowDbName && (
            <div className="mb-1 flex w-full items-center">
              <div className="min-w-[150px]">{t('database:form.database_label')}</div>
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
                placeholder={t('database:form.database_placeholder')}
              />
            </div>
          )}

          <div className="mb-3 flex w-full items-center">
            <div className="min-w-[150px]">{t('database:form.use_https_label')}</div>
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
        </>
      )}

      <div className="mt-5 flex">
        {isShowCancelButton && (
          <Button className="mr-1" danger ghost onClick={() => onCancel()}>
            {t('common:button.cancel')}
          </Button>
        )}

        {isShowBackButton && (
          <Button className="mr-auto" type="primary" ghost onClick={() => onBack()}>
            {t('common:button.back')}
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
            {t('database:form.test_connection')}
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
            {saveButtonText || t('common:button.save')}
          </Button>
        )}
      </div>
    </div>
  );
};
