package physical_service

import (
	physical_repositories "databasus-backend/internal/features/backups/backups/core/physical/repositories"
	"databasus-backend/internal/features/storages"
	"databasus-backend/internal/util/encryption"
	"databasus-backend/internal/util/logger"
)

var physicalBackupService = &PhysicalBackupService{
	physical_repositories.GetFullBackupRepository(),
	physical_repositories.GetWalSegmentRepository(),
	storages.GetStorageService(),
	encryption.GetFieldEncryptor(),
	logger.GetLogger(),
}

func GetPhysicalBackupService() *PhysicalBackupService {
	return physicalBackupService
}
