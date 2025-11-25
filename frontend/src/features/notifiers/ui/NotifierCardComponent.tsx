import { InfoCircleOutlined } from '@ant-design/icons';

import { type Notifier } from '../../../entity/notifiers';
import { getNotifierLogoFromType } from '../../../entity/notifiers/models/getNotifierLogoFromType';
import { getNotifierNameFromType } from '../../../entity/notifiers/models/getNotifierNameFromType';

interface Props {
  notifier: Notifier;
  selectedNotifierId?: string;
  setSelectedNotifierId: (notifierId: string) => void;
}

export const NotifierCardComponent = ({
  notifier,
  selectedNotifierId,
  setSelectedNotifierId,
}: Props) => {
  return (
    <div
      className={`mb-3 cursor-pointer rounded p-3 shadow ${selectedNotifierId === notifier.id ? 'bg-blue-100 dark:bg-blue-800' : 'bg-white dark:bg-gray-800'}`}
      onClick={() => setSelectedNotifierId(notifier.id)}
    >
      <div className="mb-1 font-bold">{notifier.name}</div>

      <div className="flex items-center">
        <div className="text-sm text-gray-500 dark:text-gray-400">
          Notify to {getNotifierNameFromType(notifier.notifierType)}
        </div>

        <img
          src={getNotifierLogoFromType(notifier.notifierType)}
          alt="notifyIcon"
          className="ml-1 h-4 w-4"
        />
      </div>

      {notifier.lastSendError && (
        <div className="mt-1 flex items-center text-sm text-red-600 underline dark:text-red-400">
          <InfoCircleOutlined className="mr-1" style={{ color: 'red' }} />
          Has send error
        </div>
      )}
    </div>
  );
};
