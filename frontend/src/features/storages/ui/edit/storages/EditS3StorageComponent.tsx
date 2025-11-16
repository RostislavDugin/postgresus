import { DownOutlined, InfoCircleOutlined, UpOutlined } from '@ant-design/icons';
import { Checkbox, Input, Tooltip } from 'antd';
import { useState } from 'react';

import type { Storage } from '../../../../../entity/storages';

interface Props {
  storage: Storage;
  setStorage: (storage: Storage) => void;
  setUnsaved: () => void;
}

export function EditS3StorageComponent({ storage, setStorage, setUnsaved }: Props) {
  const hasAdvancedValues =
    !!storage?.s3Storage?.s3Prefix || !!storage?.s3Storage?.s3UseVirtualHostedStyle;
  const [showAdvanced, setShowAdvanced] = useState(hasAdvancedValues);

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
            setUnsaved();
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
            setUnsaved();
          }}
          size="small"
          className="w-full max-w-[250px]"
          placeholder="us-east-1"
        />
      </div>

      <div className="mb-1 flex items-center">
        <div className="min-w-[110px]">Access key</div>
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
            setUnsaved();
          }}
          size="small"
          className="w-full max-w-[250px]"
          placeholder="AKIAIOSFODNN7EXAMPLE"
        />
      </div>

      <div className="mb-1 flex items-center">
        <div className="min-w-[110px]">Secret key</div>
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
            setUnsaved();
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
            setUnsaved();
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
          <div className="mb-1 flex items-center">
            <div className="min-w-[110px]">Folder prefix</div>
            <Input
              value={storage?.s3Storage?.s3Prefix || ''}
              onChange={(e) => {
                if (!storage?.s3Storage) return;

                setStorage({
                  ...storage,
                  s3Storage: {
                    ...storage.s3Storage,
                    s3Prefix: e.target.value.trim(),
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
              title="Optional prefix for all object keys (e.g., 'backups/' or 'my_team/'). May not work with some S3-compatible storages."
            >
              <InfoCircleOutlined className="ml-2" style={{ color: 'gray' }} />
            </Tooltip>
          </div>

          <div className="mb-1 flex items-center">
            <div className="min-w-[110px]">Virtual host</div>

            <Checkbox
              checked={storage?.s3Storage?.s3UseVirtualHostedStyle || false}
              onChange={(e) => {
                if (!storage?.s3Storage) return;

                setStorage({
                  ...storage,
                  s3Storage: {
                    ...storage.s3Storage,
                    s3UseVirtualHostedStyle: e.target.checked,
                  },
                });
                setUnsaved();
              }}
            >
              Use virtual-styled domains
            </Checkbox>

            <Tooltip
              className="cursor-pointer"
              title="Use virtual-hosted-style URLs (bucket.s3.region.amazonaws.com) instead of path-style (s3.region.amazonaws.com/bucket). May be required if you see COS errors."
            >
              <InfoCircleOutlined className="ml-2" style={{ color: 'gray' }} />
            </Tooltip>
          </div>
        </>
      )}

      <div className="mb-5" />
    </>
  );
}
