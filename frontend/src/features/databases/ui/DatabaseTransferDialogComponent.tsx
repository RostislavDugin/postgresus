import { CheckCircleOutlined, ExclamationCircleOutlined, SwapOutlined } from '@ant-design/icons';
import { Button, Modal, Radio, Select, Spin } from 'antd';
import { useEffect, useState } from 'react';

import { backupConfigApi } from '../../../entity/backups';
import type { TransferDatabaseRequest } from '../../../entity/backups';
import { type Database, databaseApi } from '../../../entity/databases';
import type { Notifier } from '../../../entity/notifiers';
import { notifierApi } from '../../../entity/notifiers';
import { type Storage, getStorageLogoFromType, storageApi } from '../../../entity/storages';
import { type WorkspaceResponse, workspaceApi } from '../../../entity/workspaces';
import { ToastHelper } from '../../../shared/toast';
import { EditNotifierComponent } from '../../notifiers/ui/edit/EditNotifierComponent';
import { EditStorageComponent } from '../../storages/ui/edit/EditStorageComponent';

interface Props {
  database: Database;
  currentStorageId?: string;
  onClose: () => void;
  onTransferred: () => void;
}

interface NotifierUsageInfo {
  notifier: Notifier;
  databaseCount: number;
  canTransfer: boolean;
}

export const DatabaseTransferDialogComponent = ({
  database,
  currentStorageId,
  onClose,
  onTransferred,
}: Props) => {
  const [isLoading, setIsLoading] = useState(true);
  const [workspaces, setWorkspaces] = useState<WorkspaceResponse[]>([]);
  const [selectedWorkspaceId, setSelectedWorkspaceId] = useState<string | undefined>();

  const [storageOption, setStorageOption] = useState<'transfer' | 'select'>('select');
  const [storageUsageCount, setStorageUsageCount] = useState<number>(0);
  const [isLoadingStorageCount, setIsLoadingStorageCount] = useState(false);
  const [targetStorages, setTargetStorages] = useState<Storage[]>([]);
  const [selectedStorageId, setSelectedStorageId] = useState<string | undefined>();
  const [isShowCreateStorage, setIsShowCreateStorage] = useState(false);
  const [storageSelectKey, setStorageSelectKey] = useState(0);

  const [notifierOption, setNotifierOption] = useState<'transfer' | 'select'>('select');
  const [notifierUsageInfo, setNotifierUsageInfo] = useState<NotifierUsageInfo[]>([]);
  const [isLoadingNotifierCounts, setIsLoadingNotifierCounts] = useState(false);
  const [targetNotifiers, setTargetNotifiers] = useState<Notifier[]>([]);
  const [selectedNotifierIds, setSelectedNotifierIds] = useState<string[]>([]);
  const [isShowCreateNotifier, setIsShowCreateNotifier] = useState(false);
  const [notifierSelectKey, setNotifierSelectKey] = useState(0);

  const [isTransferring, setIsTransferring] = useState(false);

  const hasCurrentStorage = !!currentStorageId;
  const hasCurrentNotifiers = database.notifiers && database.notifiers.length > 0;

  const loadWorkspaces = async () => {
    setIsLoading(true);

    try {
      const response = await workspaceApi.getWorkspaces();
      const filteredWorkspaces = response.workspaces.filter((w) => w.id !== database.workspaceId);
      setWorkspaces(filteredWorkspaces);
    } catch (e) {
      alert((e as Error).message);
    }

    setIsLoading(false);
  };

  const loadStorageUsageCount = async () => {
    if (!currentStorageId) return;

    setIsLoadingStorageCount(true);

    try {
      const count = await backupConfigApi.getDatabasesCountForStorage(currentStorageId);
      setStorageUsageCount(count);
    } catch (e) {
      alert((e as Error).message);
    }

    setIsLoadingStorageCount(false);
  };

  const loadNotifierUsageCounts = async () => {
    if (!database.notifiers || database.notifiers.length === 0) return;

    setIsLoadingNotifierCounts(true);

    try {
      const usageInfo: NotifierUsageInfo[] = await Promise.all(
        database.notifiers.map(async (notifier) => {
          const count = await databaseApi.getDatabasesCountForNotifier(notifier.id);
          return {
            notifier,
            databaseCount: count,
            canTransfer: count === 1,
          };
        }),
      );

      setNotifierUsageInfo(usageInfo);
    } catch (e) {
      alert((e as Error).message);
    }

    setIsLoadingNotifierCounts(false);
  };

  const loadTargetWorkspaceData = async (workspaceId: string) => {
    try {
      const [storages, notifiers] = await Promise.all([
        storageApi.getStorages(workspaceId),
        notifierApi.getNotifiers(workspaceId),
      ]);
      setTargetStorages(storages);
      setTargetNotifiers(notifiers);
    } catch (e) {
      alert((e as Error).message);
    }
  };

  const handleTransfer = async () => {
    if (!selectedWorkspaceId) return;

    const request: TransferDatabaseRequest = {
      targetWorkspaceId: selectedWorkspaceId,
      isTransferWithStorage: storageOption === 'transfer',
      isTransferWithNotifiers: notifierOption === 'transfer',
      targetStorageId: storageOption === 'select' ? selectedStorageId : undefined,
      targetNotifierIds: notifierOption === 'select' ? selectedNotifierIds : [],
    };

    setIsTransferring(true);

    try {
      await backupConfigApi.transferDatabase(database.id, request);
      ToastHelper.showToast({
        title: 'Database transferred successfully!',
        description: `"${database.name}" has been transferred to the new workspace`,
      });
      onTransferred();
    } catch (e) {
      alert((e as Error).message);
    }

    setIsTransferring(false);
  };

  useEffect(() => {
    loadWorkspaces();
    loadStorageUsageCount();
    loadNotifierUsageCounts();
  }, [database.id]);

  useEffect(() => {
    if (selectedWorkspaceId) {
      loadTargetWorkspaceData(selectedWorkspaceId);
      setSelectedStorageId(undefined);
      setSelectedNotifierIds([]);
    }
  }, [selectedWorkspaceId]);

  const canTransferWithStorage = storageUsageCount <= 1;
  const notifiersBlockingTransfer = notifierUsageInfo.filter((info) => !info.canTransfer);
  const notifiersCanTransfer = notifierUsageInfo.filter((info) => info.canTransfer);

  const isStorageValid =
    storageOption === 'transfer'
      ? canTransferWithStorage && hasCurrentStorage
      : !!selectedStorageId;

  const isFormValid = !!selectedWorkspaceId && isStorageValid;

  return (
    <Modal
      title={
        <div className="flex items-center gap-2">
          <SwapOutlined />
          Transfer database to another workspace
        </div>
      }
      footer={null}
      open={true}
      onCancel={onClose}
      maskClosable={false}
      width={550}
    >
      {isLoading ? (
        <div className="flex justify-center py-5">
          <Spin />
        </div>
      ) : (
        <div className="py-3">
          {/* Workspace Selection */}
          <div className="mb-5">
            <div className="mb-2 font-medium">Target workspace</div>
            <Select
              value={selectedWorkspaceId}
              onChange={setSelectedWorkspaceId}
              className="w-full"
              placeholder="Select workspace"
              options={workspaces.map((w) => ({ label: w.name, value: w.id }))}
            />
          </div>

          {selectedWorkspaceId && (
            <>
              {/* Storage Transfer Options */}
              <div className="mb-5">
                <div className="mb-2 font-medium">Storage</div>
                <Radio.Group
                  value={storageOption}
                  onChange={(e) => setStorageOption(e.target.value)}
                  className="flex flex-col gap-3"
                >
                  {hasCurrentStorage && (
                    <div>
                      <Radio value="transfer">
                        <span className="flex items-center gap-2">
                          Transfer with existing storage
                          {isLoadingStorageCount && <Spin size="small" />}
                        </span>
                      </Radio>
                    </div>
                  )}
                  <div>
                    <Radio value="select">Select storage from target workspace</Radio>
                  </div>
                </Radio.Group>

                {storageOption === 'transfer' && hasCurrentStorage && (
                  <div className="mt-2 ml-6">
                    {isLoadingStorageCount ? (
                      <Spin size="small" />
                    ) : !canTransferWithStorage ? (
                      <div className="flex items-center gap-2 rounded border border-red-300 bg-red-50 p-2 text-sm text-red-600 dark:border-red-600 dark:bg-red-900/20">
                        <ExclamationCircleOutlined />
                        <span>
                          This storage is used by {storageUsageCount} databases. Transfer is blocked
                          because other databases depend on it.
                        </span>
                      </div>
                    ) : (
                      <div className="flex items-center gap-2 text-sm text-green-600 dark:text-green-400">
                        <CheckCircleOutlined />
                        Storage can be transferred
                      </div>
                    )}
                  </div>
                )}

                {storageOption === 'select' && (
                  <div className="mt-2 ml-6">
                    <div className="flex items-center gap-2">
                      <Select
                        key={storageSelectKey}
                        value={selectedStorageId}
                        onChange={(storageId) => {
                          if (storageId === 'create-new-storage') {
                            setIsShowCreateStorage(true);
                            return;
                          }
                          setSelectedStorageId(storageId);
                        }}
                        className="w-full max-w-[300px]"
                        placeholder="Select storage"
                        options={[
                          ...targetStorages.map((s) => ({ label: s.name, value: s.id })),
                          { label: 'Create new storage', value: 'create-new-storage' },
                        ]}
                      />

                      {selectedStorageId &&
                        targetStorages.find((s) => s.id === selectedStorageId)?.type && (
                          <img
                            src={getStorageLogoFromType(
                              targetStorages.find((s) => s.id === selectedStorageId)!.type,
                            )}
                            alt="storageIcon"
                            className="h-4 w-4"
                          />
                        )}
                    </div>
                  </div>
                )}
              </div>

              {/* Notifier transfer options */}
              {hasCurrentNotifiers && (
                <div className="mb-5">
                  <div className="mb-2 font-medium">Notifiers (optional)</div>

                  <Radio.Group
                    value={notifierOption}
                    onChange={(e) => setNotifierOption(e.target.value)}
                    className="flex flex-col gap-3"
                  >
                    <div>
                      <Radio value="transfer">Transfer notifiers with database</Radio>
                    </div>
                    <div>
                      <Radio value="select">Select notifiers from target workspace</Radio>
                    </div>
                  </Radio.Group>

                  {notifierOption === 'transfer' && (
                    <div className="mt-2 ml-6">
                      {isLoadingNotifierCounts ? (
                        <Spin size="small" />
                      ) : (
                        <div className="space-y-1">
                          {notifiersCanTransfer.length > 0 && (
                            <div className="text-sm text-green-600 dark:text-green-400">
                              <div className="mb-1 flex items-center gap-1">
                                <CheckCircleOutlined />
                                <span>Will be transferred:</span>
                              </div>
                              <ul className="ml-5 list-disc">
                                {notifiersCanTransfer.map((info) => (
                                  <li key={info.notifier.id}>{info.notifier.name}</li>
                                ))}
                              </ul>
                            </div>
                          )}

                          {notifiersBlockingTransfer.length > 0 && (
                            <div className="mt-2 rounded border border-red-300 bg-red-50 p-2 text-sm text-red-600 dark:border-red-600 dark:bg-red-900/20 dark:text-red-600">
                              <div className="mb-1 flex items-center gap-1">
                                <ExclamationCircleOutlined />
                                <span>Will NOT be transferred (used by other databases):</span>
                              </div>
                              <ul className="ml-5 list-disc">
                                {notifiersBlockingTransfer.map((info) => (
                                  <li key={info.notifier.id}>
                                    {info.notifier.name} (used by {info.databaseCount} databases)
                                  </li>
                                ))}
                              </ul>
                            </div>
                          )}

                          {notifiersCanTransfer.length === 0 &&
                            notifiersBlockingTransfer.length > 0 && (
                              <div className="mt-1 text-sm text-gray-500 dark:text-gray-400">
                                No notifiers will be transferred. You can select notifiers from the
                                target workspace after transfer.
                              </div>
                            )}
                        </div>
                      )}
                    </div>
                  )}

                  {notifierOption === 'select' && (
                    <div className="mt-2 ml-6">
                      <Select
                        key={notifierSelectKey}
                        mode="multiple"
                        value={selectedNotifierIds}
                        onChange={(notifierIds) => {
                          if (notifierIds.includes('create-new-notifier')) {
                            setIsShowCreateNotifier(true);
                            return;
                          }
                          setSelectedNotifierIds(notifierIds);
                        }}
                        className="w-full max-w-[300px]"
                        placeholder="Select notifiers (optional)"
                        options={[
                          ...targetNotifiers.map((n) => ({ label: n.name, value: n.id })),
                          { label: 'Create new notifier', value: 'create-new-notifier' },
                        ]}
                      />
                    </div>
                  )}
                </div>
              )}
            </>
          )}

          {/* Action Buttons */}
          <div className="mt-5 flex gap-2">
            <Button type="default" onClick={onClose}>
              Cancel
            </Button>

            <Button
              type="primary"
              onClick={handleTransfer}
              loading={isTransferring}
              disabled={!isFormValid || isTransferring}
            >
              Transfer
            </Button>
          </div>
        </div>
      )}

      {/* Create Storage Modal */}
      {isShowCreateStorage && selectedWorkspaceId && (
        <Modal
          title="Add storage"
          footer={null}
          open={isShowCreateStorage}
          onCancel={() => {
            setIsShowCreateStorage(false);
            setStorageSelectKey((prev) => prev + 1);
          }}
          maskClosable={false}
        >
          <div className="my-3 max-w-[275px] text-gray-500 dark:text-gray-400">
            Storage - is a place where backups will be stored (local disk, S3, Google Drive, etc.)
          </div>

          <EditStorageComponent
            workspaceId={selectedWorkspaceId}
            isShowName
            isShowClose={false}
            onClose={() => setIsShowCreateStorage(false)}
            onChanged={(storage) => {
              loadTargetWorkspaceData(selectedWorkspaceId);
              setSelectedStorageId(storage.id);
              setIsShowCreateStorage(false);
            }}
          />
        </Modal>
      )}

      {/* Create Notifier Modal */}
      {isShowCreateNotifier && selectedWorkspaceId && (
        <Modal
          title="Add notifier"
          footer={null}
          open={isShowCreateNotifier}
          onCancel={() => {
            setIsShowCreateNotifier(false);
            setNotifierSelectKey((prev) => prev + 1);
          }}
          maskClosable={false}
        >
          <div className="my-3 max-w-[275px] text-gray-500 dark:text-gray-400">
            Notifier - is a place where notifications will be sent (email, Slack, Telegram, etc.)
          </div>

          <EditNotifierComponent
            workspaceId={selectedWorkspaceId}
            isShowName
            isShowClose={false}
            onClose={() => setIsShowCreateNotifier(false)}
            onChanged={(notifier) => {
              loadTargetWorkspaceData(selectedWorkspaceId);
              setSelectedNotifierIds([...selectedNotifierIds, notifier.id]);
              setIsShowCreateNotifier(false);
            }}
          />
        </Modal>
      )}
    </Modal>
  );
};
