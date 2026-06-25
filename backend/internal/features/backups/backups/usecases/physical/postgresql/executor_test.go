package usecases_physical_postgresql

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

func Test_BuildObjectName_FormatsReadableKey(t *testing.T) {
	backupID := uuid.MustParse("9b2c3d4e-5f60-7081-9234-56789abcdef0")
	now := time.Date(2026, 5, 30, 12, 0, 0, 0, time.UTC)

	tests := []struct {
		name         string
		databaseName string
		kind         string
		expected     string
	}{
		{
			name:         "clean name and FULL kind",
			databaseName: "production",
			kind:         "FULL",
			expected:     "production-FULL-20260530-120000-" + backupID.String(),
		},
		{
			name:         "unsafe characters are sanitized",
			databaseName: "my db/prod",
			kind:         "INCR",
			expected:     "my_db-prod-INCR-20260530-120000-" + backupID.String(),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			objectName := buildObjectName(tt.databaseName, backupID, now, tt.kind)

			assert.Equal(t, tt.expected, objectName)
		})
	}
}
