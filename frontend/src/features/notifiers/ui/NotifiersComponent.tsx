import { Button, Modal, Spin } from 'antd';
import { useEffect, useState } from 'react';
import { useTranslation } from 'react-i18next';

import { notifierApi } from '../../../entity/notifiers';
import type { Notifier } from '../../../entity/notifiers';
import { NotifierCardComponent } from './NotifierCardComponent';
import { NotifierComponent } from './NotifierComponent';
import { EditNotifierComponent } from './edit/EditNotifierComponent';

interface Props {
  contentHeight: number;
}
export const NotifiersComponent = ({ contentHeight }: Props) => {
  const { t } = useTranslation('notifier');
  const [isLoading, setIsLoading] = useState(true);
  const [notifiers, setNotifiers] = useState<Notifier[]>([]);

  const [isShowAddNotifier, setIsShowAddNotifier] = useState(false);
  const [selectedNotifierId, setSelectedNotifierId] = useState<string | undefined>(undefined);
  const loadNotifiers = () => {
    setIsLoading(true);

    notifierApi
      .getNotifiers()
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
          {notifiers.length >= 5 && addNotifierButton}

          {notifiers.map((notifier) => (
            <NotifierCardComponent
              key={notifier.id}
              notifier={notifier}
              selectedNotifierId={selectedNotifierId}
              setSelectedNotifierId={setSelectedNotifierId}
            />
          ))}

          {notifiers.length < 5 && addNotifierButton}

          <div className="mx-3 text-center text-xs text-gray-500">
            {t('modal.description')}
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
          />
        )}
      </div>

      {isShowAddNotifier && (
        <Modal
          title={t('modal.add_title')}
          footer={<div />}
          open={isShowAddNotifier}
          onCancel={() => setIsShowAddNotifier(false)}
        >
          <div className="my-3 max-w-[250px] text-gray-500">
            {t('modal.description')}
          </div>

          <EditNotifierComponent
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
