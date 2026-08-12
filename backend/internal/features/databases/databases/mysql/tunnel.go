package mysql

import (
	"context"
	"log/slog"

	"databasus-backend/internal/features/sshtunnel"
	"databasus-backend/internal/util/encryption"
)

type OpenTunnelSpec struct {
	Database  *MysqlDatabase
	Logger    *slog.Logger
	Encryptor encryption.FieldEncryptor
}

// The tunnel belongs to the operation, not to the model: buildDSN is called separately by every
// method that reaches the server, so no method down there could own a Close.
type TunneledDatabase struct {
	original              *MysqlDatabase
	databaseThroughTunnel *MysqlDatabase
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
	databaseThroughTunnel.Host = localEndpoint.Host
	databaseThroughTunnel.Port = localEndpoint.Port
	// The copy describes an already forwarded connection; leaving this on would make anything that
	// re-reads it open a second tunnel pointed at the first one's local port.
	databaseThroughTunnel.SshTunnel.IsEnabled = false

	return &TunneledDatabase{
		original:              spec.Database,
		databaseThroughTunnel: &databaseThroughTunnel,
		forwarder:             forwarder,
	}, nil
}

func (t *TunneledDatabase) GetDatabaseThroughTunnel() *MysqlDatabase {
	if t == nil {
		return nil
	}

	return t.databaseThroughTunnel
}

// TestConnection and PopulateDbData write what they detect into the model they were handed, which
// is the copy. Callers that persist the result need it carried back.
func (t *TunneledDatabase) CopyDiscoveredMetadataToOriginal() {
	if t == nil || t.original == nil || t.databaseThroughTunnel == nil {
		return
	}

	t.original.Version = t.databaseThroughTunnel.Version
	t.original.Privileges = t.databaseThroughTunnel.Privileges
}

func (t *TunneledDatabase) Close() {
	if t == nil || t.forwarder == nil {
		return
	}

	t.forwarder.Close()
}
