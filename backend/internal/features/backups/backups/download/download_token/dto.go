package download_token

import "github.com/google/uuid"

type GenerateTokenResponse struct {
	Token    string    `json:"token"`
	Filename string    `json:"filename"`
	BackupID uuid.UUID `json:"backupId"`
}
