package mariadb

import (
	"context"
	"crypto/tls"
	"database/sql"
	"errors"
	"fmt"
	"log/slog"
	"regexp"
	"sort"
	"strings"
	"time"

	"github.com/go-sql-driver/mysql"
	"github.com/google/uuid"
	"gorm.io/gorm"

	"databasus-backend/internal/util/encryption"
	"databasus-backend/internal/util/tools"
)

type MariadbDatabase struct {
	ID         uuid.UUID  `json:"id"         gorm:"primaryKey;type:uuid;default:gen_random_uuid()"`
	DatabaseID *uuid.UUID `json:"databaseId" gorm:"type:uuid;column:database_id"`

	Version tools.MariadbVersion `json:"version" gorm:"type:text;not null"`

	Host                string   `json:"host"                gorm:"type:text;not null"`
	Port                int      `json:"port"                gorm:"type:int;not null"`
	Username            string   `json:"username"            gorm:"type:text;not null"`
	Password            string   `json:"password"            gorm:"type:text;not null"`
	Database            *string  `json:"database"            gorm:"type:text"`
	IsHttps             bool     `json:"isHttps"             gorm:"type:boolean;default:false"`
	IsExcludeEvents     bool     `json:"isExcludeEvents"     gorm:"type:boolean;default:false"`
	IsUseExtendedInsert bool     `json:"isUseExtendedInsert" gorm:"column:is_use_extended_insert;type:boolean;not null;default:false"`
	IsSkipGaleraDisable bool     `json:"isSkipGaleraDisable" gorm:"column:is_skip_galera_disable;type:boolean;not null;default:false"`
	ExcludeTables       []string `json:"excludeTables"       gorm:"-"`
	ExcludeTablesString string   `json:"-"                   gorm:"column:exclude_tables;type:text;not null;default:''"`
	Privileges          string   `json:"privileges"          gorm:"column:privileges;type:text;not null;default:''"`
}

func (m *MariadbDatabase) TableName() string {
	return "mariadb_databases"
}

func (m *MariadbDatabase) BeforeSave(_ *gorm.DB) error {
	if len(m.ExcludeTables) > 0 {
		m.ExcludeTablesString = strings.Join(m.ExcludeTables, ",")
	} else {
		m.ExcludeTablesString = ""
	}

	return nil
}

func (m *MariadbDatabase) AfterFind(_ *gorm.DB) error {
	if m.ExcludeTablesString != "" {
		m.ExcludeTables = strings.Split(m.ExcludeTablesString, ",")
	} else {
		m.ExcludeTables = []string{}
	}

	return nil
}

func (m *MariadbDatabase) Validate() error {
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

func (m *MariadbDatabase) TestConnection(
	logger *slog.Logger,
	encryptor encryption.FieldEncryptor,
) error {
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	if m.Database == nil || *m.Database == "" {
		return errors.New("database name is required for MariaDB backup")
	}

	password, err := decryptPasswordIfNeeded(m.Password, encryptor)
	if err != nil {
		return fmt.Errorf("failed to decrypt password: %w", err)
	}

	dsn := m.buildDSN(password, *m.Database)

	db, err := sql.Open("mysql", dsn)
	if err != nil {
		return fmt.Errorf("failed to connect to MariaDB database '%s': %w", *m.Database, err)
	}
	defer func() {
		if closeErr := db.Close(); closeErr != nil {
			logger.Error("Failed to close MariaDB connection", "error", closeErr)
		}
	}()

	db.SetConnMaxLifetime(15 * time.Second)
	db.SetMaxOpenConns(1)
	db.SetMaxIdleConns(1)

	if err := db.PingContext(ctx); err != nil {
		return fmt.Errorf("failed to ping MariaDB database '%s': %w", *m.Database, err)
	}

	detectedVersion, err := detectMariadbVersion(ctx, db)
	if err != nil {
		return err
	}
	m.Version = detectedVersion

	privileges, err := detectPrivileges(ctx, db, *m.Database)
	if err != nil {
		return err
	}
	m.Privileges = privileges

	if err := checkBackupPermissions(m.Privileges); err != nil {
		return err
	}

	return nil
}

func (m *MariadbDatabase) GetRawDbSizeMb(
	ctx context.Context,
	logger *slog.Logger,
	encryptor encryption.FieldEncryptor,
) (float64, error) {
	if m.Database == nil || *m.Database == "" {
		return 0, nil
	}

	password, err := decryptPasswordIfNeeded(m.Password, encryptor)
	if err != nil {
		return 0, fmt.Errorf("failed to decrypt password: %w", err)
	}

	dsn := m.buildDSN(password, *m.Database)

	db, err := sql.Open("mysql", dsn)
	if err != nil {
		return 0, fmt.Errorf("failed to connect to MariaDB database '%s': %w", *m.Database, err)
	}
	defer func() {
		if closeErr := db.Close(); closeErr != nil {
			logger.Error("Failed to close MariaDB connection", "error", closeErr)
		}
	}()

	db.SetConnMaxLifetime(15 * time.Second)
	db.SetMaxOpenConns(1)
	db.SetMaxIdleConns(1)

	const query = `
		SELECT COALESCE(SUM(data_length + index_length), 0) / (1024 * 1024)
		FROM information_schema.tables
		WHERE table_schema = ?
	`

	var sizeMB float64
	if err := db.QueryRowContext(ctx, query, *m.Database).Scan(&sizeMB); err != nil {
		return 0, fmt.Errorf("failed to query MariaDB database size: %w", err)
	}

	return sizeMB, nil
}

func (m *MariadbDatabase) HideSensitiveData() {
	if m == nil {
		return
	}
	m.Password = ""
}

func (m *MariadbDatabase) Update(incoming *MariadbDatabase) {
	m.Version = incoming.Version
	m.Host = incoming.Host
	m.Port = incoming.Port
	m.Username = incoming.Username
	m.Database = incoming.Database
	m.IsHttps = incoming.IsHttps
	m.IsExcludeEvents = incoming.IsExcludeEvents
	m.IsUseExtendedInsert = incoming.IsUseExtendedInsert
	m.IsSkipGaleraDisable = incoming.IsSkipGaleraDisable
	m.ExcludeTables = incoming.ExcludeTables
	m.Privileges = incoming.Privileges

	if incoming.Password != "" {
		m.Password = incoming.Password
	}
}

func (m *MariadbDatabase) EncryptSensitiveFields(
	encryptor encryption.FieldEncryptor,
) error {
	if m.Password != "" {
		encrypted, err := encryptor.Encrypt(m.Password)
		if err != nil {
			return err
		}
		m.Password = encrypted
	}
	return nil
}

func (m *MariadbDatabase) PopulateDbData(
	logger *slog.Logger,
	encryptor encryption.FieldEncryptor,
) error {
	if m.Database == nil || *m.Database == "" {
		return nil
	}

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	password, err := decryptPasswordIfNeeded(m.Password, encryptor)
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

	detectedVersion, err := detectMariadbVersion(ctx, db)
	if err != nil {
		return err
	}
	m.Version = detectedVersion

	privileges, err := detectPrivileges(ctx, db, *m.Database)
	if err != nil {
		return err
	}
	m.Privileges = privileges

	return nil
}

func (m *MariadbDatabase) PopulateVersion(
	logger *slog.Logger,
	encryptor encryption.FieldEncryptor,
) error {
	if m.Database == nil || *m.Database == "" {
		return nil
	}

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	password, err := decryptPasswordIfNeeded(m.Password, encryptor)
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

	detectedVersion, err := detectMariadbVersion(ctx, db)
	if err != nil {
		return err
	}
	m.Version = detectedVersion

	return nil
}

func (m *MariadbDatabase) IsUserReadOnly(
	ctx context.Context,
	logger *slog.Logger,
	encryptor encryption.FieldEncryptor,
) (bool, []string, error) {
	password, err := decryptPasswordIfNeeded(m.Password, encryptor)
	if err != nil {
		return false, nil, fmt.Errorf("failed to decrypt password: %w", err)
	}

	dsn := m.buildDSN(password, *m.Database)

	db, err := sql.Open("mysql", dsn)
	if err != nil {
		return false, nil, fmt.Errorf("failed to connect to database: %w", err)
	}
	defer func() {
		if closeErr := db.Close(); closeErr != nil {
			logger.Error("Failed to close connection", "error", closeErr)
		}
	}()

	rows, err := db.QueryContext(ctx, "SHOW GRANTS FOR CURRENT_USER()")
	if err != nil {
		return false, nil, fmt.Errorf("failed to check grants: %w", err)
	}
	defer func() { _ = rows.Close() }()

	writePrivileges := map[string]bool{
		"INSERT":            true,
		"UPDATE":            true,
		"DELETE":            true,
		"CREATE":            true,
		"DROP":              true,
		"ALTER":             true,
		"INDEX":             true,
		"GRANT OPTION":      true,
		"ALL PRIVILEGES":    true,
		"SUPER":             true,
		"EXECUTE":           true,
		"FILE":              true,
		"RELOAD":            true,
		"SHUTDOWN":          true,
		"CREATE ROUTINE":    true,
		"ALTER ROUTINE":     true,
		"CREATE USER":       true,
		"CREATE TABLESPACE": true,
		"DELETE HISTORY":    true,
		"REFERENCES":        true,
	}

	detectedPrivileges := make(map[string]bool)

	for rows.Next() {
		var grant string
		if err := rows.Scan(&grant); err != nil {
			return false, nil, fmt.Errorf("failed to scan grant: %w", err)
		}

		for _, priv := range parseGrantPrivileges(grant) {
			if writePrivileges[priv] {
				detectedPrivileges[priv] = true
			}
		}
	}

	if err := rows.Err(); err != nil {
		return false, nil, fmt.Errorf("error iterating grants: %w", err)
	}

	privileges := make([]string, 0, len(detectedPrivileges))
	for priv := range detectedPrivileges {
		privileges = append(privileges, priv)
	}

	isReadOnly := len(privileges) == 0
	return isReadOnly, privileges, nil
}

func (m *MariadbDatabase) CreateReadOnlyUser(
	ctx context.Context,
	logger *slog.Logger,
	encryptor encryption.FieldEncryptor,
) (string, string, error) {
	password, err := decryptPasswordIfNeeded(m.Password, encryptor)
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
		// MariaDB 5.5 has a 16-character username limit, use shorter prefix
		newUsername := fmt.Sprintf("pgs-%s", uuid.New().String()[:8])
		newPassword := encryption.GenerateComplexPassword()

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
			"Read-only MariaDB user created successfully",
			"username", newUsername,
		)
		return newUsername, newPassword, nil
	}

	return "", "", errors.New("failed to generate unique username after 3 attempts")
}

func (m *MariadbDatabase) HasPrivilege(priv string) bool {
	return HasPrivilege(m.Privileges, priv)
}

func HasPrivilege(privileges, priv string) bool {
	for p := range strings.SplitSeq(privileges, ",") {
		if strings.TrimSpace(p) == priv {
			return true
		}
	}
	return false
}

func (m *MariadbDatabase) buildDSN(password, database string) string {
	tlsConfig := "false"

	if m.IsHttps {
		err := mysql.RegisterTLSConfig("mariadb-skip-verify", &tls.Config{
			InsecureSkipVerify: true,
		})
		if err != nil {
			// Config might already be registered, which is fine
			_ = err
		}
		tlsConfig = "mariadb-skip-verify"
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

// detectMariadbVersion parses VERSION() output to detect MariaDB version
// MariaDB returns strings like "10.11.6-MariaDB" or "11.4.2-MariaDB-1:11.4.2+maria~ubu2204"
// Minor versions are mapped to the closest supported version (e.g., 12.1 → 12.0)
func detectMariadbVersion(ctx context.Context, db *sql.DB) (tools.MariadbVersion, error) {
	var versionStr string
	err := db.QueryRowContext(ctx, "SELECT VERSION()").Scan(&versionStr)
	if err != nil {
		return "", fmt.Errorf("failed to query MariaDB version: %w", err)
	}

	if !strings.Contains(strings.ToLower(versionStr), "mariadb") {
		return "", fmt.Errorf(
			"not a MariaDB server (version: %s). Use MySQL database type instead",
			versionStr,
		)
	}

	re := regexp.MustCompile(`^(\d+)\.(\d+)`)
	matches := re.FindStringSubmatch(versionStr)
	if len(matches) < 3 {
		return "", fmt.Errorf("could not parse MariaDB version: %s", versionStr)
	}

	major := matches[1]
	minor := matches[2]

	return mapMariadbVersion(major, minor)
}

func mapMariadbVersion(major, minor string) (tools.MariadbVersion, error) {
	switch major {
	case "5":
		return tools.MariadbVersion55, nil
	case "10":
		return mapMariadb10xVersion(minor)
	case "11":
		return mapMariadb11xVersion(minor)
	case "12":
		return tools.MariadbVersion120, nil
	default:
		return "", fmt.Errorf(
			"unsupported MariaDB major version: %s (supported: 5.x, 10.x, 11.x, 12.x)",
			major,
		)
	}
}

func mapMariadb10xVersion(minor string) (tools.MariadbVersion, error) {
	switch minor {
	case "1":
		return tools.MariadbVersion101, nil
	case "2":
		return tools.MariadbVersion102, nil
	case "3":
		return tools.MariadbVersion103, nil
	case "4":
		return tools.MariadbVersion104, nil
	case "5":
		return tools.MariadbVersion105, nil
	case "6", "7", "8", "9", "10":
		return tools.MariadbVersion106, nil
	default:
		return tools.MariadbVersion1011, nil
	}
}

func mapMariadb11xVersion(minor string) (tools.MariadbVersion, error) {
	switch minor {
	case "0", "1", "2", "3", "4":
		return tools.MariadbVersion114, nil
	case "5", "6", "7", "8":
		return tools.MariadbVersion118, nil
	default:
		return tools.MariadbVersion118, nil
	}
}

// parseGrantPrivileges extracts comma-separated privilege names from a single
// SHOW GRANTS line. Returns uppercased privilege tokens with column-level
// qualifiers like "(col1, col2)" stripped. Returns nil for role grants and
// other lines without an ON clause.
//
// Parsing the privilege list is necessary because a naive substring search
// over the whole grant string falsely matches privilege keywords that appear
// inside other privilege names — e.g. "SHOW CREATE ROUTINE" contains both
// "CREATE" and "CREATE ROUTINE" as substrings.
func parseGrantPrivileges(grant string) []string {
	headRe := regexp.MustCompile(`(?is)^\s*GRANT\s+(.+?)\s+ON\s+`)
	m := headRe.FindStringSubmatch(grant)
	if m == nil {
		return nil
	}

	colRe := regexp.MustCompile(`\s*\([^)]*\)`)
	privsStr := colRe.ReplaceAllString(m[1], "")

	parts := strings.Split(privsStr, ",")
	privs := make([]string, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(strings.ToUpper(p))
		if p != "" {
			privs = append(privs, p)
		}
	}

	return privs
}

// detectPrivileges detects backup-related privileges and returns them as comma-separated string
func detectPrivileges(ctx context.Context, db *sql.DB, database string) (string, error) {
	rows, err := db.QueryContext(ctx, "SHOW GRANTS FOR CURRENT_USER()")
	if err != nil {
		return "", fmt.Errorf("failed to check grants: %w", err)
	}
	defer func() { _ = rows.Close() }()

	backupPrivileges := []string{
		"SELECT", "SHOW VIEW", "LOCK TABLES", "TRIGGER", "EVENT",
	}

	detectedPrivileges := make(map[string]bool)
	hasProcess := false
	hasAllPrivileges := false

	// Escape underscores to match MariaDB's grant output format
	// MariaDB escapes _ as \_ in SHOW GRANTS output
	// Pattern matches either literal _ or escaped \_
	escapedDbName := strings.ReplaceAll(regexp.QuoteMeta(database), "_", `(_|\\_)`)
	dbPatternStr := fmt.Sprintf(
		`(?i)ON\s+[\x60'"]?%s[\x60'"]?\s*\.\s*\*`,
		escapedDbName,
	)
	dbPattern := regexp.MustCompile(dbPatternStr)
	globalPattern := regexp.MustCompile(`(?i)ON\s+\*\s*\.\s*\*`)
	allPrivilegesPattern := regexp.MustCompile(`(?i)\bALL\s+PRIVILEGES\b`)

	for rows.Next() {
		var grant string
		if err := rows.Scan(&grant); err != nil {
			return "", fmt.Errorf("failed to scan grant: %w", err)
		}

		isRelevantGrant := globalPattern.MatchString(grant) || dbPattern.MatchString(grant)

		if allPrivilegesPattern.MatchString(grant) && isRelevantGrant {
			hasAllPrivileges = true
		}

		if isRelevantGrant {
			for _, priv := range backupPrivileges {
				privPattern := regexp.MustCompile(`(?i)\b` + regexp.QuoteMeta(priv) + `\b`)
				if privPattern.MatchString(grant) {
					detectedPrivileges[priv] = true
				}
			}
		}

		if globalPattern.MatchString(grant) {
			processPattern := regexp.MustCompile(`(?i)\bPROCESS\b`)
			if processPattern.MatchString(grant) {
				hasProcess = true
			}
		}
	}

	if err := rows.Err(); err != nil {
		return "", fmt.Errorf("error iterating grants: %w", err)
	}

	if hasAllPrivileges {
		for _, priv := range backupPrivileges {
			detectedPrivileges[priv] = true
		}
		hasProcess = true
	}

	privileges := make([]string, 0, len(detectedPrivileges)+1)
	for priv := range detectedPrivileges {
		privileges = append(privileges, priv)
	}
	if hasProcess {
		privileges = append(privileges, "PROCESS")
	}

	sort.Strings(privileges)
	return strings.Join(privileges, ","), nil
}

// checkBackupPermissions verifies the user has sufficient privileges for mariadb-dump backup.
// Required: SELECT, SHOW VIEW
func checkBackupPermissions(privileges string) error {
	requiredPrivileges := []string{"SELECT", "SHOW VIEW"}

	var missingPrivileges []string
	for _, priv := range requiredPrivileges {
		if !HasPrivilege(privileges, priv) {
			missingPrivileges = append(missingPrivileges, priv)
		}
	}

	if len(missingPrivileges) > 0 {
		return fmt.Errorf(
			"insufficient permissions for backup. Missing: %s. Required: SELECT, SHOW VIEW",
			strings.Join(missingPrivileges, ", "),
		)
	}

	return nil
}

func decryptPasswordIfNeeded(
	password string,
	encryptor encryption.FieldEncryptor,
) (string, error) {
	if encryptor == nil {
		return password, nil
	}
	return encryptor.Decrypt(password)
}
