package chain_view

import (
	physical_repositories "databasus-backend/internal/features/backups/backups/core/physical/repositories"
)

var chainViewService = &ChainViewService{
	physical_repositories.GetFullBackupRepository(),
	physical_repositories.GetIncrementalBackupRepository(),
	physical_repositories.GetWalSegmentRepository(),
	physical_repositories.GetWalHistoryRepository(),
}

func GetChainViewService() *ChainViewService {
	return chainViewService
}
