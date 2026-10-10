import {
  ArrowRightOutlined,
  CloseOutlined,
  DeleteOutlined,
  InfoCircleOutlined,
} from '@ant-design/icons';
import { Button, Input, Spin } from 'antd';
import { useState } from 'react';
import { useEffect } from 'react';
import { useTranslation } from 'react-i18next';

import { logicalBackupConfigApi } from '../../../entity/backups/logical';
import { storageApi } from '../../../entity/storages';
import type { Storage } from '../../../entity/storages';
import { translateApiError } from '../../../shared/i18n';
import { ToastHelper } from '../../../shared/toast';
import { ConfirmationComponent } from '../../../shared/ui';
import { EditStorageComponent } from './edit/EditStorageComponent';
import { ShowStorageComponent } from './show/ShowStorageComponent';
import { StorageTransferDialogComponent } from './StorageTransferDialogComponent';

interface Props {
  storageId: string;
  onStorageChanged: (storage: Storage) => void;
  onStorageDeleted: () => void;
  onStorageTransferred: () => void;
  isCanManageStorages: boolean;
}

export const StorageComponent = ({
  storageId,
  onStorageChanged,
  onStorageDeleted,
  onStorageTransferred,
  isCanManageStorages,
}: Props) => {
  const { t } = useTranslation();
  const [storage, setStorage] = useState<Storage | undefined>();

  const [isEditName, setIsEditName] = useState(false);
  const [isEditSettings, setIsEditSettings] = useState(false);

  const [editStorage, setEditStorage] = useState<Storage | undefined>();
  const [isNameUnsaved, setIsNameUnsaved] = useState(false);
  const [isSaving, setIsSaving] = useState(false);

  const [isTestingConnection, setIsTestingConnection] = useState(false);

  const [isShowRemoveConfirm, setIsShowRemoveConfirm] = useState(false);
  const [isRemoving, setIsRemoving] = useState(false);

  const [isShowTransferDialog, setIsShowTransferDialog] = useState(false);

  const testConnection = () => {
    if (!storage) return;

    setIsTestingConnection(true);
    storageApi
      .testStorageConnection(storage.id)
      .then(() => {
        ToastHelper.showToast({
          title: t('storages.connectionTestSucceeded.title'),
          description: t('storages.connectionTestSucceeded.description'),
        });

        if (storage.lastSaveError) {
          setStorage({ ...storage, lastSaveError: undefined });
          onStorageChanged(storage);
        }
      })
      .catch((e: unknown) => {
        alert(translateApiError(e, t));
      })
      .finally(() => {
        setIsTestingConnection(false);
      });
  };

  const remove = async () => {
    if (!storage) return;

    setIsRemoving(true);

    try {
      const isStorageUsing = await logicalBackupConfigApi.isStorageUsing(storage.id);
      if (isStorageUsing) {
        alert(t('storages.config.removeBlockedByDatabases'));
        setIsShowRemoveConfirm(false);
      } else {
        await storageApi.deleteStorage(storage.id);
        onStorageDeleted();
      }
    } catch (e) {
      alert(translateApiError(e, t));
    }

    setIsRemoving(false);
  };

  const startEdit = (type: 'name' | 'settings') => {
    setEditStorage(JSON.parse(JSON.stringify(storage)));
    setIsEditName(type === 'name');
    setIsEditSettings(type === 'settings');
    setIsNameUnsaved(false);
  };

  const saveName = () => {
    if (!editStorage) return;

    setIsSaving(true);
    storageApi
      .saveStorage(editStorage)
      .then(() => {
        setStorage(editStorage);
        setIsSaving(false);
        setIsNameUnsaved(false);
        setIsEditName(false);
        onStorageChanged(editStorage);
      })
      .catch((e: unknown) => {
        alert(translateApiError(e, t));
        setIsSaving(false);
      });
  };

  const loadSettings = () => {
    setStorage(undefined);
    setEditStorage(undefined);
    storageApi.getStorage(storageId).then(setStorage);
  };

  useEffect(() => {
    loadSettings();
  }, [storageId]);

  return (
    <div className="w-full">
      <div className="grow overflow-y-auto rounded bg-white p-5 shadow dark:bg-gray-800">
        {!storage ? (
          <div className="mt-10 flex justify-center">
            <Spin />
          </div>
        ) : (
          <div>
            {!isEditName ? (
              <div className="mb-5 flex items-center text-2xl font-bold">
                {storage.name}
                {isCanManageStorages && (
                  <div className="ml-2 cursor-pointer" onClick={() => startEdit('name')}>
                    <img src="/icons/pen-gray.svg" />
                  </div>
                )}
              </div>
            ) : (
              <div>
                <div className="flex items-center">
                  <Input
                    className="max-w-[250px]"
                    value={editStorage?.name}
                    onChange={(e) => {
                      if (!editStorage) return;

                      setEditStorage({ ...editStorage, name: e.target.value });
                      setIsNameUnsaved(true);
                    }}
                    placeholder={t('storages.config.namePlaceholder')}
                    size="large"
                  />

                  <div className="ml-1 flex items-center">
                    <Button
                      type="text"
                      className="flex h-6 w-6 items-center justify-center p-0"
                      onClick={() => {
                        setIsEditName(false);
                        setIsNameUnsaved(false);
                        setEditStorage(undefined);
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
                    disabled={!editStorage?.name}
                  >
                    {t('common.actions.save')}
                  </Button>
                )}
              </div>
            )}

            {storage.lastSaveError && (
              <div className="max-w-[400px] rounded border border-red-600 px-3 py-3">
                <div className="mt-1 flex items-center text-sm font-bold text-red-600">
                  <InfoCircleOutlined className="mr-2" style={{ color: 'red' }} />
                  {t('storages.config.lastSaveError.title')}
                </div>

                <div className="mt-3 text-sm">
                  {t('storages.config.lastSaveError.errorLabel')}
                  <br />
                  {storage.lastSaveError}
                </div>

                <div className="mt-3 text-sm break-words whitespace-pre-wrap text-gray-500 dark:text-gray-400">
                  {t('storages.config.lastSaveError.clearHint')}
                  <ul>
                    <li>{t('storages.config.lastSaveError.clearByTestingConnection')}</li>
                    <li>{t('storages.config.lastSaveError.clearByNextSave')}</li>
                  </ul>
                </div>
              </div>
            )}

            <div className="mt-5 flex items-center font-bold">
              <div>{t('storages.config.settingsTitle')}</div>

              {!isEditSettings && isCanManageStorages ? (
                <div className="ml-2 h-4 w-4 cursor-pointer" onClick={() => startEdit('settings')}>
                  <img src="/icons/pen-gray.svg" />
                </div>
              ) : (
                <div />
              )}
            </div>

            <div className="mt-1 text-sm">
              {isEditSettings && isCanManageStorages ? (
                <EditStorageComponent
                  workspaceId={storage.workspaceId}
                  isShowClose
                  onClose={() => {
                    setIsEditSettings(false);
                    setEditStorage(undefined);
                    loadSettings();
                  }}
                  isShowName={false}
                  editingStorage={storage}
                  onChanged={onStorageChanged}
                />
              ) : (
                <ShowStorageComponent storage={storage} />
              )}
            </div>

            {!isEditSettings && (
              <div className="mt-5">
                <Button
                  type="primary"
                  className="mr-1"
                  onClick={testConnection}
                  loading={isTestingConnection}
                  disabled={isTestingConnection}
                >
                  {t('storages.actions.testConnection')}
                </Button>

                {isCanManageStorages && (
                  <>
                    <Button
                      type="primary"
                      ghost
                      icon={<ArrowRightOutlined />}
                      onClick={() => setIsShowTransferDialog(true)}
                      className="mr-1"
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
          </div>
        )}

        {isShowRemoveConfirm && (
          <ConfirmationComponent
            onConfirm={remove}
            onDecline={() => setIsShowRemoveConfirm(false)}
            description={t('storages.config.removeConfirmation')}
            actionText={t('common.actions.remove')}
            actionButtonColor="red"
          />
        )}
      </div>

      {isShowTransferDialog && storage && (
        <StorageTransferDialogComponent
          storage={storage}
          onClose={() => setIsShowTransferDialog(false)}
          onTransferred={() => {
            setIsShowTransferDialog(false);
            onStorageTransferred();
          }}
        />
      )}
    </div>
  );
};
