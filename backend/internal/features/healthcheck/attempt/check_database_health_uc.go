package healthcheck_attempt

import (
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"databasus-backend/internal/features/databases"
	healthcheck_config "databasus-backend/internal/features/healthcheck/config"
	notifier_models "databasus-backend/internal/features/notifiers/models"
	"databasus-backend/internal/util/logger"
)

type CheckDatabaseHealthUseCase struct {
	healthcheckAttemptRepository *HealthcheckAttemptRepository
	healthcheckAttemptSender     HealthcheckAttemptSender
	databaseService              DatabaseService
}

func (uc *CheckDatabaseHealthUseCase) Execute(
	now time.Time,
	healthcheckConfig *healthcheck_config.HealthcheckConfig,
) error {
	database, err := uc.databaseService.GetDatabaseByID(healthcheckConfig.DatabaseID)
	if err != nil {
		return err
	}

	isExecuteNewAttempt, err := uc.isReadyForNewAttempt(
		now,
		database,
		healthcheckConfig,
	)
	if err != nil {
		return err
	}

	if !isExecuteNewAttempt {
		return nil
	}

	healthcheckAttempt, err := uc.healthcheckDatabase(now, database)
	if err != nil {
		return err
	}

	err = uc.healthcheckAttemptRepository.Insert(healthcheckAttempt)
	if err != nil {
		return err
	}

	err = uc.updateDatabaseHealthStatusIfChanged(
		database,
		healthcheckConfig,
		healthcheckAttempt,
	)
	if err != nil {
		return err
	}

	err = uc.healthcheckAttemptRepository.DeleteOlderThan(
		database.ID,
		time.Now().UTC().Add(-time.Duration(healthcheckConfig.StoreAttemptsDays)*24*time.Hour),
	)
	if err != nil {
		return err
	}

	return nil
}

func (uc *CheckDatabaseHealthUseCase) updateDatabaseHealthStatusIfChanged(
	database *databases.Database,
	healthcheckConfig *healthcheck_config.HealthcheckConfig,
	healthcheckAttempt *HealthcheckAttempt,
) error {
	if database.HealthStatus != nil && *database.HealthStatus == healthcheckAttempt.Status {
		return nil
	}

	if (database.HealthStatus == nil ||
		*database.HealthStatus == databases.HealthStatusUnavailable) &&
		healthcheckAttempt.Status == databases.HealthStatusAvailable {
		err := uc.databaseService.SetHealthStatus(
			database.ID,
			&healthcheckAttempt.Status,
		)
		if err != nil {
			return err
		}

		uc.sendDbStatusNotification(
			healthcheckConfig,
			database,
			healthcheckAttempt.Status,
		)
	}

	if (database.HealthStatus == nil ||
		*database.HealthStatus == databases.HealthStatusAvailable) &&
		healthcheckAttempt.Status == databases.HealthStatusUnavailable {
		if healthcheckConfig.AttemptsBeforeConcideredAsDown > 1 {
			lastHealthcheckAttempts, err := uc.healthcheckAttemptRepository.FindByDatabaseIDWithLimit(
				database.ID,
				healthcheckConfig.AttemptsBeforeConcideredAsDown,
			)
			if err != nil {
				return err
			}

			if len(lastHealthcheckAttempts) < healthcheckConfig.AttemptsBeforeConcideredAsDown {
				return nil
			}

			for _, attempt := range lastHealthcheckAttempts {
				if attempt.Status == databases.HealthStatusAvailable {
					return nil
				}
			}
		}

		err := uc.databaseService.SetHealthStatus(
			database.ID,
			&healthcheckAttempt.Status,
		)
		if err != nil {
			return err
		}

		uc.sendDbStatusNotification(
			healthcheckConfig,
			database,
			databases.HealthStatusUnavailable,
		)
	}

	return nil
}

func (uc *CheckDatabaseHealthUseCase) healthcheckDatabase(
	now time.Time,
	database *databases.Database,
) (*HealthcheckAttempt, error) {
	healthStatus := databases.HealthStatusAvailable
	err := uc.databaseService.TestDatabaseConnectionDirect(database)
	if err != nil {
		healthStatus = databases.HealthStatusUnavailable
		logger.GetLogger().
			Error(
				"Database health check failed",
				slog.String("database_id", database.ID.String()),
				slog.String("error", err.Error()),
			)
	}

	attempt := &HealthcheckAttempt{
		ID:         uuid.New(),
		DatabaseID: database.ID,
		Status:     healthStatus,
		CreatedAt:  now,
	}

	return attempt, nil
}

func (uc *CheckDatabaseHealthUseCase) isReadyForNewAttempt(
	now time.Time,
	database *databases.Database,
	healthcheckConfig *healthcheck_config.HealthcheckConfig,
) (bool, error) {
	lastHealthcheckAttempt, err := uc.healthcheckAttemptRepository.FindLastByDatabaseID(database.ID)
	if err != nil {
		// If no attempts found, it's ready for first attempt
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return true, nil
		}

		return false, err
	}

	// Check if enough time has passed since last attempt
	intervalDuration := time.Duration(healthcheckConfig.IntervalMinutes) * time.Minute
	nextAttemptTime := lastHealthcheckAttempt.CreatedAt.Add(intervalDuration)

	return now.After(nextAttemptTime.Add(-1 * time.Second)), nil
}

func (uc *CheckDatabaseHealthUseCase) sendDbStatusNotification(
	healthcheckConfig *healthcheck_config.HealthcheckConfig,
	database *databases.Database,
	newHealthStatus databases.HealthStatus,
) {
	if !healthcheckConfig.IsSentNotificationWhenUnavailable {
		return
	}

	notification := notifier_models.Notification{
		Type:    notifier_models.NotificationTypeHealthcheckFailed,
		Heading: fmt.Sprintf("❌ [%s] DB is unavailable", database.Name),
		Message: fmt.Sprintf("❌ [%s] DB is currently unavailable", database.Name),
	}

	if newHealthStatus == databases.HealthStatusAvailable {
		notification = notifier_models.Notification{
			Type:    notifier_models.NotificationTypeHealthcheckSuccess,
			Heading: fmt.Sprintf("✅ [%s] DB is online", database.Name),
			Message: fmt.Sprintf("✅ [%s] DB is back online", database.Name),
		}
	}

	for _, notifier := range database.Notifiers {
		uc.healthcheckAttemptSender.SendNotification(&notifier, notification)
	}
}
