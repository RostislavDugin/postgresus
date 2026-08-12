package usecases_physical_postgresql

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func Test_IsCompressionUnsupportedError_MatchesBuildRejection(t *testing.T) {
	cases := []struct {
		stderr             string
		isUnsupportedBuild bool
	}{
		{
			"pg_basebackup: error: could not initiate base backup: ERROR:  invalid compression specification: " +
				"this build does not support compression with ZSTD",
			true,
		},
		{
			"pg_basebackup: error: could not initiate base backup: ERROR:  invalid compression specification: " +
				"this build does not support compression with gzip",
			true,
		},
		{
			`pg_basebackup: error: invalid compression specification: compression algorithm "zstd" ` +
				`expects a compression level between 1 and 22 (default at 0)`,
			false,
		},
		{"pg_basebackup: error: could not connect to server", false},
		{"", false},
	}

	for _, testCase := range cases {
		assert.Equal(
			t,
			testCase.isUnsupportedBuild,
			isCompressionUnsupportedError([]byte(testCase.stderr)),
			testCase.stderr,
		)
	}
}
