package walmath_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"databasus-backend/internal/util/walmath"
)

func Test_ParseLSN_ValidHex_RoundtripsViaString(t *testing.T) {
	cases := []string{
		"0/0",
		"0/1",
		"1/0",
		"0/2A33FF50",
		"12345678/90ABCDEF",
		"FFFFFFFF/FFFFFFFF",
	}

	for _, c := range cases {
		lsn, err := walmath.ParseLSN(c)
		assert.NoError(t, err, c)
		assert.Equal(t, c, lsn.String(), c)
	}
}

func Test_ParseLSN_MalformedInput_ReturnsError(t *testing.T) {
	cases := []string{
		"",
		"/0",
		"0/",
		"GGG/0",
		"deadbeef",
		"0",
	}

	for _, c := range cases {
		_, err := walmath.ParseLSN(c)
		assert.Error(t, err, c)
	}
}

func Test_LSN_String_ZeroPaddingMatches_X_X_Format(t *testing.T) {
	assert.Equal(t, "12345678/90ABCDEF", walmath.LSN(0x1234567890ABCDEF).String())
	assert.Equal(t, "0/A", walmath.LSN(0xA).String())
	assert.Equal(t, "1/0", walmath.LSN(0x100000000).String())
}

func Test_ParseLSN_LowercaseHex_ParsesCorrectly(t *testing.T) {
	lsn, err := walmath.ParseLSN("0/2a33ff50")
	assert.NoError(t, err)

	assert.Equal(t, walmath.LSN(0x2A33FF50), lsn)
}
