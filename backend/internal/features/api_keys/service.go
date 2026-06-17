package api_keys

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"log/slog"
	"time"

	"github.com/google/uuid"

	"databasus-backend/internal/config"
	audit_logs "databasus-backend/internal/features/audit_logs"
	backups_core "databasus-backend/internal/features/backups/backups/core"
	backups_services "databasus-backend/internal/features/backups/backups/services"
	"databasus-backend/internal/features/databases"
	users_enums "databasus-backend/internal/features/users/enums"
	users_models "databasus-backend/internal/features/users/models"
)

type ApiKeyService struct {
	apiKeyRepository *ApiKeyRepository
	backupService    *backups_services.BackupService
	databaseService  *databases.DatabaseService
	auditLogService  *audit_logs.AuditLogService
	logger           *slog.Logger
}

func hashToken(token string) string {
	sum := sha256.Sum256([]byte(token))

	return fmt.Sprintf("%x", sum)
}

func generateToken() (plain, hashed, prefix string, err error) {
	raw := make([]byte, 32)
	if _, err = rand.Read(raw); err != nil {
		return "", "", "", err
	}

	plain = TokenPrefix + base64.RawURLEncoding.EncodeToString(raw)
	hashed = hashToken(plain)
	prefix = plain[:12]

	return plain, hashed, prefix, nil
}

func (s *ApiKeyService) CreateApiKey(
	user *users_models.User,
	request *CreateApiKeyRequestDTO,
) (*CreateApiKeyResponseDTO, error) {
	if user.Role != users_enums.UserRoleAdmin {
		return nil, ErrAdminOnly
	}

	if !request.Role.IsValid() {
		return nil, ErrInvalidRole
	}

	workspaceIDs := request.WorkspaceIDs
	if request.Role == users_enums.UserRoleAdmin {
		workspaceIDs = nil
	} else if len(workspaceIDs) == 0 {
		return nil, ErrWorkspacesRequired
	}

	plain, hashed, prefix, err := generateToken()
	if err != nil {
		return nil, err
	}

	apiKey := &ApiKey{
		Name:            request.Name,
		HashedToken:     hashed,
		TokenPrefix:     prefix,
		Role:            request.Role,
		CreatedByUserID: user.ID,
		ExpiresAt:       request.ExpiresAt,
		CreatedAt:       time.Now().UTC(),
	}

	if err := s.apiKeyRepository.Create(apiKey, workspaceIDs); err != nil {
		return nil, err
	}

	s.auditLogService.WriteAuditLog(
		fmt.Sprintf("API key created: %s (role %s)", apiKey.Name, apiKey.Role),
		&user.ID,
		nil,
	)

	return &CreateApiKeyResponseDTO{
		ID:           apiKey.ID,
		Name:         apiKey.Name,
		Role:         apiKey.Role,
		Token:        plain,
		TokenPrefix:  apiKey.TokenPrefix,
		WorkspaceIDs: workspaceIDs,
		ExpiresAt:    apiKey.ExpiresAt,
		CreatedAt:    apiKey.CreatedAt,
	}, nil
}

func (s *ApiKeyService) ListApiKeys(user *users_models.User) (*ListApiKeysResponseDTO, error) {
	if user.Role != users_enums.UserRoleAdmin {
		return nil, ErrAdminOnly
	}

	apiKeys, err := s.apiKeyRepository.FindAllActive()
	if err != nil {
		return nil, err
	}

	responses := make([]*ApiKeyResponseDTO, 0, len(apiKeys))
	for _, apiKey := range apiKeys {
		responses = append(responses, &ApiKeyResponseDTO{
			ID:           apiKey.ID,
			Name:         apiKey.Name,
			Role:         apiKey.Role,
			TokenPrefix:  apiKey.TokenPrefix,
			WorkspaceIDs: apiKey.WorkspaceIDs(),
			LastUsedAt:   apiKey.LastUsedAt,
			ExpiresAt:    apiKey.ExpiresAt,
			CreatedAt:    apiKey.CreatedAt,
		})
	}

	return &ListApiKeysResponseDTO{ApiKeys: responses}, nil
}

func (s *ApiKeyService) RevokeApiKey(user *users_models.User, id uuid.UUID) error {
	if user.Role != users_enums.UserRoleAdmin {
		return ErrAdminOnly
	}

	if err := s.apiKeyRepository.Revoke(id, time.Now().UTC()); err != nil {
		return err
	}

	s.auditLogService.WriteAuditLog(
		fmt.Sprintf("API key revoked: %s", id),
		&user.ID,
		nil,
	)

	return nil
}

func (s *ApiKeyService) AuthenticateToken(token string) (*Principal, error) {
	if len(token) <= len(TokenPrefix) || token[:len(TokenPrefix)] != TokenPrefix {
		return nil, ErrInvalidApiKey
	}

	apiKey, err := s.apiKeyRepository.FindByHashedToken(hashToken(token))
	if err != nil {
		return nil, ErrInvalidApiKey
	}

	if apiKey.RevokedAt != nil {
		return nil, ErrInvalidApiKey
	}

	if apiKey.ExpiresAt != nil && apiKey.ExpiresAt.Before(time.Now().UTC()) {
		return nil, ErrInvalidApiKey
	}

	if err := s.apiKeyRepository.TouchLastUsed(apiKey.ID, time.Now().UTC()); err != nil {
		s.logger.Warn("failed to update api key last_used_at", "api_key_id", apiKey.ID, "error", err)
	}

	return &Principal{
		ApiKeyID:     apiKey.ID,
		Name:         apiKey.Name,
		Role:         apiKey.Role,
		WorkspaceIDs: apiKey.WorkspaceIDs(),
	}, nil
}

func (s *ApiKeyService) TriggerBackupForPrincipal(
	ctx context.Context,
	principal *Principal,
	databaseID uuid.UUID,
) (*backups_core.Backup, error) {
	database, err := s.databaseService.GetDatabaseByID(databaseID)
	if err != nil {
		return nil, ErrDatabaseNotFound
	}

	if database.WorkspaceID == nil {
		return nil, ErrDatabaseWithoutWorkspace
	}

	if !principal.CanAccessWorkspace(*database.WorkspaceID) {
		return nil, ErrForbidden
	}

	backup, triggerErr := s.backupService.TriggerBackupAndWait(
		ctx,
		database,
		config.GetEnv().ApiBackupSyncTimeout,
	)

	s.auditLogService.WriteAuditLog(
		fmt.Sprintf("Backup triggered via API key '%s' for database: %s", principal.Name, database.Name),
		nil,
		database.WorkspaceID,
	)

	return backup, triggerErr
}
