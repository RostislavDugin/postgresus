package encryption

import (
	"crypto/aes"
	"crypto/cipher"
	"encoding/binary"
	"fmt"
	"io"

	"github.com/google/uuid"
)

type DecryptionReader struct {
	baseReader io.Reader
	cipher     cipher.AEAD
	buffer     []byte
	nonce      []byte
	chunkIndex uint64
	headerRead bool
	eof        bool
}

func NewDecryptionReader(
	baseReader io.Reader,
	masterKey string,
	backupID uuid.UUID,
	salt []byte,
	nonce []byte,
) (*DecryptionReader, error) {
	if len(salt) != SaltLen {
		return nil, fmt.Errorf("salt must be %d bytes, got %d", SaltLen, len(salt))
	}
	if len(nonce) != NonceLen {
		return nil, fmt.Errorf("nonce must be %d bytes, got %d", NonceLen, len(nonce))
	}

	derivedKey, err := DeriveBackupKey(masterKey, backupID, salt)
	if err != nil {
		return nil, fmt.Errorf("failed to derive backup key: %w", err)
	}

	block, err := aes.NewCipher(derivedKey)
	if err != nil {
		return nil, fmt.Errorf("failed to create cipher: %w", err)
	}

	aesgcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("failed to create GCM: %w", err)
	}

	reader := &DecryptionReader{
		baseReader,
		aesgcm,
		make([]byte, 0),
		nonce,
		0,
		false,
		false,
	}

	if err := reader.readAndValidateHeader(salt, nonce); err != nil {
		return nil, err
	}

	return reader, nil
}

func (r *DecryptionReader) Read(p []byte) (n int, err error) {
	for len(r.buffer) < len(p) && !r.eof {
		if err := r.readAndDecryptChunk(); err != nil {
			if err == io.EOF {
				r.eof = true
				break
			}
			return 0, err
		}
	}

	if len(r.buffer) == 0 {
		return 0, io.EOF
	}

	n = copy(p, r.buffer)
	r.buffer = r.buffer[n:]

	return n, nil
}

func (r *DecryptionReader) readAndValidateHeader(expectedSalt, expectedNonce []byte) error {
	header := make([]byte, HeaderLen)

	if _, err := io.ReadFull(r.baseReader, header); err != nil {
		return fmt.Errorf("failed to read header: %w", err)
	}

	magic := string(header[0:MagicBytesLen])
	if magic != MagicBytes {
		return fmt.Errorf("invalid magic bytes: expected %s, got %s", MagicBytes, magic)
	}

	salt := header[MagicBytesLen : MagicBytesLen+SaltLen]
	nonce := header[MagicBytesLen+SaltLen : MagicBytesLen+SaltLen+NonceLen]

	if string(salt) != string(expectedSalt) {
		return fmt.Errorf("salt mismatch in file header")
	}

	if string(nonce) != string(expectedNonce) {
		return fmt.Errorf("nonce mismatch in file header")
	}

	r.headerRead = true
	return nil
}

func (r *DecryptionReader) readAndDecryptChunk() error {
	lengthBuf := make([]byte, 4)
	if _, err := io.ReadFull(r.baseReader, lengthBuf); err != nil {
		return err
	}

	chunkLen := binary.BigEndian.Uint32(lengthBuf)
	if chunkLen == 0 || chunkLen > ChunkSize+16 {
		return fmt.Errorf("invalid chunk length: %d", chunkLen)
	}

	encrypted := make([]byte, chunkLen)
	if _, err := io.ReadFull(r.baseReader, encrypted); err != nil {
		return fmt.Errorf("failed to read encrypted chunk: %w", err)
	}

	chunkNonce := r.generateChunkNonce()

	decrypted, err := r.cipher.Open(nil, chunkNonce, encrypted, nil)
	if err != nil {
		return fmt.Errorf(
			"failed to decrypt chunk (authentication failed - file may be corrupted or tampered): %w",
			err,
		)
	}

	r.buffer = append(r.buffer, decrypted...)
	r.chunkIndex++

	return nil
}

func (r *DecryptionReader) generateChunkNonce() []byte {
	chunkNonce := make([]byte, NonceLen)
	copy(chunkNonce, r.nonce)

	binary.BigEndian.PutUint64(chunkNonce[4:], r.chunkIndex)

	return chunkNonce
}
