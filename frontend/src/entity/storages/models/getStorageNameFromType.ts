import { StorageType } from './StorageType';

export const getStorageNameFromType = (type: StorageType) => {
  switch (type) {
    case StorageType.LOCAL:
      return 'local storage';
    case StorageType.S3:
      return 'S3';
    case StorageType.GOOGLE_DRIVE:
      return 'Google Drive';
    case StorageType.NAS:
      return 'NAS';
    case StorageType.AZURE_BLOB:
      return 'Azure Blob Storage';
    case StorageType.FTP:
      return 'FTP';
    default:
      return '';
  }
};
