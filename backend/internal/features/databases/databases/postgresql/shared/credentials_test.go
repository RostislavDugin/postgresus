package postgresql_shared

import (
	"testing"

	"github.com/jackc/pgx/v5"
	"github.com/stretchr/testify/require"
)

func Test_BuildConnConfig_TreatsCredentialFieldsAsLiteralValues(t *testing.T) {
	spec := CredentialSpec{
		Host:     "db host=attacker\\socket'one",
		Port:     5432,
		Username: "user name=postgres\\role'one",
		SslMode:  PostgresSslModeDisable,
	}
	databaseName := "postgres dbname=databasus\\archive'one"
	password := "secret password=overridden\\value'one"

	connConfig, err := BuildConnConfig(spec, password, databaseName, nil)
	require.NoError(t, err)
	require.Equal(t, spec.Host, connConfig.Host)
	require.Equal(t, spec.Username, connConfig.User)
	require.Equal(t, databaseName, connConfig.Database)
	require.Equal(t, password, connConfig.Password)
}

func Test_BuildDatabaseNameConninfo_TreatsDuplicateParameterAsLiteralDatabaseName(t *testing.T) {
	databaseName := "postgres dbname=databasus"

	connConfig, err := pgx.ParseConfig(BuildDatabaseNameConninfo(databaseName))
	require.NoError(t, err)
	require.Equal(t, databaseName, connConfig.Database)
}
