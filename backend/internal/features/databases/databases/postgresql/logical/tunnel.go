package postgresql_logical

import (
	"context"
	"log/slog"

	"databasus-backend/internal/features/sshtunnel"
	"databasus-backend/internal/util/encryption"
)

type OpenTunnelSpec struct {
	Database  *PostgresqlLogicalDatabase
	Logger    *slog.Logger
	Encryptor encryption.FieldEncryptor
}

// The tunnel belongs to the operation, not to the model: openPgConn hands back a live connection
// that outlives the call, so no method down there could own a Close.
type TunneledDatabase struct {
	original              *PostgresqlLogicalDatabase
	databaseThroughTunnel *PostgresqlLogicalDatabase
	forwarder             *sshtunnel.Forwarder
}

// Ctx bounds opening the tunnel; the forwarder then lives until Close. With the tunnel disabled
// this still returns a usable wrapper around the original, so callers never branch.
func OpenTunnel(ctx context.Context, spec OpenTunnelSpec) (*TunneledDatabase, error) {
	if spec.Database == nil || !spec.Database.SshTunnel.IsEnabled {
		return &TunneledDatabase{original: spec.Database, databaseThroughTunnel: spec.Database}, nil
	}

	forwarder, err := sshtunnel.Open(ctx, sshtunnel.OpenSpec{
		Config:    spec.Database.SshTunnel,
		Target:    sshtunnel.Endpoint{Host: spec.Database.Host, Port: spec.Database.Port},
		Encryptor: spec.Encryptor,
		Logger:    spec.Logger,
	})
	if err != nil {
		return nil, err
	}

	localEndpoint := forwarder.GetLocalEndpoint()

	databaseThroughTunnel := *spec.Database
	// Host stays the real one: sslmode=verify-full matches the certificate against it, and libpq
	// looks up .pgpass by host and port.
	databaseThroughTunnel.Port = localEndpoint.Port
	databaseThroughTunnel.LocalTunnelEndpoint = &localEndpoint
	// The copy describes an already forwarded connection; leaving this on would make anything that
	// re-reads it open a second tunnel pointed at the first one's local port.
	databaseThroughTunnel.SshTunnel.IsEnabled = false

	return &TunneledDatabase{
		original:              spec.Database,
		databaseThroughTunnel: &databaseThroughTunnel,
		forwarder:             forwarder,
	}, nil
}

func (t *TunneledDatabase) GetDatabaseThroughTunnel() *PostgresqlLogicalDatabase {
	if t == nil {
		return nil
	}

	return t.databaseThroughTunnel
}

// TestConnection and PopulateVersion write the detected version into the model they were handed,
// which is the copy. Callers that persist the result need it carried back.
func (t *TunneledDatabase) CopyDiscoveredMetadataToOriginal() {
	if t == nil || t.original == nil || t.databaseThroughTunnel == nil {
		return
	}

	t.original.Version = t.databaseThroughTunnel.Version
}

func (t *TunneledDatabase) Close() {
	if t == nil || t.forwarder == nil {
		return
	}

	t.forwarder.Close()
}
