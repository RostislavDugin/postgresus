import { CloseOutlined, InfoCircleOutlined } from '@ant-design/icons';
import { Button, Input, Spin } from 'antd';
import { useState } from 'react';
import { useEffect } from 'react';

import { databaseApi } from '../../../entity/databases';
import { notifierApi } from '../../../entity/notifiers';
import type { Notifier } from '../../../entity/notifiers';
import { ToastHelper } from '../../../shared/toast';
import { ConfirmationComponent } from '../../../shared/ui';
import { EditNotifierComponent } from './edit/EditNotifierComponent';
import { ShowNotifierComponent } from './show/ShowNotifierComponent';

interface Props {
  notifierId: string;
  onNotifierChanged: (notifier: Notifier) => void;
  onNotifierDeleted: () => void;
  isCanManageNotifiers: boolean;
}

export const NotifierComponent = ({
  notifierId,
  onNotifierChanged,
  onNotifierDeleted,
  isCanManageNotifiers,
}: Props) => {
  const [notifier, setNotifier] = useState<Notifier | undefined>();

  const [isEditName, setIsEditName] = useState(false);
  const [isEditSettings, setIsEditSettings] = useState(false);

  const [editNotifier, setEditNotifier] = useState<Notifier | undefined>();
  const [isNameUnsaved, setIsNameUnsaved] = useState(false);
  const [isSaving, setIsSaving] = useState(false);

  const [isSendingTestNotification, setIsSendingTestNotification] = useState(false);

  const [isShowRemoveConfirm, setIsShowRemoveConfirm] = useState(false);
  const [isRemoving, setIsRemoving] = useState(false);

  const sendTestNotification = () => {
    if (!notifier) return;

    setIsSendingTestNotification(true);
    notifierApi
      .sendTestNotification(notifier.id)
      .then(() => {
        ToastHelper.showToast({
          title: 'Test notification sent!',
          description: 'Test notification sent successfully',
        });

        if (notifier.lastSendError) {
          setNotifier({ ...notifier, lastSendError: undefined });
          onNotifierChanged(notifier);
        }
      })
      .catch((e) => {
        alert(e.message);
      })
      .finally(() => {
        setIsSendingTestNotification(false);
      });
  };

  const remove = async () => {
    if (!notifier) return;

    setIsRemoving(true);

    try {
      const isNotifierUsing = await databaseApi.isNotifierUsing(notifier.id);
      if (isNotifierUsing) {
        alert(
          'Notifier is used by some databases. Please remove the notifier from databases first.',
        );
        setIsShowRemoveConfirm(false);
      } else {
        await notifierApi.deleteNotifier(notifier.id);
        onNotifierDeleted();
      }
    } catch (e) {
      alert((e as Error).message);
    }

    setIsRemoving(false);
  };

  const startEdit = (type: 'name' | 'settings') => {
    setEditNotifier(JSON.parse(JSON.stringify(notifier)));
    setIsEditName(type === 'name');
    setIsEditSettings(type === 'settings');
    setIsNameUnsaved(false);
  };

  const saveName = () => {
    if (!editNotifier) return;

    setIsSaving(true);
    notifierApi
      .saveNotifier(editNotifier)
      .then(() => {
        setNotifier(editNotifier);
        setIsSaving(false);
        setIsNameUnsaved(false);
        setIsEditName(false);
        onNotifierChanged(editNotifier);
      })
      .catch((e) => {
        alert(e.message);
        setIsSaving(false);
      });
  };

  const loadSettings = () => {
    setNotifier(undefined);
    setEditNotifier(undefined);
    notifierApi.getNotifier(notifierId).then(setNotifier);
  };

  useEffect(() => {
    loadSettings();
  }, [notifierId]);

  return (
    <div className="w-full">
      <div className="grow overflow-y-auto rounded bg-white p-5 shadow">
        {!notifier ? (
          <div className="mt-10 flex justify-center">
            <Spin />
          </div>
        ) : (
          <div>
            {!isEditName ? (
              <div className="mb-5 flex items-center text-2xl font-bold">
                {notifier.name}
                {isCanManageNotifiers && (
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
                    value={editNotifier?.name}
                    onChange={(e) => {
                      if (!editNotifier) return;

                      setEditNotifier({ ...editNotifier, name: e.target.value });
                      setIsNameUnsaved(true);
                    }}
                    placeholder="Enter name..."
                    size="large"
                  />

                  <div className="ml-1 flex items-center">
                    <Button
                      type="text"
                      className="flex h-6 w-6 items-center justify-center p-0"
                      onClick={() => {
                        setIsEditName(false);
                        setIsNameUnsaved(false);
                        setEditNotifier(undefined);
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
                    disabled={!editNotifier?.name}
                  >
                    Save
                  </Button>
                )}
              </div>
            )}

            {notifier.lastSendError && (
              <div className="max-w-[400px] rounded border border-red-600 px-3 py-3">
                <div className="mt-1 flex items-center text-sm font-bold text-red-600">
                  <InfoCircleOutlined className="mr-2" style={{ color: 'red' }} />
                  Send error
                </div>

                <div className="mt-3 text-sm">
                  The error:
                  <br />
                  {notifier.lastSendError}
                </div>

                <div className="mt-3 text-sm text-gray-500">
                  To clean this error (choose any):
                  <ul>
                    <li>
                      - send test notification via button below (even if you updated settings);
                    </li>
                    <li>- wait until the next notification is sent without errors;</li>
                  </ul>
                </div>
              </div>
            )}

            <div className="mt-5 flex items-center font-bold">
              <div>Notifier settings</div>

              {!isEditSettings && isCanManageNotifiers ? (
                <div className="ml-2 h-4 w-4 cursor-pointer" onClick={() => startEdit('settings')}>
                  <img src="/icons/pen-gray.svg" />
                </div>
              ) : (
                <div />
              )}
            </div>

            <div className="mt-1 text-sm">
              {isEditSettings ? (
                <EditNotifierComponent
                  workspaceId={notifier.workspaceId}
                  isShowClose
                  onClose={() => {
                    setIsEditSettings(false);
                    setEditNotifier(undefined);
                    loadSettings();
                  }}
                  isShowName={false}
                  editingNotifier={notifier}
                  onChanged={onNotifierChanged}
                />
              ) : (
                <ShowNotifierComponent notifier={notifier} />
              )}
            </div>

            {!isEditSettings && (
              <div className="mt-5">
                <Button
                  type="primary"
                  className="mr-1"
                  ghost
                  onClick={sendTestNotification}
                  loading={isSendingTestNotification}
                  disabled={isSendingTestNotification}
                >
                  Send test notification
                </Button>

                {isCanManageNotifiers && (
                  <Button
                    type="primary"
                    danger
                    onClick={() => setIsShowRemoveConfirm(true)}
                    ghost
                    loading={isRemoving}
                    disabled={isRemoving}
                  >
                    Remove
                  </Button>
                )}
              </div>
            )}
          </div>
        )}

        {isShowRemoveConfirm && (
          <ConfirmationComponent
            onConfirm={remove}
            onDecline={() => setIsShowRemoveConfirm(false)}
            description="Are you sure you want to remove this notifier? This action cannot be undone."
            actionText="Remove"
            actionButtonColor="red"
          />
        )}
      </div>
    </div>
  );
};
