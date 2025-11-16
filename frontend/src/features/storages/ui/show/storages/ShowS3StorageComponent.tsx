import type { Storage } from '../../../../../entity/storages';

interface Props {
  storage: Storage;
}

export function ShowS3StorageComponent({ storage }: Props) {
  return (
    <>
      <div className="mb-1 flex items-center">
        <div className="min-w-[110px]">S3 Bucket</div>
        {storage?.s3Storage?.s3Bucket}
      </div>

      <div className="mb-1 flex items-center">
        <div className="min-w-[110px]">Region</div>
        {storage?.s3Storage?.s3Region || '-'}
      </div>

      <div className="mb-1 flex items-center">
        <div className="min-w-[110px]">Access Key</div>
        {'*************'}
      </div>

      <div className="mb-1 flex items-center">
        <div className="min-w-[110px]">Secret Key</div>
        {'*************'}
      </div>

      <div className="mb-1 flex items-center">
        <div className="min-w-[110px]">Endpoint</div>
        {storage?.s3Storage?.s3Endpoint || '-'}
      </div>

      {storage?.s3Storage?.s3Prefix && (
        <div className="mb-1 flex items-center">
          <div className="min-w-[110px]">Prefix</div>
          {storage.s3Storage.s3Prefix}
        </div>
      )}

      {storage?.s3Storage?.s3UseVirtualHostedStyle && (
        <div className="mb-1 flex items-center">
          <div className="min-w-[110px]">Virtual Host</div>
          Enabled
        </div>
      )}
    </>
  );
}
