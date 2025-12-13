import { type Storage, StorageType } from '../../../../entity/storages';
import { getStorageLogoFromType } from '../../../../entity/storages/models/getStorageLogoFromType';
import { getStorageNameFromType } from '../../../../entity/storages/models/getStorageNameFromType';
import { ShowAzureBlobStorageComponent } from './storages/ShowAzureBlobStorageComponent';
import { ShowFTPStorageComponent } from './storages/ShowFTPStorageComponent';
import { ShowGoogleDriveStorageComponent } from './storages/ShowGoogleDriveStorageComponent';
import { ShowNASStorageComponent } from './storages/ShowNASStorageComponent';
import { ShowS3StorageComponent } from './storages/ShowS3StorageComponent';

interface Props {
  storage?: Storage;
}

export function ShowStorageComponent({ storage }: Props) {
  if (!storage) return null;

  return (
    <div>
      <div className="mb-1 flex items-center">
        <div className="min-w-[110px]">Type</div>

        {getStorageNameFromType(storage.type)}

        <img
          src={getStorageLogoFromType(storage.type)}
          alt="storageIcon"
          className="ml-1 h-4 w-4"
        />
      </div>

      <div>{storage?.type === StorageType.S3 && <ShowS3StorageComponent storage={storage} />}</div>

      <div>
        {storage?.type === StorageType.GOOGLE_DRIVE && (
          <ShowGoogleDriveStorageComponent storage={storage} />
        )}
      </div>

      <div>
        {storage?.type === StorageType.NAS && <ShowNASStorageComponent storage={storage} />}
      </div>

      <div>
        {storage?.type === StorageType.AZURE_BLOB && (
          <ShowAzureBlobStorageComponent storage={storage} />
        )}
      </div>

      <div>
        {storage?.type === StorageType.FTP && <ShowFTPStorageComponent storage={storage} />}
      </div>
    </div>
  );
}
