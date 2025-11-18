package encryption

import (
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

func Test_Encrypt_Decrypt_RoundTrip(t *testing.T) {
	encryptor := GetFieldEncryptor()
	itemID := uuid.New()
	plaintext := "my-secret-password"

	encrypted, err := encryptor.Encrypt(itemID, plaintext)
	assert.NoError(t, err)
	assert.NotEmpty(t, encrypted)
	assert.NotEqual(t, plaintext, encrypted)
	assert.Contains(t, encrypted, "enc:")

	decrypted, err := encryptor.Decrypt(itemID, encrypted)
	assert.NoError(t, err)
	assert.Equal(t, plaintext, decrypted)
}

func Test_Encrypt_EmptyString_ReturnsEmpty(t *testing.T) {
	encryptor := GetFieldEncryptor()
	itemID := uuid.New()

	encrypted, err := encryptor.Encrypt(itemID, "")
	assert.NoError(t, err)
	assert.Empty(t, encrypted)
}

func Test_Decrypt_EmptyString_ReturnsEmpty(t *testing.T) {
	encryptor := GetFieldEncryptor()
	itemID := uuid.New()

	decrypted, err := encryptor.Decrypt(itemID, "")
	assert.NoError(t, err)
	assert.Empty(t, decrypted)
}

func Test_Decrypt_PlaintextValue_ReturnsAsIs(t *testing.T) {
	encryptor := GetFieldEncryptor()
	itemID := uuid.New()
	plaintext := "not-encrypted-password"

	decrypted, err := encryptor.Decrypt(itemID, plaintext)
	assert.NoError(t, err)
	assert.Equal(t, plaintext, decrypted)
}

func Test_Encrypt_DetectsAlreadyEncryptedFormat(t *testing.T) {
	encryptor := GetFieldEncryptor()
	itemID := uuid.New()
	alreadyEncrypted := "enc:nonce:ciphertext"

	result, err := encryptor.Encrypt(itemID, alreadyEncrypted)
	assert.NoError(t, err)
	assert.Equal(t, alreadyEncrypted, result)
}

func Test_Encrypt_SamePlaintext_DifferentItemIDs_ProducesDifferentCiphertext(t *testing.T) {
	encryptor := GetFieldEncryptor()
	plaintext := "shared-secret"
	itemID1 := uuid.New()
	itemID2 := uuid.New()

	encrypted1, err := encryptor.Encrypt(itemID1, plaintext)
	assert.NoError(t, err)

	encrypted2, err := encryptor.Encrypt(itemID2, plaintext)
	assert.NoError(t, err)

	assert.NotEqual(t, encrypted1, encrypted2)

	decrypted1, err := encryptor.Decrypt(itemID1, encrypted1)
	assert.NoError(t, err)
	assert.Equal(t, plaintext, decrypted1)

	decrypted2, err := encryptor.Decrypt(itemID2, encrypted2)
	assert.NoError(t, err)
	assert.Equal(t, plaintext, decrypted2)
}

func Test_Encrypt_AlreadyEncrypted_ReturnsAsIs(t *testing.T) {
	encryptor := GetFieldEncryptor()
	itemID := uuid.New()
	plaintext := "my-password"

	encrypted1, err := encryptor.Encrypt(itemID, plaintext)
	assert.NoError(t, err)

	encrypted2, err := encryptor.Encrypt(itemID, encrypted1)
	assert.NoError(t, err)

	assert.Equal(t, encrypted1, encrypted2)
}

func Test_Decrypt_MalformedData_ReturnsError(t *testing.T) {
	encryptor := GetFieldEncryptor()
	itemID := uuid.New()

	_, err := encryptor.Decrypt(itemID, "enc:invalid")
	assert.Error(t, err)

	_, err = encryptor.Decrypt(itemID, "enc:invalid:invalid-base64")
	assert.Error(t, err)
}

func Test_EncryptedFormat_ContainsPrefix(t *testing.T) {
	encryptor := GetFieldEncryptor()
	itemID := uuid.New()
	plaintext := "test-secret"

	encrypted, err := encryptor.Encrypt(itemID, plaintext)
	assert.NoError(t, err)
	assert.Contains(t, encrypted, "enc:")
}
