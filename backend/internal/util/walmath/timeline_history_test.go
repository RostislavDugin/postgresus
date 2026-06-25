package walmath_test

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"

	"databasus-backend/internal/util/walmath"
)

func Test_ParseHistoryFile_SingleRecord_ParsesTimelineLsnComment(t *testing.T) {
	body := "1\t0/2A33FF50\tno recovery target specified\n"

	records, err := walmath.ParseHistoryFile(strings.NewReader(body))
	assert.NoError(t, err)
	assert.Len(t, records, 1)
	assert.Equal(t, uint32(1), records[0].Timeline)
	assert.Equal(t, walmath.LSN(0x2A33FF50), records[0].LSN)
	assert.Equal(t, "no recovery target specified", records[0].Comment)
}

func Test_ParseHistoryFile_MultipleRecords_PreservesOrder(t *testing.T) {
	body := "1\t0/2A33FF50\tfirst switch\n" +
		"2\t0/2A3400E8\tsecond switch\n" +
		"3\t1/0\tthird switch\n"

	records, err := walmath.ParseHistoryFile(strings.NewReader(body))
	assert.NoError(t, err)
	assert.Len(t, records, 3)
	assert.Equal(t, uint32(1), records[0].Timeline)
	assert.Equal(t, uint32(2), records[1].Timeline)
	assert.Equal(t, uint32(3), records[2].Timeline)
}

func Test_ParseHistoryFile_NoTrailingNewline_LastRecordParses(t *testing.T) {
	body := "1\t0/2A33FF50\tfirst\n" +
		"2\t0/2A3400E8\tlast (no trailing newline)"

	records, err := walmath.ParseHistoryFile(strings.NewReader(body))
	assert.NoError(t, err)
	assert.Len(t, records, 2)
	assert.Equal(t, uint32(2), records[1].Timeline)
}

func Test_ParseHistoryFile_EmptyLines_AreSkipped(t *testing.T) {
	body := "1\t0/2A33FF50\tfirst\n" +
		"\n" +
		"\n" +
		"2\t0/2A3400E8\tsecond\n"

	records, err := walmath.ParseHistoryFile(strings.NewReader(body))
	assert.NoError(t, err)
	assert.Len(t, records, 2)
}

func Test_ParseHistoryFile_NonMatchingLines_AreSkipped(t *testing.T) {
	body := "# this is a comment\n" +
		"1\t0/2A33FF50\tfirst\n" +
		"some non-matching free text\n" +
		"2\t0/2A3400E8\tsecond\n"

	records, err := walmath.ParseHistoryFile(strings.NewReader(body))
	assert.NoError(t, err)
	assert.Len(t, records, 2)
}

func Test_ParseHistoryFile_MalformedLsn_ReturnsError(t *testing.T) {
	body := "1\tGGG/INVALID\tbroken\n"

	_, err := walmath.ParseHistoryFile(strings.NewReader(body))
	assert.Error(t, err)
}

func Test_LSNToTimeline_LsnBeforeFirstSwitch_ReturnsOldestTimeline(t *testing.T) {
	body := []byte("1\t0/2A33FF50\tfirst switch\n" +
		"2\t0/2A3400E8\tsecond switch\n")
	tlh := walmath.NewTimelineHistFile(3, "00000003.history", body)

	timeline, err := tlh.LSNToTimeline(walmath.LSN(0x1000000))
	assert.NoError(t, err)
	assert.Equal(t, uint32(1), timeline)
}

func Test_LSNToTimeline_LsnBetweenSwitches_ReturnsMiddleTimeline(t *testing.T) {
	body := []byte("1\t0/2A33FF50\tfirst switch\n" +
		"2\t0/2A3400E8\tsecond switch\n")
	tlh := walmath.NewTimelineHistFile(3, "00000003.history", body)

	timeline, err := tlh.LSNToTimeline(walmath.LSN(0x2A34009C))
	assert.NoError(t, err)
	assert.Equal(t, uint32(2), timeline)
}

func Test_LSNToTimeline_LsnAfterAllSwitches_ReturnsFilesTimeline(t *testing.T) {
	body := []byte("1\t0/2A33FF50\tfirst switch\n" +
		"2\t0/2A3400E8\tsecond switch\n")
	tlh := walmath.NewTimelineHistFile(3, "00000003.history", body)

	timeline, err := tlh.LSNToTimeline(walmath.LSN(0x100000000))
	assert.NoError(t, err)
	assert.Equal(t, uint32(3), timeline)
}

func Test_LSNToTimeline_UnsortedRecords_StillReturnsCorrectTimeline(t *testing.T) {
	body := []byte("2\t0/2A3400E8\tsecond\n" +
		"1\t0/2A33FF50\tfirst\n")
	tlh := walmath.NewTimelineHistFile(3, "00000003.history", body)

	timeline, err := tlh.LSNToTimeline(walmath.LSN(0x1000000))
	assert.NoError(t, err)
	assert.Equal(t, uint32(1), timeline)
}

func Test_NewTimelineHistoryRecord_StoresFieldsCorrectly(t *testing.T) {
	record := walmath.NewTimelineHistoryRecord(2, walmath.LSN(0x2A33FF50), "test comment")
	assert.Equal(t, uint32(2), record.Timeline)
	assert.Equal(t, walmath.LSN(0x2A33FF50), record.LSN)
	assert.Equal(t, "test comment", record.Comment)
}

func Test_TimelineHistFile_Name_ReturnsFilename(t *testing.T) {
	tlh := walmath.NewTimelineHistFile(3, "00000003.history", []byte{})
	assert.Equal(t, "00000003.history", tlh.Name())
}

func Test_TimelineHistFile_Records_ReturnsParsedRecords(t *testing.T) {
	body := []byte("1\t0/2A33FF50\tfirst\n" +
		"2\t0/2A3400E8\tsecond\n")
	tlh := walmath.NewTimelineHistFile(3, "00000003.history", body)

	records, err := tlh.Records()
	assert.NoError(t, err)
	assert.Len(t, records, 2)
}
