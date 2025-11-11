import { InfoCircleOutlined } from '@ant-design/icons';
import { Input, Tooltip } from 'antd';

import type { Storage } from '../../../../../entity/storages';

interface Props {
  storage: Storage;
  setStorage: (storage: Storage) => void;
  setIsUnsaved: (isUnsaved: boolean) => void;
}

export function EditS3StorageComponent({ storage, setStorage, setIsUnsaved }: Props) {
  return (
    <>
      <div className="mb-2 flex items-center">
        <div className="min-w-[110px]" />

        <div className="text-xs text-blue-600">
          <a href="https://postgresus.com/storages/cloudflare-r2" target="_blank" rel="noreferrer">
            How to use with Cloudflare R2?
          </a>
        </div>
      </div>

      <div className="mb-1 flex items-center">
        <div className="min-w-[110px]">S3 Bucket</div>
        <Input
          value={storage?.s3Storage?.s3Bucket || ''}
          onChange={(e) => {
            if (!storage?.s3Storage) return;

            setStorage({
              ...storage,
              s3Storage: {
                ...storage.s3Storage,
                s3Bucket: e.target.value.trim(),
              },
            });
            setIsUnsaved(true);
          }}
          size="small"
          className="w-full max-w-[250px]"
          placeholder="my-bucket-name"
        />
      </div>

      <div className="mb-1 flex items-center">
        <div className="min-w-[110px]">Region</div>
        <Input
          value={storage?.s3Storage?.s3Region || ''}
          onChange={(e) => {
            if (!storage?.s3Storage) return;

            setStorage({
              ...storage,
              s3Storage: {
                ...storage.s3Storage,
                s3Region: e.target.value.trim(),
              },
            });
            setIsUnsaved(true);
          }}
          size="small"
          className="w-full max-w-[250px]"
          placeholder="us-east-1"
        />
      </div>

      <div className="mb-1 flex items-center">
        <div className="min-w-[110px]">Access Key</div>
        <Input.Password
          value={storage?.s3Storage?.s3AccessKey || ''}
          onChange={(e) => {
            if (!storage?.s3Storage) return;

            setStorage({
              ...storage,
              s3Storage: {
                ...storage.s3Storage,
                s3AccessKey: e.target.value.trim(),
              },
            });
            setIsUnsaved(true);
          }}
          size="small"
          className="w-full max-w-[250px]"
          placeholder="AKIAIOSFODNN7EXAMPLE"
        />
      </div>

      <div className="mb-1 flex items-center">
        <div className="min-w-[110px]">Secret Key</div>
        <Input.Password
          value={storage?.s3Storage?.s3SecretKey || ''}
          onChange={(e) => {
            if (!storage?.s3Storage) return;

            setStorage({
              ...storage,
              s3Storage: {
                ...storage.s3Storage,
                s3SecretKey: e.target.value.trim(),
              },
            });
            setIsUnsaved(true);
          }}
          size="small"
          className="w-full max-w-[250px]"
          placeholder="wJalrXUtnFEMI/K7MDENG/bPxRfiCYEXAMPLEKEY"
        />
      </div>

      <div className="mb-1 flex items-center">
        <div className="min-w-[110px]">Endpoint</div>
        <Input
          value={storage?.s3Storage?.s3Endpoint || ''}
          onChange={(e) => {
            if (!storage?.s3Storage) return;

            setStorage({
              ...storage,
              s3Storage: {
                ...storage.s3Storage,
                s3Endpoint: e.target.value.trim(),
              },
            });
            setIsUnsaved(true);
          }}
          size="small"
          className="w-full max-w-[250px]"
          placeholder="https://s3.example.com (optional)"
        />

        <Tooltip
          className="cursor-pointer"
          title="Custom S3-compatible endpoint URL (optional, leave empty for AWS S3)"
        >
          <InfoCircleOutlined className="ml-2" style={{ color: 'gray' }} />
        </Tooltip>
      </div>
    </>
  );
}
