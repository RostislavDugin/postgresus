import { Button, Modal, Spin } from 'antd';
import { useEffect, useState } from 'react';

import { storageApi } from '../../../entity/storages';
import type { Storage } from '../../../entity/storages';
import type { WorkspaceResponse } from '../../../entity/workspaces';
import { StorageCardComponent } from './StorageCardComponent';
import { StorageComponent } from './StorageComponent';
import { EditStorageComponent } from './edit/EditStorageComponent';

interface Props {
  contentHeight: number;
  workspace: WorkspaceResponse;
  isCanManageStorages: boolean;
}

export const StoragesComponent = ({ contentHeight, workspace, isCanManageStorages }: Props) => {
  const [isLoading, setIsLoading] = useState(true);
  const [storages, setStorages] = useState<Storage[]>([]);

  const [isShowAddStorage, setIsShowAddStorage] = useState(false);
  const [selectedStorageId, setSelectedStorageId] = useState<string | undefined>(undefined);

  const loadStorages = () => {
    setIsLoading(true);

    storageApi
      .getStorages(workspace.id)
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
      Add storage
    </Button>
  );

  return (
    <>
      <div className="flex grow">
        <div
          className="mx-3 w-[250px] min-w-[250px] overflow-y-auto"
          style={{ height: contentHeight }}
        >
          {storages.length >= 5 && isCanManageStorages && addStorageButton}

          {storages.map((storage) => (
            <StorageCardComponent
              key={storage.id}
              storage={storage}
              selectedStorageId={selectedStorageId}
              setSelectedStorageId={setSelectedStorageId}
            />
          ))}

          {storages.length < 5 && isCanManageStorages && addStorageButton}

          <div className="mx-3 text-center text-xs text-gray-500">
            Storage - is a place where backups will be stored (local disk, S3, etc.)
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
            isCanManageStorages={isCanManageStorages}
          />
        )}
      </div>

      {isShowAddStorage && (
        <Modal
          title="Add storage"
          footer={<div />}
          open={isShowAddStorage}
          onCancel={() => setIsShowAddStorage(false)}
        >
          <div className="my-3 max-w-[250px] text-gray-500">
            Storage - is a place where backups will be stored (local disk, S3, etc.)
          </div>

          <EditStorageComponent
            workspaceId={workspace.id}
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
