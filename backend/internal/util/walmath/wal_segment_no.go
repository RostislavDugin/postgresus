// Adapted from wal-g/internal/databases/postgres/wal_segment_no.go
// and wal-g/internal/databases/postgres/wal_segment_runner.go (WalSegmentDescription only).
// Copyright 2017 Citus Data Inc. Licensed under the Apache License, Version 2.0.

package walmath

import "fmt"

type WalSegmentNo uint64

func NewWalSegmentNo(lsn LSN) WalSegmentNo {
	return WalSegmentNo(GetSegmentNoFromLsn(lsn))
}

func GetSegmentNoFromLsn(lsn LSN) uint64 {
	return uint64(lsn) / WalSegmentSize
}

func (walSegmentNo WalSegmentNo) Next() WalSegmentNo {
	return walSegmentNo.add(1)
}

func (walSegmentNo WalSegmentNo) Previous() WalSegmentNo {
	return walSegmentNo.sub(1)
}

func (walSegmentNo WalSegmentNo) FirstLSN() LSN {
	return LSN(uint64(walSegmentNo) * WalSegmentSize)
}

func (walSegmentNo WalSegmentNo) GetFilename(timeline uint32) string {
	return fmt.Sprintf(walFileFormat,
		timeline,
		uint64(walSegmentNo)/xLogSegmentsPerXLogID,
		uint64(walSegmentNo)%xLogSegmentsPerXLogID,
	)
}

func (walSegmentNo WalSegmentNo) add(n uint64) WalSegmentNo {
	return WalSegmentNo(uint64(walSegmentNo) + n)
}

func (walSegmentNo WalSegmentNo) sub(n uint64) WalSegmentNo {
	return WalSegmentNo(uint64(walSegmentNo) - n)
}

type WalSegmentDescription struct {
	Number   WalSegmentNo
	Timeline uint32
}

func NewWalSegmentDescription(name string) (WalSegmentDescription, error) {
	timeline, segmentNo, err := ParseWALFilename(name)
	if err != nil {
		return WalSegmentDescription{}, err
	}

	return WalSegmentDescription{Timeline: timeline, Number: WalSegmentNo(segmentNo)}, nil
}

func (desc WalSegmentDescription) GetFileName() string {
	return desc.Number.GetFilename(desc.Timeline)
}
