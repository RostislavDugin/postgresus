import type { Storage } from '../../../../../entity/storages';

interface Props {
  storage: Storage;
}

export function ShowAzureBlobStorageComponent({ storage }: Props) {
  return (
    <>
      <div className="mb-1 flex items-center">
        <div className="min-w-[110px]">Auth method</div>
        {storage?.azureBlobStorage?.authMethod === 'CONNECTION_STRING'
          ? 'Connection string'
          : 'Account key'}
      </div>

      {storage?.azureBlobStorage?.authMethod === 'CONNECTION_STRING' && (
        <div className="mb-1 flex items-center">
          <div className="min-w-[110px]">Connection string</div>
          {'*************'}
        </div>
      )}

      {storage?.azureBlobStorage?.authMethod === 'ACCOUNT_KEY' && (
        <>
          <div className="mb-1 flex items-center">
            <div className="min-w-[110px]">Account name</div>
            {storage?.azureBlobStorage?.accountName || '-'}
          </div>

          <div className="mb-1 flex items-center">
            <div className="min-w-[110px]">Account key</div>
            {'*************'}
          </div>

          <div className="mb-1 flex items-center">
            <div className="min-w-[110px]">Endpoint</div>
            {storage?.azureBlobStorage?.endpoint || '-'}
          </div>
        </>
      )}

      <div className="mb-1 flex items-center">
        <div className="min-w-[110px]">Container name</div>
        {storage?.azureBlobStorage?.containerName || '-'}
      </div>

      {storage?.azureBlobStorage?.prefix && (
        <div className="mb-1 flex items-center">
          <div className="min-w-[110px]">Prefix</div>
          {storage.azureBlobStorage.prefix}
        </div>
      )}
    </>
  );
}
