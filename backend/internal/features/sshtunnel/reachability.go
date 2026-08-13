package sshtunnel

import (
	"context"
	"time"
)

// Asking must not change anything: a cached session is probed in place, and with no session this
// dials TCP only — no handshake, no authentication, nothing written back to the client cache. The
// caller is a supervision loop deciding whether a failed run was the bastion's fault, so it needs an
// answer inside its own deadline rather than the transport's 30s dial budget.
func (f *Forwarder) IsBastionReachable(ctx context.Context) bool {
	f.clientMutex.Lock()
	client := f.client
	f.clientMutex.Unlock()

	if client != nil {
		return f.isKeepaliveAnsweredWithin(budgetUntil(ctx), client)
	}

	bastionConn, err := f.dialBastionConn(ctx, f.bastionAddress)
	if err != nil {
		return false
	}

	_ = bastionConn.Close()

	return true
}

func budgetUntil(ctx context.Context) time.Duration {
	deadline, hasDeadline := ctx.Deadline()
	if !hasDeadline {
		return bastionDialTimeout
	}

	return max(time.Until(deadline), 0)
}
