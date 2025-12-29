import { DownOutlined, InfoCircleOutlined, UpOutlined } from '@ant-design/icons';
import { Input, Radio, Tooltip } from 'antd';
import { useState } from 'react';

import type { Storage } from '../../../../../entity/storages';

interface Props {
  storage: Storage;
  setStorage: (storage: Storage) => void;
  setUnsaved: () => void;
}

export function EditAzureBlobStorageComponent({ storage, setStorage, setUnsaved }: Props) {
  const hasAdvancedValues =
    !!storage?.azureBlobStorage?.prefix || !!storage?.azureBlobStorage?.endpoint;
  const [showAdvanced, setShowAdvanced] = useState(hasAdvancedValues);

  return (
    <>
      <div className="mb-1 flex w-full flex-col items-start sm:flex-row sm:items-center">
        <div className="mb-1 min-w-[110px] sm:mb-0">Auth method</div>
        <Radio.Group
          value={storage?.azureBlobStorage?.authMethod || 'ACCOUNT_KEY'}
          onChange={(e) => {
            if (!storage?.azureBlobStorage) return;

            setStorage({
              ...storage,
              azureBlobStorage: {
                ...storage.azureBlobStorage,
                authMethod: e.target.value,
              },
            });
            setUnsaved();
          }}
          size="small"
        >
          <Radio value="ACCOUNT_KEY">Account key</Radio>
          <Radio value="CONNECTION_STRING">Connection string</Radio>
        </Radio.Group>
      </div>

      {storage?.azureBlobStorage?.authMethod === 'CONNECTION_STRING' && (
        <div className="mb-1 flex w-full flex-col items-start sm:flex-row sm:items-center">
          <div className="mb-1 min-w-[110px] sm:mb-0">Connection</div>
          <div className="flex items-center">
            <Input.Password
              value={storage?.azureBlobStorage?.connectionString || ''}
              onChange={(e) => {
                if (!storage?.azureBlobStorage) return;

                setStorage({
                  ...storage,
                  azureBlobStorage: {
                    ...storage.azureBlobStorage,
                    connectionString: e.target.value.trim(),
                  },
                });
                setUnsaved();
              }}
              size="small"
              className="w-full max-w-[250px]"
              placeholder="DefaultEndpointsProtocol=https;AccountName=..."
              autoComplete="off"
              data-1p-ignore
              data-lpignore="true"
              data-form-type="other"
            />

            <Tooltip
              className="cursor-pointer"
              title="Azure Storage connection string from Azure Portal"
            >
              <InfoCircleOutlined className="ml-2" style={{ color: 'gray' }} />
            </Tooltip>
          </div>
        </div>
      )}

      {storage?.azureBlobStorage?.authMethod === 'ACCOUNT_KEY' && (
        <>
          <div className="mb-1 flex w-full flex-col items-start sm:flex-row sm:items-center">
            <div className="mb-1 min-w-[110px] sm:mb-0">Account name</div>
            <Input
              value={storage?.azureBlobStorage?.accountName || ''}
              onChange={(e) => {
                if (!storage?.azureBlobStorage) return;

                setStorage({
                  ...storage,
                  azureBlobStorage: {
                    ...storage.azureBlobStorage,
                    accountName: e.target.value.trim(),
                  },
                });
                setUnsaved();
              }}
              size="small"
              className="w-full max-w-[250px]"
              placeholder="mystorageaccount"
            />
          </div>

          <div className="mb-1 flex w-full flex-col items-start sm:flex-row sm:items-center">
            <div className="mb-1 min-w-[110px] sm:mb-0">Account key</div>
            <Input.Password
              value={storage?.azureBlobStorage?.accountKey || ''}
              onChange={(e) => {
                if (!storage?.azureBlobStorage) return;

                setStorage({
                  ...storage,
                  azureBlobStorage: {
                    ...storage.azureBlobStorage,
                    accountKey: e.target.value.trim(),
                  },
                });
                setUnsaved();
              }}
              size="small"
              className="w-full max-w-[250px]"
              placeholder="your-account-key"
              autoComplete="off"
              data-1p-ignore
              data-lpignore="true"
              data-form-type="other"
            />
          </div>
        </>
      )}

      <div className="mb-1 flex w-full flex-col items-start sm:flex-row sm:items-center">
        <div className="mb-1 min-w-[110px] sm:mb-0">Container name</div>
        <Input
          value={storage?.azureBlobStorage?.containerName || ''}
          onChange={(e) => {
            if (!storage?.azureBlobStorage) return;

            setStorage({
              ...storage,
              azureBlobStorage: {
                ...storage.azureBlobStorage,
                containerName: e.target.value.trim(),
              },
            });
            setUnsaved();
          }}
          size="small"
          className="w-full max-w-[250px]"
          placeholder="my-container"
        />
      </div>

      <div className="mt-4 mb-3 flex items-center">
        <div
          className="flex cursor-pointer items-center text-sm text-blue-600 hover:text-blue-800"
          onClick={() => setShowAdvanced(!showAdvanced)}
        >
          <span className="mr-2">Advanced settings</span>

          {showAdvanced ? (
            <UpOutlined style={{ fontSize: '12px' }} />
          ) : (
            <DownOutlined style={{ fontSize: '12px' }} />
          )}
        </div>
      </div>

      {showAdvanced && (
        <>
          {storage?.azureBlobStorage?.authMethod === 'ACCOUNT_KEY' && (
            <div className="mb-1 flex w-full flex-col items-start sm:flex-row sm:items-center">
              <div className="mb-1 min-w-[110px] sm:mb-0">Endpoint</div>
              <div className="flex items-center">
                <Input
                  value={storage?.azureBlobStorage?.endpoint || ''}
                  onChange={(e) => {
                    if (!storage?.azureBlobStorage) return;

                    setStorage({
                      ...storage,
                      azureBlobStorage: {
                        ...storage.azureBlobStorage,
                        endpoint: e.target.value.trim(),
                      },
                    });
                    setUnsaved();
                  }}
                  size="small"
                  className="w-full max-w-[250px]"
                  placeholder="https://myaccount.blob.core.windows.net (optional)"
                />

                <Tooltip
                  className="cursor-pointer"
                  title="Custom endpoint URL (optional, leave empty for standard Azure)"
                >
                  <InfoCircleOutlined className="ml-2" style={{ color: 'gray' }} />
                </Tooltip>
              </div>
            </div>
          )}

          <div className="mb-1 flex w-full flex-col items-start sm:flex-row sm:items-center">
            <div className="mb-1 min-w-[110px] sm:mb-0">Blob prefix</div>
            <div className="flex items-center">
              <Input
                value={storage?.azureBlobStorage?.prefix || ''}
                onChange={(e) => {
                  if (!storage?.azureBlobStorage) return;

                  setStorage({
                    ...storage,
                    azureBlobStorage: {
                      ...storage.azureBlobStorage,
                      prefix: e.target.value.trim(),
                    },
                  });
                  setUnsaved();
                }}
                size="small"
                className="w-full max-w-[250px]"
                placeholder="my-prefix/ (optional)"
              />

              <Tooltip
                className="cursor-pointer"
                title="Optional prefix for all blob names (e.g., 'backups/' or 'my_team/')"
              >
                <InfoCircleOutlined className="ml-2" style={{ color: 'gray' }} />
              </Tooltip>
            </div>
          </div>
        </>
      )}

      <div className="mb-5" />
    </>
  );
}
