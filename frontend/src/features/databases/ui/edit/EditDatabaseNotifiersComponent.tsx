import { Button, Modal, Select, Spin } from 'antd';
import { useEffect, useState } from 'react';

import { type Database, databaseApi } from '../../../../entity/databases';
import { type Notifier, notifierApi } from '../../../../entity/notifiers';
import { EditNotifierComponent } from '../../../notifiers/ui/edit/EditNotifierComponent';

interface Props {
  database: Database;
  workspaceId: string;

  isShowCancelButton?: boolean;
  onCancel: () => void;

  isShowBackButton: boolean;
  onBack: () => void;

  isShowSaveOnlyForUnsaved: boolean;
  saveButtonText?: string;
  isSaveToApi: boolean;
  onSaved: (database: Database) => void;
}

export const EditDatabaseNotifiersComponent = ({
  database,
  workspaceId,

  isShowCancelButton,
  onCancel,

  isShowBackButton,
  onBack,

  isShowSaveOnlyForUnsaved,
  saveButtonText,
  isSaveToApi,
  onSaved,
}: Props) => {
  const [editingDatabase, setEditingDatabase] = useState<Database>();
  const [isUnsaved, setIsUnsaved] = useState(false);
  const [isSaving, setIsSaving] = useState(false);

  const [notifiers, setNotifiers] = useState<Notifier[]>([]);
  const [isNotifiersLoading, setIsNotifiersLoading] = useState(false);
  const [isShowCreateNotifier, setShowCreateNotifier] = useState(false);
  const [notifierSelectKey, setNotifierSelectKey] = useState(0);

  const saveDatabase = async () => {
    if (!editingDatabase) return;

    if (isSaveToApi) {
      setIsSaving(true);

      try {
        await databaseApi.updateDatabase(editingDatabase);
        setIsUnsaved(false);
      } catch (e) {
        alert((e as Error).message);
      }

      setIsSaving(false);
    }

    onSaved(editingDatabase);
  };

  const loadNotifiers = async () => {
    setIsNotifiersLoading(true);

    try {
      const notifiers = await notifierApi.getNotifiers(workspaceId);
      setNotifiers(notifiers);
    } catch (e) {
      alert((e as Error).message);
    }

    setIsNotifiersLoading(false);
  };

  useEffect(() => {
    setIsSaving(false);
    setEditingDatabase({ ...database });
    loadNotifiers();
  }, [database]);

  if (!editingDatabase) return null;

  if (isNotifiersLoading)
    return (
      <div className="mb-5 flex items-center">
        <Spin />
      </div>
    );

  return (
    <div>
      <div className="mb-5 max-w-[275px] text-gray-500 dark:text-gray-400">
        Notifier - is a place where notifications will be sent (email, Slack, Telegram, etc.)
        <br />
        <br />
        You can select several notifiers, notifications will be sent to all of them.
      </div>

      <div className="mb-5 flex w-full items-center">
        <div className="min-w-[150px]">Notifiers</div>

        <Select
          key={notifierSelectKey}
          mode="multiple"
          value={editingDatabase.notifiers.map((n) => n.id)}
          onChange={(notifiersIds) => {
            if (notifiersIds.includes('create-new-notifier')) {
              setShowCreateNotifier(true);
              return;
            }

            setEditingDatabase({
              ...editingDatabase,
              notifiers: notifiers.filter((n) => notifiersIds.includes(n.id)),
            } as unknown as Database);

            setIsUnsaved(true);
          }}
          size="small"
          className="max-w-[200px] grow"
          options={[
            ...notifiers.map((n) => ({ label: n.name, value: n.id })),
            { label: 'Create new notifier', value: 'create-new-notifier' },
          ]}
          placeholder="Select notifiers"
        />
      </div>

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

        {(!isShowSaveOnlyForUnsaved || isUnsaved) && (
          <Button
            type="primary"
            onClick={() => saveDatabase()}
            loading={isSaving}
            disabled={isSaving}
            className="mr-5"
          >
            {saveButtonText || 'Save'}
          </Button>
        )}
      </div>

      {isShowCreateNotifier && (
        <Modal
          title="Add notifier"
          footer={<div />}
          open={isShowCreateNotifier}
          onCancel={() => {
            setShowCreateNotifier(false);
            setNotifierSelectKey((prev) => prev + 1);
          }}
          maskClosable={false}
        >
          <div className="my-3 max-w-[275px] text-gray-500 dark:text-gray-400">
            Notifier - is a place where notifications will be sent (email, Slack, Telegram, etc.)
          </div>

          <EditNotifierComponent
            workspaceId={workspaceId}
            isShowName
            isShowClose={false}
            onClose={() => setShowCreateNotifier(false)}
            onChanged={() => {
              loadNotifiers();
              setShowCreateNotifier(false);
            }}
          />
        </Modal>
      )}
    </div>
  );
};
