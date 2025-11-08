import type { Storage } from '../../../../../entity/storages';

interface Props {
  storage: Storage;
}

export function ShowGoogleDriveStorageComponent({ storage }: Props) {
  return (
    <>
      <div className="mb-1 flex items-center">
        <div className="min-w-[110px]">Client ID</div>
        {storage?.googleDriveStorage?.clientId
          ? `${storage?.googleDriveStorage?.clientId.slice(0, 10)}***`
          : '-'}
      </div>

      <div className="mb-1 flex items-center">
        <div className="min-w-[110px]">Client Secret</div>
        {`*************`}
      </div>

      <div className="mb-1 flex items-center">
        <div className="min-w-[110px]">User Token</div>
        {`*************`}
      </div>
    </>
  );
}
