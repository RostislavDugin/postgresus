package walmath_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"databasus-backend/internal/util/walmath"
)

func Test_ParseWALFilename_ValidNames_ExtractsTimelineAndSegNo(t *testing.T) {
	cases := []struct {
		name               string
		expectedTimelineID uint32
		expectedLogSegNo   uint64
	}{
		{"000000010000000100000001", 1, 1<<8 + 1},
		{"000000100000020000000030", 1 << 4, 2<<16 + 3<<4},
		{"10000000f0000000000000a0", 1 << 28, 15<<36 + 10<<4},
		{"ffffffffffffffff000000ff", 1<<32 - 1, 1<<40 - 1},
	}

	for _, c := range cases {
		timeline, segNo, err := walmath.ParseWALFilename(c.name)
		assert.NoError(t, err, c.name)
		assert.Equal(t, c.expectedTimelineID, timeline, c.name)
		assert.Equal(t, c.expectedLogSegNo, segNo, c.name)
	}
}

func Test_ParseWALFilename_MalformedNames_ReturnsError(t *testing.T) {
	cases := []string{
		"00000001",
		"000000010000000100000100000000010000000100000001",
		"000000010000000100000100",
		"000000010000000110000001",
		"000xYz010000000100000001",
		"0000000100xYz00100000001",
		"000000010000000100xYz001",
	}

	for _, c := range cases {
		_, _, err := walmath.ParseWALFilename(c)
		assert.Error(t, err, c)
	}
}

func Test_GetNextWalFilename_VariousSegments_Increments(t *testing.T) {
	cases := []struct {
		input    string
		expected string
	}{
		{"000000010000000000000051", "000000010000000000000052"},
		{"00000001000000000000005F", "000000010000000000000060"},
		{"0000000100000001000000FF", "000000010000000200000000"},
	}

	for _, c := range cases {
		next, err := walmath.GetNextWalFilename(c.input)
		assert.NoError(t, err, c.input)
		assert.Equal(t, c.expected, next, c.input)
	}
}

func Test_GetNextWalFilename_MalformedInput_ReturnsError(t *testing.T) {
	cases := []string{
		"0000000100000001000001FF",
		"00000001000ZZ001000000FF",
		"00000001000001000000FF",
		"asdfasdf",
	}

	for _, c := range cases {
		_, err := walmath.GetNextWalFilename(c)
		assert.Error(t, err, c)
	}
}

func Test_ParseWALFilename_WithCustomSegmentSize_HandlesBoundary(t *testing.T) {
	assert.Equal(t, uint64(16*1024*1024), walmath.WalSegmentSize)

	walmath.SetWalSize(64)
	t.Cleanup(func() { walmath.SetWalSize(16) })

	assert.Equal(t, uint64(64*1024*1024), walmath.WalSegmentSize)

	_, _, err := walmath.ParseWALFilename("10000000f0000000000000a0")
	assert.Error(t, err)

	cases := []struct {
		name               string
		expectedTimelineID uint32
		expectedLogSegNo   uint64
	}{
		{"000000010000000000000001", 1, 1},
		{"000000100000000100000001", 1 << 4, 4<<4 + 1},
		{"000000100000020000000030", 1 << 4, 2<<14 + 3<<4},
		{"10000000f000000000000030", 1 << 28, 15<<34 + 3<<4},
	}

	for _, c := range cases {
		timeline, segNo, err := walmath.ParseWALFilename(c.name)
		assert.NoError(t, err, c.name)
		assert.Equal(t, c.expectedTimelineID, timeline, c.name)
		assert.Equal(t, c.expectedLogSegNo, segNo, c.name)
	}
}

func Test_ParseWALFilenameWithSize_CustomSegmentSize_DoesNotUseGlobal(t *testing.T) {
	assert.Equal(t, uint64(16*1024*1024), walmath.WalSegmentSize)

	timelineID, segmentNo, err := walmath.ParseWALFilenameWithSize(
		"000000010000000200000003",
		64*1024*1024,
	)

	assert.NoError(t, err)
	assert.Equal(t, uint32(1), timelineID)
	assert.Equal(t, uint64(2*64+3), segmentNo)
}

func Test_IsWalFilename_ValidName_ReturnsTrue(t *testing.T) {
	assert.True(t, walmath.IsWalFilename("000000010000000100000001"))
	assert.True(t, walmath.IsWalFilename("ffffffffffffffff000000ff"))
}

func Test_IsWalFilename_InvalidName_ReturnsFalse(t *testing.T) {
	assert.False(t, walmath.IsWalFilename(""))
	assert.False(t, walmath.IsWalFilename("not-a-wal-name"))
	assert.False(t, walmath.IsWalFilename("000000010000000100000001.partial"))
}

func Test_TryFetchTimelineAndLogSegNo_FromObjectName_FindsValid(t *testing.T) {
	timeline, segNo, ok := walmath.TryFetchTimelineAndLogSegNo("db-WAL-tl1-000000010000000100000001.zst")
	assert.True(t, ok)
	assert.Equal(t, uint32(1), timeline)
	assert.Equal(t, uint64(1<<8+1), segNo)
}

func Test_TryFetchTimelineAndLogSegNo_NoMatch_ReturnsFalse(t *testing.T) {
	_, _, ok := walmath.TryFetchTimelineAndLogSegNo("no-wal-filename-here")
	assert.False(t, ok)
}

func Test_FormatHistoryFilename_KnownTimeline_FormatsAsPg(t *testing.T) {
	assert.Equal(t, "00000003.history", walmath.FormatHistoryFilename(3))
	assert.Equal(t, "0000000A.history", walmath.FormatHistoryFilename(10))
}
