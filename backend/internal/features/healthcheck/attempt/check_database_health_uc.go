package healthcheck_attempt

import (
	"errors"
	"fmt"
	"log/slog"
	"postgresus-backend/internal/features/databases"
	healthcheck_config "postgresus-backend/internal/features/healthcheck/config"
	"postgresus-backend/internal/util/logger"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
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

	err = uc.validateDatabase(database)
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

	heathcheckAttempt, err := uc.healthcheckDatabase(now, database)
	if err != nil {
		return err
	}

	// Save the attempt
	err = uc.healthcheckAttemptRepository.Insert(heathcheckAttempt)
	if err != nil {
		return err
	}

	err = uc.updateDatabaseHealthStatusIfChanged(
		database,
		healthcheckConfig,
		heathcheckAttempt,
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
	heathcheckAttempt *HealthcheckAttempt,
) error {
	if &heathcheckAttempt.Status == database.HealthStatus {
		return nil
	}

	if (database.HealthStatus == nil ||
		*database.HealthStatus == databases.HealthStatusUnavailable) &&
		heathcheckAttempt.Status == databases.HealthStatusAvailable {
		err := uc.databaseService.SetHealthStatus(
			database.ID,
			&heathcheckAttempt.Status,
		)
		if err != nil {
			return err
		}

		uc.sendDbStatusNotification(
			healthcheckConfig,
			database,
			heathcheckAttempt.Status,
		)
	}

	if (database.HealthStatus == nil ||
		*database.HealthStatus == databases.HealthStatusAvailable) &&
		heathcheckAttempt.Status == databases.HealthStatusUnavailable {
		if healthcheckConfig.AttemptsBeforeConcideredAsDown <= 1 {
			// proceed, 1 fail is enough to consider db as down
		} else {
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
			&heathcheckAttempt.Status,
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
	// Test the connection
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

	// Create health check attempt
	attempt := &HealthcheckAttempt{
		ID:         uuid.New(),
		DatabaseID: database.ID,
		Status:     healthStatus,
		CreatedAt:  now,
	}

	return attempt, nil
}

func (uc *CheckDatabaseHealthUseCase) validateDatabase(
	database *databases.Database,
) error {
	switch database.Type {
	case databases.DatabaseTypePostgres:
		if database.Postgresql == nil {
			return fmt.Errorf("database Postgresql config is not set")
		}
	case databases.DatabaseTypeMysql:
		if database.Mysql == nil {
			return fmt.Errorf("database MySQL config is not set")
		}
	case databases.DatabaseTypeMariadb:
		if database.Mariadb == nil {
			return fmt.Errorf("database MariaDB config is not set")
		}
	default:
		return fmt.Errorf("unsupported database type: %s", database.Type)
	}

	return nil
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

	messageTitle := ""
	messageBody := ""

	if newHealthStatus == databases.HealthStatusAvailable {
		messageTitle = fmt.Sprintf("✅ [%s] DB is online", database.Name)
		messageBody = fmt.Sprintf("✅ [%s] DB is back online", database.Name)
	} else {
		messageTitle = fmt.Sprintf("❌ [%s] DB is unavailable", database.Name)
		messageBody = fmt.Sprintf("❌ [%s] DB is currently unavailable", database.Name)
	}

	for _, notifier := range database.Notifiers {
		uc.healthcheckAttemptSender.SendNotification(
			&notifier,
			messageTitle,
			messageBody,
		)
	}

}
