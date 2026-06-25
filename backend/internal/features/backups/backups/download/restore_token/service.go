package restore_token

import (
	"errors"
	"log/slog"
	"time"

	"github.com/google/uuid"
	"github.com/valkey-io/valkey-go"

	"databasus-backend/internal/features/backups/backups/download/stream_guard"
)

// Service issues and consumes single-use tokens authorizing a physical restore
// stream. It embeds the SAME stream_guard.Guard as the download-token service
// (RefreshDownloadLock, ReleaseDownloadLock are promoted from it), so a user
// cannot run two heavy streams — download or restore — at once.
type Service struct {
	*stream_guard.Guard
	store  *store
	logger *slog.Logger
}

func NewService(guard *stream_guard.Guard, client valkey.Client, logger *slog.Logger) *Service {
	return &Service{
		guard,
		newStore(client),
		logger,
	}
}

// GenerateRestoreToken issues a single-use, short-TTL token authorizing a
// physical restore stream of databaseID to targetTime (nil ⇒ latest).
func (s *Service) GenerateRestoreToken(
	databaseID, userID uuid.UUID,
	targetTime *time.Time,
) (string, error) {
	if s.IsDownloadInProgress(userID) {
		return "", stream_guard.ErrDownloadAlreadyInProgress
	}

	token := stream_guard.GenerateSecureToken()

	s.store.issue(token, &Token{
		DatabaseID: databaseID,
		UserID:     userID,
		TargetTime: targetTime,
	})

	s.logger.Info("generated restore token", "database_id", databaseID, "user_id", userID)

	return token, nil
}

// GenerateBackupRestoreToken issues a single-use, short-TTL token authorizing a
// per-backup restore stream (the FULL plus its incremental ancestors, no WAL)
// of backupID within databaseID.
func (s *Service) GenerateBackupRestoreToken(
	databaseID, userID, backupID uuid.UUID,
) (string, error) {
	if s.IsDownloadInProgress(userID) {
		return "", stream_guard.ErrDownloadAlreadyInProgress
	}

	token := stream_guard.GenerateSecureToken()

	s.store.issue(token, &Token{
		DatabaseID: databaseID,
		UserID:     userID,
		BackupID:   &backupID,
	})

	s.logger.Info("generated backup restore token",
		"database_id", databaseID, "user_id", userID, "backup_id", backupID)

	return token, nil
}

// ValidateAndConsumeRestoreToken consumes a restore token, then acquires the
// per-user stream lock. The caller must release the lock when the stream ends.
//
// The token is consumed (GETDEL) BEFORE the slot is acquired, so it can be
// validated at most once even under concurrency. The trade-off: if slot
// acquisition then fails (a competing stream slipped in after the token was
// issued), the token is already spent and the user must request a new one. That
// race is narrow — GenerateRestoreToken already gates on IsDownloadInProgress —
// and burning the token beats a validate-then-consume window that two requests
// could both pass.
func (s *Service) ValidateAndConsumeRestoreToken(
	token string,
) (*Token, error) {
	restoreToken := s.store.consume(token)
	if restoreToken == nil {
		return nil, errors.New("invalid or expired restore token")
	}

	if err := s.AcquireSlot(restoreToken.UserID); err != nil {
		return nil, err
	}

	s.logger.Info("restore token validated and consumed",
		"database_id", restoreToken.DatabaseID, "user_id", restoreToken.UserID)

	return restoreToken, nil
}
