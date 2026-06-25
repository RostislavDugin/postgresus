package download_token

import (
	"errors"
	"log/slog"
	"time"

	"github.com/google/uuid"

	"databasus-backend/internal/features/backups/backups/download/stream_guard"
)

// Service issues and consumes single-use tokens for logical backup file
// downloads. It embeds stream_guard.Guard for the shared per-user single-stream
// lock (RefreshDownloadLock, ReleaseDownloadLock, IsDownloadInProgress are
// promoted from it).
type Service struct {
	*stream_guard.Guard
	repository *Repository
	logger     *slog.Logger
}

func NewService(guard *stream_guard.Guard, logger *slog.Logger) *Service {
	return &Service{
		guard,
		&Repository{},
		logger,
	}
}

func (s *Service) Generate(backupID, userID uuid.UUID) (string, error) {
	if s.IsDownloadInProgress(userID) {
		return "", stream_guard.ErrDownloadAlreadyInProgress
	}

	token := stream_guard.GenerateSecureToken()

	downloadToken := &Token{
		Token:     token,
		BackupID:  backupID,
		UserID:    userID,
		ExpiresAt: time.Now().UTC().Add(5 * time.Minute),
		Used:      false,
	}

	if err := s.repository.Create(downloadToken); err != nil {
		return "", err
	}

	s.logger.Info("Generated download token", "backupId", backupID, "userId", userID)
	return token, nil
}

func (s *Service) ValidateAndConsume(
	token string,
) (*Token, error) {
	dt, err := s.repository.FindByToken(token)
	if err != nil {
		return nil, err
	}

	if dt == nil {
		return nil, errors.New("invalid token")
	}

	if dt.Used {
		return nil, errors.New("token already used")
	}

	if time.Now().UTC().After(dt.ExpiresAt) {
		return nil, errors.New("token expired")
	}

	if err := s.AcquireSlot(dt.UserID); err != nil {
		return nil, err
	}

	dt.Used = true
	if err := s.repository.Update(dt); err != nil {
		s.logger.Error("Failed to mark token as used", "error", err)
	}

	s.logger.Info("Token validated and consumed", "backupId", dt.BackupID, "userId", dt.UserID)
	return dt, nil
}

func (s *Service) CleanExpiredTokens() error {
	if err := s.repository.DeleteExpired(time.Now().UTC()); err != nil {
		return err
	}

	s.logger.Debug("Cleaned expired download tokens")
	return nil
}
