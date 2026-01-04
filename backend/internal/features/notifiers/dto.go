package notifiers

import "github.com/google/uuid"

type TransferNotifierRequest struct {
	TargetWorkspaceID uuid.UUID `json:"targetWorkspaceId" binding:"required"`
}
