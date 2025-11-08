package storages

import (
	"io"
	"log/slog"

	"github.com/google/uuid"
)

type StorageFileSaver interface {
	SaveFile(logger *slog.Logger, fileID uuid.UUID, file io.Reader) error

	GetFile(fileID uuid.UUID) (io.ReadCloser, error)

	DeleteFile(fileID uuid.UUID) error

	Validate() error

	TestConnection() error

	HideSensitiveData()
}
