package backuping_physical

import (
	"sync/atomic"
	"time"
)

// atomicTime is a race-free holder for a time.Time that one background loop
// writes (on every tick) and a health-check goroutine reads concurrently. A
// bare time.Time field is a multi-word struct, so the unsynchronized
// write/read of lastTickTime is a data race; storing the value as unix-nanos in
// an atomic.Int64 makes both sides race-free. The zero value reads back as the
// Unix epoch, so a never-ticked holder reports "unhealthy" — matching the old
// zero-value time.Time semantics.
type atomicTime struct {
	unixNano atomic.Int64
}

func (a *atomicTime) Store(t time.Time) {
	a.unixNano.Store(t.UnixNano())
}

func (a *atomicTime) Load() time.Time {
	return time.Unix(0, a.unixNano.Load()).UTC()
}
