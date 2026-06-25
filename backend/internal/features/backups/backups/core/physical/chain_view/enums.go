package chain_view

// ChainState describes whether a chain is still extendable, dead or
// closed by a successor FULL. WAL gaps inside a chain DO NOT change this
// state — lossy chains stay EXTENDABLE; see FindWalGapsInChain to surface the
// unreachable PITR windows.
type ChainState string

const (
	ChainStateExtendable        ChainState = "EXTENDABLE"
	ChainStateBrokenByIncr      ChainState = "BROKEN_BY_INCR"
	ChainStateClosedByNewerFull ChainState = "CLOSED_BY_NEWER_FULL"
)

type ValidationStatus string

const (
	ValidationStatusOK            ValidationStatus = "OK"
	ValidationStatusOKWithWarning ValidationStatus = "OK_WITH_WARNING"
	ValidationStatusChainBroken   ValidationStatus = "CHAIN_BROKEN"
)
