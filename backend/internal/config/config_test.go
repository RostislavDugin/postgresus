package config

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func Test_IsTestProcess_UsesExecutableNameOnly(t *testing.T) {
	testCases := []struct {
		name      string
		arguments []string
		isTesting bool
	}{
		{
			name:      "Go test binary",
			arguments: []string{"/tmp/config.test", "-test.v"},
			isTesting: true,
		},
		{
			name:      "Windows Go test binary",
			arguments: []string{`C:\\Temp\\config.test.exe`, "-test.v"},
			isTesting: true,
		},
		{
			name:      "database cleanup command",
			arguments: []string{"/tmp/cleanup_test_db"},
			isTesting: true,
		},
		{
			name:      "storage command",
			arguments: []string{"/app/databasus", "--test-storage"},
			isTesting: false,
		},
		{
			name:      "unrelated argument containing test",
			arguments: []string{"/app/databasus", "contest"},
			isTesting: false,
		},
		{
			name:      "missing executable",
			arguments: nil,
			isTesting: false,
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			isTesting := isTestProcess(testCase.arguments)
			assert.Equal(t, testCase.isTesting, isTesting)
		})
	}
}
