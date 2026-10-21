package wal_health

import (
	"time"

	physical_enums "databasus-backend/internal/features/backups/backups/core/physical/enums"
	physical_models "databasus-backend/internal/features/backups/backups/core/physical/models"
)

// A RUNNING row whose heartbeat is past the takeover age may belong to a
// process that is gone, so only a fresh heartbeat counts as running.
func GetStreamerState(streamer *physical_models.PhysicalWalStreamer, now time.Time) StreamerState {
	if streamer == nil {
		return StreamerStateNotStarted
	}

	if streamer.Status == physical_enums.PhysicalWalStreamerStatusRunning &&
		now.Sub(streamer.LastHeartbeatAt) < StreamerHeartbeatStaleness {
		return StreamerStateRunning
	}

	return StreamerStateNotRunning
}

// Receiver restarts and whole-session starts both mean streaming was
// interrupted, so they add up.
func GetRestartsLastHour(
	recentSamples []*physical_models.PhysicalWalStreamSample,
	sessionStartsLastHour int,
	now time.Time,
) int {
	restarts := sessionStartsLastHour
	windowStart := now.Add(-frequentRestartsWindow)

	for _, sample := range recentSamples {
		if sample.SampledAt.After(windowStart) {
			restarts += sample.ReceiverRestarts
		}
	}

	return restarts
}
