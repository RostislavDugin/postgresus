package storages

import "github.com/google/uuid"

type TransferStorageRequest struct {
	TargetWorkspaceID uuid.UUID `json:"targetWorkspaceId" binding:"required"`
}
