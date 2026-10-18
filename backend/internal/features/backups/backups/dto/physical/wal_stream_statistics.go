package backups_dto_physical

import (
	"time"

	physical_dto "databasus-backend/internal/features/backups/backups/core/physical/dto"
	physical_enums "databasus-backend/internal/features/backups/backups/core/physical/enums"
	"databasus-backend/internal/features/backups/backups/core/physical/wal_health"
)

type WalStreamStatisticsPeriod string

const (
	WalStreamStatisticsPeriodHour WalStreamStatisticsPeriod = "HOUR"
	WalStreamStatisticsPeriodDay  WalStreamStatisticsPeriod = "DAY"
	WalStreamStatisticsPeriodWeek WalStreamStatisticsPeriod = "WEEK"
)

// One-minute buckets for the hour: 30-second ones would hold zero or two samples
// depending on the sampling phase and draw false gaps and doubled rates. The day
// and the week get 360 buckets each, more than a chart can draw distinctly.
func (p WalStreamStatisticsPeriod) GetDurations() (period, bucket time.Duration, isKnown bool) {
	switch p {
	case WalStreamStatisticsPeriodHour:
		return time.Hour, time.Minute, true
	case WalStreamStatisticsPeriodDay:
		return 24 * time.Hour, 4 * time.Minute, true
	case WalStreamStatisticsPeriodWeek:
		return 7 * 24 * time.Hour, 28 * time.Minute, true
	default:
		return 0, 0, false
	}
}

type GetWalStreamStatisticsRequest struct {
	Period WalStreamStatisticsPeriod `form:"period" binding:"required"`
}

// The WAL stream statistics of one database. Every amount carries its unit in
// its name, every time is UTC, and the verdict, states, reasons and notices are
// codes, never display text.
type GetWalStreamStatisticsResponse struct {
	Health    WalStreamHealth      `json:"health"`
	Streamer  WalStreamStreamer    `json:"streamer"`
	Current   *WalStreamCurrent    `json:"current"`
	KeepingUp wal_health.KeepingUp `json:"keepingUp"`
	Series    WalStreamSeries      `json:"series"`
}

type WalStreamHealth struct {
	Level   wal_health.HealthLevel `json:"level"`
	Reasons []wal_health.Reason    `json:"reasons"`
	Notices []wal_health.Notice    `json:"notices"`

	// A full backup is requested or running, so a broken chain is already being
	// repaired.
	IsFullBackupPending bool `json:"isFullBackupPending"`
}

type WalStreamStreamer struct {
	State            wal_health.StreamerState `json:"state"`
	SessionStartedAt *time.Time               `json:"sessionStartedAt"`
	FailingSince     *time.Time               `json:"failingSince"`
	RestartsLastHour int                      `json:"restartsLastHour"`
}

// The newest sample. Source-side values are null while the source is
// unreachable or without the slot; on a reachable sample a null
// slotSafeWalSizeBytes means the source sets no limit.
type WalStreamCurrent struct {
	SampledAt         time.Time  `json:"sampledAt"`
	LastArchivedWalAt *time.Time `json:"lastArchivedWalAt"`
	SessionStartedAt  time.Time  `json:"sessionStartedAt"`

	SourceState            physical_enums.PhysicalWalStreamSourceState `json:"sourceState"`
	SourceUnreachableSince *time.Time                                  `json:"sourceUnreachableSince"`

	SourceCurrentLsn   *string `json:"sourceCurrentLsn"`
	ArchiveProgressLsn *string `json:"archiveProgressLsn"`
	SourceWrittenBytes *int64  `json:"sourceWrittenBytes"`

	ArchivedSegments         int    `json:"archivedSegments"`
	ArchivedWalBytes         int64  `json:"archivedWalBytes"`
	ArchivedStoredBytes      int64  `json:"archivedStoredBytes"`
	WalSegmentSizeBytes      int64  `json:"walSegmentSizeBytes"`
	HighestConcurrentUploads int    `json:"highestConcurrentUploads"`
	SegmentWriteAvgMillis    *int64 `json:"segmentWriteAvgMillis"`
	SegmentWriteMaxMillis    *int64 `json:"segmentWriteMaxMillis"`

	// pg_replication_slots.wal_status as the source reports it.
	SlotWalStatus        *string    `json:"slotWalStatus"`
	SlotExtraWalSince    *time.Time `json:"slotExtraWalSince"`
	SlotLagBytes         *int64     `json:"slotLagBytes"`
	SlotSafeWalSizeBytes *int64     `json:"slotSafeWalSizeBytes"`

	UnarchivedWalBytes  *int64     `json:"unarchivedWalBytes"`
	ArchiveStalledSince *time.Time `json:"archiveStalledSince"`

	LocalQueueBytes      int64 `json:"localQueueBytes"`
	LocalQueuePauseBytes int64 `json:"localQueuePauseBytes"`

	ReceiverRestarts       int  `json:"receiverRestarts"`
	ReceiverPauses         int  `json:"receiverPauses"`
	IsReceiverRunning      bool `json:"isReceiverRunning"`
	IsForcedRotationDenied bool `json:"isForcedRotationDenied"`
}

// Exactly period ÷ bucketSeconds points, oldest first; the last one holds the
// moment of the request. A point without samples has every value null.
type WalStreamSeries struct {
	Period        WalStreamStatisticsPeriod                    `json:"period"`
	BucketSeconds int64                                        `json:"bucketSeconds"`
	From          time.Time                                    `json:"from"`
	To            time.Time                                    `json:"to"`
	Points        []*physical_dto.PhysicalWalStreamSeriesPoint `json:"points"`
}
