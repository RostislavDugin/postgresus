package usecases_physical_postgresql

import (
	"testing"

	"github.com/stretchr/testify/require"

	"databasus-backend/internal/util/logger"
)

func Test_WalStream_BackpressureWatermarks_ScaleWithWalSegmentSize(t *testing.T) {
	fixture := SetupPhysicalDBForBackup(t)
	customSegSize := int64(512 * 1024 * 1024)
	fixture.DB.PostgresqlPhysical.WalSegmentSizeBytes = &customSegSize

	supervisor := NewWalStreamSupervisor(WalStreamSpec{
		DatabaseID:   fixture.DB.ID,
		SourceDB:     fixture.DB.PostgresqlPhysical,
		WatchDirRoot: t.TempDir(),
		Logger:       logger.GetLogger(),
	})

	require.Equal(t, 8*customSegSize, supervisor.highWatermarkBytes)
	require.Equal(t, 8*customSegSize/5, supervisor.lowWatermarkBytes)
}
