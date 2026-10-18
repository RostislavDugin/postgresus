package wal_health

import (
	"time"

	"databasus-backend/internal/features/backups/backups/core/physical/chain_view"
	physical_models "databasus-backend/internal/features/backups/backups/core/physical/models"
)

type VerdictInput struct {
	// The newest sample of the database, whatever its age; nil when none exists.
	NewestSample *physical_models.PhysicalWalStreamSample

	// The samples of the last hour, oldest first. The newest sample is the last
	// one whenever it is younger than an hour.
	RecentSamples []*physical_models.PhysicalWalStreamSample

	// Nil when no streamer has ever run for the database.
	Streamer *physical_models.PhysicalWalStreamer

	SessionStartsLastHour int

	SlotRebuildLagBytes int64

	IsStreamingExpected    bool
	HasCompletedFullBackup bool

	// The WAL gaps of the newest completed full backup's chain.
	NewestChainGaps []chain_view.LSNRange

	Now time.Time
}

type Reason struct {
	Code             ReasonCode  `json:"code"`
	Level            ReasonLevel `json:"level"`
	Since            *time.Time  `json:"since,omitempty"`
	ValueBytes       *int64      `json:"valueBytes,omitempty"`
	ThresholdBytes   *int64      `json:"thresholdBytes,omitempty"`
	Count            *int        `json:"count,omitempty"`
	EstimatedSeconds *int64      `json:"estimatedSeconds,omitempty"`
}

type Notice struct {
	Code  NoticeCode  `json:"code"`
	Level ReasonLevel `json:"level"`
	Since *time.Time  `json:"since,omitempty"`
}

type Verdict struct {
	Level   HealthLevel `json:"level"`
	Reasons []Reason    `json:"reasons"`
	Notices []Notice    `json:"notices"`
}

type KeepingUp struct {
	State KeepingUpState `json:"state"`

	SourceWriteBytesPerSecond *float64 `json:"sourceWriteBytesPerSecond"`
	ArchiveBytesPerSecond     *float64 `json:"archiveBytesPerSecond"`

	UnarchivedWalBytes                *int64   `json:"unarchivedWalBytes"`
	UnarchivedWalChangeBytesPerSecond *float64 `json:"unarchivedWalChangeBytesPerSecond"`

	SlotLagBytes                *int64   `json:"slotLagBytes"`
	SlotLagChangeBytesPerSecond *float64 `json:"slotLagChangeBytesPerSecond"`

	SlotRebuildLagBytes int64 `json:"slotRebuildLagBytes"`

	EstimatedCatchUpSeconds     *int64 `json:"estimatedCatchUpSeconds"`
	EstimatedSlotRebuildSeconds *int64 `json:"estimatedSlotRebuildSeconds"`
	EstimatedSlotLimitSeconds   *int64 `json:"estimatedSlotLimitSeconds"`
}
