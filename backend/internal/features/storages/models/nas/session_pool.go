package nas_storage

import (
	"context"
	"fmt"
	"net"

	"github.com/hirochachacha/go-smb2"

	storage_connections "databasus-backend/internal/features/storages/connections"
	"databasus-backend/internal/util/encryption"
)

type nasSession struct {
	connection net.Conn
	session    *smb2.Session
	share      *smb2.Share
}

// Every call goes through the share bound to the caller's context, so the library itself gives up
// on a request when the caller does.
func (s *nasSession) getShare(ctx context.Context) *smb2.Share {
	return s.share.WithContext(ctx)
}

// The locator hands out a freshly loaded model on every write, so the pool lives at package level
// and the model only names its key.
var sessionPool = storage_connections.NewSessionPool(storage_connections.SessionOperations[*nasSession]{
	Close: closeSession,
	Probe: probeSession,
}, storage_connections.GetDefaultPoolSettings())

// Unmounting and logging off are skipped on purpose: a session is also closed to unblock a call
// stuck on a silent server, where each would wait just as long. The server drops the tree and the
// session together with the connection.
func closeSession(session *nasSession) {
	_ = session.connection.Close()
}

func probeSession(ctx context.Context, session *nasSession) error {
	_, err := session.getShare(ctx).Stat(".")

	return err
}

// Credential fields stay in their stored, encrypted form: they change whenever the user saves new
// credentials, and the fingerprint never holds a plain secret.
func (n *NASStorage) getSessionKey() storage_connections.Key {
	return storage_connections.Key{
		StorageID: n.StorageID,
		Fingerprint: storage_connections.Fingerprint(
			n.Host, n.Port, n.Share, n.Username, n.Password, n.Domain, n.UseSSL, n.Path,
		),
	}
}

func (n *NASStorage) checkoutSession(
	ctx context.Context,
	encryptor encryption.FieldEncryptor,
) (*storage_connections.Lease[*nasSession], error) {
	return sessionPool.Checkout(ctx, n.getSessionKey(), func(ctx context.Context) (*nasSession, error) {
		return n.dialSession(ctx, encryptor)
	})
}

// The share is mounted and the target directory ensured once per session rather than on every write.
func (n *NASStorage) dialSession(ctx context.Context, encryptor encryption.FieldEncryptor) (*nasSession, error) {
	connection, err := n.createConnectionWithContext(ctx)
	if err != nil {
		return nil, err
	}

	smbSession, err := n.dialSmbSession(ctx, connection, encryptor)
	if err != nil {
		_ = connection.Close()
		return nil, err
	}

	share, err := smbSession.WithContext(ctx).Mount(n.Share)
	if err != nil {
		_ = connection.Close()
		return nil, fmt.Errorf("failed to mount share '%s': %w", n.Share, err)
	}

	session := &nasSession{connection: connection, session: smbSession, share: share}

	if n.Path != "" {
		if err := n.ensureDirectory(session.getShare(ctx), n.Path); err != nil {
			closeSession(session)
			return nil, fmt.Errorf("failed to ensure directory: %w", err)
		}
	}

	return session, nil
}
