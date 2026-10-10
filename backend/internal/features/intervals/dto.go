package intervals

import "time"

// NextTrigger is the moment the scheduler will first treat a backup as due.
// When IsDue is set, the backup is due already and ScheduledAt is the moment of the check.
type NextTrigger struct {
	ScheduledAt time.Time `json:"scheduledAt"`
	IsDue       bool      `json:"isDue"`
}
