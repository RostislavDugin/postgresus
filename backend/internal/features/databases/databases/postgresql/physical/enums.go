package postgresql_physical

type BackupType string

const (
	BackupTypeFullOnly                    BackupType = "FULL"
	BackupTypeFullAndIncremental          BackupType = "FULL_INCREMENTAL"
	BackupTypeFullIncrementalAndWalStream BackupType = "FULL_INCREMENTAL_WAL_STREAM"
)

func (t BackupType) IsRequireWalSummary() bool {
	return t == BackupTypeFullAndIncremental || t == BackupTypeFullIncrementalAndWalStream
}

// platform is the detected runtime classification of the source cluster.
// Used to choose the right replication-grant SQL and to format actionable
// fix messages — managed PG can't ALTER SYSTEM, so we tell the user where
// to set the parameter instead.
type platform string

const (
	platformSelfManaged    platform = "self_managed"
	platformRds            platform = "rds"
	platformAzure          platform = "azure"
	platformGcp            platform = "gcp"
	platformUnknownManaged platform = "unknown_managed"
)
