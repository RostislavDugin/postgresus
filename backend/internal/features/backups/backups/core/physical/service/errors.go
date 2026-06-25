package physical_service

import "errors"

// ErrFullNotFound is returned by read-only methods when the requested root
// FULL does not exist. DeleteFull treats a missing FULL as a no-op (a peer
// already deleted the chain) rather than an error.
var ErrFullNotFound = errors.New("physical_service: root full backup not found")
