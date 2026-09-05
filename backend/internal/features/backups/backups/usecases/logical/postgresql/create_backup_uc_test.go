package usecases_logical_postgresql

import (
	"io"
	"log/slog"
	"testing"

	"github.com/stretchr/testify/require"

	pgtypes "databasus-backend/internal/features/databases/databases/postgresql/logical"
	"databasus-backend/internal/util/tools"
)

func Test_BuildPgDumpArgs_TreatsDuplicateParameterAsLiteralDatabaseName(t *testing.T) {
	databaseName := "postgres dbname=databasus"
	postgresDatabase := &pgtypes.PostgresqlLogicalDatabase{
		Version:  tools.PostgresqlVersion17,
		Host:     "localhost",
		Port:     5437,
		Username: "postgres",
		Database: &databaseName,
	}

	usecase := &CreatePostgresqlBackupUsecase{
		logger: slog.New(slog.NewTextHandler(io.Discard, nil)),
	}
	args := usecase.buildPgDumpArgs(postgresDatabase)

	databaseArgumentIndex := -1
	for argumentIndex, argument := range args {
		if argument == "-d" {
			databaseArgumentIndex = argumentIndex + 1
			break
		}
	}

	require.Less(t, databaseArgumentIndex, len(args))
	require.GreaterOrEqual(t, databaseArgumentIndex, 0)
	require.Equal(t, "dbname='postgres dbname=databasus'", args[databaseArgumentIndex])
}
