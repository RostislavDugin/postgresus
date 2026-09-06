package stream_guard

import (
	"context"
	"log/slog"

	"github.com/google/uuid"
)

// Guard enforces the per-user single-stream rule shared by downloads and
// restores: at most one heavy stream per user at a time. DownloadTokenService
// and RestoreTokenService embed the SAME guard, so the lock namespace is shared
// across both — a user can't run a download and a restore at once.
type Guard struct {
	tracker *Tracker
	logger  *slog.Logger
}

func NewGuard(
	tracker *Tracker,
	logger *slog.Logger,
) *Guard {
	return &Guard{tracker, logger}
}

func (g *Guard) IsDownloadInProgress(ctx context.Context, userID uuid.UUID) bool {
	isInProgress, err := g.tracker.IsDownloadInProgress(ctx, userID)
	if err != nil {
		g.logger.ErrorContext(ctx, "failed to read stream lock", "error", err)

		return true
	}

	return isInProgress
}

func (g *Guard) RefreshDownloadLock(ctx context.Context, userID uuid.UUID) {
	if err := g.tracker.RefreshDownloadLock(ctx, userID); err != nil {
		g.logger.ErrorContext(ctx, "failed to refresh stream lock", "error", err)
	}
}

func (g *Guard) ReleaseDownloadLock(ctx context.Context, userID uuid.UUID) {
	if err := g.tracker.ReleaseDownloadLock(ctx, userID); err != nil {
		g.logger.ErrorContext(ctx, "failed to release stream lock", "error", err)

		return
	}

	g.logger.InfoContext(ctx, "released stream lock", "user_id", userID)
}

func (g *Guard) AcquireSlot(ctx context.Context, userID uuid.UUID) error {
	return g.tracker.AcquireDownloadLock(ctx, userID)
}
