package physical_service

import (
	"databasus-backend/internal/features/backups/backups/core/physical/chain_view"
	physical_repositories "databasus-backend/internal/features/backups/backups/core/physical/repositories"
	"databasus-backend/internal/features/storages"
	"databasus-backend/internal/util/encryption"
	"databasus-backend/internal/util/logger"
)

var physicalBackupService = &PhysicalBackupService{
	physical_repositories.GetFullBackupRepository(),
	physical_repositories.GetIncrementalBackupRepository(),
	physical_repositories.GetWalSegmentRepository(),
	physical_repositories.GetWalHistoryRepository(),
	chain_view.GetChainViewService(),
	storages.GetStorageService(),
	encryption.GetFieldEncryptor(),
	logger.GetLogger(),
}

func GetPhysicalBackupService() *PhysicalBackupService {
	return physicalBackupService
}
