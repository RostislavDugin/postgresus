import type { Storage } from '../../../../../entity/storages';

interface Props {
  storage: Storage;
}

export function ShowNASStorageComponent({ storage }: Props) {
  return (
    <>
      <div className="mb-1 flex items-center">
        <div className="min-w-[110px]">Host</div>
        {storage?.nasStorage?.host || '-'}
      </div>

      <div className="mb-1 flex items-center">
        <div className="min-w-[110px]">Port</div>
        {storage?.nasStorage?.port || '-'}
      </div>

      <div className="mb-1 flex items-center">
        <div className="min-w-[110px]">Share</div>
        {storage?.nasStorage?.share || '-'}
      </div>

      <div className="mb-1 flex items-center">
        <div className="min-w-[110px]">Username</div>
        {storage?.nasStorage?.username || '-'}
      </div>

      <div className="mb-1 flex items-center">
        <div className="min-w-[110px]">Password</div>
        {'*************'}
      </div>

      <div className="mb-1 flex items-center">
        <div className="min-w-[110px]">Use SSL</div>
        {storage?.nasStorage?.useSsl ? 'Yes' : 'No'}
      </div>

      <div className="mb-1 flex items-center">
        <div className="min-w-[110px]">Domain</div>
        {storage?.nasStorage?.domain || '-'}
      </div>

      <div className="mb-1 flex items-center">
        <div className="min-w-[110px]">Path</div>
        {storage?.nasStorage?.path || '-'}
      </div>
    </>
  );
}
