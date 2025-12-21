package mariadb

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"strconv"
	"strings"
	"testing"

	_ "github.com/go-sql-driver/mysql"
	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"github.com/stretchr/testify/assert"

	"postgresus-backend/internal/config"
	"postgresus-backend/internal/util/tools"
)

func Test_IsUserReadOnly_AdminUser_ReturnsFalse(t *testing.T) {
	env := config.GetEnv()
	cases := []struct {
		name    string
		version tools.MariadbVersion
		port    string
	}{
		{"MariaDB 5.5", tools.MariadbVersion55, env.TestMariadb55Port},
		{"MariaDB 10.1", tools.MariadbVersion101, env.TestMariadb101Port},
		{"MariaDB 10.2", tools.MariadbVersion102, env.TestMariadb102Port},
		{"MariaDB 10.3", tools.MariadbVersion103, env.TestMariadb103Port},
		{"MariaDB 10.4", tools.MariadbVersion104, env.TestMariadb104Port},
		{"MariaDB 10.5", tools.MariadbVersion105, env.TestMariadb105Port},
		{"MariaDB 10.6", tools.MariadbVersion106, env.TestMariadb106Port},
		{"MariaDB 10.11", tools.MariadbVersion1011, env.TestMariadb1011Port},
		{"MariaDB 11.4", tools.MariadbVersion114, env.TestMariadb114Port},
		{"MariaDB 11.8", tools.MariadbVersion118, env.TestMariadb118Port},
		{"MariaDB 12.0", tools.MariadbVersion120, env.TestMariadb120Port},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			container := connectToMariadbContainer(t, tc.port, tc.version)
			defer container.DB.Close()

			mariadbModel := createMariadbModel(container)
			logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
			ctx := context.Background()

			isReadOnly, err := mariadbModel.IsUserReadOnly(ctx, logger, nil, uuid.New())
			assert.NoError(t, err)
			assert.False(t, isReadOnly, "Root user should not be read-only")
		})
	}
}

func Test_CreateReadOnlyUser_UserCanReadButNotWrite(t *testing.T) {
	env := config.GetEnv()
	cases := []struct {
		name    string
		version tools.MariadbVersion
		port    string
	}{
		{"MariaDB 5.5", tools.MariadbVersion55, env.TestMariadb55Port},
		{"MariaDB 10.1", tools.MariadbVersion101, env.TestMariadb101Port},
		{"MariaDB 10.2", tools.MariadbVersion102, env.TestMariadb102Port},
		{"MariaDB 10.3", tools.MariadbVersion103, env.TestMariadb103Port},
		{"MariaDB 10.4", tools.MariadbVersion104, env.TestMariadb104Port},
		{"MariaDB 10.5", tools.MariadbVersion105, env.TestMariadb105Port},
		{"MariaDB 10.6", tools.MariadbVersion106, env.TestMariadb106Port},
		{"MariaDB 10.11", tools.MariadbVersion1011, env.TestMariadb1011Port},
		{"MariaDB 11.4", tools.MariadbVersion114, env.TestMariadb114Port},
		{"MariaDB 11.8", tools.MariadbVersion118, env.TestMariadb118Port},
		{"MariaDB 12.0", tools.MariadbVersion120, env.TestMariadb120Port},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			container := connectToMariadbContainer(t, tc.port, tc.version)
			defer container.DB.Close()

			_, err := container.DB.Exec(`DROP TABLE IF EXISTS readonly_test`)
			assert.NoError(t, err)
			_, err = container.DB.Exec(`DROP TABLE IF EXISTS hack_table`)
			assert.NoError(t, err)
			_, err = container.DB.Exec(`DROP TABLE IF EXISTS future_table`)
			assert.NoError(t, err)

			_, err = container.DB.Exec(`
				CREATE TABLE readonly_test (
					id INT AUTO_INCREMENT PRIMARY KEY,
					data VARCHAR(255) NOT NULL
				)
			`)
			assert.NoError(t, err)

			_, err = container.DB.Exec(
				`INSERT INTO readonly_test (data) VALUES ('test1'), ('test2')`,
			)
			assert.NoError(t, err)

			mariadbModel := createMariadbModel(container)
			logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
			ctx := context.Background()

			username, password, err := mariadbModel.CreateReadOnlyUser(ctx, logger, nil, uuid.New())
			assert.NoError(t, err)
			assert.NotEmpty(t, username)
			assert.NotEmpty(t, password)
			assert.True(t, strings.HasPrefix(username, "pgs-"))

			if err != nil {
				return
			}

			readOnlyModel := &MariadbDatabase{
				Version:  mariadbModel.Version,
				Host:     mariadbModel.Host,
				Port:     mariadbModel.Port,
				Username: username,
				Password: password,
				Database: mariadbModel.Database,
				IsHttps:  false,
			}

			isReadOnly, err := readOnlyModel.IsUserReadOnly(ctx, logger, nil, uuid.New())
			assert.NoError(t, err)
			assert.True(t, isReadOnly, "Created user should be read-only")

			readOnlyDSN := fmt.Sprintf(
				"%s:%s@tcp(%s:%d)/%s?parseTime=true",
				username,
				password,
				container.Host,
				container.Port,
				container.Database,
			)
			readOnlyConn, err := sqlx.Connect("mysql", readOnlyDSN)
			assert.NoError(t, err)
			defer readOnlyConn.Close()

			var count int
			err = readOnlyConn.Get(&count, "SELECT COUNT(*) FROM readonly_test")
			assert.NoError(t, err)
			assert.Equal(t, 2, count)

			_, err = readOnlyConn.Exec("INSERT INTO readonly_test (data) VALUES ('should-fail')")
			assert.Error(t, err)
			assert.Contains(t, strings.ToLower(err.Error()), "denied")

			_, err = readOnlyConn.Exec("UPDATE readonly_test SET data = 'hacked' WHERE id = 1")
			assert.Error(t, err)
			assert.Contains(t, strings.ToLower(err.Error()), "denied")

			_, err = readOnlyConn.Exec("DELETE FROM readonly_test WHERE id = 1")
			assert.Error(t, err)
			assert.Contains(t, strings.ToLower(err.Error()), "denied")

			_, err = readOnlyConn.Exec("CREATE TABLE hack_table (id INT)")
			assert.Error(t, err)
			assert.Contains(t, strings.ToLower(err.Error()), "denied")

			dropUserSafe(container.DB, username)
		})
	}
}

func Test_ReadOnlyUser_FutureTables_NoSelectPermission(t *testing.T) {
	env := config.GetEnv()
	container := connectToMariadbContainer(t, env.TestMariadb1011Port, tools.MariadbVersion1011)
	defer container.DB.Close()

	mariadbModel := createMariadbModel(container)
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	ctx := context.Background()

	username, password, err := mariadbModel.CreateReadOnlyUser(ctx, logger, nil, uuid.New())
	assert.NoError(t, err)

	_, err = container.DB.Exec(`DROP TABLE IF EXISTS future_table`)
	assert.NoError(t, err)
	_, err = container.DB.Exec(`
		CREATE TABLE future_table (
			id INT AUTO_INCREMENT PRIMARY KEY,
			data VARCHAR(255) NOT NULL
		)
	`)
	assert.NoError(t, err)
	_, err = container.DB.Exec(`INSERT INTO future_table (data) VALUES ('future_data')`)
	assert.NoError(t, err)

	readOnlyDSN := fmt.Sprintf("%s:%s@tcp(%s:%d)/%s?parseTime=true",
		username, password, container.Host, container.Port, container.Database)
	readOnlyConn, err := sqlx.Connect("mysql", readOnlyDSN)
	assert.NoError(t, err)
	defer readOnlyConn.Close()

	var data string
	err = readOnlyConn.Get(&data, "SELECT data FROM future_table LIMIT 1")
	assert.NoError(t, err)
	assert.Equal(t, "future_data", data)

	dropUserSafe(container.DB, username)
}

func Test_CreateReadOnlyUser_DatabaseNameWithDash_Success(t *testing.T) {
	env := config.GetEnv()
	container := connectToMariadbContainer(t, env.TestMariadb1011Port, tools.MariadbVersion1011)
	defer container.DB.Close()

	dashDbName := "test-db-with-dash"

	_, err := container.DB.Exec(fmt.Sprintf("DROP DATABASE IF EXISTS `%s`", dashDbName))
	assert.NoError(t, err)

	_, err = container.DB.Exec(fmt.Sprintf("CREATE DATABASE `%s`", dashDbName))
	assert.NoError(t, err)

	defer func() {
		_, _ = container.DB.Exec(fmt.Sprintf("DROP DATABASE IF EXISTS `%s`", dashDbName))
	}()

	dashDSN := fmt.Sprintf("%s:%s@tcp(%s:%d)/%s?parseTime=true",
		container.Username, container.Password, container.Host, container.Port, dashDbName)
	dashDB, err := sqlx.Connect("mysql", dashDSN)
	assert.NoError(t, err)
	defer dashDB.Close()

	_, err = dashDB.Exec(`
		CREATE TABLE dash_test (
			id INT AUTO_INCREMENT PRIMARY KEY,
			data VARCHAR(255) NOT NULL
		)
	`)
	assert.NoError(t, err)

	_, err = dashDB.Exec(`INSERT INTO dash_test (data) VALUES ('test1'), ('test2')`)
	assert.NoError(t, err)

	mariadbModel := &MariadbDatabase{
		Version:  tools.MariadbVersion1011,
		Host:     container.Host,
		Port:     container.Port,
		Username: container.Username,
		Password: container.Password,
		Database: &dashDbName,
		IsHttps:  false,
	}

	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	ctx := context.Background()

	username, password, err := mariadbModel.CreateReadOnlyUser(ctx, logger, nil, uuid.New())
	assert.NoError(t, err)
	assert.NotEmpty(t, username)
	assert.NotEmpty(t, password)
	assert.True(t, strings.HasPrefix(username, "pgs-"))

	readOnlyDSN := fmt.Sprintf("%s:%s@tcp(%s:%d)/%s?parseTime=true",
		username, password, container.Host, container.Port, dashDbName)
	readOnlyConn, err := sqlx.Connect("mysql", readOnlyDSN)
	assert.NoError(t, err)
	defer readOnlyConn.Close()

	var count int
	err = readOnlyConn.Get(&count, "SELECT COUNT(*) FROM dash_test")
	assert.NoError(t, err)
	assert.Equal(t, 2, count)

	_, err = readOnlyConn.Exec("INSERT INTO dash_test (data) VALUES ('should-fail')")
	assert.Error(t, err)
	assert.Contains(t, strings.ToLower(err.Error()), "denied")

	dropUserSafe(dashDB, username)
}

func Test_ReadOnlyUser_CannotDropOrAlterTables(t *testing.T) {
	env := config.GetEnv()
	container := connectToMariadbContainer(t, env.TestMariadb1011Port, tools.MariadbVersion1011)
	defer container.DB.Close()

	_, err := container.DB.Exec(`DROP TABLE IF EXISTS drop_test`)
	assert.NoError(t, err)
	_, err = container.DB.Exec(`
		CREATE TABLE drop_test (
			id INT AUTO_INCREMENT PRIMARY KEY,
			data VARCHAR(255) NOT NULL
		)
	`)
	assert.NoError(t, err)
	_, err = container.DB.Exec(`INSERT INTO drop_test (data) VALUES ('test1')`)
	assert.NoError(t, err)

	mariadbModel := createMariadbModel(container)
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	ctx := context.Background()

	username, password, err := mariadbModel.CreateReadOnlyUser(ctx, logger, nil, uuid.New())
	assert.NoError(t, err)

	readOnlyDSN := fmt.Sprintf("%s:%s@tcp(%s:%d)/%s?parseTime=true",
		username, password, container.Host, container.Port, container.Database)
	readOnlyConn, err := sqlx.Connect("mysql", readOnlyDSN)
	assert.NoError(t, err)
	defer readOnlyConn.Close()

	_, err = readOnlyConn.Exec("DROP TABLE drop_test")
	assert.Error(t, err)
	assert.Contains(t, strings.ToLower(err.Error()), "denied")

	_, err = readOnlyConn.Exec("ALTER TABLE drop_test ADD COLUMN new_col VARCHAR(100)")
	assert.Error(t, err)
	assert.Contains(t, strings.ToLower(err.Error()), "denied")

	_, err = readOnlyConn.Exec("TRUNCATE TABLE drop_test")
	assert.Error(t, err)
	assert.Contains(t, strings.ToLower(err.Error()), "denied")

	dropUserSafe(container.DB, username)
}

type MariadbContainer struct {
	Host     string
	Port     int
	Username string
	Password string
	Database string
	Version  tools.MariadbVersion
	DB       *sqlx.DB
}

func connectToMariadbContainer(
	t *testing.T,
	port string,
	version tools.MariadbVersion,
) *MariadbContainer {
	if port == "" {
		t.Skipf("MariaDB port not configured for version %s", version)
	}

	dbName := "testdb"
	host := "127.0.0.1"
	username := "root"
	password := "rootpassword"

	portInt, err := strconv.Atoi(port)
	assert.NoError(t, err)

	dsn := fmt.Sprintf("%s:%s@tcp(%s:%d)/%s?parseTime=true",
		username, password, host, portInt, dbName)

	db, err := sqlx.Connect("mysql", dsn)
	if err != nil {
		t.Skipf("Failed to connect to MariaDB %s: %v", version, err)
	}

	return &MariadbContainer{
		Host:     host,
		Port:     portInt,
		Username: username,
		Password: password,
		Database: dbName,
		Version:  version,
		DB:       db,
	}
}

func createMariadbModel(container *MariadbContainer) *MariadbDatabase {
	return &MariadbDatabase{
		Version:  container.Version,
		Host:     container.Host,
		Port:     container.Port,
		Username: container.Username,
		Password: container.Password,
		Database: &container.Database,
		IsHttps:  false,
	}
}

func dropUserSafe(db *sqlx.DB, username string) {
	// MariaDB 5.5 doesn't support DROP USER IF EXISTS, so we ignore errors
	_, _ = db.Exec(fmt.Sprintf("DROP USER '%s'@'%%'", username))
}
