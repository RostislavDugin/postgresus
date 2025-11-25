import { InfoCircleOutlined } from '@ant-design/icons';
import { Input, InputNumber, Switch, Tooltip } from 'antd';

import type { Storage } from '../../../../../entity/storages';

interface Props {
  storage: Storage;
  setStorage: (storage: Storage) => void;
  setUnsaved: () => void;
}

export function EditNASStorageComponent({ storage, setStorage, setUnsaved }: Props) {
  return (
    <>
      <div className="mb-1 flex w-full flex-col items-start sm:flex-row sm:items-center">
        <div className="mb-1 min-w-[110px] sm:mb-0">Host</div>
        <Input
          value={storage?.nasStorage?.host || ''}
          onChange={(e) => {
            if (!storage?.nasStorage) return;

            setStorage({
              ...storage,
              nasStorage: {
                ...storage.nasStorage,
                host: e.target.value.trim(),
              },
            });
            setUnsaved();
          }}
          size="small"
          className="w-full max-w-[250px]"
          placeholder="192.168.1.100"
        />
      </div>

      <div className="mb-1 flex w-full flex-col items-start sm:flex-row sm:items-center">
        <div className="mb-1 min-w-[110px] sm:mb-0">Port</div>
        <InputNumber
          value={storage?.nasStorage?.port}
          onChange={(value) => {
            if (!storage?.nasStorage || !value) return;

            setStorage({
              ...storage,
              nasStorage: {
                ...storage.nasStorage,
                port: value,
              },
            });
            setUnsaved();
          }}
          size="small"
          className="w-full max-w-[250px]"
          min={1}
          max={65535}
          placeholder="445"
        />
      </div>

      <div className="mb-1 flex w-full flex-col items-start sm:flex-row sm:items-center">
        <div className="mb-1 min-w-[110px] sm:mb-0">Share</div>
        <Input
          value={storage?.nasStorage?.share || ''}
          onChange={(e) => {
            if (!storage?.nasStorage) return;

            setStorage({
              ...storage,
              nasStorage: {
                ...storage.nasStorage,
                share: e.target.value.trim(),
              },
            });
            setUnsaved();
          }}
          size="small"
          className="w-full max-w-[250px]"
          placeholder="shared_folder"
        />
      </div>

      <div className="mb-1 flex w-full flex-col items-start sm:flex-row sm:items-center">
        <div className="mb-1 min-w-[110px] sm:mb-0">Username</div>
        <Input
          value={storage?.nasStorage?.username || ''}
          onChange={(e) => {
            if (!storage?.nasStorage) return;

            setStorage({
              ...storage,
              nasStorage: {
                ...storage.nasStorage,
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
          value={storage?.nasStorage?.password || ''}
          onChange={(e) => {
            if (!storage?.nasStorage) return;

            setStorage({
              ...storage,
              nasStorage: {
                ...storage.nasStorage,
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
        <div className="mb-1 min-w-[110px] sm:mb-0">Use SSL</div>
        <div className="flex items-center">
          <Switch
            checked={storage?.nasStorage?.useSsl || false}
            onChange={(checked) => {
              if (!storage?.nasStorage) return;

              setStorage({
                ...storage,
                nasStorage: {
                  ...storage.nasStorage,
                  useSsl: checked,
                },
              });
              setUnsaved();
            }}
            size="small"
          />

          <Tooltip
            className="cursor-pointer"
            title="Enable SSL/TLS encryption for secure connection"
          >
            <InfoCircleOutlined className="ml-2" style={{ color: 'gray' }} />
          </Tooltip>
        </div>
      </div>

      <div className="mb-1 flex w-full flex-col items-start sm:flex-row sm:items-center">
        <div className="mb-1 min-w-[110px] sm:mb-0">Domain</div>
        <div className="flex items-center">
          <Input
            value={storage?.nasStorage?.domain || ''}
            onChange={(e) => {
              if (!storage?.nasStorage) return;

              setStorage({
                ...storage,
                nasStorage: {
                  ...storage.nasStorage,
                  domain: e.target.value.trim() || undefined,
                },
              });
              setUnsaved();
            }}
            size="small"
            className="w-full max-w-[250px]"
            placeholder="WORKGROUP (optional)"
          />

          <Tooltip
            className="cursor-pointer"
            title="Windows domain name (optional, leave empty if not using domain authentication)"
          >
            <InfoCircleOutlined className="ml-2" style={{ color: 'gray' }} />
          </Tooltip>
        </div>
      </div>

      <div className="mb-1 flex w-full flex-col items-start sm:flex-row sm:items-center">
        <div className="mb-1 min-w-[110px] sm:mb-0">Path</div>
        <div className="flex items-center">
          <Input
            value={storage?.nasStorage?.path || ''}
            onChange={(e) => {
              if (!storage?.nasStorage) return;

              let pathValue = e.target.value.trim();
              // Remove leading slash if present
              if (pathValue.startsWith('/')) {
                pathValue = pathValue.substring(1);
              }

              setStorage({
                ...storage,
                nasStorage: {
                  ...storage.nasStorage,
                  path: pathValue || undefined,
                },
              });
              setUnsaved();
            }}
            size="small"
            className="w-full max-w-[250px]"
            placeholder="backups (optional, no leading slash)"
          />

          <Tooltip className="cursor-pointer" title="Subdirectory path within the share (optional)">
            <InfoCircleOutlined className="ml-2" style={{ color: 'gray' }} />
          </Tooltip>
        </div>
      </div>
    </>
  );
}
