package usecases_physical_postgresql

import "sync/atomic"

// A pause for a full local queue is Databasus working as designed, so it is
// counted apart from a restart and never looks like a crash.
type receiverRespawnCounter struct {
	restarts atomic.Int64
	pauses   atomic.Int64
}

func (c *receiverRespawnCounter) recordRespawn(isAfterBackpressurePause bool) {
	if isAfterBackpressurePause {
		c.pauses.Add(1)

		return
	}

	c.restarts.Add(1)
}

func (c *receiverRespawnCounter) takeCounts() (restarts, pauses int) {
	return int(c.restarts.Swap(0)), int(c.pauses.Swap(0))
}
