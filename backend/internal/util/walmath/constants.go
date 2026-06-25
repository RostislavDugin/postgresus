package walmath

import "fmt"

var (
	WalSegmentSize = uint64(16 * 1024 * 1024)

	xLogSegmentsPerXLogID = 0x100000000 / WalSegmentSize
)

const (
	PatternTimelineAndLogSegNo = "[0-9A-F]{24}"

	walFileFormat        = "%08X%08X%08X"
	walHistoryFileFormat = "%08X.history"

	maxCountOfLSN = 2
	hexUint32Bits = 32
)

func SetWalSize(sizeMb uint64) {
	WalSegmentSize = sizeMb * 1024 * 1024
	xLogSegmentsPerXLogID = 0x100000000 / WalSegmentSize
}

func FormatHistoryFilename(timeline uint32) string {
	return fmt.Sprintf(walHistoryFileFormat, timeline)
}
