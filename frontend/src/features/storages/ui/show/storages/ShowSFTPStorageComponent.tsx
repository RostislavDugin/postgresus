import type { Storage } from '../../../../../entity/storages';

interface Props {
  storage: Storage;
}

export function ShowSFTPStorageComponent({ storage }: Props) {
  const authMethod = storage?.sftpStorage?.privateKey ? 'Private Key' : 'Password';

  return (
    <>
      <div className="mb-1 flex items-center">
        <div className="min-w-[110px]">Host</div>
        {storage?.sftpStorage?.host || '-'}
      </div>

      <div className="mb-1 flex items-center">
        <div className="min-w-[110px]">Port</div>
        {storage?.sftpStorage?.port || '-'}
      </div>

      <div className="mb-1 flex items-center">
        <div className="min-w-[110px]">Username</div>
        {storage?.sftpStorage?.username || '-'}
      </div>

      <div className="mb-1 flex items-center">
        <div className="min-w-[110px]">Auth Method</div>
        {authMethod}
      </div>

      <div className="mb-1 flex items-center">
        <div className="min-w-[110px]">Credentials</div>
        {'*************'}
      </div>

      <div className="mb-1 flex items-center">
        <div className="min-w-[110px]">Path</div>
        {storage?.sftpStorage?.path || '-'}
      </div>

      {storage?.sftpStorage?.skipHostKeyVerify && (
        <div className="mb-1 flex items-center">
          <div className="min-w-[110px]">Skip host key</div>
          Enabled
        </div>
      )}
    </>
  );
}
