package postgresql

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"postgresus-backend/internal/util/tools"
	"regexp"
	"slices"
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

func (p *PostgresqlDatabase) HideSensitiveData() {
	if p == nil {
		return
	}

	p.Password = ""
}

func (p *PostgresqlDatabase) Update(incoming *PostgresqlDatabase) {
	p.Version = incoming.Version
	p.Host = incoming.Host
	p.Port = incoming.Port
	p.Username = incoming.Username
	p.Database = incoming.Database
	p.IsHttps = incoming.IsHttps

	if incoming.Password != "" {
		p.Password = incoming.Password
	}
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

// ListAccessibleDatabases connects to the cluster and returns databases the user can CONNECT to
func (p *PostgresqlDatabase) ListAccessibleDatabases(logger *slog.Logger) ([]string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	// Try common system databases for initial connection
	candidates := []string{"postgres", "template1"}
	var lastErr error

	for _, dbName := range candidates {
		connStr := buildConnectionStringForDB(p, dbName)

		conn, err := pgx.Connect(ctx, connStr)
		if err != nil {
			lastErr = fmt.Errorf("failed to connect to database '%s': %w", dbName, err)
			continue
		}

		// Ensure we always close the connection
		defer func() {
			if closeErr := conn.Close(ctx); closeErr != nil {
				logger.Error("Failed to close connection", "error", closeErr)
			}
		}()

		// Verify version after successful connection
		if err := verifyDatabaseVersion(ctx, conn, p.Version); err != nil {
			return nil, err
		}

		// List accessible databases (exclude templates, require CONNECT privilege)
		rows, err := conn.Query(ctx, `
			SELECT datname
			FROM pg_database
			WHERE datistemplate = false
			  AND datallowconn = true
			  AND has_database_privilege(current_user, datname, 'CONNECT')
			ORDER BY datname`)
		if err != nil {
			return nil, fmt.Errorf("failed to list databases: %w", err)
		}
		defer rows.Close()

		var dbs []string
		for rows.Next() {
			var name string
			if scanErr := rows.Scan(&name); scanErr != nil {
				return nil, fmt.Errorf("failed to scan database name: %w", scanErr)
			}
			dbs = append(dbs, name)
		}
		if err := rows.Err(); err != nil {
			return nil, fmt.Errorf("error iterating over database rows: %w", err)
		}

		return dbs, nil
	}

	if lastErr != nil {
		return nil, lastErr
	}

	return nil, errors.New("unable to connect to cluster using known system databases")
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

func (p *PostgresqlDatabase) InstallExtensions(extensions []tools.PostgresqlExtension) error {
	if len(extensions) == 0 {
		return nil
	}

	if p.Database == nil || *p.Database == "" {
		return errors.New("database name is required for installing extensions")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Build connection string for the specific database
	connStr := buildConnectionStringForDB(p, *p.Database)

	// Connect to database
	conn, err := pgx.Connect(ctx, connStr)
	if err != nil {
		return fmt.Errorf("failed to connect to database '%s': %w", *p.Database, err)
	}
	defer func() {
		if closeErr := conn.Close(ctx); closeErr != nil {
			fmt.Println("failed to close connection: %w", closeErr)
		}
	}()

	// Check which extensions are already installed
	installedExtensions, err := p.getInstalledExtensions(ctx, conn)
	if err != nil {
		return fmt.Errorf("failed to check installed extensions: %w", err)
	}

	// Install missing extensions
	for _, extension := range extensions {
		if contains(installedExtensions, string(extension)) {
			continue // Extension already installed
		}

		if err := p.installExtension(ctx, conn, string(extension)); err != nil {
			return fmt.Errorf("failed to install extension '%s': %w", extension, err)
		}
	}

	return nil
}

// getInstalledExtensions queries the database for currently installed extensions
func (p *PostgresqlDatabase) getInstalledExtensions(
	ctx context.Context,
	conn *pgx.Conn,
) ([]string, error) {
	query := "SELECT extname FROM pg_extension"

	rows, err := conn.Query(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to query installed extensions: %w", err)
	}
	defer rows.Close()

	var extensions []string
	for rows.Next() {
		var extname string

		if err := rows.Scan(&extname); err != nil {
			return nil, fmt.Errorf("failed to scan extension name: %w", err)
		}

		extensions = append(extensions, extname)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating over extension rows: %w", err)
	}

	return extensions, nil
}

// installExtension installs a single PostgreSQL extension
func (p *PostgresqlDatabase) installExtension(
	ctx context.Context,
	conn *pgx.Conn,
	extensionName string,
) error {
	query := fmt.Sprintf("CREATE EXTENSION IF NOT EXISTS %s", extensionName)

	_, err := conn.Exec(ctx, query)
	if err != nil {
		return fmt.Errorf("failed to execute CREATE EXTENSION: %w", err)
	}

	return nil
}

// contains checks if a string slice contains a specific string
func contains(slice []string, item string) bool {
	return slices.Contains(slice, item)
}
