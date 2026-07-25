package databases

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"databasus-backend/internal/util/encryption"
	"databasus-backend/internal/util/logger"
)

func Test_TestConnection_ForEveryDatabaseType_WithoutEngineConfig_ReturnsErrorInsteadOfPanicking(
	t *testing.T,
) {
	missingConfigCases := []struct {
		databaseType DatabaseType
		errorMessage string
	}{
		{DatabaseTypePostgresLogical, "postgresql logical config is not set"},
		{DatabaseTypePostgresPhysical, "postgresql physical config is not set"},
		{DatabaseTypeMysql, "mysql config is not set"},
		{DatabaseTypeMariadb, "mariadb config is not set"},
		{DatabaseTypeMongodb, "mongodb config is not set"},
		{DatabaseType("CASSANDRA"), "connection test not supported for database type: CASSANDRA"},
	}

	for _, missingConfigCase := range missingConfigCases {
		t.Run(string(missingConfigCase.databaseType), func(t *testing.T) {
			databaseWithoutConfig := Database{Type: missingConfigCase.databaseType}

			err := databaseWithoutConfig.TestConnection(
				logger.GetLogger(),
				encryption.GetFieldEncryptor(),
			)

			assert.EqualError(t, err, missingConfigCase.errorMessage)
		})
	}
}

func Test_GetRawDbSizeMb_ForEveryDatabaseType_WithoutEngineConfig_ReturnsErrorInsteadOfPanicking(
	t *testing.T,
) {
	missingConfigCases := []struct {
		databaseType DatabaseType
		errorMessage string
	}{
		{DatabaseTypePostgresLogical, "postgresql logical config is not set"},
		{DatabaseTypeMysql, "mysql config is not set"},
		{DatabaseTypeMariadb, "mariadb config is not set"},
		{DatabaseTypeMongodb, "mongodb config is not set"},
		{
			DatabaseTypePostgresPhysical,
			"logical backup not supported for database type: POSTGRES_PHYSICAL",
		},
		{
			DatabaseType("CASSANDRA"),
			"logical backup not supported for database type: CASSANDRA",
		},
	}

	for _, missingConfigCase := range missingConfigCases {
		t.Run(string(missingConfigCase.databaseType), func(t *testing.T) {
			databaseWithoutConfig := Database{Type: missingConfigCase.databaseType}

			sizeMb, err := databaseWithoutConfig.GetRawDbSizeMb(
				t.Context(),
				logger.GetLogger(),
				encryption.GetFieldEncryptor(),
			)

			assert.EqualError(t, err, missingConfigCase.errorMessage)
			assert.Zero(t, sizeMb)
		})
	}
}
