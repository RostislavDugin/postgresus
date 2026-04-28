package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// +kubebuilder:validation:Enum=POSTGRES;MYSQL;MARIADB;MONGODB
type DatabaseType string

const (
	DatabaseTypePostgres DatabaseType = "POSTGRES"
	DatabaseTypeMysql    DatabaseType = "MYSQL"
	DatabaseTypeMariadb  DatabaseType = "MARIADB"
	DatabaseTypeMongodb  DatabaseType = "MONGODB"
)

// +kubebuilder:validation:Enum=HOURLY;DAILY;WEEKLY;MONTHLY;CRON
type IntervalType string

const (
	IntervalHourly  IntervalType = "HOURLY"
	IntervalDaily   IntervalType = "DAILY"
	IntervalWeekly  IntervalType = "WEEKLY"
	IntervalMonthly IntervalType = "MONTHLY"
	IntervalCron    IntervalType = "CRON"
)

// +kubebuilder:validation:Enum=TIME_PERIOD;COUNT;GFS
type RetentionPolicyType string

const (
	RetentionPolicyTypeTimePeriod RetentionPolicyType = "TIME_PERIOD"
	RetentionPolicyTypeCount     RetentionPolicyType = "COUNT"
	RetentionPolicyTypeGFS       RetentionPolicyType = "GFS"
)

// +kubebuilder:validation:Enum=NONE;ENCRYPTED
type BackupEncryption string

const (
	BackupEncryptionNone      BackupEncryption = "NONE"
	BackupEncryptionEncrypted BackupEncryption = "ENCRYPTED"
)

// +kubebuilder:validation:Enum=BACKUP_FAILED;BACKUP_SUCCESS
type BackupNotificationType string

const (
	NotificationBackupFailed  BackupNotificationType = "BACKUP_FAILED"
	NotificationBackupSuccess BackupNotificationType = "BACKUP_SUCCESS"
)

// DatabaseBackupSpec defines the desired state of a managed database backup.
type DatabaseBackupSpec struct {
	Database    DatabaseSpec    `json:"database"`
	Backup      BackupSpec      `json:"backup"`
	Healthcheck *HealthcheckSpec `json:"healthcheck,omitempty"`
}

// --- Database section ---

type DatabaseSpec struct {
	// +kubebuilder:validation:Required
	Name string `json:"name"`
	// +kubebuilder:validation:Required
	Type DatabaseType `json:"type"`

	Postgresql *PostgresqlDatabaseSpec `json:"postgresql,omitempty"`
	Mysql      *MysqlDatabaseSpec      `json:"mysql,omitempty"`
	Mariadb    *MariadbDatabaseSpec    `json:"mariadb,omitempty"`
	Mongodb    *MongodbDatabaseSpec    `json:"mongodb,omitempty"`

	// Names of Notifier CRDs in the same namespace to attach to this database.
	NotifierRefs []string `json:"notifierRefs,omitempty"`
}

type PostgresqlDatabaseSpec struct {
	Version           string       `json:"version"`
	Host              string       `json:"host"`
	Port              int          `json:"port"`
	Username          string       `json:"username"`
	PasswordSecretRef SecretKeyRef `json:"passwordSecretRef"`
	Database          string       `json:"database,omitempty"`
	// +kubebuilder:default=false
	IsHttps bool `json:"isHttps,omitempty"`
	// +kubebuilder:validation:Enum=PG_DUMP;WAL_V1
	// +kubebuilder:default=PG_DUMP
	BackupType     string   `json:"backupType,omitempty"`
	IncludeSchemas []string `json:"includeSchemas,omitempty"`
	// +kubebuilder:default=1
	CpuCount int `json:"cpuCount,omitempty"`
}

type MysqlDatabaseSpec struct {
	Version           string       `json:"version"`
	Host              string       `json:"host"`
	Port              int          `json:"port"`
	Username          string       `json:"username"`
	PasswordSecretRef SecretKeyRef `json:"passwordSecretRef"`
	Database          string       `json:"database,omitempty"`
	// +kubebuilder:default=false
	IsHttps bool `json:"isHttps,omitempty"`
}

type MariadbDatabaseSpec struct {
	Version           string       `json:"version"`
	Host              string       `json:"host"`
	Port              int          `json:"port"`
	Username          string       `json:"username"`
	PasswordSecretRef SecretKeyRef `json:"passwordSecretRef"`
	Database          string       `json:"database,omitempty"`
	// +kubebuilder:default=false
	IsHttps bool `json:"isHttps,omitempty"`
}

type MongodbDatabaseSpec struct {
	Version           string       `json:"version"`
	Host              string       `json:"host"`
	Port              *int         `json:"port,omitempty"`
	Username          string       `json:"username"`
	PasswordSecretRef SecretKeyRef `json:"passwordSecretRef"`
	Database          string       `json:"database"`
	AuthDatabase      string       `json:"authDatabase,omitempty"`
	// +kubebuilder:default=false
	IsHttps bool `json:"isHttps,omitempty"`
	// +kubebuilder:default=false
	IsSrv bool `json:"isSrv,omitempty"`
	// +kubebuilder:default=false
	IsDirectConnection bool `json:"isDirectConnection,omitempty"`
	// +kubebuilder:default=1
	CpuCount int `json:"cpuCount,omitempty"`
}

// --- Backup section ---

type BackupSpec struct {
	// +kubebuilder:default=true
	IsEnabled bool `json:"isEnabled"`

	Interval        BackupIntervalSpec  `json:"interval"`
	RetentionPolicy RetentionPolicySpec `json:"retentionPolicy"`

	// Name of the Storage CRD in the same namespace.
	StorageRef string `json:"storageRef"`

	SendNotificationsOn []BackupNotificationType `json:"sendNotificationsOn,omitempty"`

	// +kubebuilder:default=false
	IsRetryIfFailed     bool `json:"isRetryIfFailed,omitempty"`
	MaxFailedTriesCount int  `json:"maxFailedTriesCount,omitempty"`

	// +kubebuilder:validation:Enum=NONE;ENCRYPTED
	// +kubebuilder:default=NONE
	Encryption BackupEncryption `json:"encryption,omitempty"`
}

type BackupIntervalSpec struct {
	// +kubebuilder:validation:Required
	Type IntervalType `json:"type"`
	// Time in "HH:MM" format (required for DAILY, WEEKLY, MONTHLY).
	TimeOfDay *string `json:"timeOfDay,omitempty"`
	// Day of week 0-6, Sunday=0 (required for WEEKLY).
	Weekday *int `json:"weekday,omitempty"`
	// Day of month 1-31 (required for MONTHLY).
	DayOfMonth *int `json:"dayOfMonth,omitempty"`
	// 5-field cron expression (required for CRON).
	CronExpression *string `json:"cronExpression,omitempty"`
}

type RetentionPolicySpec struct {
	// +kubebuilder:validation:Required
	Type RetentionPolicyType `json:"type"`
	// Duration string e.g. "7d", "30d", "3m", "1y" (for TIME_PERIOD).
	TimePeriod string `json:"timePeriod,omitempty"`
	// Number of backups to keep (for COUNT).
	Count int `json:"count,omitempty"`

	// GFS retention fields.
	GfsHours  int `json:"gfsHours,omitempty"`
	GfsDays   int `json:"gfsDays,omitempty"`
	GfsWeeks  int `json:"gfsWeeks,omitempty"`
	GfsMonths int `json:"gfsMonths,omitempty"`
	GfsYears  int `json:"gfsYears,omitempty"`
}

// --- Healthcheck section ---

type HealthcheckSpec struct {
	// +kubebuilder:default=true
	IsEnabled bool `json:"isEnabled"`
	// +kubebuilder:default=false
	IsSentNotificationWhenUnavailable bool `json:"isSentNotificationWhenUnavailable,omitempty"`
	// +kubebuilder:validation:Minimum=1
	IntervalMinutes int `json:"intervalMinutes"`
	// +kubebuilder:validation:Minimum=1
	AttemptsBeforeConsideredAsDown int `json:"attemptsBeforeConsideredAsDown"`
	// +kubebuilder:validation:Minimum=1
	StoreAttemptsDays int `json:"storeAttemptsDays"`
}

// --- Status ---

type DatabaseBackupStatus struct {
	// IDs assigned by the databasus API.
	DatabaseID     string `json:"databaseId,omitempty"`
	BackupConfigID string `json:"backupConfigId,omitempty"`

	// +kubebuilder:validation:Enum=AVAILABLE;UNAVAILABLE;UNKNOWN;""
	HealthStatus string `json:"healthStatus,omitempty"`

	LastBackupTime         *metav1.Time `json:"lastBackupTime,omitempty"`
	LastBackupErrorMessage string       `json:"lastBackupErrorMessage,omitempty"`

	ObservedGeneration int64 `json:"observedGeneration,omitempty"`

	// +listType=map
	// +listMapKey=type
	Conditions []metav1.Condition `json:"conditions,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:printcolumn:name="DB Type",type=string,JSONPath=`.spec.database.type`
// +kubebuilder:printcolumn:name="Health",type=string,JSONPath=`.status.healthStatus`
// +kubebuilder:printcolumn:name="Ready",type=string,JSONPath=`.status.conditions[?(@.type=="Ready")].status`
// +kubebuilder:printcolumn:name="Last Backup",type=date,JSONPath=`.status.lastBackupTime`
// +kubebuilder:printcolumn:name="Age",type=date,JSONPath=`.metadata.creationTimestamp`

// DatabaseBackup is the Schema for the databasebackups API.
type DatabaseBackup struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   DatabaseBackupSpec   `json:"spec"`
	Status DatabaseBackupStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// DatabaseBackupList contains a list of DatabaseBackup.
type DatabaseBackupList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []DatabaseBackup `json:"items"`
}

func init() {
	SchemeBuilder.Register(&DatabaseBackup{}, &DatabaseBackupList{})
}
