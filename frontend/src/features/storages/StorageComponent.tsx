import { CloseOutlined, InfoCircleOutlined } from '@ant-design/icons';
import { Button, Input, Spin } from 'antd';
import { useState } from 'react';
import { useEffect } from 'react';
import { useTranslation } from 'react-i18next';

import { backupConfigApi } from '../../entity/backups';
import { storageApi } from '../../entity/storages';
import type { Storage } from '../../entity/storages';
import { ToastHelper } from '../../shared/toast';
import { ConfirmationComponent } from '../../shared/ui';
import { EditStorageComponent } from './ui/edit/EditStorageComponent';
import { ShowStorageComponent } from './ui/show/ShowStorageComponent';

interface Props {
  storageId: string;
  onStorageChanged: (storage: Storage) => void;
  onStorageDeleted: () => void;
}

export const StorageComponent = ({ storageId, onStorageChanged, onStorageDeleted }: Props) => {
  const { t } = useTranslation(['storage', 'common']);
  const [storage, setStorage] = useState<Storage | undefined>();

  const [isEditName, setIsEditName] = useState(false);
  const [isEditSettings, setIsEditSettings] = useState(false);

  const [editStorage, setEditStorage] = useState<Storage | undefined>();
  const [isNameUnsaved, setIsNameUnsaved] = useState(false);
  const [isSaving, setIsSaving] = useState(false);

  const [isTestingConnection, setIsTestingConnection] = useState(false);

  const [isShowRemoveConfirm, setIsShowRemoveConfirm] = useState(false);
  const [isRemoving, setIsRemoving] = useState(false);

  const testConnection = () => {
    if (!storage) return;

    setIsTestingConnection(true);
    storageApi
      .testStorageConnection(storage.id)
      .then(() => {
        ToastHelper.showToast({
          title: t('storage:form.connection_success_title'),
          description: t('storage:form.connection_success_description'),
        });

        if (storage.lastSaveError) {
          setStorage({ ...storage, lastSaveError: undefined });
          onStorageChanged(storage);
        }
      })
      .catch((e: Error) => {
        alert(e.message);
      })
      .finally(() => {
        setIsTestingConnection(false);
      });
  };

  const remove = async () => {
    if (!storage) return;

    setIsRemoving(true);

    try {
      const isStorageUsing = await backupConfigApi.isStorageUsing(storage.id);
      if (isStorageUsing) {
        alert(t('storage:message.storage_in_use'));
        setIsShowRemoveConfirm(false);
      } else {
        await storageApi.deleteStorage(storage.id);
        onStorageDeleted();
      }
    } catch (e) {
      alert((e as Error).message);
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
      .catch((e: Error) => {
        alert(e.message);
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
      <div className="grow overflow-y-auto rounded bg-white p-5 shadow">
        {!storage ? (
          <div className="mt-10 flex justify-center">
            <Spin />
          </div>
        ) : (
          <div>
            {!isEditName ? (
              <div className="mb-5 flex items-center text-2xl font-bold">
                {storage.name}
                <div className="ml-2 cursor-pointer" onClick={() => startEdit('name')}>
                  <img src="/icons/pen-gray.svg" />
                </div>
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
                    placeholder={t('storage:form.name_placeholder')}
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
                      <CloseOutlined className="text-gray-500" />
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
                    {t('common:button.save')}
                  </Button>
                )}
              </div>
            )}

            {storage.lastSaveError && (
              <div className="max-w-[400px] rounded border border-red-600 px-3 py-3">
                <div className="mt-1 flex items-center text-sm font-bold text-red-600">
                  <InfoCircleOutlined className="mr-2" style={{ color: 'red' }} />
                  {t('storage:details.save_error')}
                </div>

                <div className="mt-3 text-sm">
                  {t('storage:details.the_error')}
                  <br />
                  {storage.lastSaveError}
                </div>

                <div className="mt-3 text-sm text-gray-500">
                  {t('storage:details.clean_error_instruction')}
                  <ul>
                    <li>{t('storage:details.clean_error_option1')}</li>
                    <li>{t('storage:details.clean_error_option2')}</li>
                  </ul>
                </div>
              </div>
            )}

            <div className="mt-5 flex items-center font-bold">
              <div>{t('storage:details.settings')}</div>

              {!isEditSettings ? (
                <div className="ml-2 h-4 w-4 cursor-pointer" onClick={() => startEdit('settings')}>
                  <img src="/icons/pen-gray.svg" />
                </div>
              ) : (
                <div />
              )}
            </div>

            <div className="mt-1 text-sm">
              {isEditSettings ? (
                <EditStorageComponent
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
                  ghost
                  onClick={testConnection}
                  loading={isTestingConnection}
                  disabled={isTestingConnection}
                >
                  {t('storage:form.test_connection')}
                </Button>

                <Button
                  type="primary"
                  danger
                  onClick={() => setIsShowRemoveConfirm(true)}
                  ghost
                  loading={isRemoving}
                  disabled={isRemoving}
                >
                  {t('storage:action.remove')}
                </Button>
              </div>
            )}
          </div>
        )}

        {isShowRemoveConfirm && (
          <ConfirmationComponent
            onConfirm={remove}
            onDecline={() => setIsShowRemoveConfirm(false)}
            description={t('storage:message.remove_confirm_description')}
            actionText={t('storage:action.remove')}
            actionButtonColor="red"
          />
        )}
      </div>
    </div>
  );
};
