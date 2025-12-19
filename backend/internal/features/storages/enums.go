package storages

type StorageType string

const (
	StorageTypeLocal       StorageType = "LOCAL"
	StorageTypeS3          StorageType = "S3"
	StorageTypeGoogleDrive StorageType = "GOOGLE_DRIVE"
	StorageTypeNAS         StorageType = "NAS"
	StorageTypeAzureBlob   StorageType = "AZURE_BLOB"
	StorageTypeFTP         StorageType = "FTP"
	StorageTypeSFTP        StorageType = "SFTP"
	StorageTypeRclone      StorageType = "RCLONE"
)
