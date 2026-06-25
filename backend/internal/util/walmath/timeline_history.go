// Adapted from wal-g/internal/databases/postgres/timeline_history.go
// and wal-g/internal/databases/postgres/timeline_history_record.go.
// Copyright 2017 Citus Data Inc. Licensed under the Apache License, Version 2.0.

package walmath

import (
	"bufio"
	"bytes"
	"cmp"
	"fmt"
	"io"
	"regexp"
	"slices"
	"strconv"
)

var timelineHistoryRecordRegexp = regexp.MustCompile(`^(\d+)\t(.+)\t(.+)$`)

type TimelineHistoryRecord struct {
	Timeline uint32
	LSN      LSN
	Comment  string
}

func NewTimelineHistoryRecord(timeline uint32, lsn LSN, comment string) TimelineHistoryRecord {
	return TimelineHistoryRecord{Timeline: timeline, LSN: lsn, Comment: comment}
}

func newHistoryRecordFromString(row string) (*TimelineHistoryRecord, error) {
	matchResult := timelineHistoryRecordRegexp.FindStringSubmatch(row)
	if len(matchResult) < 4 {
		return nil, nil
	}

	timeline, err := strconv.ParseUint(matchResult[1], 10, hexUint32Bits)
	if err != nil {
		return nil, fmt.Errorf("walmath: invalid timeline %q in history line: %w", matchResult[1], err)
	}

	lsn, err := ParseLSN(matchResult[2])
	if err != nil {
		return nil, err
	}

	return &TimelineHistoryRecord{Timeline: uint32(timeline), LSN: lsn, Comment: matchResult[3]}, nil
}

func ParseHistoryFile(historyReader io.Reader) ([]TimelineHistoryRecord, error) {
	scanner := bufio.NewScanner(historyReader)
	records := make([]TimelineHistoryRecord, 0)

	for scanner.Scan() {
		nextRow := scanner.Text()
		if nextRow == "" {
			continue
		}

		record, err := newHistoryRecordFromString(nextRow)
		if err != nil {
			return nil, err
		}
		if record == nil {
			continue
		}

		records = append(records, *record)
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("walmath: read history file: %w", err)
	}

	return records, nil
}

type TimelineHistFile struct {
	TimelineID uint32
	Filename   string
	data       []byte
}

func NewTimelineHistFile(timelineID uint32, filename string, body []byte) TimelineHistFile {
	return TimelineHistFile{TimelineID: timelineID, Filename: filename, data: body}
}

func (tlh TimelineHistFile) Name() string {
	return tlh.Filename
}

func (tlh TimelineHistFile) Records() ([]TimelineHistoryRecord, error) {
	return ParseHistoryFile(bytes.NewReader(tlh.data))
}

func (tlh TimelineHistFile) LSNToTimeline(lsn LSN) (uint32, error) {
	records, err := tlh.Records()
	if err != nil {
		return 0, err
	}

	slices.SortFunc(records, func(a, b TimelineHistoryRecord) int {
		return cmp.Compare(a.Timeline, b.Timeline)
	})

	for _, record := range records {
		if lsn < record.LSN {
			return record.Timeline, nil
		}
	}

	return tlh.TimelineID, nil
}
