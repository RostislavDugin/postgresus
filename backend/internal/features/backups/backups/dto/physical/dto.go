package backups_dto_physical

import (
	"time"

	"github.com/google/uuid"

	physical_enums "databasus-backend/internal/features/backups/backups/core/physical/enums"
)

// PhysicalBackupListItem is one row of the flat physical-backup list: a FULL, an
// incremental, or a committed WAL segment. The backend returns all three types
// in a single list sorted by createdAt; the frontend filters by Type. Type-only
// fields (chain links for incrementals, WalFilename for WAL) are pointers so an
// unrelated row omits them rather than emitting a misleading zero value.
type PhysicalBackupListItem struct {
	ID         uuid.UUID                         `json:"id"`
	Type       physical_enums.PhysicalBackupType `json:"type"`
	Status     string                            `json:"status"`
	TimelineID int                               `json:"timelineId"`

	StartLSN string `json:"startLsn"`
	StopLSN  string `json:"stopLsn"`

	// Chain links, incremental rows only.
	RootFullBackupID          *uuid.UUID `json:"rootFullBackupId,omitempty"`
	ParentIncrementalBackupID *uuid.UUID `json:"parentIncrementalBackupId,omitempty"`

	// WalFilename is the bare PG segment name, WAL rows only.
	WalFilename *string `json:"walFilename,omitempty"`

	SizeMb float64 `json:"sizeMb"`

	// FailMessage is the human-readable failure detail, ERROR / CHAIN_BROKEN rows only.
	FailMessage *string `json:"failMessage,omitempty"`

	CreatedAt   time.Time  `json:"createdAt"`
	CompletedAt *time.Time `json:"completedAt,omitzero"`
}

// GetPhysicalBackupsRequest pages and filters the flat backup list. Filters are
// server-side because pagination makes client-side type filtering unreliable (a
// page can be all one type). Types/Statuses are optional and repeatable - a row
// matches when its type (or status) is any of the given values, and the two
// dimensions combine with AND. BeforeDate keeps only backups created strictly
// before it.
type GetPhysicalBackupsRequest struct {
	Limit      int                                 `form:"limit"`
	Offset     int                                 `form:"offset"`
	Types      []physical_enums.PhysicalBackupType `form:"type"`
	Statuses   []string                            `form:"status"`
	BeforeDate *time.Time                          `form:"beforeDate"`
}

type GetPhysicalBackupsResponse struct {
	Backups      []PhysicalBackupListItem `json:"backups"`
	TotalUsageMb float64                  `json:"totalUsageMb"`

	// Total is the count of all backups (every type) for the database, for
	// driving pagination; Limit/Offset echo the page that was served.
	Total  int64 `json:"total"`
	Limit  int   `json:"limit"`
	Offset int   `json:"offset"`
}

// TriggerBackupType selects which backup the trigger endpoint requests.
type TriggerBackupType string

const (
	TriggerBackupTypeAuto        TriggerBackupType = "auto"
	TriggerBackupTypeFull        TriggerBackupType = "full"
	TriggerBackupTypeIncremental TriggerBackupType = "incremental"
)

type TriggerBackupRequest struct {
	Type TriggerBackupType `json:"type" binding:"required,oneof=auto full incremental"`
}

type GenerateRestoreTokenRequest struct {
	// TargetTime is the PITR target; omit for a restore to the latest available
	// point.
	TargetTime *time.Time `json:"targetTime"`
}

type GenerateRestoreTokenResponse struct {
	Token string `json:"token"`
	URL   string `json:"url"`
}
