package usecases_physical_postgresql

import (
	"fmt"
	"io"

	"github.com/google/uuid"

	backups_core_enums "databasus-backend/internal/features/backups/backups/core/enums"
	backup_encryption "databasus-backend/internal/features/backups/backups/encryption"
)

// setupEncryption wraps the storage-bound writer in the backup encryption writer
// when encryption is enabled. It returns the writer the stream should tee into,
// the EncryptionWriter to Close on teardown (nil when disabled), and the fresh
// salt/nonce (base64) to persist. When encryption is off it returns base
// unchanged with empty salt/nonce.
func setupEncryption(
	base io.Writer,
	enc backups_core_enums.BackupEncryption,
	masterKey string,
	backupID uuid.UUID,
) (io.Writer, *backup_encryption.EncryptionWriter, string, string, error) {
	if enc != backups_core_enums.BackupEncryptionEncrypted {
		return base, nil, "", "", nil
	}

	encSetup, err := backup_encryption.SetupEncryptionWriter(base, masterKey, backupID)
	if err != nil {
		return nil, nil, "", "", fmt.Errorf("setup encryption: %w", err)
	}

	return encSetup.Writer, encSetup.Writer, encSetup.SaltBase64, encSetup.NonceBase64, nil
}
