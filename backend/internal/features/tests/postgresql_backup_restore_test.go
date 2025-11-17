package tests

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"postgresus-backend/internal/config"
	"postgresus-backend/internal/features/backups/backups"
	usecases_postgresql_backup "postgresus-backend/internal/features/backups/backups/usecases/postgresql"
	backups_config "postgresus-backend/internal/features/backups/config"
	"postgresus-backend/internal/features/databases"
	pgtypes "postgresus-backend/internal/features/databases/databases/postgresql"
	"postgresus-backend/internal/features/intervals"
	"postgresus-backend/internal/features/restores/models"
	usecases_postgresql_restore "postgresus-backend/internal/features/restores/usecases/postgresql"
	"postgresus-backend/internal/features/storages"
	local_storage "postgresus-backend/internal/features/storages/models/local"
	"postgresus-backend/internal/util/period"
	"postgresus-backend/internal/util/tools"
	"strconv"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq"
	"github.com/stretchr/testify/assert"
)

const createAndFillTableQuery = `
DROP TABLE IF EXISTS test_data;

CREATE TABLE test_data (
    id SERIAL PRIMARY KEY,
    name TEXT NOT NULL,
    value INTEGER NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

INSERT INTO test_data (name, value) VALUES
    ('test1', 100),
    ('test2', 200),
    ('test3', 300);
`

type PostgresContainer struct {
	Host     string
	Port     int
	Username string
	Password string
	Database string
	Version  string
	DB       *sqlx.DB
}

type TestDataItem struct {
	ID        int       `db:"id"`
	Name      string    `db:"name"`
	Value     int       `db:"value"`
	CreatedAt time.Time `db:"created_at"`
}

// Main test functions for each PostgreSQL version
func Test_BackupAndRestorePostgresql_RestoreIsSuccesful(t *testing.T) {
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
		{"PostgreSQL 18", "18", env.TestPostgres18Port},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			testBackupRestoreForVersion(t, tc.version, tc.port)
		})
	}
}

func Test_BackupAndRestorePostgresqlWithEncryption_RestoreIsSuccessful(t *testing.T) {
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
		{"PostgreSQL 18", "18", env.TestPostgres18Port},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			testBackupRestoreWithEncryptionForVersion(t, tc.version, tc.port)
		})
	}
}

func testBackupRestoreWithEncryptionForVersion(t *testing.T, pgVersion string, port string) {
	// Connect to pre-configured PostgreSQL container
	container, err := connectToPostgresContainer(pgVersion, port)
	assert.NoError(t, err)
	defer func() {
		if container.DB != nil {
			container.DB.Close()
		}
	}()

	_, err = container.DB.Exec(createAndFillTableQuery)
	assert.NoError(t, err)

	// Prepare data for backup
	backupID := uuid.New()
	pgVersionEnum := tools.GetPostgresqlVersionEnum(pgVersion)

	backupDb := &databases.Database{
		ID:   uuid.New(),
		Type: databases.DatabaseTypePostgres,
		Name: "Test Database",
		Postgresql: &pgtypes.PostgresqlDatabase{
			Version:  pgVersionEnum,
			Host:     container.Host,
			Port:     container.Port,
			Username: container.Username,
			Password: container.Password,
			Database: &container.Database,
			IsHttps:  false,
		},
	}

	storageID := uuid.New()
	backupConfig := &backups_config.BackupConfig{
		DatabaseID:       backupDb.ID,
		IsBackupsEnabled: true,
		StorePeriod:      period.PeriodDay,
		BackupInterval:   &intervals.Interval{Interval: intervals.IntervalDaily},
		StorageID:        &storageID,
		CpuCount:         1,
		Encryption:       backups_config.BackupEncryptionEncrypted,
	}

	storage := &storages.Storage{
		WorkspaceID:  uuid.New(),
		Type:         storages.StorageTypeLocal,
		Name:         "Test Storage",
		LocalStorage: &local_storage.LocalStorage{},
	}

	// Make backup
	progressTracker := func(completedMBs float64) {}
	metadata, err := usecases_postgresql_backup.GetCreatePostgresqlBackupUsecase().Execute(
		context.Background(),
		backupID,
		backupConfig,
		backupDb,
		storage,
		progressTracker,
	)
	assert.NoError(t, err)
	assert.NotNil(t, metadata)

	// Verify encryption metadata is set
	assert.Equal(t, backups_config.BackupEncryptionEncrypted, metadata.Encryption)
	assert.NotNil(t, metadata.EncryptionSalt)
	assert.NotNil(t, metadata.EncryptionIV)

	// Create new database
	newDBName := "restoreddb_encrypted"
	_, err = container.DB.Exec(fmt.Sprintf("DROP DATABASE IF EXISTS %s;", newDBName))
	assert.NoError(t, err)

	_, err = container.DB.Exec(fmt.Sprintf("CREATE DATABASE %s;", newDBName))
	assert.NoError(t, err)

	// Connect to the new database
	newDSN := fmt.Sprintf("host=%s port=%d user=%s password=%s dbname=%s sslmode=disable",
		container.Host, container.Port, container.Username, container.Password, newDBName)
	newDB, err := sqlx.Connect("postgres", newDSN)
	assert.NoError(t, err)
	defer newDB.Close()

	// Setup data for restore with encryption metadata
	completedBackup := &backups.Backup{
		ID:             backupID,
		DatabaseID:     backupDb.ID,
		StorageID:      storage.ID,
		Status:         backups.BackupStatusCompleted,
		CreatedAt:      time.Now().UTC(),
		EncryptionSalt: metadata.EncryptionSalt,
		EncryptionIV:   metadata.EncryptionIV,
		Encryption:     metadata.Encryption,
	}

	restoreID := uuid.New()
	restore := models.Restore{
		ID:     restoreID,
		Backup: completedBackup,
		Postgresql: &pgtypes.PostgresqlDatabase{
			Version:  pgVersionEnum,
			Host:     container.Host,
			Port:     container.Port,
			Username: container.Username,
			Password: container.Password,
			Database: &newDBName,
			IsHttps:  false,
		},
	}

	// Restore the encrypted backup
	restoreBackupUC := usecases_postgresql_restore.GetRestorePostgresqlBackupUsecase()
	err = restoreBackupUC.Execute(backupDb, backupConfig, restore, completedBackup, storage)
	assert.NoError(t, err)

	// Verify restored table exists
	var tableExists bool
	err = newDB.Get(
		&tableExists,
		"SELECT EXISTS (SELECT FROM information_schema.tables WHERE table_name = 'test_data')",
	)
	assert.NoError(t, err)
	assert.True(t, tableExists, "Table 'test_data' should exist in restored database")

	// Verify data integrity
	verifyDataIntegrity(t, container.DB, newDB)

	// Clean up the backup file after the test
	err = os.Remove(filepath.Join(config.GetEnv().DataFolder, backupID.String()))
	if err != nil {
		t.Logf("Warning: Failed to delete backup file: %v", err)
	}
}

// Run a test for a specific PostgreSQL version
func testBackupRestoreForVersion(t *testing.T, pgVersion string, port string) {
	// Connect to pre-configured PostgreSQL container
	container, err := connectToPostgresContainer(pgVersion, port)
	assert.NoError(t, err)
	defer func() {
		if container.DB != nil {
			container.DB.Close()
		}
	}()

	_, err = container.DB.Exec(createAndFillTableQuery)
	assert.NoError(t, err)

	// Prepare data for backup
	backupID := uuid.New()
	pgVersionEnum := tools.GetPostgresqlVersionEnum(pgVersion)

	backupDb := &databases.Database{
		ID:   uuid.New(),
		Type: databases.DatabaseTypePostgres,
		Name: "Test Database",
		Postgresql: &pgtypes.PostgresqlDatabase{
			Version:  pgVersionEnum,
			Host:     container.Host,
			Port:     container.Port,
			Username: container.Username,
			Password: container.Password,
			Database: &container.Database,
			IsHttps:  false,
		},
	}

	storageID := uuid.New()
	backupConfig := &backups_config.BackupConfig{
		DatabaseID:       backupDb.ID,
		IsBackupsEnabled: true,
		StorePeriod:      period.PeriodDay,
		BackupInterval:   &intervals.Interval{Interval: intervals.IntervalDaily},
		StorageID:        &storageID,
		CpuCount:         1,
	}

	storage := &storages.Storage{
		WorkspaceID:  uuid.New(),
		Type:         storages.StorageTypeLocal,
		Name:         "Test Storage",
		LocalStorage: &local_storage.LocalStorage{},
	}

	// Make backup
	progressTracker := func(completedMBs float64) {}
	_, err = usecases_postgresql_backup.GetCreatePostgresqlBackupUsecase().Execute(
		context.Background(),
		backupID,
		backupConfig,
		backupDb,
		storage,
		progressTracker,
	)
	assert.NoError(t, err)

	// Create new database
	newDBName := "restoreddb"
	_, err = container.DB.Exec(fmt.Sprintf("DROP DATABASE IF EXISTS %s;", newDBName))
	assert.NoError(t, err)

	_, err = container.DB.Exec(fmt.Sprintf("CREATE DATABASE %s;", newDBName))
	assert.NoError(t, err)

	// Connect to the new database
	newDSN := fmt.Sprintf("host=%s port=%d user=%s password=%s dbname=%s sslmode=disable",
		container.Host, container.Port, container.Username, container.Password, newDBName)
	newDB, err := sqlx.Connect("postgres", newDSN)
	assert.NoError(t, err)
	defer newDB.Close()

	// Setup data for restore
	completedBackup := &backups.Backup{
		ID:         backupID,
		DatabaseID: backupDb.ID,
		StorageID:  storage.ID,
		Status:     backups.BackupStatusCompleted,
		CreatedAt:  time.Now().UTC(),
	}

	restoreID := uuid.New()
	restore := models.Restore{
		ID:     restoreID,
		Backup: completedBackup,
		Postgresql: &pgtypes.PostgresqlDatabase{
			Version:  pgVersionEnum,
			Host:     container.Host,
			Port:     container.Port,
			Username: container.Username,
			Password: container.Password,
			Database: &newDBName,
			IsHttps:  false,
		},
	}

	// Restore the backup
	restoreBackupUC := usecases_postgresql_restore.GetRestorePostgresqlBackupUsecase()
	err = restoreBackupUC.Execute(backupDb, backupConfig, restore, completedBackup, storage)
	assert.NoError(t, err)

	// Verify restored table exists
	var tableExists bool
	err = newDB.Get(
		&tableExists,
		"SELECT EXISTS (SELECT FROM information_schema.tables WHERE table_name = 'test_data')",
	)
	assert.NoError(t, err)
	assert.True(t, tableExists, "Table 'test_data' should exist in restored database")

	// Verify data integrity
	verifyDataIntegrity(t, container.DB, newDB)

	// Clean up the backup file after the test
	err = os.Remove(filepath.Join(config.GetEnv().DataFolder, backupID.String()))
	if err != nil {
		t.Logf("Warning: Failed to delete backup file: %v", err)
	}
}

// verifyDataIntegrity compares data in the original and restored databases
func verifyDataIntegrity(t *testing.T, originalDB *sqlx.DB, restoredDB *sqlx.DB) {
	var originalData []TestDataItem
	var restoredData []TestDataItem

	err := originalDB.Select(&originalData, "SELECT * FROM test_data ORDER BY id")
	assert.NoError(t, err)

	err = restoredDB.Select(&restoredData, "SELECT * FROM test_data ORDER BY id")
	assert.NoError(t, err)

	assert.Equal(t, len(originalData), len(restoredData), "Should have same number of rows")

	// Only compare data if both slices have elements (to avoid panic)
	if len(originalData) > 0 && len(restoredData) > 0 {
		for i := range originalData {
			assert.Equal(t, originalData[i].ID, restoredData[i].ID, "ID should match")
			assert.Equal(t, originalData[i].Name, restoredData[i].Name, "Name should match")
			assert.Equal(t, originalData[i].Value, restoredData[i].Value, "Value should match")
		}
	}
}

func connectToPostgresContainer(version string, port string) (*PostgresContainer, error) {
	dbName := "testdb"
	password := "testpassword"
	username := "testuser"
	host := "localhost"

	portInt, err := strconv.Atoi(port)
	if err != nil {
		return nil, fmt.Errorf("failed to parse port: %w", err)
	}

	dsn := fmt.Sprintf("host=%s port=%d user=%s password=%s dbname=%s sslmode=disable",
		host, portInt, username, password, dbName)

	db, err := sqlx.Connect("postgres", dsn)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to database: %w", err)
	}

	return &PostgresContainer{
		Host:     host,
		Port:     portInt,
		Username: username,
		Password: password,
		Database: dbName,
		Version:  version,
		DB:       db,
	}, nil
}
