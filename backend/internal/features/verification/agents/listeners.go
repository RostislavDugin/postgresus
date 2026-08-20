package verification_agents

import (
	"context"

	"github.com/google/uuid"
)

type AgentHeartbeatedListener interface {
	OnAgentHeartbeated(ctx context.Context, agent *Agent, currentVerificationIDs []uuid.UUID) ([]uuid.UUID, error)
}
