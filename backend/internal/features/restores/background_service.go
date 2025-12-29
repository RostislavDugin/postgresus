package restores

import (
	"databasus-backend/internal/features/restores/enums"
	"log/slog"
)

type RestoreBackgroundService struct {
	restoreRepository *RestoreRepository
	logger            *slog.Logger
}

func (s *RestoreBackgroundService) Run() {
	if err := s.failRestoresInProgress(); err != nil {
		s.logger.Error("Failed to fail restores in progress", "error", err)
		panic(err)
	}
}

func (s *RestoreBackgroundService) failRestoresInProgress() error {
	restoresInProgress, err := s.restoreRepository.FindByStatus(enums.RestoreStatusInProgress)
	if err != nil {
		return err
	}

	for _, restore := range restoresInProgress {
		failMessage := "Restore failed due to application restart"
		restore.Status = enums.RestoreStatusFailed
		restore.FailMessage = &failMessage

		if err := s.restoreRepository.Save(restore); err != nil {
			return err
		}
	}

	return nil
}
