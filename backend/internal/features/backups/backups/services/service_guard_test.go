package backups_services

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"databasus-backend/internal/features/databases"
)

func Test_TriggerBackupAndWait_WhenDatabaseIsAgentManaged_ReturnsAgentManagedError(t *testing.T) {
	agentManagedDatabase := &databases.Database{
		Type: databases.DatabaseTypePostgresPhysical,
	}

	service := &LogicalBackupService{}

	backup, err := service.TriggerBackupAndWait(context.Background(), agentManagedDatabase, time.Second)

	require.ErrorIs(t, err, ErrAgentManagedBackup)
	require.Nil(t, backup)
}
