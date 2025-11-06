import { useTranslation } from 'react-i18next';

import type { Storage } from '../../../../../entity/storages';

interface Props {
  storage: Storage;
}

export function ShowGoogleDriveStorageComponent({ storage }: Props) {
  const { t } = useTranslation(['storage']);

  return (
    <>
      <div className="mb-1 flex items-center">
        <div className="min-w-[110px]">{t('storage:form.google_client_id_label')}</div>
        {storage?.googleDriveStorage?.clientId
          ? `${storage?.googleDriveStorage?.clientId.slice(0, 10)}***`
          : '-'}
      </div>

      <div className="mb-1 flex items-center">
        <div className="min-w-[110px]">{t('storage:form.google_client_secret_label')}</div>
        {storage?.googleDriveStorage?.clientSecret
          ? `${storage?.googleDriveStorage?.clientSecret.slice(0, 10)}***`
          : '-'}
      </div>

      <div className="mb-1 flex items-center">
        <div className="min-w-[110px]">{t('storage:form.google_user_token_label')}</div>
        {storage?.googleDriveStorage?.tokenJson
          ? `${storage?.googleDriveStorage?.tokenJson.slice(0, 10)}***`
          : '-'}
      </div>
    </>
  );
}
