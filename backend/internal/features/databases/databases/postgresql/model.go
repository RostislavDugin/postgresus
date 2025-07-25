package postgresql

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"postgresus-backend/internal/util/tools"
	"regexp"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

type PostgresqlDatabase struct {
	ID uuid.UUID `json:"id" gorm:"primaryKey;type:uuid;default:gen_random_uuid()"`

	DatabaseID *uuid.UUID `json:"databaseId" gorm:"type:uuid;column:database_id"`
	RestoreID  *uuid.UUID `json:"restoreId"  gorm:"type:uuid;column:restore_id"`

	Version tools.PostgresqlVersion `json:"version" gorm:"type:text;not null"`

	// connection data
	Host     string  `json:"host"     gorm:"type:text;not null"`
	Port     int     `json:"port"     gorm:"type:int;not null"`
	Username string  `json:"username" gorm:"type:text;not null"`
	Password string  `json:"password" gorm:"type:text;not null"`
	Database *string `json:"database" gorm:"type:text"`
	IsHttps  bool    `json:"isHttps"  gorm:"type:boolean;default:false"`
}

func (p *PostgresqlDatabase) TableName() string {
	return "postgresql_databases"
}

func (p *PostgresqlDatabase) Validate() error {
	if p.Version == "" {
		return errors.New("version is required")
	}

	if p.Host == "" {
		return errors.New("host is required")
	}

	if p.Port == 0 {
		return errors.New("port is required")
	}

	if p.Username == "" {
		return errors.New("username is required")
	}

	if p.Password == "" {
		return errors.New("password is required")
	}

	return nil
}

func (p *PostgresqlDatabase) TestConnection(logger *slog.Logger) error {
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	return testSingleDatabaseConnection(logger, ctx, p)
}

// testSingleDatabaseConnection tests connection to a specific database for pg_dump
func testSingleDatabaseConnection(
	logger *slog.Logger,
	ctx context.Context,
	postgresDb *PostgresqlDatabase,
) error {
	// For single database backup, we need to connect to the specific database
	if postgresDb.Database == nil || *postgresDb.Database == "" {
		return errors.New("database name is required for single database backup (pg_dump)")
	}

	// Build connection string for the specific database
	connStr := buildConnectionStringForDB(postgresDb, *postgresDb.Database)

	// Test connection
	conn, err := pgx.Connect(ctx, connStr)
	if err != nil {
		// TODO make more readable errors:
		// - handle wrong creds
		// - handle wrong database name
		// - handle wrong protocol
		return fmt.Errorf("failed to connect to database '%s': %w", *postgresDb.Database, err)
	}
	defer func() {
		if closeErr := conn.Close(ctx); closeErr != nil {
			logger.Error("Failed to close connection", "error", closeErr)
		}
	}()

	// Check version after successful connection
	if err := verifyDatabaseVersion(ctx, conn, postgresDb.Version); err != nil {
		return err
	}

	// Test if we can perform basic operations (like pg_dump would need)
	if err := testBasicOperations(ctx, conn, *postgresDb.Database); err != nil {
		return fmt.Errorf(
			"basic operations test failed for database '%s': %w",
			*postgresDb.Database,
			err,
		)
	}

	return nil
}

// verifyDatabaseVersion checks if the actual database version matches the specified version
func verifyDatabaseVersion(
	ctx context.Context,
	conn *pgx.Conn,
	expectedVersion tools.PostgresqlVersion,
) error {
	var versionStr string
	err := conn.QueryRow(ctx, "SELECT version()").Scan(&versionStr)
	if err != nil {
		return fmt.Errorf("failed to query database version: %w", err)
	}

	// Parse version from string like "PostgreSQL 14.2 on x86_64-pc-linux-gnu..."
	re := regexp.MustCompile(`PostgreSQL (\d+)\.`)
	matches := re.FindStringSubmatch(versionStr)
	if len(matches) < 2 {
		return fmt.Errorf("could not parse version from: %s", versionStr)
	}

	actualVersion := tools.GetPostgresqlVersionEnum(matches[1])
	if actualVersion != expectedVersion {
		return fmt.Errorf(
			"you specified wrong version. Real version is %s, but you specified %s",
			actualVersion,
			expectedVersion,
		)
	}

	return nil
}

// testBasicOperations tests basic operations that backup tools need
func testBasicOperations(ctx context.Context, conn *pgx.Conn, dbName string) error {
	var hasCreatePriv bool

	err := conn.QueryRow(ctx, "SELECT has_database_privilege(current_user, current_database(), 'CONNECT')").
		Scan(&hasCreatePriv)
	if err != nil {
		return fmt.Errorf("cannot check database privileges: %w", err)
	}

	if !hasCreatePriv {
		return fmt.Errorf("user does not have CONNECT privilege on database '%s'", dbName)
	}

	return nil
}

// buildConnectionStringForDB builds connection string for specific database
func buildConnectionStringForDB(p *PostgresqlDatabase, dbName string) string {
	sslMode := "disable"
	if p.IsHttps {
		sslMode = "require"
	}

	return fmt.Sprintf("host=%s port=%d user=%s password=%s dbname=%s sslmode=%s",
		p.Host,
		p.Port,
		p.Username,
		p.Password,
		dbName,
		sslMode,
	)
}
