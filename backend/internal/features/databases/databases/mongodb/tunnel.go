package mongodb

import (
	"context"
	"errors"
	"log/slog"

	"databasus-backend/internal/features/sshtunnel"
	"databasus-backend/internal/util/encryption"
)

type OpenTunnelSpec struct {
	Database  *MongodbDatabase
	Logger    *slog.Logger
	Encryptor encryption.FieldEncryptor
}

// The tunnel belongs to the operation, not to the model: buildURI is called separately by every
// method that reaches the server, so no method down there could own a Close.
type TunneledDatabase struct {
	original              *MongodbDatabase
	databaseThroughTunnel *MongodbDatabase
	forwarder             *sshtunnel.Forwarder
}

// Ctx bounds opening the tunnel; the forwarder then lives until Close. With the tunnel disabled
// this still returns a usable wrapper around the original, so callers never branch.
func OpenTunnel(ctx context.Context, spec OpenTunnelSpec) (*TunneledDatabase, error) {
	if spec.Database == nil || !spec.Database.SshTunnel.IsEnabled {
		return &TunneledDatabase{original: spec.Database, databaseThroughTunnel: spec.Database}, nil
	}

	// Restore targets arrive as bare models straight from the request DTO, so Validate has not
	// necessarily run and the port may still be unset.
	if spec.Database.Port == nil {
		return nil, errors.New("SSH tunnel requires an explicit MongoDB port")
	}

	forwarder, err := sshtunnel.Open(ctx, sshtunnel.OpenSpec{
		Config:    spec.Database.SshTunnel,
		Target:    sshtunnel.Endpoint{Host: spec.Database.Host, Port: *spec.Database.Port},
		Encryptor: spec.Encryptor,
		Logger:    spec.Logger,
	})
	if err != nil {
		return nil, err
	}

	localEndpoint := forwarder.GetLocalEndpoint()

	databaseThroughTunnel := *spec.Database
	databaseThroughTunnel.Host = localEndpoint.Host
	// A fresh pointer, because the shallow copy shares the original's and the original is what gets
	// persisted.
	databaseThroughTunnel.Port = new(localEndpoint.Port)
	// Topology discovery re-dials the addresses the replica set advertises, which leave the process
	// directly and never reach the forwarded port. Only the copy is pinned, so the stored config
	// keeps whatever failover behaviour the user chose.
	databaseThroughTunnel.IsDirectConnection = true
	// The copy describes an already forwarded connection; leaving this on would make anything that
	// re-reads it open a second tunnel pointed at the first one's local port.
	databaseThroughTunnel.SshTunnel.IsEnabled = false

	return &TunneledDatabase{
		original:              spec.Database,
		databaseThroughTunnel: &databaseThroughTunnel,
		forwarder:             forwarder,
	}, nil
}

func (t *TunneledDatabase) GetDatabaseThroughTunnel() *MongodbDatabase {
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
