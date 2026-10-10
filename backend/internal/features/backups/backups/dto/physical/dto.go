package backups_dto_physical

import (
	"time"

	"github.com/google/uuid"

	physical_enums "databasus-backend/internal/features/backups/backups/core/physical/enums"
	"databasus-backend/internal/features/intervals"
)

// PhysicalBackupListItem is one row of the flat physical-backup list: a FULL, an
// incremental, or a WAL range of consecutive committed segments. Type-only
// fields (chain links for incrementals, the range summary for WAL) are pointers
// so an unrelated row omits them rather than emitting a misleading zero value.
// A WAL range's CreatedAt and CompletedAt are the receive times of its first and
// last segments.
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

	// WAL range summary, WAL rows only. HasGapBefore is true when an older
	// committed segment exists on the same timeline and the range does not start
	// where it ended.
	FromWalFilename  *string `json:"fromWalFilename,omitempty"`
	ToWalFilename    *string `json:"toWalFilename,omitempty"`
	WalSegmentsCount *int    `json:"walSegmentsCount,omitempty"`
	HasGapBefore     bool    `json:"hasGapBefore,omitzero"`

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
// before it. Cursor is the NextCursor of the previous page; omit it for the
// first page.
type GetPhysicalBackupsRequest struct {
	Limit      int                                 `form:"limit"`
	Cursor     string                              `form:"cursor"`
	Types      []physical_enums.PhysicalBackupType `form:"type"`
	Statuses   []string                            `form:"status"`
	BeforeDate *time.Time                          `form:"beforeDate"`
}

type GetPhysicalBackupsResponse struct {
	Backups      []PhysicalBackupListItem `json:"backups"`
	TotalUsageMb float64                  `json:"totalUsageMb"`

	// NextCursor continues the list after this page; empty on the last page.
	NextCursor string `json:"nextCursor,omitzero"`
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

type GetNextScheduledBackupsResponse struct {
	NextFullBackup        *intervals.NextTrigger `json:"nextFullBackup"`
	NextIncrementalBackup *intervals.NextTrigger `json:"nextIncrementalBackup"`
}
