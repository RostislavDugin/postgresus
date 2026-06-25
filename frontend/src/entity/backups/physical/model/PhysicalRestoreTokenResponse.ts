export interface PhysicalRestoreTokenResponse {
  // single-use, short-TTL token; append to the restore-stream URL to download the
  // ready-to-restore tar
  token: string;
  url: string;
}
