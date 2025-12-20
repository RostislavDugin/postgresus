package mysql

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
		version tools.MysqlVersion
		port    string
	}{
		{"MySQL 5.7", tools.MysqlVersion57, env.TestMysql57Port},
		{"MySQL 8.0", tools.MysqlVersion80, env.TestMysql80Port},
		{"MySQL 8.4", tools.MysqlVersion84, env.TestMysql84Port},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			container := connectToMysqlContainer(t, tc.port, tc.version)
			defer container.DB.Close()

			mysqlModel := createMysqlModel(container)
			logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
			ctx := context.Background()

			isReadOnly, err := mysqlModel.IsUserReadOnly(ctx, logger, nil, uuid.New())
			assert.NoError(t, err)
			assert.False(t, isReadOnly, "Root user should not be read-only")
		})
	}
}

func Test_CreateReadOnlyUser_UserCanReadButNotWrite(t *testing.T) {
	env := config.GetEnv()
	cases := []struct {
		name    string
		version tools.MysqlVersion
		port    string
	}{
		{"MySQL 5.7", tools.MysqlVersion57, env.TestMysql57Port},
		{"MySQL 8.0", tools.MysqlVersion80, env.TestMysql80Port},
		{"MySQL 8.4", tools.MysqlVersion84, env.TestMysql84Port},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			container := connectToMysqlContainer(t, tc.port, tc.version)
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

			mysqlModel := createMysqlModel(container)
			logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
			ctx := context.Background()

			username, password, err := mysqlModel.CreateReadOnlyUser(ctx, logger, nil, uuid.New())
			assert.NoError(t, err)
			assert.NotEmpty(t, username)
			assert.NotEmpty(t, password)
			assert.True(t, strings.HasPrefix(username, "postgresus-"))

			readOnlyModel := &MysqlDatabase{
				Version:  mysqlModel.Version,
				Host:     mysqlModel.Host,
				Port:     mysqlModel.Port,
				Username: username,
				Password: password,
				Database: mysqlModel.Database,
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

			_, err = container.DB.Exec(fmt.Sprintf("DROP USER IF EXISTS '%s'@'%%'", username))
			assert.NoError(t, err)
		})
	}
}

func Test_ReadOnlyUser_FutureTables_NoSelectPermission(t *testing.T) {
	env := config.GetEnv()
	container := connectToMysqlContainer(t, env.TestMysql80Port, tools.MysqlVersion80)
	defer container.DB.Close()

	mysqlModel := createMysqlModel(container)
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	ctx := context.Background()

	username, password, err := mysqlModel.CreateReadOnlyUser(ctx, logger, nil, uuid.New())
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

	_, err = container.DB.Exec(fmt.Sprintf("DROP USER IF EXISTS '%s'@'%%'", username))
	assert.NoError(t, err)
}

func Test_CreateReadOnlyUser_DatabaseNameWithDash_Success(t *testing.T) {
	env := config.GetEnv()
	container := connectToMysqlContainer(t, env.TestMysql80Port, tools.MysqlVersion80)
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

	mysqlModel := &MysqlDatabase{
		Version:  tools.MysqlVersion80,
		Host:     container.Host,
		Port:     container.Port,
		Username: container.Username,
		Password: container.Password,
		Database: &dashDbName,
		IsHttps:  false,
	}

	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	ctx := context.Background()

	username, password, err := mysqlModel.CreateReadOnlyUser(ctx, logger, nil, uuid.New())
	assert.NoError(t, err)
	assert.NotEmpty(t, username)
	assert.NotEmpty(t, password)
	assert.True(t, strings.HasPrefix(username, "postgresus-"))

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

	_, err = dashDB.Exec(fmt.Sprintf("DROP USER IF EXISTS '%s'@'%%'", username))
	assert.NoError(t, err)
}

func Test_ReadOnlyUser_CannotDropOrAlterTables(t *testing.T) {
	env := config.GetEnv()
	container := connectToMysqlContainer(t, env.TestMysql80Port, tools.MysqlVersion80)
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

	mysqlModel := createMysqlModel(container)
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	ctx := context.Background()

	username, password, err := mysqlModel.CreateReadOnlyUser(ctx, logger, nil, uuid.New())
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

	_, err = container.DB.Exec(fmt.Sprintf("DROP USER IF EXISTS '%s'@'%%'", username))
	assert.NoError(t, err)
}

type MysqlContainer struct {
	Host     string
	Port     int
	Username string
	Password string
	Database string
	Version  tools.MysqlVersion
	DB       *sqlx.DB
}

func connectToMysqlContainer(
	t *testing.T,
	port string,
	version tools.MysqlVersion,
) *MysqlContainer {
	if port == "" {
		t.Skipf("MySQL port not configured for version %s", version)
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
		t.Skipf("Failed to connect to MySQL %s: %v", version, err)
	}

	return &MysqlContainer{
		Host:     host,
		Port:     portInt,
		Username: username,
		Password: password,
		Database: dbName,
		Version:  version,
		DB:       db,
	}
}

func createMysqlModel(container *MysqlContainer) *MysqlDatabase {
	return &MysqlDatabase{
		Version:  container.Version,
		Host:     container.Host,
		Port:     container.Port,
		Username: container.Username,
		Password: container.Password,
		Database: &container.Database,
		IsHttps:  false,
	}
}
