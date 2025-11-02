import type { GoogleDriveStorage } from './GoogleDriveStorage';
import type { LocalStorage } from './LocalStorage';
import type { NASStorage } from './NASStorage';
import type { S3Storage } from './S3Storage';
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
}
