import { DownOutlined, InfoCircleOutlined, UpOutlined } from '@ant-design/icons';
import { Checkbox, Input, InputNumber, Tooltip } from 'antd';
import { useState } from 'react';

import type { Storage } from '../../../../../entity/storages';

interface Props {
  storage: Storage;
  setStorage: (storage: Storage) => void;
  setUnsaved: () => void;
}

export function EditFTPStorageComponent({ storage, setStorage, setUnsaved }: Props) {
  const hasAdvancedValues = !!storage?.ftpStorage?.skipTlsVerify;
  const [showAdvanced, setShowAdvanced] = useState(hasAdvancedValues);

  return (
    <>
      <div className="mb-1 flex w-full flex-col items-start sm:flex-row sm:items-center">
        <div className="mb-1 min-w-[110px] sm:mb-0">Host</div>
        <Input
          value={storage?.ftpStorage?.host || ''}
          onChange={(e) => {
            if (!storage?.ftpStorage) return;

            setStorage({
              ...storage,
              ftpStorage: {
                ...storage.ftpStorage,
                host: e.target.value.trim(),
              },
            });
            setUnsaved();
          }}
          size="small"
          className="w-full max-w-[250px]"
          placeholder="ftp.example.com"
        />
      </div>

      <div className="mb-1 flex w-full flex-col items-start sm:flex-row sm:items-center">
        <div className="mb-1 min-w-[110px] sm:mb-0">Port</div>
        <InputNumber
          value={storage?.ftpStorage?.port}
          onChange={(value) => {
            if (!storage?.ftpStorage || !value) return;

            setStorage({
              ...storage,
              ftpStorage: {
                ...storage.ftpStorage,
                port: value,
              },
            });
            setUnsaved();
          }}
          size="small"
          className="w-full max-w-[250px]"
          min={1}
          max={65535}
          placeholder="21"
        />
      </div>

      <div className="mb-1 flex w-full flex-col items-start sm:flex-row sm:items-center">
        <div className="mb-1 min-w-[110px] sm:mb-0">Username</div>
        <Input
          value={storage?.ftpStorage?.username || ''}
          onChange={(e) => {
            if (!storage?.ftpStorage) return;

            setStorage({
              ...storage,
              ftpStorage: {
                ...storage.ftpStorage,
                username: e.target.value.trim(),
              },
            });
            setUnsaved();
          }}
          size="small"
          className="w-full max-w-[250px]"
          placeholder="username"
        />
      </div>

      <div className="mb-1 flex w-full flex-col items-start sm:flex-row sm:items-center">
        <div className="mb-1 min-w-[110px] sm:mb-0">Password</div>
        <Input.Password
          value={storage?.ftpStorage?.password || ''}
          onChange={(e) => {
            if (!storage?.ftpStorage) return;

            setStorage({
              ...storage,
              ftpStorage: {
                ...storage.ftpStorage,
                password: e.target.value,
              },
            });
            setUnsaved();
          }}
          size="small"
          className="w-full max-w-[250px]"
          placeholder="password"
        />
      </div>

      <div className="mb-1 flex w-full flex-col items-start sm:flex-row sm:items-center">
        <div className="mb-1 min-w-[110px] sm:mb-0">Path</div>
        <div className="flex items-center">
          <Input
            value={storage?.ftpStorage?.path || ''}
            onChange={(e) => {
              if (!storage?.ftpStorage) return;

              let pathValue = e.target.value.trim();
              if (pathValue.startsWith('/')) {
                pathValue = pathValue.substring(1);
              }

              setStorage({
                ...storage,
                ftpStorage: {
                  ...storage.ftpStorage,
                  path: pathValue || undefined,
                },
              });
              setUnsaved();
            }}
            size="small"
            className="w-full max-w-[250px]"
            placeholder="backups (optional)"
          />

          <Tooltip
            className="cursor-pointer"
            title="Remote directory path for storing backups (optional)"
          >
            <InfoCircleOutlined className="ml-2" style={{ color: 'gray' }} />
          </Tooltip>
        </div>
      </div>

      <div className="mb-1 flex w-full flex-col items-start sm:flex-row sm:items-center">
        <div className="mb-1 min-w-[110px] sm:mb-0">Use SSL/TLS</div>
        <div className="flex items-center">
          <Checkbox
            checked={storage?.ftpStorage?.useSsl || false}
            onChange={(e) => {
              if (!storage?.ftpStorage) return;

              setStorage({
                ...storage,
                ftpStorage: {
                  ...storage.ftpStorage,
                  useSsl: e.target.checked,
                },
              });
              setUnsaved();
            }}
          >
            Enable FTPS
          </Checkbox>

          <Tooltip
            className="cursor-pointer"
            title="Use explicit TLS encryption (FTPS) for secure file transfer"
          >
            <InfoCircleOutlined className="ml-2" style={{ color: 'gray' }} />
          </Tooltip>
        </div>
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
          {storage?.ftpStorage?.useSsl && (
            <div className="mb-1 flex w-full flex-col items-start sm:flex-row sm:items-center">
              <div className="mb-1 min-w-[110px] sm:mb-0">Skip TLS verify</div>
              <div className="flex items-center">
                <Checkbox
                  checked={storage?.ftpStorage?.skipTlsVerify || false}
                  onChange={(e) => {
                    if (!storage?.ftpStorage) return;

                    setStorage({
                      ...storage,
                      ftpStorage: {
                        ...storage.ftpStorage,
                        skipTlsVerify: e.target.checked,
                      },
                    });
                    setUnsaved();
                  }}
                >
                  Skip certificate verification
                </Checkbox>

                <Tooltip
                  className="cursor-pointer"
                  title="Skip TLS certificate verification. Enable this if your FTP server uses a self-signed certificate. Warning: this reduces security."
                >
                  <InfoCircleOutlined className="ml-2" style={{ color: 'gray' }} />
                </Tooltip>
              </div>
            </div>
          )}
        </>
      )}

      <div className="mb-5" />
    </>
  );
}
