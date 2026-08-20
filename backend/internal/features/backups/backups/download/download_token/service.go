package download_token

import (
	"context"
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

func (s *Service) Generate(ctx context.Context, backupID, userID uuid.UUID) (string, error) {
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

	s.logger.InfoContext(ctx, "generated download token", "backup_id", backupID, "user_id", userID)

	return token, nil
}

func (s *Service) ValidateAndConsume(
	ctx context.Context,
	token string,
) (*Token, error) {
	downloadToken, err := s.repository.FindByToken(ctx, token)
	if err != nil {
		return nil, err
	}

	if downloadToken == nil {
		return nil, errors.New("invalid token")
	}

	if downloadToken.Used {
		return nil, errors.New("token already used")
	}

	if time.Now().UTC().After(downloadToken.ExpiresAt) {
		return nil, errors.New("token expired")
	}

	if err := s.AcquireSlot(downloadToken.UserID); err != nil {
		return nil, err
	}

	downloadToken.Used = true
	if err := s.repository.Update(downloadToken); err != nil {
		s.logger.ErrorContext(ctx, "failed to mark token as used", "error", err)
	}

	s.logger.InfoContext(
		ctx,
		"download token validated and consumed",
		"backup_id",
		downloadToken.BackupID,
		"user_id",
		downloadToken.UserID,
	)

	return downloadToken, nil
}

func (s *Service) CleanExpiredTokens(ctx context.Context) (int64, error) {
	deletedCount, err := s.repository.DeleteExpired(ctx, time.Now().UTC())
	if err != nil {
		return 0, err
	}

	return deletedCount, nil
}
