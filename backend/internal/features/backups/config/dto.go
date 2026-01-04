package backups_config

import "github.com/google/uuid"

type TransferDatabaseRequest struct {
	TargetWorkspaceID       uuid.UUID   `json:"targetWorkspaceId"                 binding:"required"`
	TargetStorageID         *uuid.UUID  `json:"targetStorageId,omitempty"`
	IsTransferWithStorage   bool        `json:"isTransferWithStorage,omitempty"`
	IsTransferWithNotifiers bool        `json:"isTransferWithNotifiers,omitempty"`
	TargetNotifierIDs       []uuid.UUID `json:"targetNotifierIds,omitempty"`
}
