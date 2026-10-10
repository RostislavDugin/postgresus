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

import { databaseApi } from '../../../entity/databases';
import { notifierApi } from '../../../entity/notifiers';
import type { Notifier } from '../../../entity/notifiers';
import { translateApiError } from '../../../shared/i18n';
import { ToastHelper } from '../../../shared/toast';
import { ConfirmationComponent } from '../../../shared/ui';
import { EditNotifierComponent } from './edit/EditNotifierComponent';
import { NotifierTransferDialogComponent } from './NotifierTransferDialogComponent';
import { ShowNotifierComponent } from './show/ShowNotifierComponent';

interface Props {
  notifierId: string;
  onNotifierChanged: (notifier: Notifier) => void;
  onNotifierDeleted: () => void;
  onNotifierTransferred: () => void;
  isCanManageNotifiers: boolean;
}

export const NotifierComponent = ({
  notifierId,
  onNotifierChanged,
  onNotifierDeleted,
  onNotifierTransferred,
  isCanManageNotifiers,
}: Props) => {
  const { t } = useTranslation();
  const [notifier, setNotifier] = useState<Notifier | undefined>();

  const [isEditName, setIsEditName] = useState(false);
  const [isEditSettings, setIsEditSettings] = useState(false);

  const [editNotifier, setEditNotifier] = useState<Notifier | undefined>();
  const [isNameUnsaved, setIsNameUnsaved] = useState(false);
  const [isSaving, setIsSaving] = useState(false);

  const [isSendingTestNotification, setIsSendingTestNotification] = useState(false);

  const [isShowRemoveConfirm, setIsShowRemoveConfirm] = useState(false);
  const [isRemoving, setIsRemoving] = useState(false);

  const [isShowTransferDialog, setIsShowTransferDialog] = useState(false);

  const sendTestNotification = () => {
    if (!notifier) return;

    setIsSendingTestNotification(true);
    notifierApi
      .sendTestNotification(notifier.id)
      .then(() => {
        ToastHelper.showToast({
          title: t('notifiers.testNotification.sent.title'),
          description: t('notifiers.testNotification.sent.description'),
        });

        if (notifier.lastSendError) {
          setNotifier({ ...notifier, lastSendError: undefined });
          onNotifierChanged(notifier);
        }
      })
      .catch((e: unknown) => {
        alert(translateApiError(e, t));
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
        alert(t('notifiers.config.removeBlockedByDatabases'));
        setIsShowRemoveConfirm(false);
      } else {
        await notifierApi.deleteNotifier(notifier.id);
        onNotifierDeleted();
      }
    } catch (e) {
      alert(translateApiError(e, t));
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
      .catch((e: unknown) => {
        alert(translateApiError(e, t));
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
      <div className="grow overflow-y-auto rounded bg-white p-5 shadow dark:bg-gray-800">
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
                    placeholder={t('notifiers.config.namePlaceholder')}
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
                    disabled={!editNotifier?.name}
                  >
                    {t('common.actions.save')}
                  </Button>
                )}
              </div>
            )}

            {notifier.lastSendError && (
              <div className="max-w-[400px] rounded border border-red-600 px-3 py-3">
                <div className="mt-1 flex items-center text-sm font-bold text-red-600">
                  <InfoCircleOutlined className="mr-2" style={{ color: 'red' }} />
                  {t('notifiers.config.sendError.title')}
                </div>

                <div className="mt-3 text-sm">
                  {t('notifiers.config.sendError.errorLabel')}
                  <br />
                  {notifier.lastSendError}
                </div>

                <div className="mt-3 text-sm break-words whitespace-pre-wrap text-gray-500 dark:text-gray-400">
                  {t('notifiers.config.sendError.clearHint')}
                  <ul>
                    <li>{t('notifiers.config.sendError.clearBySendingTest')}</li>
                    <li>{t('notifiers.config.sendError.clearByNextNotification')}</li>
                  </ul>
                </div>
              </div>
            )}

            <div className="mt-5 flex items-center font-bold">
              <div>{t('notifiers.config.settingsTitle')}</div>

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
                  onClick={sendTestNotification}
                  loading={isSendingTestNotification}
                  disabled={isSendingTestNotification}
                >
                  {t('notifiers.testNotification.send')}
                </Button>

                {isCanManageNotifiers && (
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
            description={t('notifiers.config.removeConfirmation')}
            actionText={t('common.actions.remove')}
            actionButtonColor="red"
          />
        )}
      </div>

      {isShowTransferDialog && notifier && (
        <NotifierTransferDialogComponent
          notifier={notifier}
          onClose={() => setIsShowTransferDialog(false)}
          onTransferred={() => {
            setIsShowTransferDialog(false);
            onNotifierTransferred();
          }}
        />
      )}
    </div>
  );
};
