package backups_services

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"databasus-backend/internal/features/databases"
	"databasus-backend/internal/features/databases/databases/postgresql"
)

func Test_TriggerBackupAndWait_WhenDatabaseIsAgentManaged_ReturnsAgentManagedError(t *testing.T) {
	agentManagedDatabase := &databases.Database{
		Type:       databases.DatabaseTypePostgres,
		Postgresql: &postgresql.PostgresqlDatabase{BackupType: postgresql.PostgresBackupTypeWalV1},
	}

	service := &BackupService{}

	backup, err := service.TriggerBackupAndWait(context.Background(), agentManagedDatabase, time.Second)

	require.ErrorIs(t, err, ErrAgentManagedBackup)
	require.Nil(t, backup)
}
