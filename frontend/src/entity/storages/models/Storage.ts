import type { AzureBlobStorage } from './AzureBlobStorage';
import type { FTPStorage } from './FTPStorage';
import type { GoogleDriveStorage } from './GoogleDriveStorage';
import type { LocalStorage } from './LocalStorage';
import type { NASStorage } from './NASStorage';
import type { RcloneStorage } from './RcloneStorage';
import type { S3Storage } from './S3Storage';
import type { SFTPStorage } from './SFTPStorage';
import type { StorageType } from './StorageType';

export interface Storage {
  id: string;
  type: StorageType;
  name: string;
  lastSaveError?: string;
  workspaceId: string;

  // specific storage types
  localStorage?: LocalStorage;
  s3Storage?: S3Storage;
  googleDriveStorage?: GoogleDriveStorage;
  nasStorage?: NASStorage;
  azureBlobStorage?: AzureBlobStorage;
  ftpStorage?: FTPStorage;
  sftpStorage?: SFTPStorage;
  rcloneStorage?: RcloneStorage;
}
