import { InfoCircleOutlined } from '@ant-design/icons';
import { useTranslation } from 'react-i18next';

import { type Storage } from '../../entity/storages';
import { getStorageLogoFromType } from '../../entity/storages/models/getStorageLogoFromType';
import { getStorageNameFromType } from '../../entity/storages/models/getStorageNameFromType';

interface Props {
  storage: Storage;
  selectedStorageId?: string;
  setSelectedStorageId: (storageId: string) => void;
}

export const StorageCardComponent = ({
  storage,
  selectedStorageId,
  setSelectedStorageId,
}: Props) => {
  const { t } = useTranslation(['storage']);

  return (
    <div
      className={`mb-3 cursor-pointer rounded p-3 shadow ${selectedStorageId === storage.id ? 'bg-blue-100' : 'bg-white'}`}
      onClick={() => setSelectedStorageId(storage.id)}
    >
      <div className="mb-1 font-bold">{storage.name}</div>

      <div className="flex items-center">
        <div className="text-sm text-gray-500">{t('storage:card.type')}{getStorageNameFromType(storage.type)}</div>

        <img
          src={getStorageLogoFromType(storage.type)}
          alt="storageIcon"
          className="ml-1 h-4 w-4"
        />
      </div>

      {storage.lastSaveError && (
        <div className="mt-1 flex items-center text-sm text-red-600 underline">
          <InfoCircleOutlined className="mr-1" style={{ color: 'red' }} />
          {t('storage:card.has_save_error')}
        </div>
      )}
    </div>
  );
};
