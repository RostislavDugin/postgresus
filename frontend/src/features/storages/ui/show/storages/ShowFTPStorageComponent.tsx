import type { Storage } from '../../../../../entity/storages';

interface Props {
  storage: Storage;
}

export function ShowFTPStorageComponent({ storage }: Props) {
  return (
    <>
      <div className="mb-1 flex items-center">
        <div className="min-w-[110px]">Host</div>
        {storage?.ftpStorage?.host || '-'}
      </div>

      <div className="mb-1 flex items-center">
        <div className="min-w-[110px]">Port</div>
        {storage?.ftpStorage?.port || '-'}
      </div>

      <div className="mb-1 flex items-center">
        <div className="min-w-[110px]">Username</div>
        {storage?.ftpStorage?.username || '-'}
      </div>

      <div className="mb-1 flex items-center">
        <div className="min-w-[110px]">Password</div>
        {'*************'}
      </div>

      <div className="mb-1 flex items-center">
        <div className="min-w-[110px]">Path</div>
        {storage?.ftpStorage?.path || '-'}
      </div>

      <div className="mb-1 flex items-center">
        <div className="min-w-[110px]">Use SSL/TLS</div>
        {storage?.ftpStorage?.useSsl ? 'Yes' : 'No'}
      </div>

      {storage?.ftpStorage?.useSsl && storage?.ftpStorage?.skipTlsVerify && (
        <div className="mb-1 flex items-center">
          <div className="min-w-[110px]">Skip TLS</div>
          Enabled
        </div>
      )}
    </>
  );
}
