import { InfoCircleOutlined } from '@ant-design/icons';
import { Input, Tooltip } from 'antd';

import type { Storage } from '../../../../../entity/storages';

interface Props {
  storage: Storage;
  setStorage: (storage: Storage) => void;
  setUnsaved: () => void;
}

export function EditRcloneStorageComponent({ storage, setStorage, setUnsaved }: Props) {
  return (
    <>
      <div className="mb-1 flex w-full flex-col items-start sm:flex-row sm:items-start">
        <div className="mb-1 min-w-[110px] pt-1 sm:mb-0">Config</div>
        <div className="flex w-full flex-col">
          <div className="flex items-start">
            <Input.TextArea
              value={storage?.rcloneStorage?.configContent || ''}
              onChange={(e) => {
                if (!storage?.rcloneStorage) return;

                setStorage({
                  ...storage,
                  rcloneStorage: {
                    ...storage.rcloneStorage,
                    configContent: e.target.value,
                  },
                });
                setUnsaved();
              }}
              className="w-full max-w-[400px] font-mono text-xs"
              placeholder={`[myremote]
type = s3
provider = AWS
access_key_id = YOUR_ACCESS_KEY
secret_access_key = YOUR_SECRET_KEY
region = us-east-1`}
              rows={8}
              style={{ resize: 'vertical' }}
            />

            <Tooltip
              className="cursor-pointer"
              title="Paste your rclone.conf content here. You can get it by running 'rclone config file' and copying the contents. This config supports 70+ cloud storage providers."
            >
              <InfoCircleOutlined className="mt-2 ml-2" style={{ color: 'gray' }} />
            </Tooltip>
          </div>
        </div>
      </div>

      {!storage?.id && (
        <div className="mb-2 flex items-center">
          <div className="hidden min-w-[110px] sm:block" />

          <div className="max-w-[300px] text-xs text-gray-400">
            *content is hidden to not expose sensitive data. If you want to update existing config,
            put a new one here
          </div>
        </div>
      )}

      <div className="mb-2 flex items-center">
        <div className="hidden min-w-[110px] sm:block" />

        <div className="text-xs text-blue-600">
          <a href="https://rclone.org/docs/" target="_blank" rel="noreferrer">
            Rclone documentation
          </a>
        </div>
      </div>

      <div className="mb-1 flex w-full flex-col items-start sm:flex-row sm:items-center">
        <div className="mb-1 min-w-[110px] sm:mb-0">Remote path</div>
        <div className="flex items-center">
          <Input
            value={storage?.rcloneStorage?.remotePath || ''}
            onChange={(e) => {
              if (!storage?.rcloneStorage) return;

              setStorage({
                ...storage,
                rcloneStorage: {
                  ...storage.rcloneStorage,
                  remotePath: e.target.value.trim(),
                },
              });
              setUnsaved();
            }}
            size="small"
            className="w-full max-w-[250px]"
            placeholder="/backups (optional)"
          />

          <Tooltip
            className="cursor-pointer"
            title="Optional path prefix on the remote where backups will be stored (e.g., '/backups' or 'my-folder/backups')"
          >
            <InfoCircleOutlined className="ml-2" style={{ color: 'gray' }} />
          </Tooltip>
        </div>
      </div>

      <div className="mb-5" />
    </>
  );
}
