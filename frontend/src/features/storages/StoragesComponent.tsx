import { Button, Modal, Spin } from 'antd';
import { useEffect, useState } from 'react';
import { useTranslation } from 'react-i18next';

import { storageApi } from '../../entity/storages';
import type { Storage } from '../../entity/storages';
import { StorageCardComponent } from './StorageCardComponent';
import { StorageComponent } from './StorageComponent';
import { EditStorageComponent } from './ui/edit/EditStorageComponent';

interface Props {
  contentHeight: number;
}

export const StoragesComponent = ({ contentHeight }: Props) => {
  const { t } = useTranslation('storage');
  const [isLoading, setIsLoading] = useState(true);
  const [storages, setStorages] = useState<Storage[]>([]);

  const [isShowAddStorage, setIsShowAddStorage] = useState(false);
  const [selectedStorageId, setSelectedStorageId] = useState<string | undefined>(undefined);

  const loadStorages = () => {
    setIsLoading(true);

    storageApi
      .getStorages()
      .then((storages: Storage[]) => {
        setStorages(storages);
        if (!selectedStorageId) {
          setSelectedStorageId(storages[0]?.id);
        }
      })
      .catch((e: Error) => alert(e.message))
      .finally(() => setIsLoading(false));
  };

  useEffect(() => {
    loadStorages();
  }, []);

  if (isLoading) {
    return (
      <div className="mx-3 my-3 flex w-[250px] justify-center">
        <Spin />
      </div>
    );
  }

  const addStorageButton = (
    <Button type="primary" className="mb-2 w-full" onClick={() => setIsShowAddStorage(true)}>
      {t('add_button')}
    </Button>
  );

  return (
    <>
      <div className="flex grow">
        <div
          className="mx-3 w-[250px] min-w-[250px] overflow-y-auto"
          style={{ height: contentHeight }}
        >
          {storages.length >= 5 && addStorageButton}

          {storages.map((storage) => (
            <StorageCardComponent
              key={storage.id}
              storage={storage}
              selectedStorageId={selectedStorageId}
              setSelectedStorageId={setSelectedStorageId}
            />
          ))}

          {storages.length < 5 && addStorageButton}

          <div className="mx-3 text-center text-xs text-gray-500">
            {t('modal.description')}
          </div>
        </div>

        {selectedStorageId && (
          <StorageComponent
            storageId={selectedStorageId}
            onStorageChanged={() => {
              loadStorages();
            }}
            onStorageDeleted={() => {
              loadStorages();
              setSelectedStorageId(
                storages.filter((storage) => storage.id !== selectedStorageId)[0]?.id,
              );
            }}
          />
        )}
      </div>

      {isShowAddStorage && (
        <Modal
          title={t('modal.add_title')}
          footer={<div />}
          open={isShowAddStorage}
          onCancel={() => setIsShowAddStorage(false)}
        >
          <div className="my-3 max-w-[250px] text-gray-500">
            {t('modal.description')}
          </div>

          <EditStorageComponent
            isShowName
            isShowClose={false}
            onClose={() => setIsShowAddStorage(false)}
            onChanged={() => {
              loadStorages();
              setIsShowAddStorage(false);
            }}
          />
        </Modal>
      )}
    </>
  );
};
