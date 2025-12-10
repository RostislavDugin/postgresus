package clusters

import (
	"errors"
	"postgresus-backend/internal/features/intervals"
	"postgresus-backend/internal/features/notifiers"
	"postgresus-backend/internal/util/period"
	"postgresus-backend/internal/util/tools"
	"time"

	"github.com/google/uuid"
)

type Cluster struct {
	ID uuid.UUID `json:"id" gorm:"column:id;primaryKey;type:uuid;default:gen_random_uuid()"`

	WorkspaceID *uuid.UUID `json:"workspaceId" gorm:"column:workspace_id;type:uuid;not null"`
	Name        string     `json:"name"        gorm:"column:name;type:text;not null"`

	// PostgreSQL connection settings
	Postgresql PostgresqlCluster `json:"postgresql" gorm:"foreignKey:ClusterID"`

	// Default backup settings applied to newly discovered databases during cluster backups
	IsBackupsEnabled    bool                `json:"isBackupsEnabled"   gorm:"column:is_backups_enabled;type:boolean;not null;default:false"`
	StorePeriod         period.Period       `json:"storePeriod"        gorm:"column:store_period;type:text;not null"`
	BackupIntervalID    *uuid.UUID          `json:"backupIntervalId"   gorm:"column:backup_interval_id;type:uuid"`
	BackupInterval      *intervals.Interval `json:"backupInterval,omitempty" gorm:"foreignKey:BackupIntervalID"`
	StorageID           *uuid.UUID          `json:"storageId"          gorm:"column:storage_id;type:uuid"`
	SendNotificationsOn string              `json:"sendNotificationsOn" gorm:"column:send_notifications_on;type:text;not null"`
	CpuCount            int                 `json:"cpuCount"           gorm:"column:cpu_count;type:int;not null;default:1"`

	LastRunAt *time.Time `json:"lastRunAt" gorm:"column:last_run_at"`

	Notifiers         []notifiers.Notifier      `json:"notifiers" gorm:"many2many:cluster_notifiers;"`
	ExcludedDatabases []ClusterExcludedDatabase `json:"excludedDatabases" gorm:"foreignKey:ClusterID"`
}

func (c *Cluster) TableName() string { return "clusters" }

func (c *Cluster) ValidateBasic() error {
	if c.WorkspaceID == nil {
		return errors.New("workspaceId is required")
	}
	if c.Name == "" {
		return errors.New("name is required")
	}
	return c.Postgresql.Validate()
}

func (c *Cluster) HideSensitiveData() {
	c.Postgresql.HideSensitiveData()
}

type PostgresqlCluster struct {
	ID        uuid.UUID `json:"id" gorm:"primaryKey;type:uuid;default:gen_random_uuid()"`
	ClusterID uuid.UUID `json:"clusterId" gorm:"type:uuid;column:cluster_id;not null"`

	Version  tools.PostgresqlVersion `json:"version" gorm:"type:text;not null"`
	Host     string                  `json:"host"    gorm:"type:text;not null"`
	Port     int                     `json:"port"    gorm:"type:int;not null"`
	Username string                  `json:"username" gorm:"type:text;not null"`
	Password string                  `json:"password" gorm:"type:text;not null"`
	IsHttps  bool                    `json:"isHttps"  gorm:"type:boolean;default:false"`
}

func (p *PostgresqlCluster) TableName() string { return "postgresql_clusters" }

func (p *PostgresqlCluster) Validate() error {
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

func (p *PostgresqlCluster) HideSensitiveData() { p.Password = "" }

type ClusterExcludedDatabase struct {
	ID        uuid.UUID `json:"id" gorm:"primaryKey;type:uuid;default:gen_random_uuid()"`
	ClusterID uuid.UUID `json:"clusterId" gorm:"type:uuid;column:cluster_id;not null"`
	Name      string    `json:"name" gorm:"type:text;not null"`
}

func (c *ClusterExcludedDatabase) TableName() string { return "cluster_excluded_databases" }

type PropagationOptions struct {
	ApplyStorage       bool `json:"applyStorage"`
	ApplySchedule      bool `json:"applySchedule"`
	ApplyEnableBackups bool `json:"applyEnableBackups"`
	RespectExclusions  bool `json:"respectExclusions"`
}

type PropagationChange struct {
	DatabaseID     uuid.UUID `json:"databaseId"`
	Name           string    `json:"name"`
	ChangeStorage  bool      `json:"changeStorage"`
	ChangeSchedule bool      `json:"changeSchedule"`
	ChangeEnabled  bool      `json:"changeEnabled"`
}
