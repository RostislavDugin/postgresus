package sshtunnel

import "errors"

// Returned when a connection reaches the forwarder after Close: an expected shutdown race, not a
// tunnel failure, so callers can tell it apart from a bastion that is genuinely unreachable.
var ErrForwarderClosed = errors.New("ssh tunnel forwarder is closed")
