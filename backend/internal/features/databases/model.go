package databases

import (
	"context"
	"errors"
	"log/slog"
	"time"

	"github.com/google/uuid"

	"databasus-backend/internal/features/databases/databases/clickhouse"
	"databasus-backend/internal/features/databases/databases/mariadb"
	"databasus-backend/internal/features/databases/databases/mongodb"
	"databasus-backend/internal/features/databases/databases/mysql"
	"databasus-backend/internal/features/databases/databases/postgresql"
	"databasus-backend/internal/features/notifiers"
	"databasus-backend/internal/util/encryption"
)

type Database struct {
	ID uuid.UUID `json:"id" gorm:"column:id;primaryKey;type:uuid;default:gen_random_uuid()"`

	// WorkspaceID can be null when a database is created via restore operation
	// outside the context of any workspace
	WorkspaceID *uuid.UUID   `json:"workspaceId" gorm:"column:workspace_id;type:uuid"`
	Name        string       `json:"name"        gorm:"column:name;type:text;not null"`
	Type        DatabaseType `json:"type"        gorm:"column:type;type:text;not null"`

	Postgresql *postgresql.PostgresqlDatabase `json:"postgresql,omitzero" gorm:"foreignKey:DatabaseID"`
	Mysql      *mysql.MysqlDatabase           `json:"mysql,omitzero"      gorm:"foreignKey:DatabaseID"`
	Mariadb    *mariadb.MariadbDatabase       `json:"mariadb,omitzero"    gorm:"foreignKey:DatabaseID"`
	Mongodb    *mongodb.MongodbDatabase       `json:"mongodb,omitzero"    gorm:"foreignKey:DatabaseID"`
	Clickhouse *clickhouse.ClickhouseDatabase `json:"clickhouse,omitzero" gorm:"foreignKey:DatabaseID"`

	Notifiers []notifiers.Notifier `json:"notifiers" gorm:"many2many:database_notifiers;"`

	// these fields are not reliable, but
	// they are used for pretty UI
	LastBackupTime         *time.Time `json:"lastBackupTime,omitzero"          gorm:"column:last_backup_time;type:timestamp with time zone"`
	LastBackupErrorMessage *string    `json:"lastBackupErrorMessage,omitempty" gorm:"column:last_backup_error_message;type:text"`

	HealthStatus *HealthStatus `json:"healthStatus" gorm:"column:health_status;type:text;not null"`

	AgentToken            *string `json:"-"                     gorm:"column:agent_token;type:text"`
	IsAgentTokenGenerated bool    `json:"isAgentTokenGenerated" gorm:"column:is_agent_token_generated;not null;default:false"`
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
	case DatabaseTypeMongodb:
		if d.Mongodb == nil {
			return errors.New("mongodb database is required")
		}
		return d.Mongodb.Validate()
	case DatabaseTypeClickhouse:
		if d.Clickhouse == nil {
			return errors.New("clickhouse database is required")
		}
		return d.Clickhouse.Validate()
	default:
		return errors.New("invalid database type: " + string(d.Type))
	}
}

func (d *Database) ValidateUpdate(old, new Database) error {
	// Database type cannot be changed after creation — the entire backup
	// structure (storage files, schedulers, WAL hierarchy, etc.) is tied to
	// the type at creation time. Recreating that state automatically is
	// error-prone; it is safer for the user to create a new database and
	// remove the old one.
	if old.Type != new.Type {
		return errors.New("database type cannot be changed; create a new database instead")
	}

	if old.Type == DatabaseTypePostgres && old.Postgresql != nil && new.Postgresql != nil {
		if err := new.Postgresql.ValidateUpdate(old.Postgresql); err != nil {
			return err
		}
	}

	return nil
}

func (d *Database) TestConnection(
	logger *slog.Logger,
	encryptor encryption.FieldEncryptor,
) error {
	return d.getSpecificDatabase().TestConnection(logger, encryptor, d.ID)
}

func (d *Database) IsUserReadOnly(
	ctx context.Context,
	logger *slog.Logger,
	encryptor encryption.FieldEncryptor,
) (bool, []string, error) {
	switch d.Type {
	case DatabaseTypePostgres:
		return d.Postgresql.IsUserReadOnly(ctx, logger, encryptor, d.ID)
	case DatabaseTypeMysql:
		return d.Mysql.IsUserReadOnly(ctx, logger, encryptor, d.ID)
	case DatabaseTypeMariadb:
		return d.Mariadb.IsUserReadOnly(ctx, logger, encryptor, d.ID)
	case DatabaseTypeMongodb:
		return d.Mongodb.IsUserReadOnly(ctx, logger, encryptor, d.ID)
	case DatabaseTypeClickhouse:
		return d.Clickhouse.IsUserReadOnly(ctx, logger, encryptor, d.ID)
	default:
		return false, nil, errors.New("read-only check not supported for this database type")
	}
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
	if d.Mongodb != nil {
		return d.Mongodb.EncryptSensitiveFields(d.ID, encryptor)
	}
	if d.Clickhouse != nil {
		return d.Clickhouse.EncryptSensitiveFields(d.ID, encryptor)
	}
	return nil
}

func (d *Database) PopulateDbData(
	logger *slog.Logger,
	encryptor encryption.FieldEncryptor,
) error {
	if d.Postgresql != nil {
		return d.Postgresql.PopulateDbData(logger, encryptor, d.ID)
	}
	if d.Mysql != nil {
		return d.Mysql.PopulateDbData(logger, encryptor, d.ID)
	}
	if d.Mariadb != nil {
		return d.Mariadb.PopulateDbData(logger, encryptor, d.ID)
	}
	if d.Mongodb != nil {
		return d.Mongodb.PopulateDbData(logger, encryptor, d.ID)
	}
	if d.Clickhouse != nil {
		return d.Clickhouse.PopulateDbData(logger, encryptor, d.ID)
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
	case DatabaseTypeMongodb:
		if d.Mongodb != nil && incoming.Mongodb != nil {
			d.Mongodb.Update(incoming.Mongodb)
		}
	case DatabaseTypeClickhouse:
		if d.Clickhouse != nil && incoming.Clickhouse != nil {
			d.Clickhouse.Update(incoming.Clickhouse)
		}
	}
}

func (d *Database) IsAgentManagedBackup() bool {
	return d.Type == DatabaseTypePostgres &&
		d.Postgresql != nil &&
		d.Postgresql.BackupType == postgresql.PostgresBackupTypeWalV1
}

func (d *Database) getSpecificDatabase() DatabaseConnector {
	switch d.Type {
	case DatabaseTypePostgres:
		return d.Postgresql
	case DatabaseTypeMysql:
		return d.Mysql
	case DatabaseTypeMariadb:
		return d.Mariadb
	case DatabaseTypeMongodb:
		return d.Mongodb
	case DatabaseTypeClickhouse:
		return d.Clickhouse
	}

	panic("invalid database type: " + string(d.Type))
}
