package databases

import (
	"errors"
	"log/slog"
	"postgresus-backend/internal/features/databases/databases/mariadb"
	"postgresus-backend/internal/features/databases/databases/mysql"
	"postgresus-backend/internal/features/databases/databases/postgresql"
	"postgresus-backend/internal/features/notifiers"
	"postgresus-backend/internal/util/encryption"
	"time"

	"github.com/google/uuid"
)

type Database struct {
	ID uuid.UUID `json:"id" gorm:"column:id;primaryKey;type:uuid;default:gen_random_uuid()"`

	// WorkspaceID can be null when a database is created via restore operation
	// outside the context of any workspace
	WorkspaceID *uuid.UUID   `json:"workspaceId" gorm:"column:workspace_id;type:uuid"`
	Name        string       `json:"name"        gorm:"column:name;type:text;not null"`
	Type        DatabaseType `json:"type"        gorm:"column:type;type:text;not null"`

	Postgresql *postgresql.PostgresqlDatabase `json:"postgresql,omitempty" gorm:"foreignKey:DatabaseID"`
	Mysql      *mysql.MysqlDatabase           `json:"mysql,omitempty"      gorm:"foreignKey:DatabaseID"`
	Mariadb    *mariadb.MariadbDatabase       `json:"mariadb,omitempty"    gorm:"foreignKey:DatabaseID"`

	Notifiers []notifiers.Notifier `json:"notifiers" gorm:"many2many:database_notifiers;"`

	// these fields are not reliable, but
	// they are used for pretty UI
	LastBackupTime         *time.Time `json:"lastBackupTime,omitempty"         gorm:"column:last_backup_time;type:timestamp with time zone"`
	LastBackupErrorMessage *string    `json:"lastBackupErrorMessage,omitempty" gorm:"column:last_backup_error_message;type:text"`

	HealthStatus *HealthStatus `json:"healthStatus" gorm:"column:health_status;type:text;not null"`
}

func (d *Database) Validate() error {
	if d.Name == "" {
		return errors.New("name is required")
	}

	switch d.Type {
	case DatabaseTypePostgres:
		if d.Postgresql == nil {
			return errors.New("postgresql database is required")
		}
		return d.Postgresql.Validate()
	case DatabaseTypeMysql:
		if d.Mysql == nil {
			return errors.New("mysql database is required")
		}
		return d.Mysql.Validate()
	case DatabaseTypeMariadb:
		if d.Mariadb == nil {
			return errors.New("mariadb database is required")
		}
		return d.Mariadb.Validate()
	default:
		return errors.New("invalid database type: " + string(d.Type))
	}
}

func (d *Database) ValidateUpdate(old, new Database) error {
	if old.Type != new.Type {
		return errors.New("database type is not allowed to change")
	}

	return nil
}

func (d *Database) TestConnection(
	logger *slog.Logger,
	encryptor encryption.FieldEncryptor,
) error {
	return d.getSpecificDatabase().TestConnection(logger, encryptor, d.ID)
}

func (d *Database) HideSensitiveData() {
	d.getSpecificDatabase().HideSensitiveData()
}

func (d *Database) EncryptSensitiveFields(encryptor encryption.FieldEncryptor) error {
	if d.Postgresql != nil {
		return d.Postgresql.EncryptSensitiveFields(d.ID, encryptor)
	}
	if d.Mysql != nil {
		return d.Mysql.EncryptSensitiveFields(d.ID, encryptor)
	}
	if d.Mariadb != nil {
		return d.Mariadb.EncryptSensitiveFields(d.ID, encryptor)
	}
	return nil
}

func (d *Database) PopulateVersionIfEmpty(
	logger *slog.Logger,
	encryptor encryption.FieldEncryptor,
) error {
	if d.Postgresql != nil {
		return d.Postgresql.PopulateVersionIfEmpty(logger, encryptor, d.ID)
	}
	if d.Mysql != nil {
		return d.Mysql.PopulateVersionIfEmpty(logger, encryptor, d.ID)
	}
	if d.Mariadb != nil {
		return d.Mariadb.PopulateVersionIfEmpty(logger, encryptor, d.ID)
	}
	return nil
}

func (d *Database) Update(incoming *Database) {
	d.Name = incoming.Name
	d.Type = incoming.Type
	d.Notifiers = incoming.Notifiers

	switch d.Type {
	case DatabaseTypePostgres:
		if d.Postgresql != nil && incoming.Postgresql != nil {
			d.Postgresql.Update(incoming.Postgresql)
		}
	case DatabaseTypeMysql:
		if d.Mysql != nil && incoming.Mysql != nil {
			d.Mysql.Update(incoming.Mysql)
		}
	case DatabaseTypeMariadb:
		if d.Mariadb != nil && incoming.Mariadb != nil {
			d.Mariadb.Update(incoming.Mariadb)
		}
	}
}

func (d *Database) getSpecificDatabase() DatabaseConnector {
	switch d.Type {
	case DatabaseTypePostgres:
		return d.Postgresql
	case DatabaseTypeMysql:
		return d.Mysql
	case DatabaseTypeMariadb:
		return d.Mariadb
	}

	panic("invalid database type: " + string(d.Type))
}
