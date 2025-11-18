package encryption

import "github.com/google/uuid"

type FieldEncryptor interface {
	// Encrypt encrypts a plaintext string and returns an encrypted string.
	// If the string is already encrypted, returns it as-is.
	// Empty strings are returned unchanged.
	Encrypt(itemID uuid.UUID, plaintext string) (string, error)

	// Decrypt decrypts an encrypted string and returns a plaintext string.
	// If the string is not encrypted, returns it as-is.
	// Empty strings are returned unchanged.
	Decrypt(itemID uuid.UUID, ciphertext string) (string, error)
}
