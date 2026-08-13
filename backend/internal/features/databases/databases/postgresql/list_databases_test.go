package postgresql

import (
	"log/slog"
	"os"
	"sort"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"postgresus-backend/internal/config"
)

func Test_ListDatabases_WithValidCredentials_ReturnsServerDatabases(t *testing.T) {
	env := config.GetEnv()
	cases := []struct {
		name string
		addr string
	}{
		{"PostgreSQL 12", env.TestPostgres12Addr},
		{"PostgreSQL 16", env.TestPostgres16Addr},
		{"PostgreSQL 17", env.TestPostgres17Addr},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			container := connectToPostgresContainer(t, tc.addr)
			defer container.DB.Close()

			pgModel := createPostgresModel(container)
			logger := slog.New(slog.NewTextHandler(os.Stdout, nil))

			names, err := pgModel.ListDatabases(logger, nil, uuid.New())

			require.NoError(t, err)
			assert.Contains(t, names, "testdb")
		})
	}
}

func Test_ListDatabases_ExcludesTemplateDatabases(t *testing.T) {
	env := config.GetEnv()

	container := connectToPostgresContainer(t, env.TestPostgres16Addr)
	defer container.DB.Close()

	pgModel := createPostgresModel(container)
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))

	names, err := pgModel.ListDatabases(logger, nil, uuid.New())

	require.NoError(t, err)
	assert.NotContains(t, names, "template0")
	assert.NotContains(t, names, "template1")
}

func Test_ListDatabases_ReturnsNamesSortedAlphabetically(t *testing.T) {
	env := config.GetEnv()

	container := connectToPostgresContainer(t, env.TestPostgres16Addr)
	defer container.DB.Close()

	pgModel := createPostgresModel(container)
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))

	names, err := pgModel.ListDatabases(logger, nil, uuid.New())

	require.NoError(t, err)
	require.NotEmpty(t, names)

	sorted := append([]string(nil), names...)
	sort.Strings(sorted)
	assert.Equal(t, sorted, names)
}

func Test_ListDatabases_WithoutMaintenanceDatabase_ReturnsError(t *testing.T) {
	env := config.GetEnv()

	container := connectToPostgresContainer(t, env.TestPostgres16Addr)
	defer container.DB.Close()

	pgModel := createPostgresModel(container)
	pgModel.Database = nil
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))

	_, err := pgModel.ListDatabases(logger, nil, uuid.New())

	require.Error(t, err)
	assert.Contains(t, err.Error(), "maintenance database name is required")
}

func Test_ListDatabases_WithWrongPassword_ReturnsError(t *testing.T) {
	env := config.GetEnv()

	container := connectToPostgresContainer(t, env.TestPostgres16Addr)
	defer container.DB.Close()

	pgModel := createPostgresModel(container)
	pgModel.Password = "definitely-wrong-password"
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))

	_, err := pgModel.ListDatabases(logger, nil, uuid.New())

	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to connect")
}
