import { CopyOutlined, InfoCircleOutlined } from '@ant-design/icons';
import { App, Button, Input, InputNumber, Switch, Tooltip } from 'antd';
import { useEffect, useState } from 'react';

import { IS_CLOUD } from '../../../../constants';
import { type Database, databaseApi } from '../../../../entity/databases';
import { ClickhouseConnectionStringParser } from '../../../../entity/databases/model/clickhouse/ClickhouseConnectionStringParser';
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

export const EditClickhouseSpecificDataComponent = ({
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

  const [isShowPasteModal, setIsShowPasteModal] = useState(false);

  const applyConnectionString = (text: string) => {
    const trimmedText = text.trim();

    if (!trimmedText) {
      message.error('Clipboard is empty');
      return;
    }

    const result = ClickhouseConnectionStringParser.parse(trimmedText);

    if ('error' in result) {
      message.error(result.error);
      return;
    }

    if (!editingDatabase?.clickhouse) return;

    const updatedDatabase: Database = {
      ...editingDatabase,
      clickhouse: {
        ...editingDatabase.clickhouse,
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
    if (!editingDatabase?.clickhouse) return;
    setIsTestingConnection(true);
    setIsConnectionFailed(false);

    const trimmedDatabase = {
      ...editingDatabase,
      clickhouse: {
        ...editingDatabase.clickhouse,
        password: editingDatabase.clickhouse.password?.trim(),
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
    if (!editingDatabase?.clickhouse) return;

    const trimmedDatabase = {
      ...editingDatabase,
      clickhouse: {
        ...editingDatabase.clickhouse,
        password: editingDatabase.clickhouse.password?.trim(),
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
  if (!editingDatabase.clickhouse?.host) isAllFieldsFilled = false;
  if (!editingDatabase.clickhouse?.port) isAllFieldsFilled = false;
  if (!editingDatabase.clickhouse?.username) isAllFieldsFilled = false;
  if (!editingDatabase.clickhouse?.password) isAllFieldsFilled = false;
  if (!editingDatabase.clickhouse?.database) isAllFieldsFilled = false;

  const isLocalhostDb =
    editingDatabase.clickhouse?.host?.includes('localhost') ||
    editingDatabase.clickhouse?.host?.includes('127.0.0.1');

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
          value={editingDatabase.clickhouse?.host}
          onChange={(e) => {
            if (!editingDatabase.clickhouse) return;
            setEditingDatabase({
              ...editingDatabase,
              clickhouse: {
                ...editingDatabase.clickhouse,
                host: e.target.value.trim().replace('https://', '').replace('http://', ''),
              },
            });
            setIsConnectionTested(false);
          }}
          size="small"
          className="max-w-[200px] grow"
          placeholder="Enter ClickHouse host"
        />
      </div>

      {isLocalhostDb && !IS_CLOUD && (
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
          value={editingDatabase.clickhouse?.port}
          onChange={(e) => {
            if (!editingDatabase.clickhouse || e === null) return;
            setEditingDatabase({
              ...editingDatabase,
              clickhouse: { ...editingDatabase.clickhouse, port: e },
            });
            setIsConnectionTested(false);
          }}
          size="small"
          className="max-w-[200px] grow"
          placeholder="9000 for native, 9440 with TLS"
        />
      </div>

      <div className="mb-1 flex w-full items-center">
        <div className="min-w-[150px]">Username</div>
        <Input
          value={editingDatabase.clickhouse?.username}
          onChange={(e) => {
            if (!editingDatabase.clickhouse) return;
            setEditingDatabase({
              ...editingDatabase,
              clickhouse: { ...editingDatabase.clickhouse, username: e.target.value.trim() },
            });
            setIsConnectionTested(false);
          }}
          size="small"
          className="max-w-[200px] grow"
          placeholder="default"
        />
      </div>

      <div className="mb-1 flex w-full items-center">
        <div className="min-w-[150px]">Password</div>
        <Input.Password
          value={editingDatabase.clickhouse?.password}
          onChange={(e) => {
            if (!editingDatabase.clickhouse) return;
            setEditingDatabase({
              ...editingDatabase,
              clickhouse: { ...editingDatabase.clickhouse, password: e.target.value },
            });
            setIsConnectionTested(false);
          }}
          size="small"
          className="max-w-[200px] grow"
          placeholder="Password"
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
            value={editingDatabase.clickhouse?.database}
            onChange={(e) => {
              if (!editingDatabase.clickhouse) return;
              setEditingDatabase({
                ...editingDatabase,
                clickhouse: { ...editingDatabase.clickhouse, database: e.target.value.trim() },
              });
              setIsConnectionTested(false);
            }}
            size="small"
            className="max-w-[200px] grow"
            placeholder="Enter ClickHouse database name"
          />
        </div>
      )}

      <div className="mb-3 flex w-full items-center">
        <div className="min-w-[150px]">
          Use TLS{' '}
          <Tooltip title="Native protocol over TLS (port 9440). HTTP/HTTPS support is deferred.">
            <InfoCircleOutlined className="ml-1 text-gray-400" />
          </Tooltip>
        </div>
        <Switch
          checked={editingDatabase.clickhouse?.isHttps}
          onChange={(checked) => {
            if (!editingDatabase.clickhouse) return;
            setEditingDatabase({
              ...editingDatabase,
              clickhouse: { ...editingDatabase.clickhouse, isHttps: checked },
            });
            setIsConnectionTested(false);
          }}
          size="small"
        />
      </div>

      {editingDatabase.clickhouse?.isHttps && (
        <div className="mb-3 flex w-full items-center">
          <div className="min-w-[150px]">
            Strict TLS verify{' '}
            <Tooltip title="Reject self-signed and untrusted certificates. Defaults off (matches PostgreSQL sslmode=require).">
              <InfoCircleOutlined className="ml-1 text-gray-400" />
            </Tooltip>
          </div>
          <Switch
            checked={editingDatabase.clickhouse?.isStrictTls ?? false}
            onChange={(checked) => {
              if (!editingDatabase.clickhouse) return;
              setEditingDatabase({
                ...editingDatabase,
                clickhouse: { ...editingDatabase.clickhouse, isStrictTls: checked },
              });
              setIsConnectionTested(false);
            }}
            size="small"
          />
        </div>
      )}

      {isRestoreMode && (
        <>
          <div className="mb-1 flex w-full items-center">
            <div className="min-w-[150px]">Drop existing</div>
            <Switch
              checked={editingDatabase.clickhouse?.isDropExisting ?? false}
              onChange={(checked) => {
                if (!editingDatabase.clickhouse) return;
                setEditingDatabase({
                  ...editingDatabase,
                  clickhouse: { ...editingDatabase.clickhouse, isDropExisting: checked },
                });
              }}
              size="small"
            />
            <Tooltip title="Drop the target database before restore. Otherwise restore fails fast if the target already has tables.">
              <InfoCircleOutlined className="ml-2 text-gray-400" />
            </Tooltip>
          </div>

          <div className="mb-3 flex w-full items-center">
            <div className="min-w-[150px]">Keep Replicated*</div>
            <Switch
              checked={editingDatabase.clickhouse?.isKeepReplicatedDDL ?? false}
              onChange={(checked) => {
                if (!editingDatabase.clickhouse) return;
                setEditingDatabase({
                  ...editingDatabase,
                  clickhouse: { ...editingDatabase.clickhouse, isKeepReplicatedDDL: checked },
                });
              }}
              size="small"
            />
            <Tooltip title="Preserve Replicated*MergeTree / Shared*MergeTree engine names on restore. Default rewrites them to plain *MergeTree (no Keeper coordinates).">
              <InfoCircleOutlined className="ml-2 text-gray-400" />
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

      {isConnectionFailed && !IS_CLOUD && (
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
