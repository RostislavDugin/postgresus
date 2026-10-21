package backups_dto_physical

import (
	"time"

	physical_enums "databasus-backend/internal/features/backups/backups/core/physical/enums"
)

// The current values of a database's running WAL stream, read at the time of
// the request and never stored. Current is null while no streamer runs.
type GetWalStreamLiveResponse struct {
	IsStreamerRunning bool           `json:"isStreamerRunning"`
	Current           *WalStreamLive `json:"current"`
}

type WalStreamLive struct {
	MeasuredAt time.Time `json:"measuredAt"`

	// The newest probe of the source, made at least every 10 seconds; null until
	// the streaming session's first probe. The source-side values below are null
	// unless that probe found the source reachable with its slot.
	SourceProbedAt       *time.Time                                   `json:"sourceProbedAt"`
	SourceState          *physical_enums.PhysicalWalStreamSourceState `json:"sourceState"`
	SourceCurrentLsn     *string                                      `json:"sourceCurrentLsn"`
	SlotWalStatus        *string                                      `json:"slotWalStatus"`
	SlotLagBytes         *int64                                       `json:"slotLagBytes"`
	SlotSafeWalSizeBytes *int64                                       `json:"slotSafeWalSizeBytes"`

	ArchiveProgressLsn *string    `json:"archiveProgressLsn"`
	LastArchivedWalAt  *time.Time `json:"lastArchivedWalAt"`
	UnarchivedWalBytes *int64     `json:"unarchivedWalBytes"`

	LocalQueueBytes      int64 `json:"localQueueBytes"`
	LocalQueuePauseBytes int64 `json:"localQueuePauseBytes"`
	IsReceiverRunning    bool  `json:"isReceiverRunning"`
}
