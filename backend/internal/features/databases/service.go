package databases

import (
	"context"
	"crypto/sha256"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/google/uuid"

	"databasus-backend/internal/config"
	audit_logs "databasus-backend/internal/features/audit_logs"
	"databasus-backend/internal/features/databases/databases/clickhouse"
	"databasus-backend/internal/features/databases/databases/mariadb"
	"databasus-backend/internal/features/databases/databases/mongodb"
	"databasus-backend/internal/features/databases/databases/mysql"
	"databasus-backend/internal/features/databases/databases/postgresql"
	"databasus-backend/internal/features/notifiers"
	users_models "databasus-backend/internal/features/users/models"
	workspaces_services "databasus-backend/internal/features/workspaces/services"
	"databasus-backend/internal/util/encryption"
)

type DatabaseService struct {
	dbRepository    *DatabaseRepository
	notifierService *notifiers.NotifierService
	logger          *slog.Logger

	dbCreationListener []DatabaseCreationListener
	dbRemoveListener   []DatabaseRemoveListener
	dbCopyListener     []DatabaseCopyListener

	workspaceService *workspaces_services.WorkspaceService
	auditLogService  *audit_logs.AuditLogService
	fieldEncryptor   encryption.FieldEncryptor
}

func (s *DatabaseService) AddDbCreationListener(
	dbCreationListener DatabaseCreationListener,
) {
	s.dbCreationListener = append(s.dbCreationListener, dbCreationListener)
}

func (s *DatabaseService) AddDbRemoveListener(
	dbRemoveListener DatabaseRemoveListener,
) {
	s.dbRemoveListener = append(s.dbRemoveListener, dbRemoveListener)
}

func (s *DatabaseService) AddDbCopyListener(
	dbCopyListener DatabaseCopyListener,
) {
	s.dbCopyListener = append(s.dbCopyListener, dbCopyListener)
}

func (s *DatabaseService) GetNotifierAttachedDatabasesIDs(
	notifierID uuid.UUID,
) ([]uuid.UUID, error) {
	databasesIDs, err := s.dbRepository.GetDatabasesIDsByNotifierID(notifierID)
	if err != nil {
		return nil, err
	}

	return databasesIDs, nil
}

func (s *DatabaseService) CreateDatabase(
	user *users_models.User,
	workspaceID uuid.UUID,
	database *Database,
) (*Database, error) {
	canManage, err := s.workspaceService.CanUserManageDBs(workspaceID, user)
	if err != nil {
		return nil, err
	}
	if !canManage {
		return nil, errors.New("insufficient permissions to create database in this workspace")
	}

	database.WorkspaceID = &workspaceID

	if err := database.Validate(); err != nil {
		return nil, err
	}

	if err := database.PopulateDbData(s.logger, s.fieldEncryptor); err != nil {
		return nil, fmt.Errorf("failed to auto-detect database data: %w", err)
	}

	if err := s.verifyReadOnlyUserIfNeeded(database); err != nil {
		return nil, err
	}

	if err := database.EncryptSensitiveFields(s.fieldEncryptor); err != nil {
		return nil, fmt.Errorf("failed to encrypt sensitive fields: %w", err)
	}

	database, err = s.dbRepository.Save(database)
	if err != nil {
		return nil, err
	}

	for _, listener := range s.dbCreationListener {
		listener.OnDatabaseCreated(database.ID)
	}

	s.auditLogService.WriteAuditLog(
		fmt.Sprintf("Database created: %s", database.Name),
		&user.ID,
		&workspaceID,
	)

	return database, nil
}

func (s *DatabaseService) UpdateDatabase(
	user *users_models.User,
	database *Database,
) error {
	if database.ID == uuid.Nil {
		return errors.New("database ID is required for update")
	}

	existingDatabase, err := s.dbRepository.FindByID(database.ID)
	if err != nil {
		return err
	}

	if existingDatabase.WorkspaceID == nil {
		return errors.New("cannot update database without workspace")
	}

	canManage, err := s.workspaceService.CanUserManageDBs(*existingDatabase.WorkspaceID, user)
	if err != nil {
		return err
	}
	if !canManage {
		return errors.New("insufficient permissions to update this database")
	}

	if err := database.ValidateUpdate(*existingDatabase, *database); err != nil {
		return err
	}

	for _, notifier := range database.Notifiers {
		if notifier.WorkspaceID != *existingDatabase.WorkspaceID {
			return errors.New("notifier does not belong to this workspace")
		}
	}

	existingDatabase.Update(database)

	if err := existingDatabase.Validate(); err != nil {
		return err
	}

	if err := existingDatabase.PopulateDbData(s.logger, s.fieldEncryptor); err != nil {
		return fmt.Errorf("failed to auto-detect database data: %w", err)
	}

	if err := s.verifyReadOnlyUserIfNeeded(existingDatabase); err != nil {
		return err
	}

	oldName := existingDatabase.Name

	if err := existingDatabase.EncryptSensitiveFields(s.fieldEncryptor); err != nil {
		return fmt.Errorf("failed to encrypt sensitive fields: %w", err)
	}

	_, err = s.dbRepository.Save(existingDatabase)
	if err != nil {
		return err
	}

	if oldName != existingDatabase.Name {
		s.auditLogService.WriteAuditLog(
			fmt.Sprintf(
				"Database updated and renamed from '%s' to '%s'",
				oldName,
				existingDatabase.Name,
			),
			&user.ID,
			existingDatabase.WorkspaceID,
		)
	} else {
		s.auditLogService.WriteAuditLog(
			fmt.Sprintf("Database updated: %s", existingDatabase.Name),
			&user.ID,
			existingDatabase.WorkspaceID,
		)
	}

	return nil
}

func (s *DatabaseService) DeleteDatabase(
	user *users_models.User,
	id uuid.UUID,
) error {
	existingDatabase, err := s.dbRepository.FindByID(id)
	if err != nil {
		return err
	}

	if existingDatabase.WorkspaceID == nil {
		return errors.New("cannot delete database without workspace")
	}

	canManage, err := s.workspaceService.CanUserManageDBs(*existingDatabase.WorkspaceID, user)
	if err != nil {
		return err
	}
	if !canManage {
		return errors.New("insufficient permissions to delete this database")
	}

	for _, listener := range s.dbRemoveListener {
		if err := listener.OnBeforeDatabaseRemove(id); err != nil {
			return err
		}
	}

	s.auditLogService.WriteAuditLog(
		fmt.Sprintf("Database deleted: %s", existingDatabase.Name),
		&user.ID,
		existingDatabase.WorkspaceID,
	)

	return s.dbRepository.Delete(id)
}

func (s *DatabaseService) GetDatabase(
	user *users_models.User,
	id uuid.UUID,
) (*Database, error) {
	database, err := s.dbRepository.FindByID(id)
	if err != nil {
		return nil, err
	}

	if database.WorkspaceID == nil {
		return nil, errors.New("cannot access database without workspace")
	}

	canAccess, _, err := s.workspaceService.CanUserAccessWorkspace(*database.WorkspaceID, user)
	if err != nil {
		return nil, err
	}
	if !canAccess {
		return nil, errors.New("insufficient permissions to access this database")
	}

	database.HideSensitiveData()
	return database, nil
}

func (s *DatabaseService) GetDatabasesByWorkspace(
	user *users_models.User,
	workspaceID uuid.UUID,
) ([]*Database, error) {
	canAccess, _, err := s.workspaceService.CanUserAccessWorkspace(workspaceID, user)
	if err != nil {
		return nil, err
	}
	if !canAccess {
		return nil, errors.New("insufficient permissions to access this workspace")
	}

	databases, err := s.dbRepository.FindByWorkspaceID(workspaceID)
	if err != nil {
		return nil, err
	}

	for _, database := range databases {
		database.HideSensitiveData()
	}

	return databases, nil
}

func (s *DatabaseService) IsNotifierUsing(
	user *users_models.User,
	notifierID uuid.UUID,
) (bool, error) {
	_, err := s.notifierService.GetNotifier(user, notifierID)
	if err != nil {
		return false, err
	}

	return s.dbRepository.IsNotifierUsing(notifierID)
}

func (s *DatabaseService) CountDatabasesByNotifier(
	user *users_models.User,
	notifierID uuid.UUID,
) (int, error) {
	_, err := s.notifierService.GetNotifier(user, notifierID)
	if err != nil {
		return 0, err
	}

	databaseIDs, err := s.dbRepository.GetDatabasesIDsByNotifierID(notifierID)
	if err != nil {
		return 0, err
	}

	return len(databaseIDs), nil
}

func (s *DatabaseService) TestDatabaseConnection(
	user *users_models.User,
	databaseID uuid.UUID,
) error {
	database, err := s.dbRepository.FindByID(databaseID)
	if err != nil {
		return err
	}

	if database.WorkspaceID == nil {
		return errors.New("cannot test connection for database without workspace")
	}

	canAccess, _, err := s.workspaceService.CanUserAccessWorkspace(*database.WorkspaceID, user)
	if err != nil {
		return err
	}
	if !canAccess {
		return errors.New("insufficient permissions to test connection for this database")
	}

	err = database.TestConnection(s.logger, s.fieldEncryptor)
	if err != nil {
		lastSaveError := err.Error()
		database.LastBackupErrorMessage = &lastSaveError
		return err
	}

	database.LastBackupErrorMessage = nil

	_, err = s.dbRepository.Save(database)
	if err != nil {
		return err
	}

	return nil
}

func (s *DatabaseService) TestDatabaseConnectionDirect(
	database *Database,
) error {
	var usingDatabase *Database

	if database.ID != uuid.Nil {
		existingDatabase, err := s.dbRepository.FindByID(database.ID)
		if err != nil {
			return err
		}

		if database.WorkspaceID != nil && existingDatabase.WorkspaceID != nil &&
			*existingDatabase.WorkspaceID != *database.WorkspaceID {
			return errors.New("database does not belong to this workspace")
		}

		existingDatabase.Update(database)

		if err := existingDatabase.Validate(); err != nil {
			return err
		}

		usingDatabase = existingDatabase
	} else {
		usingDatabase = database
	}

	return usingDatabase.TestConnection(s.logger, s.fieldEncryptor)
}

func (s *DatabaseService) GetDatabaseByID(
	id uuid.UUID,
) (*Database, error) {
	return s.dbRepository.FindByID(id)
}

func (s *DatabaseService) GetAllDatabases() ([]*Database, error) {
	return s.dbRepository.GetAllDatabases()
}

func (s *DatabaseService) SetBackupError(databaseID uuid.UUID, errorMessage string) error {
	database, err := s.dbRepository.FindByID(databaseID)
	if err != nil {
		return err
	}

	database.LastBackupErrorMessage = &errorMessage
	_, err = s.dbRepository.Save(database)
	if err != nil {
		return err
	}

	return nil
}

func (s *DatabaseService) SetLastBackupTime(databaseID uuid.UUID, backupTime time.Time) error {
	database, err := s.dbRepository.FindByID(databaseID)
	if err != nil {
		return err
	}

	database.LastBackupTime = &backupTime
	database.LastBackupErrorMessage = nil // Clear any previous error
	_, err = s.dbRepository.Save(database)
	if err != nil {
		return err
	}

	return nil
}

func (s *DatabaseService) CopyDatabase(
	user *users_models.User,
	databaseID uuid.UUID,
) (*Database, error) {
	existingDatabase, err := s.dbRepository.FindByID(databaseID)
	if err != nil {
		return nil, err
	}

	if existingDatabase.WorkspaceID == nil {
		return nil, errors.New("cannot copy database without workspace")
	}

	canManage, err := s.workspaceService.CanUserManageDBs(*existingDatabase.WorkspaceID, user)
	if err != nil {
		return nil, err
	}
	if !canManage {
		return nil, errors.New("insufficient permissions to copy this database")
	}

	newDatabase := &Database{
		ID:                     uuid.Nil,
		WorkspaceID:            existingDatabase.WorkspaceID,
		Name:                   existingDatabase.Name + " (Copy)",
		Type:                   existingDatabase.Type,
		Notifiers:              existingDatabase.Notifiers,
		LastBackupTime:         nil,
		LastBackupErrorMessage: nil,
		HealthStatus:           existingDatabase.HealthStatus,
	}

	switch existingDatabase.Type {
	case DatabaseTypePostgres:
		if existingDatabase.Postgresql != nil {
			newDatabase.Postgresql = &postgresql.PostgresqlDatabase{
				ID:             uuid.Nil,
				DatabaseID:     nil,
				BackupType:     existingDatabase.Postgresql.BackupType,
				Version:        existingDatabase.Postgresql.Version,
				Host:           existingDatabase.Postgresql.Host,
				Port:           existingDatabase.Postgresql.Port,
				Username:       existingDatabase.Postgresql.Username,
				Password:       existingDatabase.Postgresql.Password,
				Database:       existingDatabase.Postgresql.Database,
				IsHttps:        existingDatabase.Postgresql.IsHttps,
				IncludeSchemas: existingDatabase.Postgresql.IncludeSchemas,
				CpuCount:       existingDatabase.Postgresql.CpuCount,
			}
		}
	case DatabaseTypeMysql:
		if existingDatabase.Mysql != nil {
			newDatabase.Mysql = &mysql.MysqlDatabase{
				ID:         uuid.Nil,
				DatabaseID: nil,
				Version:    existingDatabase.Mysql.Version,
				Host:       existingDatabase.Mysql.Host,
				Port:       existingDatabase.Mysql.Port,
				Username:   existingDatabase.Mysql.Username,
				Password:   existingDatabase.Mysql.Password,
				Database:   existingDatabase.Mysql.Database,
				IsHttps:    existingDatabase.Mysql.IsHttps,
			}
		}
	case DatabaseTypeMariadb:
		if existingDatabase.Mariadb != nil {
			newDatabase.Mariadb = &mariadb.MariadbDatabase{
				ID:         uuid.Nil,
				DatabaseID: nil,
				Version:    existingDatabase.Mariadb.Version,
				Host:       existingDatabase.Mariadb.Host,
				Port:       existingDatabase.Mariadb.Port,
				Username:   existingDatabase.Mariadb.Username,
				Password:   existingDatabase.Mariadb.Password,
				Database:   existingDatabase.Mariadb.Database,
				IsHttps:    existingDatabase.Mariadb.IsHttps,
			}
		}
	case DatabaseTypeMongodb:
		if existingDatabase.Mongodb != nil {
			newDatabase.Mongodb = &mongodb.MongodbDatabase{
				ID:           uuid.Nil,
				DatabaseID:   nil,
				Version:      existingDatabase.Mongodb.Version,
				Host:         existingDatabase.Mongodb.Host,
				Port:         existingDatabase.Mongodb.Port,
				Username:     existingDatabase.Mongodb.Username,
				Password:     existingDatabase.Mongodb.Password,
				Database:     existingDatabase.Mongodb.Database,
				AuthDatabase: existingDatabase.Mongodb.AuthDatabase,
				IsHttps:      existingDatabase.Mongodb.IsHttps,
				CpuCount:     existingDatabase.Mongodb.CpuCount,
			}
		}
	case DatabaseTypeClickhouse:
		if existingDatabase.Clickhouse != nil {
			newDatabase.Clickhouse = &clickhouse.ClickhouseDatabase{
				ID:         uuid.Nil,
				DatabaseID: nil,
				Version:    existingDatabase.Clickhouse.Version,
				Host:       existingDatabase.Clickhouse.Host,
				Port:       existingDatabase.Clickhouse.Port,
				Username:   existingDatabase.Clickhouse.Username,
				Password:   existingDatabase.Clickhouse.Password,
				Database:   existingDatabase.Clickhouse.Database,
				IsHttps:    existingDatabase.Clickhouse.IsHttps,
			}
		}
	}

	if err := newDatabase.Validate(); err != nil {
		return nil, err
	}

	copiedDatabase, err := s.dbRepository.Save(newDatabase)
	if err != nil {
		return nil, err
	}

	for _, listener := range s.dbCreationListener {
		listener.OnDatabaseCreated(copiedDatabase.ID)
	}

	for _, listener := range s.dbCopyListener {
		listener.OnDatabaseCopied(databaseID, copiedDatabase.ID)
	}

	s.auditLogService.WriteAuditLog(
		fmt.Sprintf("Database copied: %s to %s", existingDatabase.Name, copiedDatabase.Name),
		&user.ID,
		existingDatabase.WorkspaceID,
	)

	return copiedDatabase, nil
}

func (s *DatabaseService) TransferDatabaseToWorkspace(
	databaseID uuid.UUID,
	targetWorkspaceID uuid.UUID,
) error {
	database, err := s.dbRepository.FindByID(databaseID)
	if err != nil {
		return err
	}

	sourceWorkspaceID := database.WorkspaceID
	database.WorkspaceID = &targetWorkspaceID

	_, err = s.dbRepository.Save(database)
	if err != nil {
		return err
	}

	sourceWorkspace, err := s.workspaceService.GetWorkspaceByID(*sourceWorkspaceID)
	if err != nil {
		return fmt.Errorf("failed to get source workspace: %w", err)
	}

	targetWorkspace, err := s.workspaceService.GetWorkspaceByID(targetWorkspaceID)
	if err != nil {
		return fmt.Errorf("failed to get target workspace: %w", err)
	}

	s.auditLogService.WriteAuditLog(
		fmt.Sprintf("Database transferred: %s from workspace '%s' to workspace '%s'",
			database.Name, sourceWorkspace.Name, targetWorkspace.Name),
		nil,
		&targetWorkspaceID,
	)

	return nil
}

func (s *DatabaseService) UpdateDatabaseNotifiers(
	databaseID uuid.UUID,
	newNotifiers []notifiers.Notifier,
) error {
	database, err := s.dbRepository.FindByID(databaseID)
	if err != nil {
		return err
	}

	database.Notifiers = newNotifiers

	_, err = s.dbRepository.Save(database)
	return err
}

func (s *DatabaseService) SetHealthStatus(
	databaseID uuid.UUID,
	healthStatus *HealthStatus,
) error {
	database, err := s.dbRepository.FindByID(databaseID)
	if err != nil {
		return err
	}

	database.HealthStatus = healthStatus
	_, err = s.dbRepository.Save(database)
	if err != nil {
		return err
	}

	return nil
}

func (s *DatabaseService) RegenerateAgentToken(
	user *users_models.User,
	databaseID uuid.UUID,
) (string, error) {
	database, err := s.dbRepository.FindByID(databaseID)
	if err != nil {
		return "", err
	}

	if database.WorkspaceID == nil {
		return "", errors.New("cannot regenerate token for database without workspace")
	}

	canManage, err := s.workspaceService.CanUserManageDBs(*database.WorkspaceID, user)
	if err != nil {
		return "", err
	}
	if !canManage {
		return "", errors.New(
			"insufficient permissions to regenerate agent token for this database",
		)
	}

	plainToken := strings.ReplaceAll(uuid.New().String(), "-", "")
	tokenHash := hashAgentToken(plainToken)

	database.AgentToken = &tokenHash
	database.IsAgentTokenGenerated = true

	_, err = s.dbRepository.Save(database)
	if err != nil {
		return "", err
	}

	s.auditLogService.WriteAuditLog(
		fmt.Sprintf("Agent token regenerated for database: %s", database.Name),
		&user.ID,
		database.WorkspaceID,
	)

	return plainToken, nil
}

func (s *DatabaseService) VerifyAgentToken(token string) error {
	hash := hashAgentToken(token)

	_, err := s.dbRepository.FindByAgentTokenHash(hash)
	if err != nil {
		return errors.New("invalid token")
	}

	return nil
}

func (s *DatabaseService) GetDatabaseByAgentToken(token string) (*Database, error) {
	hash := hashAgentToken(token)

	partial, err := s.dbRepository.FindByAgentTokenHash(hash)
	if err != nil {
		return nil, errors.New("invalid agent token")
	}

	return s.dbRepository.FindByID(partial.ID)
}

func (s *DatabaseService) OnBeforeWorkspaceDeletion(workspaceID uuid.UUID) error {
	databases, err := s.dbRepository.FindByWorkspaceID(workspaceID)
	if err != nil {
		return err
	}

	if len(databases) > 0 {
		return fmt.Errorf(
			"workspace contains %d databases that must be deleted",
			len(databases),
		)
	}

	return nil
}

func (s *DatabaseService) IsUserReadOnly(
	user *users_models.User,
	database *Database,
) (bool, []string, error) {
	var usingDatabase *Database

	if database.ID != uuid.Nil {
		existingDatabase, err := s.dbRepository.FindByID(database.ID)
		if err != nil {
			return false, nil, err
		}

		if existingDatabase.WorkspaceID == nil {
			return false, nil, errors.New("cannot check user for database without workspace")
		}

		canAccess, _, err := s.workspaceService.CanUserAccessWorkspace(
			*existingDatabase.WorkspaceID,
			user,
		)
		if err != nil {
			return false, nil, err
		}
		if !canAccess {
			return false, nil, errors.New("insufficient permissions to access this database")
		}

		if database.WorkspaceID != nil && *existingDatabase.WorkspaceID != *database.WorkspaceID {
			return false, nil, errors.New("database does not belong to this workspace")
		}

		existingDatabase.Update(database)

		if err := existingDatabase.Validate(); err != nil {
			return false, nil, err
		}

		usingDatabase = existingDatabase
	} else {
		if database.WorkspaceID != nil {
			canAccess, _, err := s.workspaceService.CanUserAccessWorkspace(
				*database.WorkspaceID,
				user,
			)
			if err != nil {
				return false, nil, err
			}
			if !canAccess {
				return false, nil, errors.New("insufficient permissions to access this workspace")
			}
		}

		usingDatabase = database
	}

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	return usingDatabase.IsUserReadOnly(ctx, s.logger, s.fieldEncryptor)
}

func (s *DatabaseService) CreateReadOnlyUser(
	user *users_models.User,
	database *Database,
) (string, string, error) {
	var usingDatabase *Database

	if database.ID != uuid.Nil {
		existingDatabase, err := s.dbRepository.FindByID(database.ID)
		if err != nil {
			return "", "", err
		}

		if existingDatabase.WorkspaceID == nil {
			return "", "", errors.New("cannot create user for database without workspace")
		}

		canManage, err := s.workspaceService.CanUserManageDBs(*existingDatabase.WorkspaceID, user)
		if err != nil {
			return "", "", err
		}
		if !canManage {
			return "", "", errors.New("insufficient permissions to manage this database")
		}

		if database.WorkspaceID != nil && *existingDatabase.WorkspaceID != *database.WorkspaceID {
			return "", "", errors.New("database does not belong to this workspace")
		}

		existingDatabase.Update(database)

		if err := existingDatabase.Validate(); err != nil {
			return "", "", err
		}

		usingDatabase = existingDatabase
	} else {
		if database.WorkspaceID != nil {
			canManage, err := s.workspaceService.CanUserManageDBs(*database.WorkspaceID, user)
			if err != nil {
				return "", "", err
			}
			if !canManage {
				return "", "", errors.New("insufficient permissions to manage this workspace")
			}
		}

		usingDatabase = database
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	var username, password string
	var err error

	switch usingDatabase.Type {
	case DatabaseTypePostgres:
		username, password, err = usingDatabase.Postgresql.CreateReadOnlyUser(
			ctx, s.logger, s.fieldEncryptor, usingDatabase.ID,
		)
	case DatabaseTypeMysql:
		username, password, err = usingDatabase.Mysql.CreateReadOnlyUser(
			ctx, s.logger, s.fieldEncryptor, usingDatabase.ID,
		)
	case DatabaseTypeMariadb:
		username, password, err = usingDatabase.Mariadb.CreateReadOnlyUser(
			ctx, s.logger, s.fieldEncryptor, usingDatabase.ID,
		)
	case DatabaseTypeMongodb:
		username, password, err = usingDatabase.Mongodb.CreateReadOnlyUser(
			ctx, s.logger, s.fieldEncryptor, usingDatabase.ID,
		)
	case DatabaseTypeClickhouse:
		username, password, err = usingDatabase.Clickhouse.CreateReadOnlyUser(
			ctx, s.logger, s.fieldEncryptor, usingDatabase.ID,
		)
	default:
		return "", "", errors.New("read-only user creation not supported for this database type")
	}

	if err != nil {
		return "", "", err
	}

	if usingDatabase.WorkspaceID != nil {
		s.auditLogService.WriteAuditLog(
			fmt.Sprintf(
				"Read-only user created for database: %s (username: %s)",
				usingDatabase.Name,
				username,
			),
			&user.ID,
			usingDatabase.WorkspaceID,
		)
	}

	return username, password, nil
}

func (s *DatabaseService) verifyReadOnlyUserIfNeeded(database *Database) error {
	if !config.GetEnv().IsCloud {
		return nil
	}

	if database.Postgresql != nil &&
		database.Postgresql.BackupType == postgresql.PostgresBackupTypeWalV1 {
		return nil
	}

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	isReadOnly, permissions, err := database.IsUserReadOnly(ctx, s.logger, s.fieldEncryptor)
	if err != nil {
		return fmt.Errorf("failed to verify user permissions: %w", err)
	}

	if !isReadOnly {
		return fmt.Errorf(
			"in cloud mode, only read-only database users are allowed (user has permissions: %v)",
			permissions,
		)
	}

	return nil
}

func hashAgentToken(token string) string {
	hash := sha256.Sum256([]byte(token))
	return fmt.Sprintf("%x", hash)
}
