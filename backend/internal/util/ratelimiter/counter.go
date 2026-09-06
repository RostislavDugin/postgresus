package ratelimiter

import (
	"context"
	"time"
)

type Attempt struct {
	Scope      string
	Identifier string
	Limit      int
	Window     time.Duration
}

type Counter interface {
	RecordAttemptAndCheckIsAllowed(ctx context.Context, attempt Attempt) (bool, error)
}
