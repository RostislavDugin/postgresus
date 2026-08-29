package postgresql_logical

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"regexp"
	"slices"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"gorm.io/gorm"

	postgresql_shared "databasus-backend/internal/features/databases/databases/postgresql/shared"
	"databasus-backend/internal/features/sshtunnel"
	"databasus-backend/internal/util/encryption"
	"databasus-backend/internal/util/namelist"
	"databasus-backend/internal/util/tools"
)

type PostgresqlLogicalDatabase struct {
	ID uuid.UUID `json:"id" gorm:"primaryKey;type:uuid;default:gen_random_uuid()"`

	DatabaseID *uuid.UUID `json:"databaseId" gorm:"type:uuid;column:database_id"`

	Version tools.PostgresqlVersion `json:"version" gorm:"type:text;not null"`

	Host     string  `json:"host"     gorm:"type:text;not null"`
	Port     int     `json:"port"     gorm:"type:int;not null"`
	Username string  `json:"username" gorm:"type:text;not null"`
	Password string  `json:"password" gorm:"type:text;not null"`
	Database *string `json:"database" gorm:"type:text"`

	// SSL / TLS connection settings
	SslMode       postgresql_shared.PostgresSslMode `json:"sslMode"       gorm:"column:ssl_mode;type:text;not null;default:'disable'"`
	SslClientCert string                            `json:"sslClientCert" gorm:"column:ssl_client_cert;type:text;not null;default:''"`
	SslClientKey  string                            `json:"sslClientKey"  gorm:"column:ssl_client_key;type:text;not null;default:''"`
	SslRootCert   string                            `json:"sslRootCert"   gorm:"column:ssl_root_cert;type:text;not null;default:''"`

	// When the tunnel is enabled, Host and Port above address the database as the bastion sees it.
	SshTunnel sshtunnel.Config `json:"sshTunnel" gorm:"embedded;embeddedPrefix:ssh_"`

	// Set only on the copy handed out by OpenTunnel, so CredentialSpec can point libpq at the
	// forwarded port while Host keeps the name that TLS and .pgpass are matched against.
	LocalTunnelEndpoint *sshtunnel.Endpoint `json:"-" gorm:"-"`

	// backup settings
	IncludeSchemas       []string `json:"includeSchemas"     gorm:"-"`
	IncludeSchemasString string   `json:"-"                  gorm:"column:include_schemas;type:text;not null;default:''"`
	ExcludeTables        []string `json:"excludeTables"      gorm:"-"`
	ExcludeTablesString  string   `json:"-"                  gorm:"column:exclude_tables;type:text;not null;default:''"`
	CpuCount             int      `json:"cpuCount"           gorm:"column:cpu_count;type:int;not null;default:1"`
	IsSkipUserMappings   bool     `json:"isSkipUserMappings" gorm:"column:is_skip_user_mappings;type:bool;not null;default:false"`

	// restore settings (not saved to DB)
	IsExcludeExtensions bool `json:"isExcludeExtensions" gorm:"-"`
	IsRestoreOwnership  bool `json:"isRestoreOwnership"  gorm:"-"`
	IsRestorePrivileges bool `json:"isRestorePrivileges" gorm:"-"`
}

func (p *PostgresqlLogicalDatabase) TableName() string {
	return "postgresql_logical_databases"
}

func (p *PostgresqlLogicalDatabase) BeforeSave(_ *gorm.DB) error {
	p.IncludeSchemasString = namelist.FormatUniqueNames(p.IncludeSchemas)
	p.ExcludeTablesString = namelist.FormatUniqueNames(p.ExcludeTables)

	return nil
}

func (p *PostgresqlLogicalDatabase) AfterFind(_ *gorm.DB) error {
	p.IncludeSchemas = namelist.ParseUniqueNames(p.IncludeSchemasString)
	p.ExcludeTables = namelist.ParseUniqueNames(p.ExcludeTablesString)

	return nil
}

func (p *PostgresqlLogicalDatabase) Validate() error {
	if p.SslMode == "" {
		p.SslMode = postgresql_shared.PostgresSslModeDisable
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

	if p.CpuCount <= 0 {
		return errors.New("cpu count must be greater than 0")
	}

	if err := p.validateSslConfig(); err != nil {
		return err
	}

	if err := p.SshTunnel.Validate(); err != nil {
		return err
	}

	// Prevent Databasus from backing up itself
	// Databasus runs an internal PostgreSQL instance that should not be backed up through the UI
	// because it would expose internal metadata to non-system administrators.
	// To properly backup Databasus, see: https://databasus.com/faq#backup-databasus
	// Only a remote bastion relaxes it: there a loopback address names a database on the bastion,
	// whereas a bastion on this machine would forward straight back to the instance being guarded.
	if !p.isReachedThroughARemoteBastion() &&
		p.Database != nil &&
		isLocalhostAddress(p.Host) &&
		strings.EqualFold(*p.Database, "databasus") {
		return errors.New(
			"backing up Databasus internal database is not allowed. To backup Databasus itself, see https://databasus.com/faq#backup-databasus",
		)
	}

	return nil
}

func (p *PostgresqlLogicalDatabase) TestConnection(
	logger *slog.Logger,
	encryptor encryption.FieldEncryptor,
) error {
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	return testSingleDatabaseConnection(logger, ctx, p, encryptor)
}

// The value feeds verification disk planning and the restored-size sanity check,
// so it must cover exactly the relations pg_dump emits for this database.
func (p *PostgresqlLogicalDatabase) GetRawDbSizeMb(
	ctx context.Context,
	logger *slog.Logger,
	encryptor encryption.FieldEncryptor,
) (float64, error) {
	if p.Database == nil || *p.Database == "" {
		return 0, nil
	}

	conn, err := openPgConn(ctx, p, *p.Database, encryptor)
	if err != nil {
		return 0, fmt.Errorf("failed to connect to database '%s': %w", *p.Database, err)
	}
	defer func() {
		if closeErr := conn.Close(ctx); closeErr != nil {
			logger.ErrorContext(ctx, "failed to close connection", "error", closeErr)
		}
	}()

	sizeBytes, err := getDumpedRelationsSizeBytes(ctx, conn, DumpFilter{
		IncludeSchemas:       p.IncludeSchemas,
		ExcludeTablePatterns: p.ExcludeTables,
	})
	if err != nil {
		return 0, err
	}

	return float64(sizeBytes) / (1024 * 1024), nil
}

func (p *PostgresqlLogicalDatabase) HideSensitiveData() {
	if p == nil {
		return
	}

	p.Password = ""
	p.SslClientKey = ""
	p.SshTunnel.HideSensitiveData()
}

func (p *PostgresqlLogicalDatabase) ValidateUpdate(_ *PostgresqlLogicalDatabase) error {
	return nil
}

func (p *PostgresqlLogicalDatabase) Update(incoming *PostgresqlLogicalDatabase) {
	p.Version = incoming.Version
	p.Host = incoming.Host
	p.Port = incoming.Port
	p.Username = incoming.Username
	p.Database = incoming.Database
	p.SslMode = incoming.SslMode
	p.SslClientCert = incoming.SslClientCert
	p.SslRootCert = incoming.SslRootCert
	p.IncludeSchemas = incoming.IncludeSchemas
	p.ExcludeTables = incoming.ExcludeTables
	p.CpuCount = incoming.CpuCount
	p.IsSkipUserMappings = incoming.IsSkipUserMappings
	p.SshTunnel.Update(&incoming.SshTunnel)

	if incoming.Password != "" {
		p.Password = incoming.Password
	}

	if incoming.SslClientKey != "" {
		p.SslClientKey = incoming.SslClientKey
	}
}

// LocalTunnelEndpoint belongs to the operation that opened the tunnel, not to the configuration.
func (p *PostgresqlLogicalDatabase) CopyForNewDatabase() *PostgresqlLogicalDatabase {
	if p == nil {
		return nil
	}

	copiedDatabase := *p
	copiedDatabase.ID = uuid.Nil
	copiedDatabase.DatabaseID = nil
	copiedDatabase.IncludeSchemas = slices.Clone(p.IncludeSchemas)
	copiedDatabase.ExcludeTables = slices.Clone(p.ExcludeTables)
	copiedDatabase.LocalTunnelEndpoint = nil

	if p.Database != nil {
		copiedDatabase.Database = new(*p.Database)
	}

	return &copiedDatabase
}

func (p *PostgresqlLogicalDatabase) EncryptSensitiveFields(
	encryptor encryption.FieldEncryptor,
) error {
	for _, field := range []*string{
		&p.Password,
		&p.SslClientCert,
		&p.SslClientKey,
		&p.SslRootCert,
	} {
		if *field == "" {
			continue
		}

		encrypted, err := encryptor.Encrypt(*field)
		if err != nil {
			return err
		}

		*field = encrypted
	}

	return p.SshTunnel.EncryptSensitiveFields(encryptor)
}

// PopulateDbData detects and sets the PostgreSQL version.
// This should be called before encrypting sensitive fields.
func (p *PostgresqlLogicalDatabase) PopulateDbData(
	logger *slog.Logger,
	encryptor encryption.FieldEncryptor,
) error {
	return p.PopulateVersion(logger, encryptor)
}

// PopulateVersion detects and sets the PostgreSQL version by querying the database.
func (p *PostgresqlLogicalDatabase) PopulateVersion(
	logger *slog.Logger,
	encryptor encryption.FieldEncryptor,
) error {
	if p.Database == nil || *p.Database == "" {
		return nil
	}

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	conn, err := openPgConn(ctx, p, *p.Database, encryptor)
	if err != nil {
		return fmt.Errorf("failed to connect to database: %w", err)
	}
	defer func() {
		if closeErr := conn.Close(ctx); closeErr != nil {
			logger.Error("failed to close connection", "error", closeErr)
		}
	}()

	detectedVersion, err := detectDatabaseVersion(ctx, conn)
	if err != nil {
		return err
	}

	p.Version = detectedVersion
	return nil
}

func (p *PostgresqlLogicalDatabase) ShouldSuggestReadOnlyUser(
	ctx context.Context,
	logger *slog.Logger,
	encryptor encryption.FieldEncryptor,
) (bool, []string, error) {
	conn, err := openPgConn(ctx, p, *p.Database, encryptor)
	if err != nil {
		return false, nil, fmt.Errorf("failed to connect to database: %w", err)
	}
	defer func() {
		if closeErr := conn.Close(ctx); closeErr != nil {
			logger.ErrorContext(ctx, "failed to close connection", "error", closeErr)
		}
	}()

	var privileges []string

	// LEVEL 1: Check role-level attributes
	var isSuperuser, canCreateRole, canCreateDB, canBypassRLS, canReplication bool
	err = conn.QueryRow(ctx, `
		SELECT
			rolsuper,
			rolcreaterole,
			rolcreatedb,
			rolbypassrls,
			rolreplication
		FROM pg_roles
		WHERE rolname = current_user
	`).Scan(&isSuperuser, &canCreateRole, &canCreateDB, &canBypassRLS, &canReplication)
	if err != nil {
		return false, nil, fmt.Errorf("failed to check role attributes: %w", err)
	}

	if isSuperuser {
		privileges = append(privileges, "SUPERUSER")
	}
	if canCreateRole {
		privileges = append(privileges, "CREATEROLE")
	}
	if canCreateDB {
		privileges = append(privileges, "CREATEDB")
	}
	if canBypassRLS {
		privileges = append(privileges, "BYPASSRLS")
	}
	if canReplication {
		privileges = append(privileges, "REPLICATION")
	}

	// LEVEL 2: Check database-level privileges
	var canCreate, canTemp bool
	err = conn.QueryRow(ctx, `
		SELECT
			has_database_privilege(current_user, current_database(), 'CREATE') as can_create,
			has_database_privilege(current_user, current_database(), 'TEMP') as can_temp
	`).Scan(&canCreate, &canTemp)
	if err != nil {
		return false, nil, fmt.Errorf("failed to check database privileges: %w", err)
	}

	if canCreate {
		privileges = append(privileges, "CREATE (database)")
	}
	if canTemp {
		privileges = append(privileges, "TEMP")
	}

	// LEVEL 2.5: Check schema-level CREATE privileges
	var hasSchemaCreate bool
	err = conn.QueryRow(ctx, `
		SELECT EXISTS(
			SELECT 1
			FROM pg_namespace n
			WHERE has_schema_privilege(current_user, n.nspname, 'CREATE')
			AND nspname NOT IN ('pg_catalog', 'information_schema', 'pg_toast')
		)
	`).Scan(&hasSchemaCreate)
	if err != nil {
		return false, nil, fmt.Errorf("failed to check schema privileges: %w", err)
	}
	if hasSchemaCreate {
		privileges = append(privileges, "CREATE (schema)")
	}

	// LEVEL 3: Check table-level write permissions
	writePrivileges := map[string]bool{
		"INSERT":     true,
		"UPDATE":     true,
		"DELETE":     true,
		"TRUNCATE":   true,
		"REFERENCES": true,
		"TRIGGER":    true,
	}

	var tablePrivileges []string
	rows, err := conn.Query(ctx, `
		SELECT DISTINCT privilege_type
		FROM information_schema.role_table_grants
		WHERE grantee = current_user
		AND table_schema NOT IN ('pg_catalog', 'information_schema')
	`)
	if err != nil {
		return false, nil, fmt.Errorf("failed to check table privileges: %w", err)
	}

	for rows.Next() {
		var privilege string
		if err := rows.Scan(&privilege); err != nil {
			rows.Close()
			return false, nil, fmt.Errorf("failed to scan privilege: %w", err)
		}
		tablePrivileges = append(tablePrivileges, privilege)
	}
	rows.Close()

	if err := rows.Err(); err != nil {
		return false, nil, fmt.Errorf("error iterating privileges: %w", err)
	}

	for _, privilege := range tablePrivileges {
		if writePrivileges[privilege] {
			privileges = append(privileges, privilege)
		}
	}

	// LEVEL 4: Check for EXECUTE privilege on functions that are SECURITY DEFINER
	var funcCount int
	err = conn.QueryRow(ctx, `
		SELECT COUNT(*)
		FROM pg_proc p
		JOIN pg_namespace n ON p.pronamespace = n.oid
		WHERE n.nspname NOT IN ('pg_catalog', 'information_schema')
		AND p.prosecdef = true
		AND has_function_privilege(current_user, p.oid, 'EXECUTE')
	`).Scan(&funcCount)
	if err != nil {
		return false, nil, fmt.Errorf("failed to check function privileges: %w", err)
	}
	if funcCount > 0 {
		privileges = append(privileges, "EXECUTE (SECURITY DEFINER)")
	}

	rowLevelSecurityTables, err := getRowLevelSecurityTables(ctx, conn, p.IncludeSchemas)
	if err != nil {
		return false, nil, err
	}

	shouldSuggestReadOnlyUser := len(privileges) > 0 && len(rowLevelSecurityTables) == 0

	return shouldSuggestReadOnlyUser, privileges, nil
}

func (p *PostgresqlLogicalDatabase) CreateReadOnlyUser(
	ctx context.Context,
	logger *slog.Logger,
	encryptor encryption.FieldEncryptor,
) (string, string, error) {
	conn, err := openPgConn(ctx, p, *p.Database, encryptor)
	if err != nil {
		return "", "", fmt.Errorf("failed to connect to database: %w", err)
	}
	defer func() {
		if closeErr := conn.Close(ctx); closeErr != nil {
			logger.ErrorContext(ctx, "failed to close connection", "error", closeErr)
		}
	}()

	// Pre-validation runs on the bare connection so a refusal costs no transaction and leaks no
	// generated credentials.
	var canCreateRole, isSuperuser bool
	err = conn.QueryRow(ctx, `
		SELECT rolcreaterole, rolsuper
		FROM pg_roles
		WHERE rolname = current_user
	`).Scan(&canCreateRole, &isSuperuser)
	if err != nil {
		return "", "", fmt.Errorf("failed to check permissions: %w", err)
	}
	if !canCreateRole && !isSuperuser {
		return "", "", errors.New("current database user lacks CREATEROLE privilege")
	}

	rowLevelSecurityTables, err := getRowLevelSecurityTables(ctx, conn, p.IncludeSchemas)
	if err != nil {
		return "", "", err
	}

	if len(rowLevelSecurityTables) > 0 {
		return "", "", fmt.Errorf(
			"row-level security is enabled on %s; grant BYPASSRLS to a role you create yourself, or back up with a role that owns these tables",
			namelist.FormatTruncatedNames(rowLevelSecurityTables),
		)
	}

	// Retry logic for username collision
	maxRetries := 3
	for attempt := range maxRetries {
		// Generate base username for PostgreSQL user creation
		baseUsername := fmt.Sprintf("databasus-%s", uuid.New().String()[:8])

		// For Supabase session pooler, the username format for connection is "username.projectid"
		// but the actual PostgreSQL user must be created with just the base name.
		// The pooler will strip the ".projectid" suffix when authenticating.
		connectionUsername := baseUsername
		if isSupabaseConnection(p.Host, p.Username) {
			if supabaseProjectID := extractSupabaseProjectID(p.Username); supabaseProjectID != "" {
				connectionUsername = fmt.Sprintf("%s.%s", baseUsername, supabaseProjectID)
			}
		}

		newPassword := encryption.GenerateComplexPassword()

		tx, err := conn.Begin(ctx)
		if err != nil {
			return "", "", fmt.Errorf("failed to begin transaction: %w", err)
		}

		success := false
		defer func() {
			if !success {
				if rollbackErr := tx.Rollback(ctx); rollbackErr != nil {
					logger.ErrorContext(ctx, "failed to rollback transaction", "error", rollbackErr)
				}
			}
		}()

		// Step 1: Create PostgreSQL user with LOGIN privilege
		// Note: We use baseUsername for the actual PostgreSQL user name if Supabase is used
		_, err = tx.Exec(
			ctx,
			fmt.Sprintf(`CREATE USER "%s" WITH PASSWORD '%s' LOGIN`, baseUsername, newPassword),
		)
		if err != nil {
			if err.Error() != "" && attempt < maxRetries-1 {
				continue
			}
			return "", "", fmt.Errorf("failed to create user: %w", err)
		}

		// Step 2: Check if public schema exists and revoke CREATE privilege if it does
		// This is necessary because all PostgreSQL users inherit CREATE privilege on the
		// public schema through the PUBLIC role. This is a one-time operation that affects
		// the entire database, making it more secure by default.
		// Note: This only affects the public schema; other schemas are unaffected.
		var publicSchemaExists bool
		err = tx.QueryRow(ctx, `
			SELECT EXISTS(
				SELECT 1 FROM information_schema.schemata 
				WHERE schema_name = 'public'
			)
		`).Scan(&publicSchemaExists)
		if err != nil {
			return "", "", fmt.Errorf("failed to check if public schema exists: %w", err)
		}

		if publicSchemaExists {
			// Revoke CREATE from PUBLIC role (affects all users)
			_, err = tx.Exec(ctx, `REVOKE CREATE ON SCHEMA public FROM PUBLIC`)
			if err != nil {
				if strings.Contains(err.Error(), "permission denied") {
					logger.WarnContext(
						ctx,
						"failed to revoke CREATE on public from PUBLIC (permission denied)",
						"error",
						err,
					)
				} else {
					return "", "", fmt.Errorf("failed to revoke CREATE from PUBLIC on existing public schema: %w", err)
				}
			}

			// Now revoke from the specific user as well (belt and suspenders)
			_, err = tx.Exec(
				ctx,
				fmt.Sprintf(`REVOKE CREATE ON SCHEMA public FROM "%s"`, baseUsername),
			)
			if err != nil {
				logger.WarnContext(
					ctx,
					"failed to revoke CREATE on public schema from user",
					"error",
					err,
					"username",
					baseUsername,
				)
			}
		} else {
			logger.InfoContext(ctx, "public schema does not exist, skipping CREATE privilege revocation")
		}

		// Step 3: Grant database connection privilege and revoke TEMP
		_, err = tx.Exec(
			ctx,
			fmt.Sprintf(`GRANT CONNECT ON DATABASE "%s" TO "%s"`, *p.Database, baseUsername),
		)
		if err != nil {
			return "", "", fmt.Errorf("failed to grant connect privilege: %w", err)
		}

		// Revoke TEMP privilege from PUBLIC role (like CREATE on public schema, TEMP is granted to PUBLIC by default)
		_, err = tx.Exec(ctx, fmt.Sprintf(`REVOKE TEMP ON DATABASE "%s" FROM PUBLIC`, *p.Database))
		if err != nil {
			logger.WarnContext(ctx, "failed to revoke TEMP from PUBLIC", "error", err)
		}

		// Also revoke from the specific user (belt and suspenders)
		_, err = tx.Exec(
			ctx,
			fmt.Sprintf(`REVOKE TEMP ON DATABASE "%s" FROM "%s"`, *p.Database, baseUsername),
		)
		if err != nil {
			logger.WarnContext(ctx, "failed to revoke TEMP privilege", "error", err, "username", baseUsername)
		}

		// Step 4: Discover schemas to grant privileges on
		// If IncludeSchemas is specified, only use those schemas; otherwise use all non-system schemas
		var rows pgx.Rows
		if len(p.IncludeSchemas) > 0 {
			rows, err = tx.Query(ctx, `
				SELECT schema_name
				FROM information_schema.schemata
				WHERE schema_name NOT IN ('pg_catalog', 'information_schema')
				AND schema_name = ANY($1::text[])
			`, p.IncludeSchemas)
		} else {
			rows, err = tx.Query(ctx, `
				SELECT schema_name
				FROM information_schema.schemata
				WHERE schema_name NOT IN ('pg_catalog', 'information_schema')
			`)
		}
		if err != nil {
			return "", "", fmt.Errorf("failed to get schemas: %w", err)
		}

		var schemas []string
		for rows.Next() {
			var schema string
			if err := rows.Scan(&schema); err != nil {
				rows.Close()
				return "", "", fmt.Errorf("failed to scan schema: %w", err)
			}
			schemas = append(schemas, schema)
		}
		rows.Close()

		if err := rows.Err(); err != nil {
			return "", "", fmt.Errorf("error iterating schemas: %w", err)
		}

		// Step 5: Grant USAGE on each schema and explicitly prevent CREATE
		for _, schema := range schemas {
			// Revoke CREATE specifically (handles inheritance from PUBLIC role)
			_, err = tx.Exec(
				ctx,
				fmt.Sprintf(`REVOKE CREATE ON SCHEMA "%s" FROM "%s"`, schema, baseUsername),
			)
			if err != nil {
				logger.WarnContext(
					ctx,
					"failed to revoke CREATE on schema",
					"error",
					err,
					"schema",
					schema,
					"username",
					baseUsername,
				)
			}

			// Grant only USAGE (not CREATE)
			_, err = tx.Exec(
				ctx,
				fmt.Sprintf(`GRANT USAGE ON SCHEMA "%s" TO "%s"`, schema, baseUsername),
			)
			if err != nil {
				return "", "", fmt.Errorf("failed to grant usage on schema %s: %w", schema, err)
			}
		}

		// Step 6: Grant SELECT on ALL existing tables and sequences
		// Use the already-filtered schemas list from Step 4
		for _, schema := range schemas {
			_, err = tx.Exec(
				ctx,
				fmt.Sprintf(
					`GRANT SELECT ON ALL TABLES IN SCHEMA "%s" TO "%s"`,
					schema,
					baseUsername,
				),
			)
			if err != nil {
				return "", "", fmt.Errorf(
					"failed to grant select on tables in schema %s: %w",
					schema,
					err,
				)
			}

			_, err = tx.Exec(
				ctx,
				fmt.Sprintf(
					`GRANT SELECT ON ALL SEQUENCES IN SCHEMA "%s" TO "%s"`,
					schema,
					baseUsername,
				),
			)
			if err != nil {
				return "", "", fmt.Errorf(
					"failed to grant select on sequences in schema %s: %w",
					schema,
					err,
				)
			}
		}

		// Step 7: Set default privileges for FUTURE tables and sequences
		// First, set default privileges for objects created by the current user
		// Use the already-filtered schemas list from Step 4
		for _, schema := range schemas {
			_, err = tx.Exec(
				ctx,
				fmt.Sprintf(
					`ALTER DEFAULT PRIVILEGES IN SCHEMA "%s" GRANT SELECT ON TABLES TO "%s"`,
					schema,
					baseUsername,
				),
			)
			if err != nil {
				return "", "", fmt.Errorf(
					"failed to set default privileges for tables in schema %s: %w",
					schema,
					err,
				)
			}

			_, err = tx.Exec(
				ctx,
				fmt.Sprintf(
					`ALTER DEFAULT PRIVILEGES IN SCHEMA "%s" GRANT SELECT ON SEQUENCES TO "%s"`,
					schema,
					baseUsername,
				),
			)
			if err != nil {
				return "", "", fmt.Errorf(
					"failed to set default privileges for sequences in schema %s: %w",
					schema,
					err,
				)
			}
		}

		// Step 8: Discover all roles that own objects in each schema
		// This is needed because ALTER DEFAULT PRIVILEGES only applies to objects created by the current role.
		// To handle tables created by OTHER users (like the GitHub issue with partitioned tables),
		// we need to set "ALTER DEFAULT PRIVILEGES FOR ROLE <owner>" for each object owner.
		// Filter by IncludeSchemas if specified.
		type SchemaOwner struct {
			SchemaName string
			RoleName   string
		}

		var ownerRows pgx.Rows
		if len(p.IncludeSchemas) > 0 {
			ownerRows, err = tx.Query(ctx, `
				SELECT DISTINCT n.nspname as schema_name, pg_get_userbyid(c.relowner) as role_name
				FROM pg_class c
				JOIN pg_namespace n ON c.relnamespace = n.oid
				WHERE n.nspname NOT IN ('pg_catalog', 'information_schema', 'pg_toast')
				  AND n.nspname = ANY($1::text[])
				  AND c.relkind IN ('r', 'p', 'v', 'm', 'f')
				  AND pg_get_userbyid(c.relowner) != current_user
				ORDER BY n.nspname, role_name
			`, p.IncludeSchemas)
		} else {
			ownerRows, err = tx.Query(ctx, `
				SELECT DISTINCT n.nspname as schema_name, pg_get_userbyid(c.relowner) as role_name
				FROM pg_class c
				JOIN pg_namespace n ON c.relnamespace = n.oid
				WHERE n.nspname NOT IN ('pg_catalog', 'information_schema', 'pg_toast')
				  AND c.relkind IN ('r', 'p', 'v', 'm', 'f')
				  AND pg_get_userbyid(c.relowner) != current_user
				ORDER BY n.nspname, role_name
			`)
		}

		if err != nil {
			// Log warning but continue - this is a best-effort enhancement
			logger.WarnContext(ctx, "failed to query object owners for default privileges", "error", err)
		} else {
			var schemaOwners []SchemaOwner
			for ownerRows.Next() {
				var so SchemaOwner
				if err := ownerRows.Scan(&so.SchemaName, &so.RoleName); err != nil {
					ownerRows.Close()
					logger.WarnContext(ctx, "failed to scan schema owner", "error", err)
					break
				}
				schemaOwners = append(schemaOwners, so)
			}
			ownerRows.Close()

			if err := ownerRows.Err(); err != nil {
				logger.WarnContext(ctx, "error iterating schema owners", "error", err)
			}

			// Step 9: Set default privileges FOR ROLE for each object owner
			// Note: This may fail for some roles due to permission issues (e.g., roles owned by other superusers)
			// We log warnings but continue - user creation should succeed even if some roles can't be configured
			for _, so := range schemaOwners {
				// Try to set default privileges for tables
				_, err = tx.Exec(
					ctx,
					fmt.Sprintf(
						`ALTER DEFAULT PRIVILEGES FOR ROLE "%s" IN SCHEMA "%s" GRANT SELECT ON TABLES TO "%s"`,
						so.RoleName,
						so.SchemaName,
						baseUsername,
					),
				)
				if err != nil {
					logger.WarnContext(
						ctx,
						"failed to set default privileges for role (tables)",
						"error",
						err,
						"role",
						so.RoleName,
						"schema",
						so.SchemaName,
						"readonly_user",
						baseUsername,
					)
				}

				// Try to set default privileges for sequences
				_, err = tx.Exec(
					ctx,
					fmt.Sprintf(
						`ALTER DEFAULT PRIVILEGES FOR ROLE "%s" IN SCHEMA "%s" GRANT SELECT ON SEQUENCES TO "%s"`,
						so.RoleName,
						so.SchemaName,
						baseUsername,
					),
				)
				if err != nil {
					logger.WarnContext(
						ctx,
						"failed to set default privileges for role (sequences)",
						"error",
						err,
						"role",
						so.RoleName,
						"schema",
						so.SchemaName,
						"readonly_user",
						baseUsername,
					)
				}
			}

			if len(schemaOwners) > 0 {
				logger.InfoContext(
					ctx,
					"set default privileges for existing object owners",
					"readonly_user",
					baseUsername,
					"owner_count",
					len(schemaOwners),
				)
			}
		}

		// Step 10: Verify user creation before committing
		var verifyUsername string
		err = tx.QueryRow(ctx, fmt.Sprintf(`SELECT rolname FROM pg_roles WHERE rolname = '%s'`, baseUsername)).
			Scan(&verifyUsername)
		if err != nil {
			return "", "", fmt.Errorf("failed to verify user creation: %w", err)
		}

		if err := tx.Commit(ctx); err != nil {
			return "", "", fmt.Errorf("failed to commit transaction: %w", err)
		}

		success = true
		// Return connectionUsername (with project ID suffix for Supabase) for the caller to use when connecting
		logger.InfoContext(
			ctx,
			"read-only user created successfully",
			"username",
			baseUsername,
			"connectionUsername",
			connectionUsername,
		)
		return connectionUsername, newPassword, nil
	}

	return "", "", errors.New("failed to generate unique username after 3 attempts")
}

func (p *PostgresqlLogicalDatabase) validateSslConfig() error {
	return postgresql_shared.ValidateSslConfig(
		p.SslMode,
		p.SslClientCert,
		p.SslClientKey,
		p.SslRootCert,
	)
}

func testSingleDatabaseConnection(
	logger *slog.Logger,
	ctx context.Context,
	postgresDb *PostgresqlLogicalDatabase,
	encryptor encryption.FieldEncryptor,
) error {
	if postgresDb.Database == nil || *postgresDb.Database == "" {
		return errors.New("database name is required for single database backup (pg_dump)")
	}

	conn, err := openPgConn(ctx, postgresDb, *postgresDb.Database, encryptor)
	if err != nil {
		// TODO make more readable errors:
		// - handle wrong creds
		// - handle wrong database name
		// - handle wrong protocol
		return fmt.Errorf("failed to connect to database '%s': %w", *postgresDb.Database, err)
	}
	defer func() {
		if closeErr := conn.Close(ctx); closeErr != nil {
			logger.ErrorContext(ctx, "failed to close connection", "error", closeErr)
		}
	}()

	detectedVersion, err := detectDatabaseVersion(ctx, conn)
	if err != nil {
		return err
	}
	postgresDb.Version = detectedVersion

	if err := checkDumpReadPrivileges(ctx, conn, DumpFilter{
		IncludeSchemas:       postgresDb.IncludeSchemas,
		ExcludeTablePatterns: postgresDb.ExcludeTables,
	}); err != nil {
		return err
	}

	if !postgresDb.IsSkipUserMappings {
		if err := checkUserMappingsReadable(ctx, conn); err != nil {
			return err
		}
	}

	return nil
}

func detectDatabaseVersion(ctx context.Context, conn *pgx.Conn) (tools.PostgresqlVersion, error) {
	var versionStr string
	err := conn.QueryRow(ctx, "SELECT version()").Scan(&versionStr)
	if err != nil {
		return "", fmt.Errorf("failed to query database version: %w", err)
	}

	// Parse version from string like "PostgreSQL 14.2 on x86_64-pc-linux-gnu..."
	// or "PostgreSQL 16 maintained by Postgre BY..." (some builds omit minor version)
	re := regexp.MustCompile(`PostgreSQL (\d+)`)
	matches := re.FindStringSubmatch(versionStr)
	if len(matches) < 2 {
		return "", fmt.Errorf("could not parse version from: %s", versionStr)
	}

	majorVersion := matches[1]

	// Map to known PostgresqlVersion enum values
	switch majorVersion {
	case "12", "13", "14", "15", "16", "17", "18":
		return tools.PostgresqlVersion(majorVersion), nil
	default:
		return "", fmt.Errorf("unsupported PostgreSQL version: %s", majorVersion)
	}
}

func isSupabaseConnection(host, username string) bool {
	return strings.Contains(strings.ToLower(host), "supabase") ||
		strings.Contains(strings.ToLower(username), "supabase")
}

func extractSupabaseProjectID(username string) string {
	if _, after, found := strings.Cut(username, "."); found {
		return after
	}
	return ""
}

func (p *PostgresqlLogicalDatabase) isReachedThroughARemoteBastion() bool {
	return p.SshTunnel.IsEnabled && !isLocalhostAddress(p.SshTunnel.Host)
}

func isLocalhostAddress(host string) bool {
	localhostHosts := []string{
		"localhost",
		"127.0.0.1",
		"172.17.0.1",
		"host.docker.internal",
		"::1",     // IPv6 loopback (equivalent to 127.0.0.1)
		"::",      // IPv6 all interfaces (equivalent to 0.0.0.0)
		"0.0.0.0", // IPv4 all interfaces
	}

	for _, localhostHost := range localhostHosts {
		if strings.EqualFold(host, localhostHost) {
			return true
		}
	}

	// The entire 127.0.0.0/8 loopback range, not just 127.0.0.1
	return strings.HasPrefix(host, "127.")
}
