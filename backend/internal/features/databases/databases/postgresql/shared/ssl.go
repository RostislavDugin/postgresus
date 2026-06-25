package postgresql_shared

import (
	"errors"
	"fmt"
)

type PostgresSslMode string

const (
	PostgresSslModeDisable    PostgresSslMode = "disable"
	PostgresSslModeRequire    PostgresSslMode = "require"
	PostgresSslModeVerifyCA   PostgresSslMode = "verify-ca"
	PostgresSslModeVerifyFull PostgresSslMode = "verify-full"
)

// ValidateSslConfig is the cross-strategy SSL rule: known mode, paired
// client cert + key, and certs only when SSL is enabled. Logical (pg_dump)
// and physical (pg_basebackup / replication) callers must apply the same rule.
func ValidateSslConfig(mode PostgresSslMode, clientCert, clientKey, rootCert string) error {
	switch mode {
	case PostgresSslModeDisable, PostgresSslModeRequire, PostgresSslModeVerifyCA, PostgresSslModeVerifyFull:
	default:
		return fmt.Errorf("invalid SSL mode: %s", mode)
	}

	hasClientCert := clientCert != ""
	hasClientKey := clientKey != ""

	if hasClientCert != hasClientKey {
		return errors.New("client certificate and client key must be provided together")
	}

	if mode == PostgresSslModeDisable && (hasClientCert || hasClientKey || rootCert != "") {
		return errors.New(
			"SSL certificates require SSL to be enabled (set SSL mode to require, verify-ca or verify-full)",
		)
	}

	return nil
}
