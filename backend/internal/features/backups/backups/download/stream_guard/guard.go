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

func (g *Guard) IsDownloadInProgress(userID uuid.UUID) bool {
	return g.tracker.IsDownloadInProgress(userID)
}

func (g *Guard) RefreshDownloadLock(ctx context.Context, userID uuid.UUID) {
	g.tracker.RefreshDownloadLock(ctx, userID)
}

func (g *Guard) ReleaseDownloadLock(ctx context.Context, userID uuid.UUID) {
	g.tracker.ReleaseDownloadLock(ctx, userID)
	g.logger.InfoContext(ctx, "released stream lock", "user_id", userID)
}

// AcquireSlot takes the per-user single-stream lock.
func (g *Guard) AcquireSlot(userID uuid.UUID) error {
	return g.tracker.AcquireDownloadLock(userID)
}
