import { Button, Input } from 'antd';

import { GOOGLE_DRIVE_OAUTH_REDIRECT_URL } from '../../../../../constants';
import type { Storage } from '../../../../../entity/storages';
import type { StorageOauthDto } from '../../../../../entity/storages/models/StorageOauthDto';

interface Props {
  storage: Storage;
  setStorage: (storage: Storage) => void;
  setIsUnsaved: (isUnsaved: boolean) => void;
}

export function EditGoogleDriveStorageComponent({ storage, setStorage, setIsUnsaved }: Props) {
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
          <a href="https://postgresus.com/storages/google-drive" target="_blank" rel="noreferrer">
            How to connect Google Drive?
          </a>
        </div>
      </div>

      <div className="mb-1 flex items-center">
        <div className="min-w-[110px]">Client ID</div>
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
          placeholder="my-client-id"
          disabled={!!storage?.googleDriveStorage?.tokenJson}
        />
      </div>

      <div className="mb-1 flex items-center">
        <div className="min-w-[110px]">Client Secret</div>
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
          placeholder="my-client-secret"
          disabled={!!storage?.googleDriveStorage?.tokenJson}
        />
      </div>

      {storage?.googleDriveStorage?.tokenJson && (
        <>
          <div className="mb-1 flex items-center">
            <div className="min-w-[110px]">User Token</div>
            <Input
              value={storage?.googleDriveStorage?.tokenJson || ''}
              disabled
              size="small"
              className="w-full max-w-[250px]"
              placeholder="my-user-token"
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
          Authorize
        </Button>
      )}
    </>
  );
}
