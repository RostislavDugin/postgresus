package clusters

import (
	"log/slog"
	"time"

	"github.com/google/uuid"
)

// ClusterBackgroundService periodically checks clusters and triggers RunBackup on their intervals.
type ClusterBackgroundService struct {
	service *ClusterService
	repo    *ClusterRepository
	logger  *slog.Logger
}

func (s *ClusterBackgroundService) Run() {
	for {
		if err := s.tick(); err != nil {
			s.logger.Error("Cluster scheduler tick failed", "error", err)
		}
		time.Sleep(1 * time.Minute)
	}
}

func (s *ClusterBackgroundService) tick() error {
	clusters, err := s.repo.FindAll()
	if err != nil {
		return err
	}

	now := time.Now().UTC()
	for _, c := range clusters {
		// require interval and backups enabled
		if !c.IsBackupsEnabled || c.BackupInterval == nil {
			continue
		}

		var last *time.Time
		if c.LastRunAt != nil && !c.LastRunAt.IsZero() {
			last = c.LastRunAt
		}

		if c.BackupInterval.ShouldTriggerBackup(now, last) {
			if err := s.service.RunBackupScheduled(c.ID); err != nil {
				s.logger.Error("Failed to run cluster backup", "clusterId", c.ID, "error", err)
			}
			// Update last run regardless of outcome to avoid tight loop
			_ = s.repo.UpdateLastRunAt(c.ID, now)
		}
	}

	return nil
}

// For tests or manual invocations
func (s *ClusterBackgroundService) RunOnceFor(clusterID uuid.UUID) error {
	return s.service.RunBackupScheduled(clusterID)
}
