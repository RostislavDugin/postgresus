import { Modal, Spin } from 'antd';
import { useEffect, useState } from 'react';

import { GOOGLE_DRIVE_OAUTH_REDIRECT_URL } from '../constants';
import { type Storage, StorageType } from '../entity/storages';
import type { StorageOauthDto } from '../entity/storages/models/StorageOauthDto';
import { EditStorageComponent } from '../features/storages/ui/edit/EditStorageComponent';

export function OauthStorageComponent() {
  const [storage, setStorage] = useState<Storage | undefined>();

  const exchangeGoogleOauthCode = async (oauthDto: StorageOauthDto) => {
    if (!oauthDto.storage.googleDriveStorage) {
      alert('Google Drive storage configuration not found');
      return;
    }

    const { clientId, clientSecret } = oauthDto.storage.googleDriveStorage;
    const { authCode } = oauthDto;

    try {
      // Exchange authorization code for access token
      const response = await fetch('https://oauth2.googleapis.com/token', {
        method: 'POST',
        headers: {
          'Content-Type': 'application/x-www-form-urlencoded',
        },
        body: new URLSearchParams({
          code: authCode,
          client_id: clientId,
          client_secret: clientSecret,
          redirect_uri: GOOGLE_DRIVE_OAUTH_REDIRECT_URL,
          grant_type: 'authorization_code',
        }),
      });

      if (!response.ok) {
        throw new Error(`OAuth exchange failed: ${response.statusText}`);
      }

      const tokenData = await response.json();

      oauthDto.storage.googleDriveStorage.tokenJson = JSON.stringify(tokenData);
      setStorage(oauthDto.storage);
    } catch (error) {
      alert(`Failed to exchange OAuth code: ${error}`);
    }
  };

  useEffect(() => {
    const oauthDtoParam = new URLSearchParams(window.location.search).get('oauthDto');
    if (!oauthDtoParam) {
      alert('OAuth param not found');
      return;
    }

    const decodedParam = decodeURIComponent(oauthDtoParam);
    const oauthDto: StorageOauthDto = JSON.parse(decodedParam);

    if (oauthDto.storage.type === StorageType.GOOGLE_DRIVE) {
      if (!oauthDto.storage.googleDriveStorage) {
        alert('Google Drive storage not found');
        return;
      }

      exchangeGoogleOauthCode(oauthDto);
    }
  }, []);

  if (!storage) {
    return (
      <div className="mt-20 flex justify-center">
        <Spin />
      </div>
    );
  }

  return (
    <div>
      <Modal
        title="Add storage"
        footer={<div />}
        open
        onCancel={() => {
          window.location.href = '/';
        }}
      >
        <div className="my-3 max-w-[250px] text-gray-500 dark:text-gray-400">
          Storage - is a place where backups will be stored (local disk, S3, etc.)
        </div>

        <EditStorageComponent
          workspaceId={storage.workspaceId}
          isShowClose={false}
          onClose={() => {}}
          isShowName={false}
          editingStorage={storage}
          onChanged={() => {
            window.location.href = '/';
          }}
        />
      </Modal>
    </div>
  );
}
