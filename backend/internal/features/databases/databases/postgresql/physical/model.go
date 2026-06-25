package postgresql_physical

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"gorm.io/gorm"

	postgresql_shared "databasus-backend/internal/features/databases/databases/postgresql/shared"
	"databasus-backend/internal/util/encryption"
	"databasus-backend/internal/util/tools"
)

type PostgresqlPhysicalDatabase struct {
	ID uuid.UUID `json:"id" gorm:"primaryKey;type:uuid;default:gen_random_uuid()"`

	DatabaseID *uuid.UUID `json:"databaseId" gorm:"type:uuid;column:database_id"`

	Version    tools.PostgresqlVersion `json:"version"    gorm:"type:text;not null"`
	BackupType BackupType              `json:"backupType" gorm:"column:backup_type;type:text;not null;default:'FULL'"`

	Host     string `json:"host"     gorm:"type:text;not null"`
	Port     int    `json:"port"     gorm:"type:int;not null"`
	Username string `json:"username" gorm:"type:text;not null"`
	Password string `json:"password" gorm:"type:text;not null"`

	// SSL / TLS connection settings
	SslMode       postgresql_shared.PostgresSslMode `json:"sslMode"       gorm:"column:ssl_mode;type:text;not null;default:'disable'"`
	SslClientCert string                            `json:"sslClientCert" gorm:"column:ssl_client_cert;type:text;not null;default:''"`
	SslClientKey  string                            `json:"sslClientKey"  gorm:"column:ssl_client_key;type:text;not null;default:''"`
	SslRootCert   string                            `json:"sslRootCert"   gorm:"column:ssl_root_cert;type:text;not null;default:''"`

	ReplicationSlotName string  `json:"-"                gorm:"column:replication_slot_name;type:text;not null"`
	SystemIdentifier    *string `json:"systemIdentifier" gorm:"column:system_identifier;type:text"`

	// WalSegmentSizeBytes captures the source cluster's wal_segment_size at first connect.
	WalSegmentSizeBytes *int64 `json:"walSegmentSizeBytes" gorm:"column:wal_segment_size_bytes;type:bigint"`
}

func (p *PostgresqlPhysicalDatabase) TableName() string {
	return "postgresql_physical_databases"
}

func (p *PostgresqlPhysicalDatabase) Validate() error {
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

	if err := postgresql_shared.ValidateSslConfig(
		p.SslMode,
		p.SslClientCert,
		p.SslClientKey,
		p.SslRootCert,
	); err != nil {
		return err
	}

	switch p.BackupType {
	case BackupTypeFullOnly, BackupTypeFullAndIncremental, BackupTypeFullIncrementalAndWalStream:
	case "":
		p.BackupType = BackupTypeFullOnly
	default:
		return fmt.Errorf("invalid backup type: %q", p.BackupType)
	}

	return nil
}

func (p *PostgresqlPhysicalDatabase) ValidateUpdate(old *PostgresqlPhysicalDatabase) error {
	if old == nil {
		return nil
	}

	if old.SystemIdentifier != nil && p.SystemIdentifier != nil &&
		*old.SystemIdentifier != *p.SystemIdentifier {
		return errors.New(
			"system_identifier is immutable; cluster swap refused",
		)
	}

	if old.WalSegmentSizeBytes != nil && p.WalSegmentSizeBytes != nil &&
		*old.WalSegmentSizeBytes != *p.WalSegmentSizeBytes {
		return errors.New(
			"wal_segment_size_bytes is immutable; cluster swap refused",
		)
	}

	return nil
}

func (p *PostgresqlPhysicalDatabase) BeforeCreate(_ *gorm.DB) error {
	if p.ReplicationSlotName == "" {
		p.ReplicationSlotName = "databasus_slot_" + strings.ReplaceAll(uuid.New().String(), "-", "")
	}

	return nil
}

func (p *PostgresqlPhysicalDatabase) Update(incoming *PostgresqlPhysicalDatabase) {
	p.Host = incoming.Host
	p.Port = incoming.Port
	p.Username = incoming.Username
	p.SslMode = incoming.SslMode
	p.SslClientCert = incoming.SslClientCert
	p.SslRootCert = incoming.SslRootCert
	p.BackupType = incoming.BackupType

	if incoming.Password != "" {
		p.Password = incoming.Password
	}

	if incoming.SslClientKey != "" {
		p.SslClientKey = incoming.SslClientKey
	}

	// SystemIdentifier and ReplicationSlotName are server-managed; never overwritten from UI input.
}

func (p *PostgresqlPhysicalDatabase) HideSensitiveData() {
	if p == nil {
		return
	}

	p.Password = ""
	p.SslClientKey = ""
}

func (p *PostgresqlPhysicalDatabase) EncryptSensitiveFields(
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

	return nil
}

func (p *PostgresqlPhysicalDatabase) PopulateDbData(
	logger *slog.Logger,
	encryptor encryption.FieldEncryptor,
) error {
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	conn, err := openConn(ctx, p, encryptor)
	if err != nil {
		return fmt.Errorf("failed to connect to cluster: %w", err)
	}
	defer closeConnQuietly(ctx, conn, logger)

	detectedVersion, err := detectVersion(ctx, conn)
	if err != nil {
		return err
	}
	p.Version = detectedVersion

	if p.SystemIdentifier == nil {
		var sysID string

		if err := conn.QueryRow(ctx, "SELECT system_identifier::text FROM pg_control_system()").
			Scan(&sysID); err != nil {
			return fmt.Errorf("failed to read system_identifier: %w", err)
		}

		p.SystemIdentifier = &sysID
	}

	if p.WalSegmentSizeBytes == nil {
		var sizeBytes int64

		if err := conn.QueryRow(ctx,
			"SELECT setting::bigint FROM pg_settings WHERE name = 'wal_segment_size'",
		).Scan(&sizeBytes); err != nil {
			return fmt.Errorf("failed to read wal_segment_size: %w", err)
		}

		p.WalSegmentSizeBytes = &sizeBytes
	}

	return nil
}

// ParentDatabaseID returns the parent databases.id this physical record hangs
// off — the target of every physical catalog FK (fk_physical_wal_*_database_id,
// fk_physical_full_backups_database_id, …), never the
// postgresql_physical_databases PK. The fallback to ID covers an unassociated
// in-memory row whose DatabaseID was not loaded, mirroring
// receivewalApplicationName's fallback.
func (p *PostgresqlPhysicalDatabase) ParentDatabaseID() uuid.UUID {
	if p.DatabaseID != nil {
		return *p.DatabaseID
	}

	return p.ID
}

func (p *PostgresqlPhysicalDatabase) SystemIdentifierUint64() uint64 {
	if p.SystemIdentifier == nil {
		return 0
	}

	if value, err := strconv.ParseUint(*p.SystemIdentifier, 10, 64); err == nil {
		return value
	}

	if value, err := strconv.ParseInt(*p.SystemIdentifier, 10, 64); err == nil {
		return uint64(value)
	}

	return 0
}

// Rationale for the summarize_wal=on gate: see
// adr/0008-why-pg17-native-backups-with-mandatory-wal-summary.md.
// Rationale for the custom-tablespace refusal: see
// adr/0010-no-support-for-customer-tablespaces.md.
func (p *PostgresqlPhysicalDatabase) TestReplicationConnection(
	logger *slog.Logger,
	encryptor encryption.FieldEncryptor,
) error {
	return p.checkReplicationReadiness(logger, encryptor)
}

// VerifyWalSlot ensures a physical replication slot named p.ReplicationSlotName
// exists on the target cluster. Idempotent: a second call after first create
// is a no-op. Caller is the backup runner — slot existence is a backup-time
// precondition, intentionally NOT called from TestReplicationConnection
// (which only probes server settings, not state).
//
// Refuses to overwrite a same-named logical slot — that would clobber another
// tool's state. The caller (or the user) must resolve such conflicts manually.
func (p *PostgresqlPhysicalDatabase) VerifyWalSlot(
	ctx context.Context,
	logger *slog.Logger,
	encryptor encryption.FieldEncryptor,
) error {
	if p.ReplicationSlotName == "" {
		return errors.New("replication slot name is empty; row corruption or BeforeCreate bypassed")
	}

	conn, err := openConn(ctx, p, encryptor)
	if err != nil {
		return fmt.Errorf("open conn: %w", err)
	}
	defer closeConnQuietly(ctx, conn, logger)

	var slotType string
	err = conn.QueryRow(ctx,
		"SELECT slot_type FROM pg_replication_slots WHERE slot_name = $1",
		p.ReplicationSlotName,
	).Scan(&slotType)

	switch {
	case err == nil:
		if slotType != "physical" {
			return fmt.Errorf(
				"replication slot %q exists with type %q (expected physical); resolve manually",
				p.ReplicationSlotName, slotType,
			)
		}

		logger.Debug("replication slot already exists", "slot_name", p.ReplicationSlotName)

		return nil

	case errors.Is(err, pgx.ErrNoRows):

	default:
		return fmt.Errorf("query pg_replication_slots: %w", err)
	}

	_, err = conn.Exec(ctx,
		"SELECT pg_create_physical_replication_slot($1, true)",
		p.ReplicationSlotName,
	)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "42710" {
			logger.Debug("replication slot created by concurrent verify",
				"slot_name", p.ReplicationSlotName)

			return nil
		}

		return fmt.Errorf("create physical replication slot: %w", err)
	}

	logger.Info("replication slot created", "slot_name", p.ReplicationSlotName)

	return nil
}

// DropWalSlot removes the physical replication slot named p.ReplicationSlotName
// from the upstream cluster. Idempotent: no-op if the slot is already gone.
// Caller (typically the OnBeforeDatabaseRemove listener — see BACKUPS-PLAN.md)
// must ensure any local pg_receivewal against this slot is stopped first;
// DropWalSlot refuses to terminate other sessions to avoid clobbering external
// tools that may share the same name.
//
// Refuses on logical-slot collision (same defensive reasoning as VerifyWalSlot).
func (p *PostgresqlPhysicalDatabase) DropWalSlot(
	ctx context.Context,
	logger *slog.Logger,
	encryptor encryption.FieldEncryptor,
) error {
	if p.ReplicationSlotName == "" {
		return errors.New("replication slot name is empty; row corruption or BeforeCreate bypassed")
	}

	conn, err := openConn(ctx, p, encryptor)
	if err != nil {
		return fmt.Errorf("open conn: %w", err)
	}
	defer closeConnQuietly(ctx, conn, logger)

	var slotType string
	var isActive bool
	err = conn.QueryRow(ctx,
		"SELECT slot_type, active FROM pg_replication_slots WHERE slot_name = $1",
		p.ReplicationSlotName,
	).Scan(&slotType, &isActive)

	switch {
	case errors.Is(err, pgx.ErrNoRows):
		logger.Debug("replication slot already absent", "slot_name", p.ReplicationSlotName)

		return nil

	case err != nil:
		return fmt.Errorf("query pg_replication_slots: %w", err)
	}

	if slotType != "physical" {
		return fmt.Errorf(
			"replication slot %q exists with type %q (expected physical); refusing to drop",
			p.ReplicationSlotName, slotType,
		)
	}

	if isActive {
		return fmt.Errorf(
			"replication slot %q is active; stop the consumer (pg_receivewal) before deletion",
			p.ReplicationSlotName,
		)
	}

	_, err = conn.Exec(ctx,
		"SELECT pg_drop_replication_slot($1)",
		p.ReplicationSlotName,
	)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "42704" {
			logger.Debug("replication slot dropped by concurrent caller",
				"slot_name", p.ReplicationSlotName)

			return nil
		}

		return fmt.Errorf("drop replication slot: %w", err)
	}

	logger.Info("replication slot dropped", "slot_name", p.ReplicationSlotName)

	return nil
}

// DropWalSlotForRemoval drops the persistent WAL slot when its owning database is
// being removed. Unlike DropWalSlot it does NOT refuse an active slot: on database
// deletion the local pg_receivewal is torn down concurrently (cancelled by the
// removal listener), so the consumer may still be attached or mid-detach when this
// runs. It terminates any remaining consumer of *this* slot — safe because the slot
// name (databasus_slot_<uuid>) is owned exclusively by this database — waits for the
// slot to go inactive, then drops it. This guarantees a deleted database never
// leaves a slot pinning WAL on the source. Bounded by ctx; the caller passes a
// deadline. Refuses on logical-slot collision (same reasoning as VerifyWalSlot).
func (p *PostgresqlPhysicalDatabase) DropWalSlotForRemoval(
	ctx context.Context,
	logger *slog.Logger,
	encryptor encryption.FieldEncryptor,
) error {
	if p.ReplicationSlotName == "" {
		return errors.New("replication slot name is empty; row corruption or BeforeCreate bypassed")
	}

	conn, err := openConn(ctx, p, encryptor)
	if err != nil {
		return fmt.Errorf("open conn: %w", err)
	}
	defer closeConnQuietly(ctx, conn, logger)

	for {
		var slotType string
		var isActive bool
		var activePID *int
		err = conn.QueryRow(ctx,
			"SELECT slot_type, active, active_pid FROM pg_replication_slots WHERE slot_name = $1",
			p.ReplicationSlotName,
		).Scan(&slotType, &isActive, &activePID)

		switch {
		case errors.Is(err, pgx.ErrNoRows):
			logger.Debug("replication slot already absent", "slot_name", p.ReplicationSlotName)

			return nil

		case err != nil:
			return fmt.Errorf("query pg_replication_slots: %w", err)
		}

		if slotType != "physical" {
			return fmt.Errorf(
				"replication slot %q exists with type %q (expected physical); refusing to drop",
				p.ReplicationSlotName, slotType,
			)
		}

		if !isActive {
			break
		}

		// The slot is ours and the database is going away — evict the consumer.
		// pg_receivewal clears the active flag a moment after SIGTERM, so poll
		// rather than assume the detach is instantaneous.
		if activePID != nil {
			if _, termErr := conn.Exec(ctx, "SELECT pg_terminate_backend($1)", *activePID); termErr != nil {
				logger.Warn("failed to terminate replication slot consumer",
					"slot_name", p.ReplicationSlotName, "active_pid", *activePID, "error", termErr)
			}
		}

		select {
		case <-ctx.Done():
			return fmt.Errorf("waiting for replication slot %q to detach: %w", p.ReplicationSlotName, ctx.Err())
		case <-time.After(250 * time.Millisecond):
		}
	}

	_, err = conn.Exec(ctx, "SELECT pg_drop_replication_slot($1)", p.ReplicationSlotName)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "42704" {
			logger.Debug("replication slot dropped by concurrent caller", "slot_name", p.ReplicationSlotName)

			return nil
		}

		return fmt.Errorf("drop replication slot: %w", err)
	}

	logger.Info("replication slot dropped", "slot_name", p.ReplicationSlotName)

	return nil
}

func (p *PostgresqlPhysicalDatabase) GetClusterSizeMb(
	ctx context.Context,
	logger *slog.Logger,
	encryptor encryption.FieldEncryptor,
) (float64, error) {
	conn, err := openConn(ctx, p, encryptor)
	if err != nil {
		return 0, fmt.Errorf("failed to connect to cluster: %w", err)
	}
	defer closeConnQuietly(ctx, conn, logger)

	var sizeBytes int64
	if err := conn.QueryRow(ctx, `
		SELECT COALESCE(SUM(pg_database_size(datname)), 0)::bigint
		FROM pg_database
		WHERE datistemplate = false
	`).Scan(&sizeBytes); err != nil {
		return 0, fmt.Errorf("failed to query cluster size: %w", err)
	}

	return float64(sizeBytes) / (1024 * 1024), nil
}

// IsUserReplicationOnly reports whether the configured user has only
// LOGIN + REPLICATION (or its cloud equivalent) and nothing more. Mirrors
// logical's IsUserReadOnly but tuned for physical: cloud admin roles
// (rds_superuser, azure_pg_admin, cloudsqlsuperuser) count as "excessive"
// even though they don't flip rolsuper on managed PG.
//
// Returns (isMinimal, excessivePrivileges, error). REPLICATION itself is the
// baseline and is NOT in the excessive list — it is validated separately by
// TestReplicationConnection.
func (p *PostgresqlPhysicalDatabase) IsUserReplicationOnly(
	ctx context.Context,
	logger *slog.Logger,
	encryptor encryption.FieldEncryptor,
) (bool, []string, error) {
	conn, err := openConn(ctx, p, encryptor)
	if err != nil {
		return false, nil, fmt.Errorf("failed to connect to cluster: %w", err)
	}
	defer closeConnQuietly(ctx, conn, logger)

	var excessive []string

	var isSuper, canCreateRole, canCreateDB, canBypassRLS bool
	err = conn.QueryRow(ctx, `
		SELECT rolsuper, rolcreaterole, rolcreatedb, rolbypassrls
		FROM pg_roles
		WHERE rolname = current_user
	`).Scan(&isSuper, &canCreateRole, &canCreateDB, &canBypassRLS)
	if err != nil {
		return false, nil, fmt.Errorf("failed to read role attributes: %w", err)
	}

	if isSuper {
		excessive = append(excessive, "SUPERUSER")
	}
	if canCreateRole {
		excessive = append(excessive, "CREATEROLE")
	}
	if canCreateDB {
		excessive = append(excessive, "CREATEDB")
	}
	if canBypassRLS {
		excessive = append(excessive, "BYPASSRLS")
	}

	type cloudRole struct {
		name  string
		label string
	}

	cloudRoles := []cloudRole{
		{"rds_superuser", "rds_superuser (RDS admin)"},
		{"azure_pg_admin", "azure_pg_admin (Azure admin)"},
		{"cloudsqlsuperuser", "cloudsqlsuperuser (GCP admin)"},
		{"pg_write_all_data", "pg_write_all_data"},
	}

	cloudRoleNames := make([]string, 0, len(cloudRoles))
	for _, r := range cloudRoles {
		cloudRoleNames = append(cloudRoleNames, r.name)
	}

	rows, err := conn.Query(ctx, `
		SELECT rolname
		FROM pg_roles
		WHERE rolname = ANY($1)
		  AND pg_has_role(current_user, rolname, 'MEMBER')
	`, cloudRoleNames)
	if err != nil {
		return false, nil, fmt.Errorf("failed to check cloud admin role membership: %w", err)
	}

	memberOf := map[string]bool{}
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			rows.Close()
			return false, nil, fmt.Errorf("failed to scan admin role name: %w", err)
		}
		memberOf[name] = true
	}
	rows.Close()

	if err := rows.Err(); err != nil {
		return false, nil, fmt.Errorf("error iterating admin roles: %w", err)
	}

	for _, r := range cloudRoles {
		if memberOf[r.name] {
			excessive = append(excessive, r.label)
		}
	}

	writePrivileges := map[string]bool{
		"INSERT":     true,
		"UPDATE":     true,
		"DELETE":     true,
		"TRUNCATE":   true,
		"REFERENCES": true,
		"TRIGGER":    true,
	}

	tableRows, err := conn.Query(ctx, `
		SELECT DISTINCT privilege_type
		FROM information_schema.role_table_grants
		WHERE grantee = current_user
		  AND table_schema NOT IN ('pg_catalog', 'information_schema')
	`)
	if err != nil {
		return false, nil, fmt.Errorf("failed to check table grants: %w", err)
	}

	for tableRows.Next() {
		var p string
		if err := tableRows.Scan(&p); err != nil {
			tableRows.Close()
			return false, nil, fmt.Errorf("failed to scan privilege: %w", err)
		}
		if writePrivileges[p] {
			excessive = append(excessive, p)
		}
	}
	tableRows.Close()

	if err := tableRows.Err(); err != nil {
		return false, nil, fmt.Errorf("error iterating table grants: %w", err)
	}

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
		return false, nil, fmt.Errorf("failed to check schema CREATE: %w", err)
	}
	if hasSchemaCreate {
		excessive = append(excessive, "CREATE (schema)")
	}

	var secdefCount int
	err = conn.QueryRow(ctx, `
		SELECT COUNT(*)
		FROM pg_proc proc
		JOIN pg_namespace n ON proc.pronamespace = n.oid
		WHERE n.nspname NOT IN ('pg_catalog', 'information_schema')
		  AND proc.prosecdef = true
		  AND has_function_privilege(current_user, proc.oid, 'EXECUTE')
	`).Scan(&secdefCount)
	if err != nil {
		return false, nil, fmt.Errorf("failed to check SECURITY DEFINER grants: %w", err)
	}
	if secdefCount > 0 {
		excessive = append(excessive, "EXECUTE (SECURITY DEFINER)")
	}

	return len(excessive) == 0, excessive, nil
}

// CreateReplicationOnlyUser provisions a fresh role with exactly
// LOGIN + REPLICATION (or its cloud equivalent). Mirrors logical's
// CreateReadOnlyUser but trivially smaller — physical doesn't need schema /
// table grants. Cloud-aware: each platform grants replication differently
// and some (Azure / GCP) require operator action in the console first.
func (p *PostgresqlPhysicalDatabase) CreateReplicationOnlyUser(
	ctx context.Context,
	logger *slog.Logger,
	encryptor encryption.FieldEncryptor,
) (string, string, error) {
	conn, err := openConn(ctx, p, encryptor)
	if err != nil {
		return "", "", fmt.Errorf("failed to connect to cluster: %w", err)
	}
	defer closeConnQuietly(ctx, conn, logger)

	if err := assertCanCreateRole(ctx, conn); err != nil {
		return "", "", err
	}

	platform := detectPlatform(ctx, conn)

	const maxRetries = 3
	for attempt := range maxRetries {
		baseUsername := "databasus-" + uuid.New().String()[:8]
		newPassword := encryption.GenerateComplexPassword()

		tx, err := conn.Begin(ctx)
		if err != nil {
			return "", "", fmt.Errorf("failed to begin transaction: %w", err)
		}

		isCommitted := false
		defer func() {
			if !isCommitted {
				if rollbackErr := tx.Rollback(ctx); rollbackErr != nil {
					logger.Warn("failed to rollback transaction", "error", rollbackErr)
				}
			}
		}()

		_, err = tx.Exec(
			ctx,
			fmt.Sprintf(`CREATE USER "%s" WITH PASSWORD '%s' LOGIN`, baseUsername, newPassword),
		)
		if err != nil {
			if strings.Contains(err.Error(), "already exists") && attempt < maxRetries-1 {
				continue
			}
			return "", "", fmt.Errorf("failed to create user: %w", err)
		}

		if err := grantReplication(ctx, tx, baseUsername, platform); err != nil {
			return "", "", err
		}

		var verifyName string
		if err := tx.QueryRow(
			ctx,
			`SELECT rolname FROM pg_roles WHERE rolname = $1`,
			baseUsername,
		).Scan(&verifyName); err != nil {
			return "", "", fmt.Errorf("failed to verify user creation: %w", err)
		}

		if err := tx.Commit(ctx); err != nil {
			return "", "", fmt.Errorf("failed to commit transaction: %w", err)
		}
		isCommitted = true

		logger.Info("replication-only user created", "username", baseUsername, "platform", platform)
		return baseUsername, newPassword, nil
	}

	return "", "", errors.New("failed to generate unique username after 3 attempts")
}

// OpenInspectionConn opens a regular (non-replication) connection to the
// source cluster. Used by the FULL/INCR executors for pre-flight checks
// (timeline.CheckTimelineCompatibility), .history file reads, and
// post-stream LSN validation.
func (p *PostgresqlPhysicalDatabase) OpenInspectionConn(
	ctx context.Context,
	encryptor encryption.FieldEncryptor,
) (*pgx.Conn, error) {
	return openConn(ctx, p, encryptor)
}

func (p *PostgresqlPhysicalDatabase) checkReplicationReadiness(
	logger *slog.Logger,
	encryptor encryption.FieldEncryptor,
) error {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	replConn, err := openPhysicalReplicationConn(ctx, p, encryptor)
	if err != nil {
		return classifyReplicationConnectError(err, p.Username)
	}
	defer func() {
		if err := replConn.Close(ctx); err != nil {
			logger.Warn("failed to close replication connection", "error", err)
		}
	}()

	// pg_replication_slots is not in the replication command set, so the
	// inspection queries need an ordinary connection.
	conn, err := openConn(ctx, p, encryptor)
	if err != nil {
		return fmt.Errorf("failed to open inspection connection: %w", err)
	}
	defer closeConnQuietly(ctx, conn, logger)

	if err := p.checkNoCustomTablespaces(ctx, conn); err != nil {
		return err
	}

	settings, err := readReplicationSettings(ctx, conn)
	if err != nil {
		return err
	}

	if settings.walLevel != "replica" && settings.walLevel != "logical" {
		return &postgresql_shared.ConnectionTestError{Code: postgresql_shared.ConnErrWalLevelInvalid}
	}

	if settings.maxWalSenders == 0 {
		return &postgresql_shared.ConnectionTestError{Code: postgresql_shared.ConnErrNoWalSenders}
	}

	if settings.maxReplicationSlots == 0 {
		return &postgresql_shared.ConnectionTestError{Code: postgresql_shared.ConnErrNoReplicationSlots}
	}

	if p.BackupType.IsRequireWalSummary() && settings.summarizeWal != "on" {
		return &postgresql_shared.ConnectionTestError{Code: postgresql_shared.ConnErrWalSummaryDisabled}
	}

	if p.SystemIdentifier != nil {
		var currentSysID string
		if err := conn.QueryRow(ctx, "SELECT system_identifier::text FROM pg_control_system()").
			Scan(&currentSysID); err != nil {
			return fmt.Errorf("failed to read cluster system_identifier: %w", err)
		}

		if currentSysID != *p.SystemIdentifier {
			return &postgresql_shared.ConnectionTestError{Code: postgresql_shared.ConnErrSystemIdentifierMismatch}
		}
	}

	return nil
}

// checkNoCustomTablespaces enforces ADR-0010: physical backups refuse any
// cluster that has tablespaces outside pg_default / pg_global, because
// pg_basebackup --pgdata=- -F tar cannot multiplex multi-tablespace output
// into a single stdout stream.
func (p *PostgresqlPhysicalDatabase) checkNoCustomTablespaces(
	ctx context.Context,
	conn *pgx.Conn,
) error {
	rows, err := conn.Query(ctx, `
		SELECT spcname
		FROM pg_tablespace
		WHERE spcname NOT IN ('pg_default', 'pg_global')
		ORDER BY spcname
	`)
	if err != nil {
		return fmt.Errorf("query pg_tablespace: %w", err)
	}
	defer rows.Close()

	var spcnames []string

	for rows.Next() {
		var name string

		if err := rows.Scan(&name); err != nil {
			return fmt.Errorf("scan pg_tablespace row: %w", err)
		}

		spcnames = append(spcnames, name)
	}

	if err := rows.Err(); err != nil {
		return fmt.Errorf("iterate pg_tablespace: %w", err)
	}

	if len(spcnames) > 0 {
		return &postgresql_shared.ConnectionTestError{Code: postgresql_shared.ConnErrCustomTablespaces}
	}

	return nil
}

func detectPlatform(ctx context.Context, conn *pgx.Conn) platform {
	if roleExists(ctx, conn, "rds_replication") {
		return platformRds
	}
	if roleExists(ctx, conn, "azure_pg_admin") {
		return platformAzure
	}
	if roleExists(ctx, conn, "cloudsqlsuperuser") {
		return platformGcp
	}

	var isSuper string
	if err := conn.QueryRow(ctx, "SHOW is_superuser").Scan(&isSuper); err == nil {
		if strings.EqualFold(isSuper, "off") {
			return platformUnknownManaged
		}
	}

	return platformSelfManaged
}

func roleExists(ctx context.Context, conn *pgx.Conn, name string) bool {
	var exists bool

	err := conn.QueryRow(
		ctx,
		`SELECT EXISTS(SELECT 1 FROM pg_roles WHERE rolname = $1)`,
		name,
	).Scan(&exists)

	return err == nil && exists
}

func assertCanCreateRole(ctx context.Context, conn *pgx.Conn) error {
	var canCreate, isSuper bool

	if err := conn.QueryRow(ctx, `
		SELECT rolcreaterole, rolsuper
		FROM pg_roles
		WHERE rolname = current_user
	`).Scan(&canCreate, &isSuper); err != nil {
		return fmt.Errorf("failed to check role-create capability: %w", err)
	}

	if canCreate || isSuper {
		return nil
	}

	// Cloud admins frequently have rolcreaterole=false yet can create users
	// via membership in the platform admin role.
	cloudAdmins := []string{"rds_superuser", "azure_pg_admin", "cloudsqlsuperuser"}

	for _, role := range cloudAdmins {
		if !roleExists(ctx, conn, role) {
			continue
		}

		var isMember bool
		if err := conn.QueryRow(
			ctx,
			`SELECT pg_has_role(current_user, $1, 'MEMBER')`,
			role,
		).Scan(&isMember); err == nil && isMember {
			return nil
		}
	}

	return errors.New("current user cannot create roles — connect as a platform admin")
}

func grantReplication(ctx context.Context, tx pgx.Tx, username string, plat platform) error {
	switch plat {
	case platformSelfManaged:
		_, err := tx.Exec(ctx, fmt.Sprintf(`ALTER ROLE "%s" REPLICATION`, username))
		if err != nil {
			return fmt.Errorf("failed to grant REPLICATION: %w", err)
		}

	case platformRds:
		_, err := tx.Exec(ctx, fmt.Sprintf(`GRANT rds_replication TO "%s"`, username))
		if err != nil {
			return fmt.Errorf("failed to grant rds_replication: %w", err)
		}

	case platformAzure:
		_, err := tx.Exec(ctx, fmt.Sprintf(`ALTER ROLE "%s" REPLICATION`, username))
		if err != nil {
			if strings.Contains(err.Error(), "permission denied") ||
				strings.Contains(err.Error(), "must have") {
				return errors.New(
					"replication must be enabled at the server level in the Azure portal (Server parameters → wal_level=logical → restart); after that, retry",
				)
			}
			return fmt.Errorf("failed to grant REPLICATION on Azure: %w", err)
		}

	case platformGcp:
		_, err := tx.Exec(ctx, fmt.Sprintf(`ALTER ROLE "%s" REPLICATION`, username))
		if err != nil {
			if strings.Contains(err.Error(), "permission denied") ||
				strings.Contains(err.Error(), "must have") {
				return errors.New(
					"GCP Cloud SQL: external replication requires console-side enablement (see cloud.google.com/sql/docs/postgres/replication/configure-external-replica); provision the user via gcloud after enabling, or use REPLICATION-capable credentials directly",
				)
			}
			return fmt.Errorf("failed to grant REPLICATION on GCP: %w", err)
		}

	case platformUnknownManaged:
		_, err := tx.Exec(ctx, fmt.Sprintf(`ALTER ROLE "%s" REPLICATION`, username))
		if err != nil {
			if strings.Contains(err.Error(), "permission denied") ||
				strings.Contains(err.Error(), "must have") {
				return errors.New(
					"replication grant denied on this managed PG; consult the platform docs for how to grant REPLICATION (or membership in its replication role)",
				)
			}
			return fmt.Errorf("failed to grant REPLICATION: %w", err)
		}

	default:
		return fmt.Errorf("unknown platform: %s", plat)
	}

	return nil
}

func readReplicationSettings(ctx context.Context, conn *pgx.Conn) (*replicationSettings, error) {
	s := &replicationSettings{}

	if err := conn.QueryRow(ctx, `
		SELECT
			(SELECT setting FROM pg_settings WHERE name = 'wal_level'),
			COALESCE((SELECT setting FROM pg_settings WHERE name = 'summarize_wal'), 'off'),
			(SELECT setting::int FROM pg_settings WHERE name = 'max_wal_senders'),
			(SELECT setting::int FROM pg_settings WHERE name = 'max_replication_slots')
	`).Scan(
		&s.walLevel,
		&s.summarizeWal,
		&s.maxWalSenders,
		&s.maxReplicationSlots,
	); err != nil {
		return nil, fmt.Errorf("failed to read replication settings: %w", err)
	}

	return s, nil
}

func classifyReplicationConnectError(err error, username string) error {
	msg := err.Error()

	switch {
	case strings.Contains(msg, "no pg_hba.conf entry"):
		return &postgresql_shared.ConnectionTestError{
			Code: postgresql_shared.ConnErrPgHbaNoEntry,
			Message: fmt.Sprintf(
				"pg_hba.conf has no replication entry for user %q. "+
					"Add \"host replication %s all scram-sha-256\" to pg_hba.conf and reload PostgreSQL",
				username, username,
			),
		}
	case strings.Contains(msg, "password authentication failed"):
		return &postgresql_shared.ConnectionTestError{Code: postgresql_shared.ConnErrBadCredentials}
	case strings.Contains(msg, "must have replication privilege") ||
		strings.Contains(msg, "must be replication role") ||
		strings.Contains(msg, "permission denied to start WAL sender"):
		return &postgresql_shared.ConnectionTestError{Code: postgresql_shared.ConnErrNoReplicationPrivilege}
	default:
		return &postgresql_shared.ConnectionTestError{Code: postgresql_shared.ConnErrConnectionFailed}
	}
}

// openConn opens a regular pgx connection to the `postgres` database (always
// exists; physical model has no per-DB selection), carrying the source's client
// certificates the same way pg_basebackup does via the shared credential files.
func openConn(
	ctx context.Context,
	p *PostgresqlPhysicalDatabase,
	encryptor encryption.FieldEncryptor,
) (*pgx.Conn, error) {
	password, err := postgresql_shared.DecryptFieldIfNeeded(p.Password, encryptor)
	if err != nil {
		return nil, fmt.Errorf("failed to decrypt password: %w", err)
	}

	files, err := postgresql_shared.WriteCredentialFilesToTempDir(p.CredentialSpec(), password, encryptor)
	if err != nil {
		return nil, err
	}
	defer files.Remove()

	return pgx.Connect(ctx, postgresql_shared.BuildConnString(p.CredentialSpec(), password, "postgres", files))
}

// openPhysicalReplicationConn opens a PHYSICAL replication connection (replication=true) — the same
// mode pg_basebackup / pg_receivewal use. This is what exercises the "host replication" pg_hba path
// and the REPLICATION privilege at connect time; an ordinary "host all" rule does NOT cover it, so a
// logical (replication=database) probe would wrongly accept a cluster that real backups cannot stream.
// Uses the low-level pgconn because no ordinary SQL is allowed on a physical replication connection.
func openPhysicalReplicationConn(
	ctx context.Context,
	p *PostgresqlPhysicalDatabase,
	encryptor encryption.FieldEncryptor,
) (*pgconn.PgConn, error) {
	password, err := postgresql_shared.DecryptFieldIfNeeded(p.Password, encryptor)
	if err != nil {
		return nil, fmt.Errorf("failed to decrypt password: %w", err)
	}

	files, err := postgresql_shared.WriteCredentialFilesToTempDir(p.CredentialSpec(), password, encryptor)
	if err != nil {
		return nil, err
	}
	defer files.Remove()

	return pgconn.Connect(
		ctx,
		postgresql_shared.BuildPhysicalReplicationConnString(p.CredentialSpec(), password, "postgres", files),
	)
}

func closeConnQuietly(ctx context.Context, conn *pgx.Conn, logger *slog.Logger) {
	if err := conn.Close(ctx); err != nil {
		logger.Warn("failed to close connection", "error", err)
	}
}

var versionRegexp = regexp.MustCompile(`PostgreSQL (\d+)`)

func detectVersion(ctx context.Context, conn *pgx.Conn) (tools.PostgresqlVersion, error) {
	var versionStr string
	if err := conn.QueryRow(ctx, "SELECT version()").Scan(&versionStr); err != nil {
		return "", fmt.Errorf("failed to query version(): %w", err)
	}

	matches := versionRegexp.FindStringSubmatch(versionStr)
	if len(matches) < 2 {
		return "", fmt.Errorf("could not parse version from: %s", versionStr)
	}

	switch matches[1] {
	case "17":
		return tools.PostgresqlVersion17, nil
	case "18":
		return tools.PostgresqlVersion18, nil
	default:
		return "", fmt.Errorf("physical backup requires PostgreSQL 17 or 18, detected %s", matches[1])
	}
}
