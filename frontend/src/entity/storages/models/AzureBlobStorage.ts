export interface AzureBlobStorage {
  authMethod: 'CONNECTION_STRING' | 'ACCOUNT_KEY';
  connectionString: string;
  accountName: string;
  accountKey: string;
  containerName: string;
  endpoint?: string;
  prefix?: string;
}
