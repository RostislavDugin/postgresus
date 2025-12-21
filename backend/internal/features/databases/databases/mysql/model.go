package mysql

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log/slog"
	"regexp"
	"time"

	"postgresus-backend/internal/util/encryption"
	"postgresus-backend/internal/util/tools"

	_ "github.com/go-sql-driver/mysql"
	"github.com/google/uuid"
)

type MysqlDatabase struct {
	ID         uuid.UUID  `json:"id"         gorm:"primaryKey;type:uuid;default:gen_random_uuid()"`
	DatabaseID *uuid.UUID `json:"databaseId" gorm:"type:uuid;column:database_id"`

	Version tools.MysqlVersion `json:"version" gorm:"type:text;not null"`

	Host     string  `json:"host"     gorm:"type:text;not null"`
	Port     int     `json:"port"     gorm:"type:int;not null"`
	Username string  `json:"username" gorm:"type:text;not null"`
	Password string  `json:"password" gorm:"type:text;not null"`
	Database *string `json:"database" gorm:"type:text"`
	IsHttps  bool    `json:"isHttps"  gorm:"type:boolean;default:false"`
}

func (m *MysqlDatabase) TableName() string {
	return "mysql_databases"
}

func (m *MysqlDatabase) Validate() error {
	if m.Host == "" {
		return errors.New("host is required")
	}
	if m.Port == 0 {
		return errors.New("port is required")
	}
	if m.Username == "" {
		return errors.New("username is required")
	}
	if m.Password == "" {
		return errors.New("password is required")
	}
	return nil
}

func (m *MysqlDatabase) TestConnection(
	logger *slog.Logger,
	encryptor encryption.FieldEncryptor,
	databaseID uuid.UUID,
) error {
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	if m.Database == nil || *m.Database == "" {
		return errors.New("database name is required for MySQL backup")
	}

	password, err := decryptPasswordIfNeeded(m.Password, encryptor, databaseID)
	if err != nil {
		return fmt.Errorf("failed to decrypt password: %w", err)
	}

	dsn := m.buildDSN(password, *m.Database)

	db, err := sql.Open("mysql", dsn)
	if err != nil {
		return fmt.Errorf("failed to connect to MySQL database '%s': %w", *m.Database, err)
	}
	defer func() {
		if closeErr := db.Close(); closeErr != nil {
			logger.Error("Failed to close MySQL connection", "error", closeErr)
		}
	}()

	db.SetConnMaxLifetime(15 * time.Second)
	db.SetMaxOpenConns(1)
	db.SetMaxIdleConns(1)

	if err := db.PingContext(ctx); err != nil {
		return fmt.Errorf("failed to ping MySQL database '%s': %w", *m.Database, err)
	}

	detectedVersion, err := detectMysqlVersion(ctx, db)
	if err != nil {
		return err
	}
	m.Version = detectedVersion

	return nil
}

func (m *MysqlDatabase) HideSensitiveData() {
	if m == nil {
		return
	}
	m.Password = ""
}

func (m *MysqlDatabase) Update(incoming *MysqlDatabase) {
	m.Version = incoming.Version
	m.Host = incoming.Host
	m.Port = incoming.Port
	m.Username = incoming.Username
	m.Database = incoming.Database
	m.IsHttps = incoming.IsHttps

	if incoming.Password != "" {
		m.Password = incoming.Password
	}
}

func (m *MysqlDatabase) EncryptSensitiveFields(
	databaseID uuid.UUID,
	encryptor encryption.FieldEncryptor,
) error {
	if m.Password != "" {
		encrypted, err := encryptor.Encrypt(databaseID, m.Password)
		if err != nil {
			return err
		}
		m.Password = encrypted
	}
	return nil
}

func (m *MysqlDatabase) PopulateVersionIfEmpty(
	logger *slog.Logger,
	encryptor encryption.FieldEncryptor,
	databaseID uuid.UUID,
) error {
	if m.Version != "" {
		return nil
	}
	return m.PopulateVersion(logger, encryptor, databaseID)
}

func (m *MysqlDatabase) PopulateVersion(
	logger *slog.Logger,
	encryptor encryption.FieldEncryptor,
	databaseID uuid.UUID,
) error {
	if m.Database == nil || *m.Database == "" {
		return nil
	}

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	password, err := decryptPasswordIfNeeded(m.Password, encryptor, databaseID)
	if err != nil {
		return fmt.Errorf("failed to decrypt password: %w", err)
	}

	dsn := m.buildDSN(password, *m.Database)

	db, err := sql.Open("mysql", dsn)
	if err != nil {
		return fmt.Errorf("failed to connect to database: %w", err)
	}
	defer func() {
		if closeErr := db.Close(); closeErr != nil {
			logger.Error("Failed to close connection", "error", closeErr)
		}
	}()

	detectedVersion, err := detectMysqlVersion(ctx, db)
	if err != nil {
		return err
	}

	m.Version = detectedVersion
	return nil
}

func (m *MysqlDatabase) IsUserReadOnly(
	ctx context.Context,
	logger *slog.Logger,
	encryptor encryption.FieldEncryptor,
	databaseID uuid.UUID,
) (bool, error) {
	password, err := decryptPasswordIfNeeded(m.Password, encryptor, databaseID)
	if err != nil {
		return false, fmt.Errorf("failed to decrypt password: %w", err)
	}

	dsn := m.buildDSN(password, *m.Database)

	db, err := sql.Open("mysql", dsn)
	if err != nil {
		return false, fmt.Errorf("failed to connect to database: %w", err)
	}
	defer func() {
		if closeErr := db.Close(); closeErr != nil {
			logger.Error("Failed to close connection", "error", closeErr)
		}
	}()

	rows, err := db.QueryContext(ctx, "SHOW GRANTS FOR CURRENT_USER()")
	if err != nil {
		return false, fmt.Errorf("failed to check grants: %w", err)
	}
	defer func() { _ = rows.Close() }()

	writePrivileges := []string{
		"INSERT", "UPDATE", "DELETE", "CREATE", "DROP", "ALTER",
		"INDEX", "GRANT OPTION", "ALL PRIVILEGES", "SUPER",
	}

	for rows.Next() {
		var grant string
		if err := rows.Scan(&grant); err != nil {
			return false, fmt.Errorf("failed to scan grant: %w", err)
		}

		for _, priv := range writePrivileges {
			if regexp.MustCompile(`(?i)\b` + priv + `\b`).MatchString(grant) {
				return false, nil
			}
		}
	}

	if err := rows.Err(); err != nil {
		return false, fmt.Errorf("error iterating grants: %w", err)
	}

	return true, nil
}

func (m *MysqlDatabase) CreateReadOnlyUser(
	ctx context.Context,
	logger *slog.Logger,
	encryptor encryption.FieldEncryptor,
	databaseID uuid.UUID,
) (string, string, error) {
	password, err := decryptPasswordIfNeeded(m.Password, encryptor, databaseID)
	if err != nil {
		return "", "", fmt.Errorf("failed to decrypt password: %w", err)
	}

	dsn := m.buildDSN(password, *m.Database)

	db, err := sql.Open("mysql", dsn)
	if err != nil {
		return "", "", fmt.Errorf("failed to connect to database: %w", err)
	}
	defer func() {
		if closeErr := db.Close(); closeErr != nil {
			logger.Error("Failed to close connection", "error", closeErr)
		}
	}()

	maxRetries := 3
	for attempt := range maxRetries {
		newUsername := fmt.Sprintf("postgresus-%s", uuid.New().String()[:8])
		newPassword := uuid.New().String()

		tx, err := db.BeginTx(ctx, nil)
		if err != nil {
			return "", "", fmt.Errorf("failed to begin transaction: %w", err)
		}

		success := false
		defer func() {
			if !success {
				if rollbackErr := tx.Rollback(); rollbackErr != nil {
					logger.Error("Failed to rollback transaction", "error", rollbackErr)
				}
			}
		}()

		_, err = tx.ExecContext(ctx, fmt.Sprintf(
			"CREATE USER '%s'@'%%' IDENTIFIED BY '%s'",
			newUsername,
			newPassword,
		))
		if err != nil {
			if attempt < maxRetries-1 {
				continue
			}
			return "", "", fmt.Errorf("failed to create user: %w", err)
		}

		_, err = tx.ExecContext(ctx, fmt.Sprintf(
			"GRANT SELECT, SHOW VIEW, LOCK TABLES, TRIGGER, EVENT ON `%s`.* TO '%s'@'%%'",
			*m.Database,
			newUsername,
		))
		if err != nil {
			return "", "", fmt.Errorf("failed to grant database privileges: %w", err)
		}

		_, err = tx.ExecContext(ctx, fmt.Sprintf(
			"GRANT PROCESS ON *.* TO '%s'@'%%'",
			newUsername,
		))
		if err != nil {
			return "", "", fmt.Errorf("failed to grant PROCESS privilege: %w", err)
		}

		_, err = tx.ExecContext(ctx, "FLUSH PRIVILEGES")
		if err != nil {
			return "", "", fmt.Errorf("failed to flush privileges: %w", err)
		}

		if err := tx.Commit(); err != nil {
			return "", "", fmt.Errorf("failed to commit transaction: %w", err)
		}

		success = true
		logger.Info(
			"Read-only MySQL user created successfully",
			"username",
			newUsername,
		)
		return newUsername, newPassword, nil
	}

	return "", "", errors.New("failed to generate unique username after 3 attempts")
}

func (m *MysqlDatabase) buildDSN(password string, database string) string {
	tlsConfig := "false"
	if m.IsHttps {
		tlsConfig = "true"
	}

	return fmt.Sprintf(
		"%s:%s@tcp(%s:%d)/%s?parseTime=true&timeout=15s&tls=%s&charset=utf8mb4",
		m.Username,
		password,
		m.Host,
		m.Port,
		database,
		tlsConfig,
	)
}

// detectMysqlVersion parses VERSION() output to detect MySQL version
// Minor versions are mapped to the closest supported version (e.g., 8.1 → 8.0, 8.4+ → 8.4)
func detectMysqlVersion(ctx context.Context, db *sql.DB) (tools.MysqlVersion, error) {
	var versionStr string
	err := db.QueryRowContext(ctx, "SELECT VERSION()").Scan(&versionStr)
	if err != nil {
		return "", fmt.Errorf("failed to query MySQL version: %w", err)
	}

	re := regexp.MustCompile(`^(\d+)\.(\d+)`)
	matches := re.FindStringSubmatch(versionStr)
	if len(matches) < 3 {
		return "", fmt.Errorf("could not parse MySQL version: %s", versionStr)
	}

	major := matches[1]
	minor := matches[2]

	return mapMysqlVersion(major, minor)
}

func mapMysqlVersion(major, minor string) (tools.MysqlVersion, error) {
	switch major {
	case "5":
		return tools.MysqlVersion57, nil
	case "8":
		return mapMysql8xVersion(minor), nil
	case "9":
		return tools.MysqlVersion84, nil
	default:
		return "", fmt.Errorf(
			"unsupported MySQL major version: %s (supported: 5.x, 8.x, 9.x)",
			major,
		)
	}
}

func mapMysql8xVersion(minor string) tools.MysqlVersion {
	switch minor {
	case "0", "1", "2", "3":
		return tools.MysqlVersion80
	default:
		return tools.MysqlVersion84
	}
}

func decryptPasswordIfNeeded(
	password string,
	encryptor encryption.FieldEncryptor,
	databaseID uuid.UUID,
) (string, error) {
	if encryptor == nil {
		return password, nil
	}
	return encryptor.Decrypt(databaseID, password)
}
