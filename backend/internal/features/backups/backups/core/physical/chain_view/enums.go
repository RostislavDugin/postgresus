package chain_view

type ValidationStatus string

const (
	ValidationStatusOK            ValidationStatus = "OK"
	ValidationStatusOKWithWarning ValidationStatus = "OK_WITH_WARNING"
	ValidationStatusChainBroken   ValidationStatus = "CHAIN_BROKEN"
)
