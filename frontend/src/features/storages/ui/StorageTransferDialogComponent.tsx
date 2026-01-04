import { Button, Modal, Select, Spin } from 'antd';
import { useEffect, useState } from 'react';

import { backupConfigApi } from '../../../entity/backups';
import { type Storage, storageApi } from '../../../entity/storages';
import { type WorkspaceResponse, workspaceApi } from '../../../entity/workspaces';

interface Props {
  storage: Storage;
  onClose: () => void;
  onTransferred: () => void;
}

export const StorageTransferDialogComponent = ({ storage, onClose, onTransferred }: Props) => {
  const [isLoading, setIsLoading] = useState(true);
  const [isStorageInUse, setIsStorageInUse] = useState(false);
  const [workspaces, setWorkspaces] = useState<WorkspaceResponse[]>([]);
  const [selectedWorkspaceId, setSelectedWorkspaceId] = useState<string | undefined>();
  const [isTransferring, setIsTransferring] = useState(false);

  const loadData = async () => {
    setIsLoading(true);

    try {
      const isUsing = await backupConfigApi.isStorageUsing(storage.id);
      setIsStorageInUse(isUsing);

      if (!isUsing) {
        const response = await workspaceApi.getWorkspaces();
        const filteredWorkspaces = response.workspaces.filter((w) => w.id !== storage.workspaceId);
        setWorkspaces(filteredWorkspaces);
      }
    } catch (e) {
      alert((e as Error).message);
    }

    setIsLoading(false);
  };

  const transferStorage = async () => {
    if (!selectedWorkspaceId) return;

    setIsTransferring(true);

    try {
      await storageApi.transferStorage(storage.id, selectedWorkspaceId);
      onTransferred();
    } catch (e) {
      alert((e as Error).message);
    }

    setIsTransferring(false);
  };

  useEffect(() => {
    loadData();
  }, [storage.id]);

  return (
    <Modal
      title="Transfer storage to another workspace"
      footer={null}
      open={true}
      onCancel={onClose}
      maskClosable={false}
    >
      {isLoading ? (
        <div className="flex justify-center py-5">
          <Spin />
        </div>
      ) : isStorageInUse ? (
        <div className="py-3">
          <div className="text-gray-700 dark:text-gray-300">
            This storage is used by some databases. Please transfer or remove related databases
            first.
          </div>

          <div className="mt-5">
            <Button type="primary" onClick={onClose}>
              OK
            </Button>
          </div>
        </div>
      ) : (
        <div className="py-3">
          <div className="mb-3 text-gray-500 dark:text-gray-400">
            Select a workspace to transfer this storage to.
          </div>

          <div className="mb-5 flex items-center">
            <div className="min-w-[120px]">Target workspace</div>

            <Select
              value={selectedWorkspaceId}
              onChange={setSelectedWorkspaceId}
              className="min-w-[200px] grow"
              placeholder="Select workspace"
              options={workspaces.map((w) => ({ label: w.name, value: w.id }))}
            />
          </div>

          <div className="flex gap-2">
            <Button type="default" onClick={onClose}>
              Cancel
            </Button>

            <Button
              type="primary"
              onClick={transferStorage}
              loading={isTransferring}
              disabled={!selectedWorkspaceId || isTransferring}
            >
              Transfer
            </Button>
          </div>
        </div>
      )}
    </Modal>
  );
};
