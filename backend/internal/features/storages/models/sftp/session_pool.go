package sftp_storage

import (
	"context"
	"fmt"

	"github.com/pkg/sftp"
	"golang.org/x/crypto/ssh"

	storage_connections "databasus-backend/internal/features/storages/connections"
	"databasus-backend/internal/util/encryption"
)

type sftpSession struct {
	sshClient  *ssh.Client
	sftpClient *sftp.Client
}

// The locator hands out a freshly loaded model on every write, so the pool lives at package level
// and the model only names its key.
var sessionPool = storage_connections.NewSessionPool(storage_connections.SessionOperations[*sftpSession]{
	Close: closeSession,
	Probe: probeSession,
}, storage_connections.GetDefaultPoolSettings())

// The SFTP client's Close waits for its receive loop, which only ends once the SSH connection is
// gone, so the SSH side closes first or a session stuck on a silent server never closes.
func closeSession(session *sftpSession) {
	_ = session.sshClient.Close()
	_ = session.sftpClient.Close()
}

// The SFTP client takes no context, so the probe deadline is enforced by closing the session.
func probeSession(ctx context.Context, session *sftpSession) error {
	stopCloseOnDeadline := context.AfterFunc(ctx, func() {
		closeSession(session)
	})
	defer stopCloseOnDeadline()

	_, err := session.sftpClient.Getwd()

	return err
}

// Credential fields stay in their stored, encrypted form: they change whenever the user saves new
// credentials, and the fingerprint never holds a plain secret.
func (s *SFTPStorage) getSessionKey() storage_connections.Key {
	return storage_connections.Key{
		StorageID:   s.StorageID,
		Fingerprint: storage_connections.Fingerprint(s.Host, s.Port, s.Username, s.Password, s.PrivateKey, s.Path),
	}
}

func (s *SFTPStorage) checkoutSession(
	ctx context.Context,
	encryptor encryption.FieldEncryptor,
) (*storage_connections.Lease[*sftpSession], error) {
	return sessionPool.Checkout(ctx, s.getSessionKey(), func(ctx context.Context) (*sftpSession, error) {
		return s.dialSession(ctx, encryptor)
	})
}

// The target directory is ensured once per session rather than walked on every write.
func (s *SFTPStorage) dialSession(ctx context.Context, encryptor encryption.FieldEncryptor) (*sftpSession, error) {
	sftpClient, sshClient, err := s.connectWithContext(ctx, encryptor, sftpConnectTimeout)
	if err != nil {
		return nil, err
	}

	session := &sftpSession{sshClient: sshClient, sftpClient: sftpClient}

	if s.Path != "" {
		if err := s.ensureDirectory(sftpClient, s.Path); err != nil {
			closeSession(session)
			return nil, fmt.Errorf("failed to ensure directory: %w", err)
		}
	}

	return session, nil
}
