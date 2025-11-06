import { Button, Input } from 'antd';
import { useTranslation } from 'react-i18next';

import { GOOGLE_DRIVE_OAUTH_REDIRECT_URL } from '../../../../../constants';
import type { Storage } from '../../../../../entity/storages';
import type { StorageOauthDto } from '../../../../../entity/storages/models/StorageOauthDto';

interface Props {
  storage: Storage;
  setStorage: (storage: Storage) => void;
  setIsUnsaved: (isUnsaved: boolean) => void;
}

export function EditGoogleDriveStorageComponent({ storage, setStorage, setIsUnsaved }: Props) {
  const { t } = useTranslation(['storage']);

  const goToAuthUrl = () => {
    if (!storage?.googleDriveStorage?.clientId || !storage?.googleDriveStorage?.clientSecret) {
      return;
    }

    const redirectUri = GOOGLE_DRIVE_OAUTH_REDIRECT_URL;
    const clientId = storage.googleDriveStorage.clientId;
    const scope = 'https://www.googleapis.com/auth/drive.file';
    const originUrl = `${window.location.origin}/storages/google-oauth`;

    const oauthDto: StorageOauthDto = {
      redirectUrl: originUrl,
      storage: storage,
      authCode: '',
    };

    const authUrl = `https://accounts.google.com/o/oauth2/v2/auth?client_id=${
      clientId
    }&redirect_uri=${redirectUri}&response_type=code&scope=${encodeURIComponent(scope)}&access_type=offline&prompt=consent&state=${encodeURIComponent(JSON.stringify(oauthDto))}`;

    window.open(authUrl);
  };

  return (
    <>
      <div className="mb-2 flex items-center">
        <div className="min-w-[110px]" />

        <div className="text-xs text-blue-600">
          <a href="https://postgresus.com/google-drive-storage" target="_blank" rel="noreferrer">
            {t('storage:form.google_drive_link')}
          </a>
        </div>
      </div>

      <div className="mb-1 flex items-center">
        <div className="min-w-[110px]">{t('storage:form.client_id_label')}</div>
        <Input
          value={storage?.googleDriveStorage?.clientId || ''}
          onChange={(e: React.ChangeEvent<HTMLInputElement>) => {
            if (!storage?.googleDriveStorage) return;

            setStorage({
              ...storage,
              googleDriveStorage: {
                ...storage.googleDriveStorage,
                clientId: e.target.value.trim(),
              },
            });
            setIsUnsaved(true);
          }}
          size="small"
          className="w-full max-w-[250px]"
          placeholder={t('storage:form.client_id_placeholder')}
          disabled={!!storage?.googleDriveStorage?.tokenJson}
        />
      </div>

      <div className="mb-1 flex items-center">
        <div className="min-w-[110px]">{t('storage:form.client_secret_label')}</div>
        <Input
          value={storage?.googleDriveStorage?.clientSecret || ''}
          onChange={(e: React.ChangeEvent<HTMLInputElement>) => {
            if (!storage?.googleDriveStorage) return;

            setStorage({
              ...storage,
              googleDriveStorage: {
                ...storage.googleDriveStorage,
                clientSecret: e.target.value.trim(),
              },
            });
            setIsUnsaved(true);
          }}
          size="small"
          className="w-full max-w-[250px]"
          placeholder={t('storage:form.client_secret_placeholder')}
          disabled={!!storage?.googleDriveStorage?.tokenJson}
        />
      </div>

      {storage?.googleDriveStorage?.tokenJson && (
        <>
          <div className="mb-1 flex items-center">
            <div className="min-w-[110px]">{t('storage:form.user_token_label')}</div>
            <Input
              value={storage?.googleDriveStorage?.tokenJson || ''}
              disabled
              size="small"
              className="w-full max-w-[250px]"
              placeholder={t('storage:form.user_token_placeholder')}
            />
          </div>
        </>
      )}

      {!storage?.googleDriveStorage?.tokenJson && (
        <Button
          type="primary"
          disabled={
            !storage?.googleDriveStorage?.clientId || !storage?.googleDriveStorage?.clientSecret
          }
          onClick={goToAuthUrl}
        >
          {t('storage:form.authorize_button')}
        </Button>
      )}
    </>
  );
}
