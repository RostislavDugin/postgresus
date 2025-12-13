import { Button, Input, Select } from 'antd';
import { useEffect, useState } from 'react';

import {
  type Storage,
  StorageType,
  getStorageLogoFromType,
  storageApi,
} from '../../../../entity/storages';
import { ToastHelper } from '../../../../shared/toast';
import { EditAzureBlobStorageComponent } from './storages/EditAzureBlobStorageComponent';
import { EditFTPStorageComponent } from './storages/EditFTPStorageComponent';
import { EditGoogleDriveStorageComponent } from './storages/EditGoogleDriveStorageComponent';
import { EditNASStorageComponent } from './storages/EditNASStorageComponent';
import { EditS3StorageComponent } from './storages/EditS3StorageComponent';

interface Props {
  workspaceId: string;

  isShowClose: boolean;
  onClose: () => void;

  isShowName: boolean;

  editingStorage?: Storage;
  onChanged: (storage: Storage) => void;
}

export function EditStorageComponent({
  workspaceId,
  isShowClose,
  onClose,
  isShowName,
  editingStorage,
  onChanged,
}: Props) {
  const [storage, setStorage] = useState<Storage | undefined>();
  const [isUnsaved, setIsUnsaved] = useState(false);
  const [isSaving, setIsSaving] = useState(false);

  const [isTestingConnection, setIsTestingConnection] = useState(false);
  const [isTestConnectionSuccess, setIsTestConnectionSuccess] = useState(false);
  const [connectionError, setConnectionError] = useState<string | undefined>();

  const save = async () => {
    if (!storage) return;

    setIsSaving(true);

    try {
      await storageApi.saveStorage(storage);
      onChanged(storage);
      setIsUnsaved(false);
    } catch (e) {
      alert((e as Error).message);
    }

    setIsSaving(false);
  };

  const testConnection = async () => {
    if (!storage) return;

    setIsTestingConnection(true);
    setConnectionError(undefined);

    try {
      await storageApi.testStorageConnectionDirect(storage);
      setIsTestConnectionSuccess(true);
      ToastHelper.showToast({
        title: 'Connection test successful!',
        description: 'Storage connection tested successfully',
      });
    } catch (e) {
      const errorMessage = (e as Error).message;
      setConnectionError(errorMessage);
      alert(errorMessage);
    }

    setIsTestingConnection(false);
  };

  const setStorageType = (type: StorageType) => {
    if (!storage) return;

    storage.localStorage = undefined;
    storage.s3Storage = undefined;
    storage.googleDriveStorage = undefined;
    storage.azureBlobStorage = undefined;
    storage.ftpStorage = undefined;

    if (type === StorageType.LOCAL) {
      storage.localStorage = {};
    }

    if (type === StorageType.S3) {
      storage.s3Storage = {
        s3Bucket: '',
        s3Region: '',
        s3AccessKey: '',
        s3SecretKey: '',
        s3Endpoint: '',
      };
    }

    if (type === StorageType.GOOGLE_DRIVE) {
      storage.googleDriveStorage = {
        clientId: '',
        clientSecret: '',
      };
    }

    if (type === StorageType.NAS) {
      storage.nasStorage = {
        host: '',
        port: 445,
        share: '',
        username: '',
        password: '',
        useSsl: false,
        domain: '',
        path: '',
      };
    }

    if (type === StorageType.AZURE_BLOB) {
      storage.azureBlobStorage = {
        authMethod: 'ACCOUNT_KEY',
        connectionString: '',
        accountName: '',
        accountKey: '',
        containerName: '',
        endpoint: '',
        prefix: '',
      };
    }

    if (type === StorageType.FTP) {
      storage.ftpStorage = {
        host: '',
        port: 21,
        username: '',
        password: '',
        useSsl: false,
        path: '',
      };
    }

    setStorage(
      JSON.parse(
        JSON.stringify({
          ...storage,
          type: type,
        }),
      ),
    );
  };

  useEffect(() => {
    setIsUnsaved(true);

    setStorage(
      editingStorage
        ? JSON.parse(JSON.stringify(editingStorage))
        : {
            id: undefined as unknown as string,
            workspaceId,
            name: '',
            type: StorageType.LOCAL,
            localStorage: {},
          },
    );
  }, [editingStorage]);

  const isAllDataFilled = () => {
    if (!storage) {
      return false;
    }

    if (!storage.name) {
      return false;
    }

    if (storage.type === StorageType.LOCAL) {
      return true; // No additional settings required for local storage
    }

    if (storage.type === StorageType.S3) {
      if (storage.id) {
        return storage.s3Storage?.s3Bucket;
      }

      return (
        storage.s3Storage?.s3Bucket &&
        storage.s3Storage?.s3AccessKey &&
        storage.s3Storage?.s3SecretKey
      );
    }

    if (storage.type === StorageType.GOOGLE_DRIVE) {
      if (storage.id) {
        return storage.googleDriveStorage?.clientId;
      }

      return (
        storage.googleDriveStorage?.clientId &&
        storage.googleDriveStorage?.clientSecret &&
        storage.googleDriveStorage?.tokenJson
      );
    }

    if (storage.type === StorageType.NAS) {
      if (storage.id) {
        return (
          storage.nasStorage?.host &&
          storage.nasStorage?.port &&
          storage.nasStorage?.share &&
          storage.nasStorage?.username
        );
      }

      return (
        storage.nasStorage?.host &&
        storage.nasStorage?.port &&
        storage.nasStorage?.share &&
        storage.nasStorage?.username &&
        storage.nasStorage?.password
      );
    }

    if (storage.type === StorageType.AZURE_BLOB) {
      if (storage.id) {
        return storage.azureBlobStorage?.containerName;
      }

      const isContainerNameFilled = storage.azureBlobStorage?.containerName;

      if (storage.azureBlobStorage?.authMethod === 'CONNECTION_STRING') {
        return isContainerNameFilled && storage.azureBlobStorage?.connectionString;
      }

      if (storage.azureBlobStorage?.authMethod === 'ACCOUNT_KEY') {
        return (
          isContainerNameFilled &&
          storage.azureBlobStorage?.accountName &&
          storage.azureBlobStorage?.accountKey
        );
      }
    }

    if (storage.type === StorageType.FTP) {
      if (storage.id) {
        return storage.ftpStorage?.host && storage.ftpStorage?.port && storage.ftpStorage?.username;
      }

      return (
        storage.ftpStorage?.host &&
        storage.ftpStorage?.port &&
        storage.ftpStorage?.username &&
        storage.ftpStorage?.password
      );
    }

    return false;
  };

  if (!storage) return <div />;

  return (
    <div>
      {isShowName && (
        <div className="mb-1 flex w-full flex-col items-start sm:flex-row sm:items-center">
          <div className="mb-1 min-w-[110px] sm:mb-0">Name</div>

          <Input
            value={storage?.name || ''}
            onChange={(e) => {
              setStorage({ ...storage, name: e.target.value });
              setIsUnsaved(true);
            }}
            size="small"
            className="w-full max-w-[250px]"
            placeholder="My Storage"
          />
        </div>
      )}

      <div className="mb-1 flex w-full flex-col items-start sm:flex-row sm:items-center">
        <div className="mb-1 min-w-[110px] sm:mb-0">Type</div>

        <div className="flex items-center">
          <Select
            value={storage?.type}
            options={[
              { label: 'Local storage', value: StorageType.LOCAL },
              { label: 'S3', value: StorageType.S3 },
              { label: 'Google Drive', value: StorageType.GOOGLE_DRIVE },
              { label: 'NAS', value: StorageType.NAS },
              { label: 'Azure Blob Storage', value: StorageType.AZURE_BLOB },
              { label: 'FTP', value: StorageType.FTP },
            ]}
            onChange={(value) => {
              setStorageType(value);
              setIsUnsaved(true);
            }}
            size="small"
            className="w-[250px] max-w-[250px]"
          />

          <img src={getStorageLogoFromType(storage?.type)} className="ml-2 h-4 w-4" />
        </div>
      </div>

      <div className="mt-5" />

      <div>
        {storage?.type === StorageType.S3 && (
          <EditS3StorageComponent
            storage={storage}
            setStorage={setStorage}
            setUnsaved={() => {
              setIsUnsaved(true);
              setIsTestConnectionSuccess(false);
              setConnectionError(undefined);
            }}
            connectionError={connectionError}
          />
        )}

        {storage?.type === StorageType.GOOGLE_DRIVE && (
          <EditGoogleDriveStorageComponent
            storage={storage}
            setStorage={setStorage}
            setUnsaved={() => {
              setIsUnsaved(true);
              setIsTestConnectionSuccess(false);
            }}
          />
        )}

        {storage?.type === StorageType.NAS && (
          <EditNASStorageComponent
            storage={storage}
            setStorage={setStorage}
            setUnsaved={() => {
              setIsUnsaved(true);
              setIsTestConnectionSuccess(false);
            }}
          />
        )}

        {storage?.type === StorageType.AZURE_BLOB && (
          <EditAzureBlobStorageComponent
            storage={storage}
            setStorage={setStorage}
            setUnsaved={() => {
              setIsUnsaved(true);
              setIsTestConnectionSuccess(false);
            }}
          />
        )}

        {storage?.type === StorageType.FTP && (
          <EditFTPStorageComponent
            storage={storage}
            setStorage={setStorage}
            setUnsaved={() => {
              setIsUnsaved(true);
              setIsTestConnectionSuccess(false);
            }}
          />
        )}
      </div>

      <div className="mt-3 flex">
        {isUnsaved && !isTestConnectionSuccess ? (
          <Button
            className="mr-1"
            disabled={isTestingConnection || !isAllDataFilled()}
            loading={isTestingConnection}
            type="primary"
            onClick={testConnection}
          >
            Test connection
          </Button>
        ) : (
          <div />
        )}

        {isUnsaved && isTestConnectionSuccess ? (
          <Button
            className="mr-1"
            disabled={isSaving || !isAllDataFilled()}
            loading={isSaving}
            type="primary"
            onClick={save}
          >
            Save
          </Button>
        ) : (
          <div />
        )}

        {isShowClose ? (
          <Button
            className="mr-1"
            disabled={isSaving}
            type="primary"
            danger
            ghost
            onClick={onClose}
          >
            Cancel
          </Button>
        ) : (
          <div />
        )}
      </div>
    </div>
  );
}
