package encryption

import (
	"crypto/aes"
	"crypto/cipher"
	"encoding/binary"
	"fmt"
	"io"

	"github.com/google/uuid"
)

type EncryptionWriter struct {
	baseWriter    io.Writer
	cipher        cipher.AEAD
	buffer        []byte
	nonce         []byte
	salt          []byte
	chunkIndex    uint64
	headerWritten bool
}

func NewEncryptionWriter(
	baseWriter io.Writer,
	masterKey string,
	backupID uuid.UUID,
	salt []byte,
	nonce []byte,
) (*EncryptionWriter, error) {
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

	writer := &EncryptionWriter{
		baseWriter:    baseWriter,
		cipher:        aesgcm,
		buffer:        make([]byte, 0, ChunkSize),
		nonce:         nonce,
		chunkIndex:    0,
		headerWritten: false,
		salt:          salt, // Store salt for lazy header writing
	}

	return writer, nil
}

func (w *EncryptionWriter) Write(p []byte) (n int, err error) {
	// Write header on first write (lazy initialization)
	if !w.headerWritten {
		if err := w.writeHeader(w.salt, w.nonce); err != nil {
			return 0, fmt.Errorf("failed to write header: %w", err)
		}
	}

	n = len(p)
	w.buffer = append(w.buffer, p...)

	for len(w.buffer) >= ChunkSize {
		chunk := w.buffer[:ChunkSize]
		if err := w.encryptAndWriteChunk(chunk); err != nil {
			return 0, err
		}
		w.buffer = w.buffer[ChunkSize:]
	}

	return n, nil
}

func (w *EncryptionWriter) Close() error {
	// Write header if it hasn't been written yet (in case Close is called without any writes)
	if !w.headerWritten {
		if err := w.writeHeader(w.salt, w.nonce); err != nil {
			return fmt.Errorf("failed to write header: %w", err)
		}
	}

	if len(w.buffer) > 0 {
		if err := w.encryptAndWriteChunk(w.buffer); err != nil {
			return err
		}
		w.buffer = nil
	}
	return nil
}

func (w *EncryptionWriter) writeHeader(salt, nonce []byte) error {
	header := make([]byte, HeaderLen)

	copy(header[0:MagicBytesLen], []byte(MagicBytes))
	copy(header[MagicBytesLen:MagicBytesLen+SaltLen], salt)
	copy(header[MagicBytesLen+SaltLen:MagicBytesLen+SaltLen+NonceLen], nonce)

	_, err := w.baseWriter.Write(header)
	if err != nil {
		return fmt.Errorf("failed to write header: %w", err)
	}

	w.headerWritten = true
	return nil
}

func (w *EncryptionWriter) encryptAndWriteChunk(chunk []byte) error {
	chunkNonce := w.generateChunkNonce()

	encrypted := w.cipher.Seal(nil, chunkNonce, chunk, nil)

	lengthBuf := make([]byte, 4)
	binary.BigEndian.PutUint32(lengthBuf, uint32(len(encrypted)))

	if _, err := w.baseWriter.Write(lengthBuf); err != nil {
		return fmt.Errorf("failed to write chunk length: %w", err)
	}

	if _, err := w.baseWriter.Write(encrypted); err != nil {
		return fmt.Errorf("failed to write encrypted chunk: %w", err)
	}

	w.chunkIndex++
	return nil
}

func (w *EncryptionWriter) generateChunkNonce() []byte {
	chunkNonce := make([]byte, NonceLen)
	copy(chunkNonce, w.nonce)

	binary.BigEndian.PutUint64(chunkNonce[4:], w.chunkIndex)

	return chunkNonce
}
