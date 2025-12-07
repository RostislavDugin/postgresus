package encryption

import (
	"crypto/rand"
	"crypto/sha256"
	"fmt"

	"github.com/google/uuid"
	"golang.org/x/crypto/pbkdf2"
)

const (
	MagicBytes       = "PGRSUS01"
	MagicBytesLen    = 8
	SaltLen          = 32
	NonceLen         = 12
	ReservedLen      = 12
	HeaderLen        = MagicBytesLen + SaltLen + NonceLen + ReservedLen
	ChunkSize        = 1 * 1024 * 1024
	PBKDF2Iterations = 100000
)

func DeriveBackupKey(masterKey string, backupID uuid.UUID, salt []byte) ([]byte, error) {
	if masterKey == "" {
		return nil, fmt.Errorf("master key cannot be empty")
	}
	if len(salt) != SaltLen {
		return nil, fmt.Errorf("salt must be %d bytes", SaltLen)
	}

	keyMaterial := []byte(masterKey + backupID.String())

	derivedKey := pbkdf2.Key(keyMaterial, salt, PBKDF2Iterations, 32, sha256.New)

	return derivedKey, nil
}

func GenerateSalt() ([]byte, error) {
	salt := make([]byte, SaltLen)
	if _, err := rand.Read(salt); err != nil {
		return nil, fmt.Errorf("failed to generate salt: %w", err)
	}
	return salt, nil
}

func GenerateNonce() ([]byte, error) {
	nonce := make([]byte, NonceLen)
	if _, err := rand.Read(nonce); err != nil {
		return nil, fmt.Errorf("failed to generate nonce: %w", err)
	}
	return nonce, nil
}
