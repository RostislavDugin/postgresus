package usecases_physical_postgresql

import (
	"context"
	"io"
	"log/slog"

	util_encryption "databasus-backend/internal/util/encryption"
)

// parentManifestFetcher is the storage capability an incremental backup needs to
// pull its parent's reconstructed manifest sidecar: a read-only GetFile. The
// concrete storages.StorageFileSaver is asserted to this narrower seam in
// downloadParentManifest so the executor depends only on what it uses.
type parentManifestFetcher interface {
	GetFile(
		ctx context.Context,
		encryptor util_encryption.FieldEncryptor,
		logger *slog.Logger,
		fileName string,
	) (io.ReadCloser, error)
}
