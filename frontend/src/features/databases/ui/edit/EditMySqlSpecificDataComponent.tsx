import { CopyOutlined, InfoCircleOutlined } from '@ant-design/icons';
import { App, Button, Input, InputNumber, Select, Switch, Tooltip } from 'antd';
import { useEffect, useState } from 'react';

import {
  type Database,
  databaseApi,
  hasStoredSshTunnelSecretsForAuthType,
  isSshTunnelReadyToTest,
} from '../../../../entity/databases';
import { MySqlConnectionStringParser } from '../../../../entity/databases/model/mysql/MySqlConnectionStringParser';
import { NAME_LIST_TOKEN_SEPARATORS, normalizeNameList } from '../../../../shared/lib';
import { ClipboardHelper } from '../../../../shared/lib/ClipboardHelper';
import { ToastHelper } from '../../../../shared/toast';
import { ClipboardPasteModalComponent } from '../../../../shared/ui';
import { AdvancedSettingsToggleComponent } from './AdvancedSettingsToggleComponent';
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
      message.error('Clipboard is empty');
      return;
    }

    const result = MySqlConnectionStringParser.parse(trimmedText);

    if ('error' in result) {
      message.error(result.error);
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
          placeholder="Enter MySQL host"
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
          placeholder="Enter MySQL port"
        />
      </div>

      <div className="mb-1 flex w-full items-center">
        <div className="min-w-[150px]">Username</div>
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
          placeholder="Enter MySQL username"
        />
      </div>

      <div className="mb-1 flex w-full items-center">
        <div className="min-w-[150px]">Password</div>
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
          placeholder="Enter MySQL password"
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
            placeholder="Enter MySQL database name"
          />
        </div>
      )}

      <div className="mb-3 flex w-full items-center">
        <div className="min-w-[150px]">Use HTTPS</div>
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
            <div className="min-w-[150px]">Exclude tables</div>
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
              placeholder="No tables excluded"
              tokenSeparators={NAME_LIST_TOKEN_SEPARATORS}
            />

            <Tooltip
              className="cursor-pointer"
              title="Table names to exclude from the backup. You can paste a list separated by commas or new lines."
            >
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
