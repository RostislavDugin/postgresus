package physical_repositories

var (
	fullBackupRepository        = &PhysicalFullBackupRepository{}
	incrementalBackupRepository = &PhysicalIncrementalBackupRepository{}
	inFlightBackupRepository    = &PhysicalInFlightBackupRepository{}
	walSegmentRepository        = &PhysicalWalSegmentRepository{}
	walHistoryRepository        = &PhysicalWalHistoryRepository{}
	walStreamerRepository       = &PhysicalWalStreamerRepository{}
)

func GetFullBackupRepository() *PhysicalFullBackupRepository {
	return fullBackupRepository
}

func GetIncrementalBackupRepository() *PhysicalIncrementalBackupRepository {
	return incrementalBackupRepository
}

func GetInFlightBackupRepository() *PhysicalInFlightBackupRepository {
	return inFlightBackupRepository
}

func GetWalSegmentRepository() *PhysicalWalSegmentRepository {
	return walSegmentRepository
}

func GetWalHistoryRepository() *PhysicalWalHistoryRepository {
	return walHistoryRepository
}

func GetWalStreamerRepository() *PhysicalWalStreamerRepository {
	return walStreamerRepository
}
