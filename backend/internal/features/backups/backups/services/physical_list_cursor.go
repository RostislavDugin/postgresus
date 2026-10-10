package backups_services

import (
	"encoding/base64"
	"encoding/json"
	"time"

	"github.com/google/uuid"

	physical_core_service "databasus-backend/internal/features/backups/backups/core/physical/service"
)

// physicalListCursor is the JSON body of the opaque page cursor. It carries only
// a position in the list: the database always comes from the request path and
// passes the access check, so a cursor taken from another database selects
// nothing of that database.
type physicalListCursor struct {
	SortTime time.Time `json:"t"`
	ID       uuid.UUID `json:"id"`
}

func encodePhysicalListCursor(position physical_core_service.BackupListPosition) (string, error) {
	cursorJSON, err := json.Marshal(physicalListCursor{SortTime: position.SortTime, ID: position.ID})
	if err != nil {
		return "", err
	}

	return base64.RawURLEncoding.EncodeToString(cursorJSON), nil
}

func decodePhysicalListCursor(encodedCursor string) (*physical_core_service.BackupListPosition, error) {
	if encodedCursor == "" {
		return nil, nil
	}

	cursorJSON, err := base64.RawURLEncoding.DecodeString(encodedCursor)
	if err != nil {
		return nil, ErrInvalidBackupsCursor
	}

	var cursor physicalListCursor
	if err := json.Unmarshal(cursorJSON, &cursor); err != nil {
		return nil, ErrInvalidBackupsCursor
	}

	if cursor.SortTime.IsZero() || cursor.ID == uuid.Nil {
		return nil, ErrInvalidBackupsCursor
	}

	return &physical_core_service.BackupListPosition{SortTime: cursor.SortTime, ID: cursor.ID}, nil
}
