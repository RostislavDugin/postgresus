package restores_core

import (
	"time"

	"github.com/google/uuid"

	backups_core_logical "databasus-backend/internal/features/backups/backups/core/logical"
	"databasus-backend/internal/features/databases/databases/mariadb"
	"databasus-backend/internal/features/databases/databases/mongodb"
	"databasus-backend/internal/features/databases/databases/mysql"
	postgresql_logical "databasus-backend/internal/features/databases/databases/postgresql/logical"
)

type Restore struct {
	ID     uuid.UUID     `json:"id"     gorm:"column:id;type:uuid;primaryKey"`
	Status RestoreStatus `json:"status" gorm:"column:status;type:text;not null"`

	BackupID uuid.UUID `json:"backupId" gorm:"column:backup_id;type:uuid;not null"`
	Backup   *backups_core_logical.LogicalBackup

	PostgresqlLogicalDatabase *postgresql_logical.PostgresqlLogicalDatabase `json:"postgresqlDatabase" gorm:"-"`
	MysqlDatabase             *mysql.MysqlDatabase                          `json:"mysqlDatabase"      gorm:"-"`
	MariadbDatabase           *mariadb.MariadbDatabase                      `json:"mariadbDatabase"    gorm:"-"`
	MongodbDatabase           *mongodb.MongodbDatabase                      `json:"mongodbDatabase"    gorm:"-"`

	FailMessage *string `json:"failMessage" gorm:"column:fail_message"`

	RestoreDurationMs int64     `json:"restoreDurationMs" gorm:"column:restore_duration_ms;default:0"`
	CreatedAt         time.Time `json:"createdAt"         gorm:"column:created_at;default:now()"`
}
