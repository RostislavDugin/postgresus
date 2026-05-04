package system_healthcheck

import (
	"context"
	"errors"
	"time"

	"databasus-backend/internal/config"
	"databasus-backend/internal/features/backups/backups/backuping"
	"databasus-backend/internal/features/disk"
	"databasus-backend/internal/storage"
	cache_utils "databasus-backend/internal/util/cache"
	"databasus-backend/internal/util/tools"
)

type HealthcheckService struct {
	diskService             *disk.DiskService
	backupBackgroundService *backuping.BackupsScheduler
	backuperNode            *backuping.BackuperNode
}

func (s *HealthcheckService) IsHealthy() error {
	return s.performHealthCheck()
}

func (s *HealthcheckService) performHealthCheck() error {
	// Check if cache is available with PING
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	client := cache_utils.GetValkeyClient()
	pingResult := client.Do(ctx, client.B().Ping().Build())
	if pingResult.Error() != nil {
		return errors.New("cannot connect to valkey")
	}

	diskUsage, err := s.diskService.GetDiskUsage()
	if err != nil {
		return errors.New("cannot get disk usage")
	}

	if float64(diskUsage.UsedSpaceBytes) >= float64(diskUsage.TotalSpaceBytes)*0.95 {
		return errors.New("more than 95% of the disk is used")
	}

	if err := tools.ClientToolsHealthError(); err != nil {
		return err
	}

	db := storage.GetDb()
	err = db.Raw("SELECT 1").Error
	if err != nil {
		return errors.New("cannot connect to the database")
	}

	if config.GetEnv().IsPrimaryNode {
		if !s.backupBackgroundService.IsSchedulerRunning() {
			return errors.New("backups are not running for more than 5 minutes")
		}

		if !s.backupBackgroundService.IsBackupNodesAvailable() {
			return errors.New("no backup nodes available")
		}
	}

	if config.GetEnv().IsProcessingNode {
		if !s.backuperNode.IsBackuperRunning() {
			return errors.New("backuper node is not running for more than 5 minutes")
		}
	}

	return nil
}
