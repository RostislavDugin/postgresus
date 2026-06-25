// Adapted from wal-g/internal/databases/postgres/timeline.go.
// Copyright 2017 Citus Data Inc. Licensed under the Apache License, Version 2.0.

package walmath

import (
	"fmt"
	"regexp"
	"strconv"
)

var regexpTimelineAndLogSegNo = regexp.MustCompile(PatternTimelineAndLogSegNo)

func ParseWALFilename(name string) (timelineID uint32, logSegNo uint64, err error) {
	return ParseWALFilenameWithSize(name, WalSegmentSize)
}

func ParseWALFilenameWithSize(name string, segmentSize uint64) (timelineID uint32, logSegNo uint64, err error) {
	if len(name) != 24 {
		return 0, 0, NotWalFilenameError{Filename: name}
	}

	if segmentSize == 0 || 0x100000000%segmentSize != 0 {
		return 0, 0, IncorrectLogSegNoError{Filename: name}
	}

	timelineID64, err := strconv.ParseUint(name[0:8], 0x10, hexUint32Bits)
	if err != nil {
		return 0, 0, NotWalFilenameError{Filename: name}
	}

	logSegNoHi, err := strconv.ParseUint(name[8:16], 0x10, hexUint32Bits)
	if err != nil {
		return 0, 0, NotWalFilenameError{Filename: name}
	}

	logSegNoLo, err := strconv.ParseUint(name[16:24], 0x10, hexUint32Bits)
	if err != nil {
		return 0, 0, NotWalFilenameError{Filename: name}
	}

	segmentsPerLogID := 0x100000000 / segmentSize
	if logSegNoLo >= segmentsPerLogID {
		return 0, 0, IncorrectLogSegNoError{Filename: name}
	}

	timelineID = uint32(timelineID64)
	logSegNo = logSegNoHi*segmentsPerLogID + logSegNoLo

	return timelineID, logSegNo, nil
}

func formatWALFileName(timeline uint32, logSegNo uint64) string {
	return fmt.Sprintf(walFileFormat, timeline, logSegNo/xLogSegmentsPerXLogID, logSegNo%xLogSegmentsPerXLogID)
}

func GetNextWalFilename(name string) (string, error) {
	timelineID, logSegNo, err := ParseWALFilename(name)
	if err != nil {
		return "", err
	}

	return formatWALFileName(timelineID, logSegNo+1), nil
}

func TryFetchTimelineAndLogSegNo(objectName string) (uint32, uint64, bool) {
	foundLsn := regexpTimelineAndLogSegNo.FindAllString(objectName, maxCountOfLSN)
	if len(foundLsn) > 0 {
		timelineID, logSegNo, err := ParseWALFilename(foundLsn[0])
		if err == nil {
			return timelineID, logSegNo, true
		}
	}

	return 0, 0, false
}

func IsWalFilename(filename string) bool {
	_, _, err := ParseWALFilename(filename)
	return err == nil
}
