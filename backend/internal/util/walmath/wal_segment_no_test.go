package walmath_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"databasus-backend/internal/util/walmath"
)

func Test_NewWalSegmentNo_LsnInsideSegment_ReturnsSegmentNumber(t *testing.T) {
	segNo := walmath.NewWalSegmentNo(walmath.LSN(walmath.WalSegmentSize * 5))
	assert.Equal(t, walmath.WalSegmentNo(5), segNo)

	segNo = walmath.NewWalSegmentNo(walmath.LSN(walmath.WalSegmentSize*5 + 12345))
	assert.Equal(t, walmath.WalSegmentNo(5), segNo)
}

func Test_GetSegmentNoFromLsn_LsnAtSegmentBoundary_ReturnsNextSegment(t *testing.T) {
	assert.Equal(t, uint64(0), walmath.GetSegmentNoFromLsn(walmath.LSN(walmath.WalSegmentSize-1)))
	assert.Equal(t, uint64(1), walmath.GetSegmentNoFromLsn(walmath.LSN(walmath.WalSegmentSize)))
	assert.Equal(t, uint64(1), walmath.GetSegmentNoFromLsn(walmath.LSN(walmath.WalSegmentSize+1)))
}

func Test_WalSegmentNo_Next_AdvancesByOne(t *testing.T) {
	assert.Equal(t, walmath.WalSegmentNo(6), walmath.WalSegmentNo(5).Next())
	assert.Equal(t, walmath.WalSegmentNo(1), walmath.WalSegmentNo(0).Next())
}

func Test_WalSegmentNo_Previous_DecrementsByOne(t *testing.T) {
	assert.Equal(t, walmath.WalSegmentNo(4), walmath.WalSegmentNo(5).Previous())
	assert.Equal(t, walmath.WalSegmentNo(0), walmath.WalSegmentNo(1).Previous())
}

func Test_WalSegmentNo_FirstLSN_FromSegnoZero_ReturnsZero(t *testing.T) {
	assert.Equal(t, walmath.LSN(0), walmath.WalSegmentNo(0).FirstLSN())
}

func Test_WalSegmentNo_FirstLSN_FromSegnoN_ReturnsNxSegmentSize(t *testing.T) {
	assert.Equal(t, walmath.LSN(walmath.WalSegmentSize), walmath.WalSegmentNo(1).FirstLSN())
	assert.Equal(t, walmath.LSN(walmath.WalSegmentSize*42), walmath.WalSegmentNo(42).FirstLSN())
}

func Test_WalSegmentNo_GetFilename_KnownTimelineAndSegno_MatchesPgFormat(t *testing.T) {
	assert.Equal(t, "000000010000000000000042", walmath.WalSegmentNo(0x42).GetFilename(1))
	assert.Equal(t, "00000007000000000000002A", walmath.WalSegmentNo(0x2A).GetFilename(7))
	assert.Equal(t, "000000010000000100000001", walmath.WalSegmentNo(0x101).GetFilename(1))
}

func Test_NewWalSegmentDescription_FromValidName_RoundtripsToFilename(t *testing.T) {
	desc, err := walmath.NewWalSegmentDescription("000000010000000100000001")
	assert.NoError(t, err)
	assert.Equal(t, "000000010000000100000001", desc.GetFileName())
	assert.Equal(t, uint32(1), desc.Timeline)
}

func Test_NewWalSegmentDescription_FromInvalidName_ReturnsError(t *testing.T) {
	_, err := walmath.NewWalSegmentDescription("not-a-wal-name")
	assert.Error(t, err)
}
