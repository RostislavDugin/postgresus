package encryption

import (
	"bytes"
	"crypto/rand"
	"io"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func Test_EncryptDecryptRoundTrip_ReturnsOriginalData(t *testing.T) {
	masterKey := uuid.New().String() + uuid.New().String()
	backupID := uuid.New()
	salt, err := GenerateSalt()
	require.NoError(t, err)
	nonce, err := GenerateNonce()
	require.NoError(t, err)

	originalData := []byte(
		"This is a test backup data that should be encrypted and then decrypted successfully.",
	)

	var encrypted bytes.Buffer
	writer, err := NewEncryptionWriter(&encrypted, masterKey, backupID, salt, nonce)
	require.NoError(t, err)

	n, err := writer.Write(originalData)
	require.NoError(t, err)
	assert.Equal(t, len(originalData), n)

	err = writer.Close()
	require.NoError(t, err)

	reader, err := NewDecryptionReader(&encrypted, masterKey, backupID, salt, nonce)
	require.NoError(t, err)

	decrypted := make([]byte, len(originalData))
	n, err = io.ReadFull(reader, decrypted)
	require.NoError(t, err)
	assert.Equal(t, len(originalData), n)
	assert.Equal(t, originalData, decrypted)
}

func Test_EncryptDecryptRoundTrip_LargeData_WorksCorrectly(t *testing.T) {
	masterKey := uuid.New().String() + uuid.New().String()
	backupID := uuid.New()
	salt, err := GenerateSalt()
	require.NoError(t, err)
	nonce, err := GenerateNonce()
	require.NoError(t, err)

	originalData := make([]byte, 100*1024)
	_, err = rand.Read(originalData)
	require.NoError(t, err)

	var encrypted bytes.Buffer
	writer, err := NewEncryptionWriter(&encrypted, masterKey, backupID, salt, nonce)
	require.NoError(t, err)

	n, err := writer.Write(originalData)
	require.NoError(t, err)
	assert.Equal(t, len(originalData), n)

	err = writer.Close()
	require.NoError(t, err)

	reader, err := NewDecryptionReader(&encrypted, masterKey, backupID, salt, nonce)
	require.NoError(t, err)

	decrypted, err := io.ReadAll(reader)
	require.NoError(t, err)
	assert.Equal(t, originalData, decrypted)
}

func Test_EncryptionWriter_MultipleWrites_CombinesCorrectly(t *testing.T) {
	masterKey := uuid.New().String() + uuid.New().String()
	backupID := uuid.New()
	salt, err := GenerateSalt()
	require.NoError(t, err)
	nonce, err := GenerateNonce()
	require.NoError(t, err)

	part1 := []byte("First part of data. ")
	part2 := []byte("Second part of data. ")
	part3 := []byte("Third part of data.")
	expectedData := append(append(part1, part2...), part3...)

	var encrypted bytes.Buffer
	writer, err := NewEncryptionWriter(&encrypted, masterKey, backupID, salt, nonce)
	require.NoError(t, err)

	_, err = writer.Write(part1)
	require.NoError(t, err)
	_, err = writer.Write(part2)
	require.NoError(t, err)
	_, err = writer.Write(part3)
	require.NoError(t, err)

	err = writer.Close()
	require.NoError(t, err)

	reader, err := NewDecryptionReader(&encrypted, masterKey, backupID, salt, nonce)
	require.NoError(t, err)

	decrypted, err := io.ReadAll(reader)
	require.NoError(t, err)
	assert.Equal(t, expectedData, decrypted)
}

func Test_DecryptionReader_InvalidHeader_ReturnsError(t *testing.T) {
	masterKey := uuid.New().String() + uuid.New().String()
	backupID := uuid.New()
	salt, err := GenerateSalt()
	require.NoError(t, err)
	nonce, err := GenerateNonce()
	require.NoError(t, err)

	invalidHeader := make([]byte, HeaderLen)
	copy(invalidHeader, []byte("INVALID!"))

	invalidData := bytes.NewBuffer(invalidHeader)

	_, err = NewDecryptionReader(invalidData, masterKey, backupID, salt, nonce)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid magic bytes")
}

func Test_DecryptionReader_TamperedData_ReturnsError(t *testing.T) {
	masterKey := uuid.New().String() + uuid.New().String()
	backupID := uuid.New()
	salt, err := GenerateSalt()
	require.NoError(t, err)
	nonce, err := GenerateNonce()
	require.NoError(t, err)

	originalData := []byte("This data will be tampered with.")

	var encrypted bytes.Buffer
	writer, err := NewEncryptionWriter(&encrypted, masterKey, backupID, salt, nonce)
	require.NoError(t, err)

	_, err = writer.Write(originalData)
	require.NoError(t, err)

	err = writer.Close()
	require.NoError(t, err)

	encryptedBytes := encrypted.Bytes()
	if len(encryptedBytes) > HeaderLen+10 {
		encryptedBytes[HeaderLen+10] ^= 0xFF
	}

	tamperedBuffer := bytes.NewBuffer(encryptedBytes)

	reader, err := NewDecryptionReader(tamperedBuffer, masterKey, backupID, salt, nonce)
	require.NoError(t, err)

	_, err = io.ReadAll(reader)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "authentication failed")
}

func Test_DeriveBackupKey_SameInputs_ReturnsSameKey(t *testing.T) {
	masterKey := uuid.New().String() + uuid.New().String()
	backupID := uuid.New()
	salt, err := GenerateSalt()
	require.NoError(t, err)

	key1, err := DeriveBackupKey(masterKey, backupID, salt)
	require.NoError(t, err)

	key2, err := DeriveBackupKey(masterKey, backupID, salt)
	require.NoError(t, err)

	assert.Equal(t, key1, key2)
}

func Test_DeriveBackupKey_DifferentInputs_ReturnsDifferentKeys(t *testing.T) {
	masterKey1 := uuid.New().String() + uuid.New().String()
	masterKey2 := uuid.New().String() + uuid.New().String()
	backupID1 := uuid.New()
	backupID2 := uuid.New()
	salt1, err := GenerateSalt()
	require.NoError(t, err)
	salt2, err := GenerateSalt()
	require.NoError(t, err)

	key1, err := DeriveBackupKey(masterKey1, backupID1, salt1)
	require.NoError(t, err)

	key2, err := DeriveBackupKey(masterKey2, backupID1, salt1)
	require.NoError(t, err)
	assert.NotEqual(t, key1, key2)

	key3, err := DeriveBackupKey(masterKey1, backupID2, salt1)
	require.NoError(t, err)
	assert.NotEqual(t, key1, key3)

	key4, err := DeriveBackupKey(masterKey1, backupID1, salt2)
	require.NoError(t, err)
	assert.NotEqual(t, key1, key4)
}

func Test_EncryptionWriter_PartialChunk_HandledCorrectly(t *testing.T) {
	masterKey := uuid.New().String() + uuid.New().String()
	backupID := uuid.New()
	salt, err := GenerateSalt()
	require.NoError(t, err)
	nonce, err := GenerateNonce()
	require.NoError(t, err)

	smallData := []byte("Small data less than chunk size")

	var encrypted bytes.Buffer
	writer, err := NewEncryptionWriter(&encrypted, masterKey, backupID, salt, nonce)
	require.NoError(t, err)

	_, err = writer.Write(smallData)
	require.NoError(t, err)

	err = writer.Close()
	require.NoError(t, err)

	reader, err := NewDecryptionReader(&encrypted, masterKey, backupID, salt, nonce)
	require.NoError(t, err)

	decrypted, err := io.ReadAll(reader)
	require.NoError(t, err)
	assert.Equal(t, smallData, decrypted)
}

func Test_GenerateSalt_ReturnsCorrectLength(t *testing.T) {
	salt, err := GenerateSalt()
	require.NoError(t, err)
	assert.Equal(t, SaltLen, len(salt))
}

func Test_GenerateSalt_GeneratesUniqueSalts(t *testing.T) {
	salt1, err := GenerateSalt()
	require.NoError(t, err)

	salt2, err := GenerateSalt()
	require.NoError(t, err)

	assert.NotEqual(t, salt1, salt2)
}

func Test_GenerateNonce_ReturnsCorrectLength(t *testing.T) {
	nonce, err := GenerateNonce()
	require.NoError(t, err)
	assert.Equal(t, NonceLen, len(nonce))
}

func Test_GenerateNonce_GeneratesUniqueNonces(t *testing.T) {
	nonce1, err := GenerateNonce()
	require.NoError(t, err)

	nonce2, err := GenerateNonce()
	require.NoError(t, err)

	assert.NotEqual(t, nonce1, nonce2)
}

func Test_DecryptionReader_WrongMasterKey_ReturnsError(t *testing.T) {
	masterKey1 := uuid.New().String() + uuid.New().String()
	masterKey2 := uuid.New().String() + uuid.New().String()
	backupID := uuid.New()
	salt, err := GenerateSalt()
	require.NoError(t, err)
	nonce, err := GenerateNonce()
	require.NoError(t, err)

	originalData := []byte("Secret data")

	var encrypted bytes.Buffer
	writer, err := NewEncryptionWriter(&encrypted, masterKey1, backupID, salt, nonce)
	require.NoError(t, err)

	_, err = writer.Write(originalData)
	require.NoError(t, err)

	err = writer.Close()
	require.NoError(t, err)

	reader, err := NewDecryptionReader(&encrypted, masterKey2, backupID, salt, nonce)
	require.NoError(t, err)

	_, err = io.ReadAll(reader)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "authentication failed")
}

func Test_EncryptionWriter_EmptyData_WorksCorrectly(t *testing.T) {
	masterKey := uuid.New().String() + uuid.New().String()
	backupID := uuid.New()
	salt, err := GenerateSalt()
	require.NoError(t, err)
	nonce, err := GenerateNonce()
	require.NoError(t, err)

	var encrypted bytes.Buffer
	writer, err := NewEncryptionWriter(&encrypted, masterKey, backupID, salt, nonce)
	require.NoError(t, err)

	err = writer.Close()
	require.NoError(t, err)

	reader, err := NewDecryptionReader(&encrypted, masterKey, backupID, salt, nonce)
	require.NoError(t, err)

	decrypted, err := io.ReadAll(reader)
	require.NoError(t, err)
	assert.Equal(t, 0, len(decrypted))
}

func Test_EncryptionWriter_MultipleChunks_WorksCorrectly(t *testing.T) {
	masterKey := uuid.New().String() + uuid.New().String()
	backupID := uuid.New()
	salt, err := GenerateSalt()
	require.NoError(t, err)
	nonce, err := GenerateNonce()
	require.NoError(t, err)

	dataSize := ChunkSize*3 + 1000
	originalData := make([]byte, dataSize)
	_, err = rand.Read(originalData)
	require.NoError(t, err)

	var encrypted bytes.Buffer
	writer, err := NewEncryptionWriter(&encrypted, masterKey, backupID, salt, nonce)
	require.NoError(t, err)

	_, err = writer.Write(originalData)
	require.NoError(t, err)

	err = writer.Close()
	require.NoError(t, err)

	reader, err := NewDecryptionReader(&encrypted, masterKey, backupID, salt, nonce)
	require.NoError(t, err)

	decrypted, err := io.ReadAll(reader)
	require.NoError(t, err)
	assert.Equal(t, originalData, decrypted)
}

func Test_DecryptionReader_SmallReads_WorksCorrectly(t *testing.T) {
	masterKey := uuid.New().String() + uuid.New().String()
	backupID := uuid.New()
	salt, err := GenerateSalt()
	require.NoError(t, err)
	nonce, err := GenerateNonce()
	require.NoError(t, err)

	originalData := []byte("This is test data that will be read in small chunks.")

	var encrypted bytes.Buffer
	writer, err := NewEncryptionWriter(&encrypted, masterKey, backupID, salt, nonce)
	require.NoError(t, err)

	_, err = writer.Write(originalData)
	require.NoError(t, err)

	err = writer.Close()
	require.NoError(t, err)

	reader, err := NewDecryptionReader(&encrypted, masterKey, backupID, salt, nonce)
	require.NoError(t, err)

	var decrypted []byte
	buf := make([]byte, 5)
	for {
		n, err := reader.Read(buf)
		if n > 0 {
			decrypted = append(decrypted, buf[:n]...)
		}
		if err == io.EOF {
			break
		}
		require.NoError(t, err)
	}

	assert.Equal(t, originalData, decrypted)
}
