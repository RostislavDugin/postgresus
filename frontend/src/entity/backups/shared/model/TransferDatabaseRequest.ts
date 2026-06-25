export interface TransferDatabaseRequest {
  targetWorkspaceId: string;
  targetStorageId?: string;
  isTransferWithStorage: boolean;
  isTransferWithNotifiers: boolean;
  targetNotifierIds: string[];
}
