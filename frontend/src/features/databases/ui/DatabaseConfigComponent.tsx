import {
  ArrowRightOutlined,
  CloseOutlined,
  CopyOutlined,
  DeleteOutlined,
  InfoCircleOutlined,
} from '@ant-design/icons';
import { Button, Input } from 'antd';
import { useEffect, useState } from 'react';

import { logicalBackupConfigApi } from '../../../entity/backups/logical';
import { physicalBackupConfigApi } from '../../../entity/backups/physical';
import { type Database, DatabaseType, databaseApi } from '../../../entity/databases';
import { ToastHelper } from '../../../shared/toast';
import { ConfirmationComponent } from '../../../shared/ui';
import {
  EditLogicalBackupConfigComponent,
  ShowLogicalBackupConfigComponent,
} from '../../backups/logical';
import {
  EditPhysicalBackupConfigComponent,
  ShowPhysicalBackupConfigComponent,
} from '../../backups/physical';
import { EditHealthcheckConfigComponent, ShowHealthcheckConfigComponent } from '../../healthcheck';
import {
  EditBackupVerificationConfigComponent,
  ShowBackupVerificationConfigComponent,
} from '../../verification/config';
import { DatabaseTransferDialogComponent } from './DatabaseTransferDialogComponent';
import { EditDatabaseNotifiersComponent } from './edit/EditDatabaseNotifiersComponent';
import { EditDatabaseSpecificDataComponent } from './edit/EditDatabaseSpecificDataComponent';
import { ShowDatabaseNotifiersComponent } from './show/ShowDatabaseNotifiersComponent';
import { ShowDatabaseSpecificDataComponent } from './show/ShowDatabaseSpecificDataComponent';

interface Props {
  database: Database;
  setDatabase: (database?: Database | undefined) => void;
  onDatabaseChanged: (database: Database) => void;
  onDatabaseDeleted: () => void;
  editDatabase: Database | undefined;
  setEditDatabase: (database: Database | undefined) => void;

  isCanManageDBs: boolean;
}

export const DatabaseConfigComponent = ({
  database,
  setDatabase,
  onDatabaseChanged,
  onDatabaseDeleted,
  editDatabase,
  setEditDatabase,
  isCanManageDBs,
}: Props) => {
  const [isEditName, setIsEditName] = useState(false);
  const [isEditDatabaseSpecificDataSettings, setIsEditDatabaseSpecificDataSettings] =
    useState(false);
  const [isEditBackupConfig, setIsEditBackupConfig] = useState(false);
  const [isEditNotifiersSettings, setIsEditNotifiersSettings] = useState(false);
  const [isEditHealthcheckSettings, setIsEditHealthcheckSettings] = useState(false);
  const [isEditVerificationConfig, setIsEditVerificationConfig] = useState(false);

  const [isNameUnsaved, setIsNameUnsaved] = useState(false);
  const [isSaving, setIsSaving] = useState(false);

  const [isTestingConnection, setIsTestingConnection] = useState(false);
  const [isCopying, setIsCopying] = useState(false);
  const [isShowRemoveConfirm, setIsShowRemoveConfirm] = useState(false);
  const [isRemoving, setIsRemoving] = useState(false);
  const [isShowCopyConfirm, setIsShowCopyConfirm] = useState(false);
  const [isShowTransferDialog, setIsShowTransferDialog] = useState(false);
  const [currentStorageId, setCurrentStorageId] = useState<string | undefined>();

  const isPhysicalDatabase = database.type === DatabaseType.POSTGRES_PHYSICAL;

  useEffect(() => {
    const loadStorageId = isPhysicalDatabase
      ? physicalBackupConfigApi.getPhysicalBackupConfigByDbId(database.id)
      : logicalBackupConfigApi.getBackupConfigByDbID(database.id);

    loadStorageId.then((config) => {
      setCurrentStorageId(config.storage?.id);
    });
  }, [database.id, isPhysicalDatabase]);

  const loadSettings = () => {
    setDatabase(undefined);
    setEditDatabase(undefined);
    databaseApi.getDatabase(database.id).then(setDatabase);
  };

  const copyDatabase = () => {
    if (!database) return;

    setIsCopying(true);
    setIsShowCopyConfirm(false);

    databaseApi
      .copyDatabase(database.id)
      .then((copiedDatabase) => {
        ToastHelper.showToast({
          title: 'Database copied successfully!',
          description: `"${copiedDatabase.name}" has been created successfully`,
        });
        window.location.reload();
      })
      .catch((e: Error) => {
        alert(e.message);
      })
      .finally(() => {
        setIsCopying(false);
      });
  };

  const testConnection = () => {
    if (!database) return;

    setIsTestingConnection(true);
    databaseApi
      .testDatabaseConnection(database.id)
      .then(() => {
        ToastHelper.showToast({
          title: 'Connection test successful!',
          description: 'Database connection tested successfully',
        });

        if (database.lastBackupErrorMessage) {
          setDatabase({ ...database, lastBackupErrorMessage: undefined });
          onDatabaseChanged(database);
        }
      })
      .catch((e: Error) => {
        alert(e.message);
      })
      .finally(() => {
        setIsTestingConnection(false);
      });
  };

  const remove = () => {
    if (!database) return;

    setIsShowRemoveConfirm(false);
    setIsRemoving(true);
    databaseApi
      .deleteDatabase(database.id)
      .then(() => {
        onDatabaseDeleted();
      })
      .catch((e: Error) => {
        alert(e.message);
      })
      .finally(() => {
        setIsRemoving(false);
      });
  };

  const startEdit = (
    type:
      | 'name'
      | 'database'
      | 'backup-config'
      | 'notifiers'
      | 'healthcheck'
      | 'verification-config',
  ) => {
    setEditDatabase(JSON.parse(JSON.stringify(database)));
    setIsEditName(type === 'name');
    setIsEditDatabaseSpecificDataSettings(type === 'database');
    setIsEditBackupConfig(type === 'backup-config');
    setIsEditNotifiersSettings(type === 'notifiers');
    setIsEditHealthcheckSettings(type === 'healthcheck');
    setIsEditVerificationConfig(type === 'verification-config');
    setIsNameUnsaved(false);
  };

  const saveName = () => {
    if (!editDatabase) return;

    setIsSaving(true);
    databaseApi
      .updateDatabase(editDatabase)
      .then(() => {
        setDatabase(editDatabase);
        setIsSaving(false);
        setIsNameUnsaved(false);
        setIsEditName(false);
        onDatabaseChanged(editDatabase);
      })
      .catch((e: Error) => {
        alert(e.message);
        setIsSaving(false);
      });
  };

  const isLogicalDatabase = database.type === DatabaseType.POSTGRES_LOGICAL;

  return (
    <div className="relative w-full rounded-tr-md rounded-br-md rounded-bl-md bg-white p-3 shadow sm:p-5 dark:bg-gray-800">
      {isRemoving && (
        <div className="absolute inset-0 z-10 flex items-center justify-center rounded-tr-md rounded-br-md rounded-bl-md bg-white/80 dark:bg-gray-800/80">
          <div className="flex flex-col items-center gap-3">
            <div className="h-8 w-8 animate-spin rounded-full border-4 border-gray-300 border-t-blue-500" />
            <span className="text-sm font-medium text-gray-600 dark:text-gray-300">
              Removing database...
            </span>
          </div>
        </div>
      )}

      {!isEditName ? (
        <div className="mb-5 flex items-center text-xl font-bold sm:text-2xl">
          {database.name}

          {isCanManageDBs && (
            <div className="ml-2 cursor-pointer" onClick={() => startEdit('name')}>
              <img src="/icons/pen-gray.svg" />
            </div>
          )}
        </div>
      ) : (
        <div>
          <div className="flex items-center">
            <Input
              className="max-w-full sm:max-w-[250px]"
              value={editDatabase?.name}
              onChange={(e) => {
                if (!editDatabase) return;

                setEditDatabase({ ...editDatabase, name: e.target.value });
                setIsNameUnsaved(true);
              }}
              placeholder="Enter name..."
              size="large"
            />

            <div className="ml-1 flex flex-shrink-0 items-center">
              <Button
                type="text"
                className="flex h-6 w-6 items-center justify-center p-0"
                onClick={() => {
                  setIsEditName(false);
                  setIsNameUnsaved(false);
                  setEditDatabase(undefined);
                }}
              >
                <CloseOutlined className="text-gray-500 dark:text-gray-400" />
              </Button>
            </div>
          </div>

          {isNameUnsaved && (
            <Button
              className="mt-1"
              type="primary"
              onClick={() => saveName()}
              loading={isSaving}
              disabled={!editDatabase?.name}
            >
              Save
            </Button>
          )}
        </div>
      )}

      {database.lastBackupErrorMessage && (
        <div className="mb-4 max-w-full rounded border border-red-600 px-3 py-3 sm:max-w-[400px]">
          <div className="mt-1 flex items-center text-sm font-bold text-red-600">
            <InfoCircleOutlined className="mr-2" style={{ color: 'red' }} />
            Last backup error
          </div>

          <div className="mt-3 text-sm break-words whitespace-pre-wrap">
            The error:
            <br />
            {database.lastBackupErrorMessage}
          </div>

          <div className="mt-3 text-sm text-gray-500 dark:text-gray-400">
            To clean this error (choose any):
            <ul>
              <li>- test connection via button below (even if you updated settings);</li>
              <li>- wait until the next backup is done without errors;</li>
            </ul>
          </div>
        </div>
      )}

      <div className="flex flex-col gap-6 lg:flex-row lg:flex-wrap lg:gap-10">
        <div className="w-full lg:w-[400px]">
          <div className="mt-5 flex items-center font-bold">
            <div>Database settings</div>

            {!isEditDatabaseSpecificDataSettings && isCanManageDBs ? (
              <div className="ml-2 h-4 w-4 cursor-pointer" onClick={() => startEdit('database')}>
                <img src="/icons/pen-gray.svg" />
              </div>
            ) : (
              <div />
            )}
          </div>

          <div className="mt-1 text-sm">
            {isEditDatabaseSpecificDataSettings ? (
              <EditDatabaseSpecificDataComponent
                database={database}
                isShowCancelButton
                isShowBackButton={false}
                onBack={() => {}}
                onCancel={() => {
                  setIsEditDatabaseSpecificDataSettings(false);
                  loadSettings();
                }}
                isSaveToApi={true}
                onSaved={onDatabaseChanged}
              />
            ) : (
              <ShowDatabaseSpecificDataComponent database={database} />
            )}
          </div>
        </div>

        <div className="w-full lg:w-[400px]">
          <div className="mt-5 flex items-center font-bold">
            <div>Backup config</div>

            {!isEditBackupConfig && isCanManageDBs ? (
              <div
                className="ml-2 h-4 w-4 cursor-pointer"
                onClick={() => startEdit('backup-config')}
              >
                <img src="/icons/pen-gray.svg" />
              </div>
            ) : (
              <div />
            )}
          </div>

          <div>
            <div className="mt-1 text-sm">
              {isEditBackupConfig ? (
                isPhysicalDatabase ? (
                  <EditPhysicalBackupConfigComponent
                    database={database}
                    isShowCancelButton
                    onCancel={() => {
                      setIsEditBackupConfig(false);
                      loadSettings();
                    }}
                    isSaveToApi={true}
                    onSaved={() => onDatabaseChanged(database)}
                    isShowBackButton={false}
                    onBack={() => {}}
                  />
                ) : (
                  <EditLogicalBackupConfigComponent
                    database={database}
                    isShowCancelButton
                    onCancel={() => {
                      setIsEditBackupConfig(false);
                      loadSettings();
                    }}
                    isSaveToApi={true}
                    onSaved={() => onDatabaseChanged(database)}
                    isShowBackButton={false}
                    onBack={() => {}}
                  />
                )
              ) : isPhysicalDatabase ? (
                <ShowPhysicalBackupConfigComponent database={database} />
              ) : (
                <ShowLogicalBackupConfigComponent database={database} />
              )}
            </div>
          </div>
        </div>
      </div>

      <div className="flex flex-col gap-6 lg:flex-row lg:flex-wrap lg:gap-10">
        <div className="w-full lg:w-[400px]">
          <div className="mt-5 flex items-center font-bold">
            <div>Healthcheck settings</div>

            {!isEditHealthcheckSettings && isCanManageDBs ? (
              <div className="ml-2 h-4 w-4 cursor-pointer" onClick={() => startEdit('healthcheck')}>
                <img src="/icons/pen-gray.svg" />
              </div>
            ) : (
              <div />
            )}
          </div>

          <div className="mt-1 text-sm">
            {isEditHealthcheckSettings ? (
              <EditHealthcheckConfigComponent
                databaseId={database.id}
                onClose={() => {
                  setIsEditHealthcheckSettings(false);
                  loadSettings();
                }}
              />
            ) : (
              <ShowHealthcheckConfigComponent databaseId={database.id} />
            )}
          </div>
        </div>

        {isLogicalDatabase && (
          <div className="w-full lg:w-[400px]">
            <div className="mt-5 flex items-center font-bold">
              <div>Restore verification</div>

              {!isEditVerificationConfig && isCanManageDBs ? (
                <div
                  className="ml-2 h-4 w-4 cursor-pointer"
                  onClick={() => startEdit('verification-config')}
                >
                  <img src="/icons/pen-gray.svg" />
                </div>
              ) : (
                <div />
              )}
            </div>

            <div className="mt-1 text-sm">
              {isEditVerificationConfig ? (
                <EditBackupVerificationConfigComponent
                  databaseId={database.id}
                  onClose={() => {
                    setIsEditVerificationConfig(false);
                    loadSettings();
                  }}
                />
              ) : (
                <ShowBackupVerificationConfigComponent databaseId={database.id} />
              )}
            </div>
          </div>
        )}

        <div className="w-full lg:w-[400px]">
          <div className="mt-5 flex items-center font-bold">
            <div>Notifiers settings</div>

            {!isEditNotifiersSettings && isCanManageDBs ? (
              <div className="ml-2 h-4 w-4 cursor-pointer" onClick={() => startEdit('notifiers')}>
                <img src="/icons/pen-gray.svg" />
              </div>
            ) : (
              <div />
            )}
          </div>

          <div className="mt-1 text-sm">
            {isEditNotifiersSettings ? (
              <EditDatabaseNotifiersComponent
                workspaceId={database.workspaceId}
                database={database}
                isShowCancelButton
                isShowBackButton={false}
                isShowSaveOnlyForUnsaved={true}
                onBack={() => {}}
                onCancel={() => {
                  setIsEditNotifiersSettings(false);
                  loadSettings();
                }}
                isSaveToApi={true}
                saveButtonText="Save"
                onSaved={onDatabaseChanged}
              />
            ) : (
              <ShowDatabaseNotifiersComponent database={database} />
            )}
          </div>
        </div>
      </div>

      {!isEditDatabaseSpecificDataSettings && (
        <div className="mt-10 flex flex-col gap-2 sm:flex-row sm:gap-0">
          <Button
            type="primary"
            className="w-full sm:mr-1 sm:w-auto"
            onClick={testConnection}
            loading={isTestingConnection}
            disabled={isTestingConnection}
          >
            Test connection
          </Button>

          {isCanManageDBs && (
            <>
              <Button
                type="primary"
                ghost
                icon={<ArrowRightOutlined />}
                onClick={() => setIsShowTransferDialog(true)}
                className="sm:mr-1"
              />

              <Button
                type="primary"
                ghost
                icon={<CopyOutlined />}
                onClick={() => setIsShowCopyConfirm(true)}
                loading={isCopying}
                disabled={isCopying}
                className="sm:mr-1"
              />

              <Button
                type="primary"
                ghost
                danger
                icon={<DeleteOutlined />}
                onClick={() => setIsShowRemoveConfirm(true)}
                loading={isRemoving}
                disabled={isRemoving}
              />
            </>
          )}
        </div>
      )}

      {isShowCopyConfirm && (
        <ConfirmationComponent
          onConfirm={copyDatabase}
          onDecline={() => setIsShowCopyConfirm(false)}
          description="Are you sure you want to copy this database? A new database with the same settings will be created."
          actionText="Copy"
          actionButtonColor="blue"
        />
      )}

      {isShowRemoveConfirm && (
        <ConfirmationComponent
          onConfirm={remove}
          onDecline={() => setIsShowRemoveConfirm(false)}
          description="Are you sure you want to remove this database? This action cannot be undone."
          actionText="Remove"
          actionButtonColor="red"
        />
      )}

      {isShowTransferDialog && (
        <DatabaseTransferDialogComponent
          database={database}
          currentStorageId={currentStorageId}
          onClose={() => setIsShowTransferDialog(false)}
          onTransferred={() => {
            setIsShowTransferDialog(false);
            window.location.reload();
          }}
        />
      )}
    </div>
  );
};
