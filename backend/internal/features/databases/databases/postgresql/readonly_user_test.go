package postgresql

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"strconv"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq"
	"github.com/stretchr/testify/assert"

	"postgresus-backend/internal/config"
	"postgresus-backend/internal/util/tools"
)

func Test_IsUserReadOnly_AdminUser_ReturnsFalse(t *testing.T) {
	env := config.GetEnv()
	cases := []struct {
		name    string
		version string
		port    string
	}{
		{"PostgreSQL 12", "12", env.TestPostgres12Port},
		{"PostgreSQL 13", "13", env.TestPostgres13Port},
		{"PostgreSQL 14", "14", env.TestPostgres14Port},
		{"PostgreSQL 15", "15", env.TestPostgres15Port},
		{"PostgreSQL 16", "16", env.TestPostgres16Port},
		{"PostgreSQL 17", "17", env.TestPostgres17Port},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			container := connectToPostgresContainer(t, tc.port)
			defer container.DB.Close()

			pgModel := createPostgresModel(container)
			logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
			ctx := context.Background()

			isReadOnly, err := pgModel.IsUserReadOnly(ctx, logger, nil, uuid.New())
			assert.NoError(t, err)
			assert.False(t, isReadOnly, "Admin user should not be read-only")
		})
	}
}

func Test_CreateReadOnlyUser_UserCanReadButNotWrite(t *testing.T) {
	env := config.GetEnv()
	cases := []struct {
		name    string
		version string
		port    string
	}{
		{"PostgreSQL 12", "12", env.TestPostgres12Port},
		{"PostgreSQL 13", "13", env.TestPostgres13Port},
		{"PostgreSQL 14", "14", env.TestPostgres14Port},
		{"PostgreSQL 15", "15", env.TestPostgres15Port},
		{"PostgreSQL 16", "16", env.TestPostgres16Port},
		{"PostgreSQL 17", "17", env.TestPostgres17Port},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			container := connectToPostgresContainer(t, tc.port)
			defer container.DB.Close()

			_, err := container.DB.Exec(`
			DROP TABLE IF EXISTS readonly_test CASCADE;
			DROP TABLE IF EXISTS hack_table CASCADE;
			DROP TABLE IF EXISTS future_table CASCADE;
			CREATE TABLE readonly_test (
				id SERIAL PRIMARY KEY,
				data TEXT NOT NULL
			);
			INSERT INTO readonly_test (data) VALUES ('test1'), ('test2');
		`)
			assert.NoError(t, err)

			pgModel := createPostgresModel(container)
			logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
			ctx := context.Background()

			username, password, err := pgModel.CreateReadOnlyUser(ctx, logger, nil, uuid.New())
			assert.NoError(t, err)
			assert.NotEmpty(t, username)
			assert.NotEmpty(t, password)
			assert.True(t, strings.HasPrefix(username, "postgresus-"))

			readOnlyModel := &PostgresqlDatabase{
				Version:  pgModel.Version,
				Host:     pgModel.Host,
				Port:     pgModel.Port,
				Username: username,
				Password: password,
				Database: pgModel.Database,
				IsHttps:  false,
			}

			isReadOnly, err := readOnlyModel.IsUserReadOnly(ctx, logger, nil, uuid.New())
			assert.NoError(t, err)
			assert.True(t, isReadOnly, "Created user should be read-only")

			readOnlyDSN := fmt.Sprintf(
				"host=%s port=%d user=%s password=%s dbname=%s sslmode=disable",
				container.Host,
				container.Port,
				username,
				password,
				container.Database,
			)
			readOnlyConn, err := sqlx.Connect("postgres", readOnlyDSN)
			assert.NoError(t, err)
			defer readOnlyConn.Close()

			var count int
			err = readOnlyConn.Get(&count, "SELECT COUNT(*) FROM readonly_test")
			assert.NoError(t, err)
			assert.Equal(t, 2, count)

			_, err = readOnlyConn.Exec("INSERT INTO readonly_test (data) VALUES ('should-fail')")
			assert.Error(t, err)
			assert.Contains(t, err.Error(), "permission denied")

			_, err = readOnlyConn.Exec("UPDATE readonly_test SET data = 'hacked' WHERE id = 1")
			assert.Error(t, err)
			assert.Contains(t, err.Error(), "permission denied")

			_, err = readOnlyConn.Exec("DELETE FROM readonly_test WHERE id = 1")
			assert.Error(t, err)
			assert.Contains(t, err.Error(), "permission denied")

			_, err = readOnlyConn.Exec("CREATE TABLE hack_table (id INT)")
			assert.Error(t, err)
			assert.Contains(t, err.Error(), "permission denied")

			// Clean up: Drop user with CASCADE to handle default privilege dependencies
			_, err = container.DB.Exec(fmt.Sprintf(`DROP OWNED BY "%s" CASCADE`, username))
			if err != nil {
				t.Logf("Warning: Failed to drop owned objects: %v", err)
			}

			_, err = container.DB.Exec(fmt.Sprintf(`DROP USER IF EXISTS "%s"`, username))
			assert.NoError(t, err)
		})
	}
}

func Test_ReadOnlyUser_FutureTables_HaveSelectPermission(t *testing.T) {
	env := config.GetEnv()
	container := connectToPostgresContainer(t, env.TestPostgres16Port)
	defer container.DB.Close()

	pgModel := createPostgresModel(container)
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	ctx := context.Background()

	username, password, err := pgModel.CreateReadOnlyUser(ctx, logger, nil, uuid.New())
	assert.NoError(t, err)

	_, err = container.DB.Exec(`
		CREATE TABLE future_table (
			id SERIAL PRIMARY KEY,
			data TEXT NOT NULL
		);
		INSERT INTO future_table (data) VALUES ('future_data');
	`)
	assert.NoError(t, err)

	readOnlyDSN := fmt.Sprintf("host=%s port=%d user=%s password=%s dbname=%s sslmode=disable",
		container.Host, container.Port, username, password, container.Database)
	readOnlyConn, err := sqlx.Connect("postgres", readOnlyDSN)
	assert.NoError(t, err)
	defer readOnlyConn.Close()

	var data string
	err = readOnlyConn.Get(&data, "SELECT data FROM future_table LIMIT 1")
	assert.NoError(t, err)
	assert.Equal(t, "future_data", data)

	// Clean up: Drop user with CASCADE to handle default privilege dependencies
	_, err = container.DB.Exec(fmt.Sprintf(`DROP OWNED BY "%s" CASCADE`, username))
	if err != nil {
		t.Logf("Warning: Failed to drop owned objects: %v", err)
	}

	_, err = container.DB.Exec(fmt.Sprintf(`DROP USER IF EXISTS "%s"`, username))
	assert.NoError(t, err)
}

func Test_ReadOnlyUser_MultipleSchemas_AllAccessible(t *testing.T) {
	env := config.GetEnv()
	container := connectToPostgresContainer(t, env.TestPostgres16Port)
	defer container.DB.Close()

	_, err := container.DB.Exec(`
		CREATE SCHEMA IF NOT EXISTS schema_a;
		CREATE SCHEMA IF NOT EXISTS schema_b;
		CREATE TABLE schema_a.table_a (id INT, data TEXT);
		CREATE TABLE schema_b.table_b (id INT, data TEXT);
		INSERT INTO schema_a.table_a VALUES (1, 'data_a');
		INSERT INTO schema_b.table_b VALUES (2, 'data_b');
	`)
	assert.NoError(t, err)

	pgModel := createPostgresModel(container)
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	ctx := context.Background()

	username, password, err := pgModel.CreateReadOnlyUser(ctx, logger, nil, uuid.New())
	assert.NoError(t, err)

	readOnlyDSN := fmt.Sprintf("host=%s port=%d user=%s password=%s dbname=%s sslmode=disable",
		container.Host, container.Port, username, password, container.Database)
	readOnlyConn, err := sqlx.Connect("postgres", readOnlyDSN)
	assert.NoError(t, err)
	defer readOnlyConn.Close()

	var dataA string
	err = readOnlyConn.Get(&dataA, "SELECT data FROM schema_a.table_a LIMIT 1")
	assert.NoError(t, err)
	assert.Equal(t, "data_a", dataA)

	var dataB string
	err = readOnlyConn.Get(&dataB, "SELECT data FROM schema_b.table_b LIMIT 1")
	assert.NoError(t, err)
	assert.Equal(t, "data_b", dataB)

	// Clean up: Drop user with CASCADE to handle default privilege dependencies
	_, err = container.DB.Exec(fmt.Sprintf(`DROP OWNED BY "%s" CASCADE`, username))
	if err != nil {
		t.Logf("Warning: Failed to drop owned objects: %v", err)
	}

	_, err = container.DB.Exec(fmt.Sprintf(`DROP USER IF EXISTS "%s"`, username))
	assert.NoError(t, err)
	_, err = container.DB.Exec(`DROP SCHEMA schema_a CASCADE; DROP SCHEMA schema_b CASCADE;`)
	assert.NoError(t, err)
}

type PostgresContainer struct {
	Host     string
	Port     int
	Username string
	Password string
	Database string
	DB       *sqlx.DB
}

func connectToPostgresContainer(t *testing.T, port string) *PostgresContainer {
	dbName := "testdb"
	password := "testpassword"
	username := "testuser"
	host := "localhost"

	portInt, err := strconv.Atoi(port)
	assert.NoError(t, err)

	dsn := fmt.Sprintf("host=%s port=%d user=%s password=%s dbname=%s sslmode=disable",
		host, portInt, username, password, dbName)

	db, err := sqlx.Connect("postgres", dsn)
	assert.NoError(t, err)

	var versionStr string
	err = db.Get(&versionStr, "SELECT version()")
	assert.NoError(t, err)

	return &PostgresContainer{
		Host:     host,
		Port:     portInt,
		Username: username,
		Password: password,
		Database: dbName,
		DB:       db,
	}
}

func createPostgresModel(container *PostgresContainer) *PostgresqlDatabase {
	var versionStr string
	err := container.DB.Get(&versionStr, "SELECT version()")
	if err != nil {
		return nil
	}

	version := extractPostgresVersion(versionStr)

	return &PostgresqlDatabase{
		Version:  version,
		Host:     container.Host,
		Port:     container.Port,
		Username: container.Username,
		Password: container.Password,
		Database: &container.Database,
		IsHttps:  false,
	}
}

func extractPostgresVersion(versionStr string) tools.PostgresqlVersion {
	if strings.Contains(versionStr, "PostgreSQL 12") {
		return tools.GetPostgresqlVersionEnum("12")
	} else if strings.Contains(versionStr, "PostgreSQL 13") {
		return tools.GetPostgresqlVersionEnum("13")
	} else if strings.Contains(versionStr, "PostgreSQL 14") {
		return tools.GetPostgresqlVersionEnum("14")
	} else if strings.Contains(versionStr, "PostgreSQL 15") {
		return tools.GetPostgresqlVersionEnum("15")
	} else if strings.Contains(versionStr, "PostgreSQL 16") {
		return tools.GetPostgresqlVersionEnum("16")
	} else if strings.Contains(versionStr, "PostgreSQL 17") {
		return tools.GetPostgresqlVersionEnum("17")
	}

	return tools.GetPostgresqlVersionEnum("16")
}
