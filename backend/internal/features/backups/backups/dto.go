package backups

import (
	"databasus-backend/internal/features/backups/backups/encryption"
	"io"
)

type GetBackupsRequest struct {
	DatabaseID string `form:"database_id" binding:"required"`
	Limit      int    `form:"limit"`
	Offset     int    `form:"offset"`
}

type GetBackupsResponse struct {
	Backups []*Backup `json:"backups"`
	Total   int64     `json:"total"`
	Limit   int       `json:"limit"`
	Offset  int       `json:"offset"`
}

type decryptionReaderCloser struct {
	*encryption.DecryptionReader
	baseReader io.ReadCloser
}

func (r *decryptionReaderCloser) Close() error {
	return r.baseReader.Close()
}
