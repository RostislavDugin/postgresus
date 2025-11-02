import { Button, Modal, Spin } from 'antd';
import { useEffect, useState } from 'react';

import { notifierApi } from '../../../entity/notifiers';
import type { Notifier } from '../../../entity/notifiers';
import type { WorkspaceResponse } from '../../../entity/workspaces';
import { NotifierCardComponent } from './NotifierCardComponent';
import { NotifierComponent } from './NotifierComponent';
import { EditNotifierComponent } from './edit/EditNotifierComponent';

interface Props {
  contentHeight: number;
  workspace: WorkspaceResponse;
  isCanManageNotifiers: boolean;
}

export const NotifiersComponent = ({ contentHeight, workspace, isCanManageNotifiers }: Props) => {
  const [isLoading, setIsLoading] = useState(true);
  const [notifiers, setNotifiers] = useState<Notifier[]>([]);

  const [isShowAddNotifier, setIsShowAddNotifier] = useState(false);
  const [selectedNotifierId, setSelectedNotifierId] = useState<string | undefined>(undefined);
  const loadNotifiers = () => {
    setIsLoading(true);

    notifierApi
      .getNotifiers(workspace.id)
      .then((notifiers) => {
        setNotifiers(notifiers);
        if (!selectedNotifierId) {
          setSelectedNotifierId(notifiers[0]?.id);
        }
      })
      .catch((e) => alert(e.message))
      .finally(() => setIsLoading(false));
  };

  useEffect(() => {
    loadNotifiers();
  }, []);

  if (isLoading) {
    return (
      <div className="mx-3 my-3 flex w-[250px] justify-center">
        <Spin />
      </div>
    );
  }

  const addNotifierButton = (
    <Button type="primary" className="mb-2 w-full" onClick={() => setIsShowAddNotifier(true)}>
      Add notifier
    </Button>
  );

  return (
    <>
      <div className="flex grow">
        <div
          className="mx-3 w-[250px] min-w-[250px] overflow-y-auto"
          style={{ height: contentHeight }}
        >
          {notifiers.length >= 5 && isCanManageNotifiers && addNotifierButton}

          {notifiers.map((notifier) => (
            <NotifierCardComponent
              key={notifier.id}
              notifier={notifier}
              selectedNotifierId={selectedNotifierId}
              setSelectedNotifierId={setSelectedNotifierId}
            />
          ))}

          {notifiers.length < 5 && isCanManageNotifiers && addNotifierButton}

          <div className="mx-3 text-center text-xs text-gray-500">
            Notifier - is a place where notifications will be sent (email, Slack, Telegram, etc.)
          </div>
        </div>

        {selectedNotifierId && (
          <NotifierComponent
            notifierId={selectedNotifierId}
            onNotifierChanged={() => {
              loadNotifiers();
            }}
            onNotifierDeleted={() => {
              loadNotifiers();
              setSelectedNotifierId(
                notifiers.filter((notifier) => notifier.id !== selectedNotifierId)[0]?.id,
              );
            }}
            isCanManageNotifiers={isCanManageNotifiers}
          />
        )}
      </div>

      {isShowAddNotifier && (
        <Modal
          title="Add notifier"
          footer={<div />}
          open={isShowAddNotifier}
          onCancel={() => setIsShowAddNotifier(false)}
        >
          <div className="my-3 max-w-[250px] text-gray-500">
            Notifier - is a place where notifications will be sent (email, Slack, Telegram, etc.)
          </div>

          <EditNotifierComponent
            workspaceId={workspace.id}
            isShowName
            isShowClose={false}
            onClose={() => setIsShowAddNotifier(false)}
            onChanged={() => {
              loadNotifiers();
              setIsShowAddNotifier(false);
            }}
          />
        </Modal>
      )}
    </>
  );
};
