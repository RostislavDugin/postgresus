export interface S3Storage {
  s3Bucket: string;
  s3Region: string;
  s3AccessKey: string;
  s3SecretKey: string;
  s3Endpoint?: string;
  s3Prefix?: string;
  s3UseVirtualHostedStyle?: boolean;
  skipTLSVerify?: boolean;
}
