package physical_models

import (
	"time"

	"github.com/google/uuid"

	physical_enums "databasus-backend/internal/features/backups/backups/core/physical/enums"
)

// Sidecar row for the long-running pg_receivewal supervisor. The PK is
// database_id (one streamer per database). Heartbeat > 90 s means the row is
// stale and a later supervisor tick reclaims it; heartbeat alone answers
// liveness, and runtime PG-slot semantics (active_pid != ours ⇒ refuse)
// prevent concurrent-claim races.
type PhysicalWalStreamer struct {
	DatabaseID      uuid.UUID                                `json:"databaseId"      gorm:"column:database_id;type:uuid;primaryKey"`
	StartedAt       time.Time                                `json:"startedAt"       gorm:"column:started_at"`
	LastHeartbeatAt time.Time                                `json:"lastHeartbeatAt" gorm:"column:last_heartbeat_at"`
	Status          physical_enums.PhysicalWalStreamerStatus `json:"status"          gorm:"column:status;type:text;not null"`
}

func (PhysicalWalStreamer) TableName() string {
	return "physical_wal_streamers"
}
