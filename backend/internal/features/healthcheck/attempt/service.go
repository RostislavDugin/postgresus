package healthcheck_attempt

import (
	"errors"
	"postgresus-backend/internal/features/databases"
	users_models "postgresus-backend/internal/features/users/models"
	workspaces_services "postgresus-backend/internal/features/workspaces/services"
	"time"

	"github.com/google/uuid"
)

type HealthcheckAttemptService struct {
	healthcheckAttemptRepository *HealthcheckAttemptRepository
	databaseService              *databases.DatabaseService
	workspaceService             *workspaces_services.WorkspaceService
}

func (s *HealthcheckAttemptService) GetAttemptsByDatabase(
	user users_models.User,
	databaseID uuid.UUID,
	afterDate time.Time,
) ([]*HealthcheckAttempt, error) {
	database, err := s.databaseService.GetDatabaseByID(databaseID)
	if err != nil {
		return nil, err
	}

	if database.WorkspaceID == nil {
		return nil, errors.New("cannot access healthcheck attempts for databases without workspace")
	}

	canAccess, _, err := s.workspaceService.CanUserAccessWorkspace(*database.WorkspaceID, &user)
	if err != nil {
		return nil, err
	}
	if !canAccess {
		return nil, errors.New("forbidden")
	}

	return s.healthcheckAttemptRepository.FindByDatabaseIdOrderByCreatedAtDesc(
		databaseID,
		afterDate,
	)
}
