package databases

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"time"

	audit_logs "postgresus-backend/internal/features/audit_logs"
	"postgresus-backend/internal/features/databases/databases/postgresql"
	"postgresus-backend/internal/features/notifiers"
	users_models "postgresus-backend/internal/features/users/models"
	workspaces_services "postgresus-backend/internal/features/workspaces/services"
	"postgresus-backend/internal/util/encryption"

	"github.com/google/uuid"
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

	if err := database.PopulateVersionIfEmpty(s.logger, s.fieldEncryptor); err != nil {
		return nil, fmt.Errorf("failed to auto-detect database version: %w", err)
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

	existingDatabase.Update(database)

	if err := existingDatabase.Validate(); err != nil {
		return err
	}

	if err := existingDatabase.PopulateVersionIfEmpty(s.logger, s.fieldEncryptor); err != nil {
		return fmt.Errorf("failed to auto-detect database version: %w", err)
	}

	if err := existingDatabase.EncryptSensitiveFields(s.fieldEncryptor); err != nil {
		return fmt.Errorf("failed to encrypt sensitive fields: %w", err)
	}

	_, err = s.dbRepository.Save(existingDatabase)
	if err != nil {
		return err
	}

	s.auditLogService.WriteAuditLog(
		fmt.Sprintf("Database updated: %s", existingDatabase.Name),
		&user.ID,
		existingDatabase.WorkspaceID,
	)

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
				ID:         uuid.Nil,
				DatabaseID: nil,
				Version:    existingDatabase.Postgresql.Version,
				Host:       existingDatabase.Postgresql.Host,
				Port:       existingDatabase.Postgresql.Port,
				Username:   existingDatabase.Postgresql.Username,
				Password:   existingDatabase.Postgresql.Password,
				Database:   existingDatabase.Postgresql.Database,
				IsHttps:    existingDatabase.Postgresql.IsHttps,
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
) (bool, error) {
	var usingDatabase *Database

	if database.ID != uuid.Nil {
		existingDatabase, err := s.dbRepository.FindByID(database.ID)
		if err != nil {
			return false, err
		}

		if existingDatabase.WorkspaceID == nil {
			return false, errors.New("cannot check user for database without workspace")
		}

		canAccess, _, err := s.workspaceService.CanUserAccessWorkspace(
			*existingDatabase.WorkspaceID,
			user,
		)
		if err != nil {
			return false, err
		}
		if !canAccess {
			return false, errors.New("insufficient permissions to access this database")
		}

		if database.WorkspaceID != nil && *existingDatabase.WorkspaceID != *database.WorkspaceID {
			return false, errors.New("database does not belong to this workspace")
		}

		existingDatabase.Update(database)

		if err := existingDatabase.Validate(); err != nil {
			return false, err
		}

		usingDatabase = existingDatabase
	} else {
		if database.WorkspaceID != nil {
			canAccess, _, err := s.workspaceService.CanUserAccessWorkspace(*database.WorkspaceID, user)
			if err != nil {
				return false, err
			}
			if !canAccess {
				return false, errors.New("insufficient permissions to access this workspace")
			}
		}

		usingDatabase = database
	}

	if usingDatabase.Type != DatabaseTypePostgres {
		return false, errors.New("read-only check only supported for PostgreSQL databases")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	return usingDatabase.Postgresql.IsUserReadOnly(
		ctx,
		s.logger,
		s.fieldEncryptor,
		usingDatabase.ID,
	)
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

	if usingDatabase.Type != DatabaseTypePostgres {
		return "", "", errors.New("read-only user creation only supported for PostgreSQL")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	username, password, err := usingDatabase.Postgresql.CreateReadOnlyUser(
		ctx, s.logger, s.fieldEncryptor, usingDatabase.ID,
	)
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
