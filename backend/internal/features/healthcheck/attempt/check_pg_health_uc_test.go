package healthcheck_attempt

import (
	"errors"
	"fmt"
	"testing"
	"time"

	"postgresus-backend/internal/features/databases"
	healthcheck_config "postgresus-backend/internal/features/healthcheck/config"
	"postgresus-backend/internal/features/notifiers"
	"postgresus-backend/internal/features/storages"
	users_enums "postgresus-backend/internal/features/users/enums"
	users_testing "postgresus-backend/internal/features/users/testing"
	workspaces_testing "postgresus-backend/internal/features/workspaces/testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func Test_CheckPgHealthUseCase(t *testing.T) {
	user := users_testing.CreateTestUser(users_enums.UserRoleAdmin)

	// Create workspace directly via service
	workspace, err := workspaces_testing.CreateTestWorkspaceDirect("Test Workspace", user.UserID)
	if err != nil {
		t.Fatalf("Failed to create workspace: %v", err)
	}

	storage := storages.CreateTestStorage(workspace.ID)
	notifier := notifiers.CreateTestNotifier(workspace.ID)

	defer func() {
		storages.RemoveTestStorage(storage.ID)
		notifiers.RemoveTestNotifier(notifier)
		workspaces_testing.RemoveTestWorkspaceDirect(workspace.ID)
	}()

	t.Run("Test_DbAttemptFailed_DbMarkedAsUnavailable", func(t *testing.T) {
		database := databases.CreateTestDatabase(workspace.ID, storage, notifier)
		defer databases.RemoveTestDatabase(database)

		// Setup mock notifier sender
		mockSender := &MockHealthcheckAttemptSender{}
		mockSender.On("SendNotification", mock.Anything, mock.Anything, mock.Anything).Return()

		// Setup mock database service
		mockDatabaseService := &MockDatabaseService{}
		mockDatabaseService.On("TestDatabaseConnectionDirect", database).
			Return(errors.New("test error"))
		unavailableStatus := databases.HealthStatusUnavailable
		mockDatabaseService.On("SetHealthStatus", database.ID, &unavailableStatus).
			Return(nil)
		mockDatabaseService.On("GetDatabaseByID", database.ID).
			Return(database, nil)

		// Setup healthcheck config
		healthcheckConfig := &healthcheck_config.HealthcheckConfig{
			DatabaseID:                        database.ID,
			IsHealthcheckEnabled:              true,
			IsSentNotificationWhenUnavailable: true,
			IntervalMinutes:                   1,
			AttemptsBeforeConcideredAsDown:    1,
			StoreAttemptsDays:                 7,
		}

		// Create use case with mock sender
		useCase := &CheckPgHealthUseCase{
			healthcheckAttemptRepository: &HealthcheckAttemptRepository{},
			healthcheckAttemptSender:     mockSender,
			databaseService:              mockDatabaseService,
		}

		// Execute healthcheck
		err := useCase.Execute(time.Now().UTC(), healthcheckConfig)
		assert.NoError(t, err)

		// Verify attempt was created and marked as unavailable
		attempts, err := useCase.healthcheckAttemptRepository.FindByDatabaseIDWithLimit(
			database.ID,
			1,
		)
		assert.NoError(t, err)
		assert.Len(t, attempts, 1)
		assert.Equal(t, databases.HealthStatusUnavailable, attempts[0].Status)

		// Verify database health status was updated (via mock)
		mockDatabaseService.AssertCalled(
			t,
			"SetHealthStatus",
			database.ID,
			&unavailableStatus,
		)

		// Verify notification was sent
		mockSender.AssertCalled(
			t,
			"SendNotification",
			mock.Anything,
			fmt.Sprintf("❌ [%s] DB is unavailable", database.Name),
			fmt.Sprintf("❌ [%s] DB is currently unavailable", database.Name),
		)
	})

	t.Run(
		"Test_DbShouldBeConsideredAsDownOnThirdFailedAttempt_DbNotMarkerdAsDownAfterFirstAttempt",
		func(t *testing.T) {
			database := databases.CreateTestDatabase(workspace.ID, storage, notifier)
			defer databases.RemoveTestDatabase(database)

			// Setup mock notifier sender
			mockSender := &MockHealthcheckAttemptSender{}

			// Setup mock database service - connection fails but SetHealthStatus should not be called
			mockDatabaseService := &MockDatabaseService{}
			mockDatabaseService.On("TestDatabaseConnectionDirect", database).
				Return(errors.New("test error"))
			mockDatabaseService.On("GetDatabaseByID", database.ID).
				Return(database, nil)

			// Setup healthcheck config requiring 3 attempts before marking as down
			healthcheckConfig := &healthcheck_config.HealthcheckConfig{
				DatabaseID:                        database.ID,
				IsHealthcheckEnabled:              true,
				IsSentNotificationWhenUnavailable: true,
				IntervalMinutes:                   1,
				AttemptsBeforeConcideredAsDown:    3,
				StoreAttemptsDays:                 7,
			}

			// Create use case with mock sender
			useCase := &CheckPgHealthUseCase{
				healthcheckAttemptRepository: &HealthcheckAttemptRepository{},
				healthcheckAttemptSender:     mockSender,
				databaseService:              mockDatabaseService,
			}

			// Execute first healthcheck
			err := useCase.Execute(time.Now().UTC(), healthcheckConfig)
			assert.NoError(t, err)

			// Verify attempt was created and marked as unavailable
			attempts, err := useCase.healthcheckAttemptRepository.FindByDatabaseIDWithLimit(
				database.ID,
				1,
			)
			assert.NoError(t, err)
			assert.Len(t, attempts, 1)
			assert.Equal(t, databases.HealthStatusUnavailable, attempts[0].Status)

			// Verify database health status was NOT updated (SetHealthStatus should not be called)
			unavailableStatus := databases.HealthStatusUnavailable
			mockDatabaseService.AssertNotCalled(
				t,
				"SetHealthStatus",
				database.ID,
				&unavailableStatus,
			)

			// Verify no notification was sent (not marked as down yet)
			mockSender.AssertNotCalled(
				t,
				"SendNotification",
				mock.Anything,
				fmt.Sprintf("❌ [%s] DB is unavailable", database.Name),
				fmt.Sprintf("❌ [%s] DB is currently unavailable", database.Name),
			)
		},
	)

	t.Run(
		"Test_DbShouldBeConsideredAsDownOnThirdFailedAttempt_DbMarkerdAsDownAfterThirdFailedAttempt",
		func(t *testing.T) {
			database := databases.CreateTestDatabase(workspace.ID, storage, notifier)
			defer databases.RemoveTestDatabase(database)

			// Make sure DB is available
			available := databases.HealthStatusAvailable
			database.HealthStatus = &available
			err := databases.GetDatabaseService().
				SetHealthStatus(database.ID, &available)
			assert.NoError(t, err)

			// Setup mock notifier sender
			mockSender := &MockHealthcheckAttemptSender{}
			mockSender.On("SendNotification", mock.Anything, mock.Anything, mock.Anything).Return()

			// Setup mock database service
			mockDatabaseService := &MockDatabaseService{}
			mockDatabaseService.On("TestDatabaseConnectionDirect", database).
				Return(errors.New("test error"))
			unavailableStatus := databases.HealthStatusUnavailable
			mockDatabaseService.On("SetHealthStatus", database.ID, &unavailableStatus).
				Return(nil)
			mockDatabaseService.On("GetDatabaseByID", database.ID).
				Return(database, nil)

			// Setup healthcheck config requiring 3 attempts before marking as down
			healthcheckConfig := &healthcheck_config.HealthcheckConfig{
				DatabaseID:                        database.ID,
				IsHealthcheckEnabled:              true,
				IsSentNotificationWhenUnavailable: true,
				IntervalMinutes:                   1,
				AttemptsBeforeConcideredAsDown:    3,
				StoreAttemptsDays:                 7,
			}

			// Create use case with mock sender
			useCase := &CheckPgHealthUseCase{
				healthcheckAttemptRepository: &HealthcheckAttemptRepository{},
				healthcheckAttemptSender:     mockSender,
				databaseService:              mockDatabaseService,
			}

			// Execute three failed healthchecks
			now := time.Now().UTC()
			for i := range 3 {
				err := useCase.Execute(now.Add(time.Duration(i)*time.Minute), healthcheckConfig)
				assert.NoError(t, err)

				// Verify attempt was created
				attempts, err := useCase.healthcheckAttemptRepository.FindByDatabaseIDWithLimit(
					database.ID,
					1,
				)
				assert.NoError(t, err)
				assert.Len(t, attempts, 1)
				assert.Equal(t, databases.HealthStatusUnavailable, attempts[0].Status)
			}

			// Verify database health status was updated to unavailable after 3rd attempt
			mockDatabaseService.AssertCalled(
				t,
				"SetHealthStatus",
				database.ID,
				&unavailableStatus,
			)

			// Verify notification was sent
			mockSender.AssertCalled(
				t,
				"SendNotification",
				mock.Anything,
				fmt.Sprintf("❌ [%s] DB is unavailable", database.Name),
				fmt.Sprintf("❌ [%s] DB is currently unavailable", database.Name),
			)
		},
	)

	t.Run("Test_UnavailableDbAttemptSucceed_DbMarkedAsAvailable", func(t *testing.T) {
		database := databases.CreateTestDatabase(workspace.ID, storage, notifier)
		defer databases.RemoveTestDatabase(database)

		// Make sure DB is unavailable
		unavailable := databases.HealthStatusUnavailable
		database.HealthStatus = &unavailable
		err := databases.GetDatabaseService().
			SetHealthStatus(database.ID, &unavailable)
		assert.NoError(t, err)

		// Setup mock notifier sender
		mockSender := &MockHealthcheckAttemptSender{}
		mockSender.On("SendNotification", mock.Anything, mock.Anything, mock.Anything).Return()

		// Setup mock database service - connection succeeds
		mockDatabaseService := &MockDatabaseService{}
		mockDatabaseService.On("TestDatabaseConnectionDirect", database).Return(nil)
		availableStatus := databases.HealthStatusAvailable
		mockDatabaseService.On("SetHealthStatus", database.ID, &availableStatus).
			Return(nil)
		mockDatabaseService.On("GetDatabaseByID", database.ID).
			Return(database, nil)

		// Setup healthcheck config
		healthcheckConfig := &healthcheck_config.HealthcheckConfig{
			DatabaseID:                        database.ID,
			IsHealthcheckEnabled:              true,
			IsSentNotificationWhenUnavailable: true,
			IntervalMinutes:                   1,
			AttemptsBeforeConcideredAsDown:    1,
			StoreAttemptsDays:                 7,
		}

		// Create use case with mock sender
		useCase := &CheckPgHealthUseCase{
			healthcheckAttemptRepository: &HealthcheckAttemptRepository{},
			healthcheckAttemptSender:     mockSender,
			databaseService:              mockDatabaseService,
		}

		// Execute healthcheck (should succeed)
		err = useCase.Execute(time.Now().UTC(), healthcheckConfig)
		assert.NoError(t, err)

		// Verify attempt was created and marked as available
		attempts, err := useCase.healthcheckAttemptRepository.FindByDatabaseIDWithLimit(
			database.ID,
			1,
		)
		assert.NoError(t, err)
		assert.Len(t, attempts, 1)
		assert.Equal(t, databases.HealthStatusAvailable, attempts[0].Status)

		// Verify database health status was updated to available
		mockDatabaseService.AssertCalled(
			t,
			"SetHealthStatus",
			database.ID,
			&availableStatus,
		)

		// Verify notification was sent for recovery
		mockSender.AssertCalled(
			t,
			"SendNotification",
			mock.Anything,
			fmt.Sprintf("✅ [%s] DB is online", database.Name),
			fmt.Sprintf("✅ [%s] DB is back online", database.Name),
		)
	})

	t.Run(
		"Test_DbHealthcheckExecutedFast_HealthcheckNotExecutedFasterThanInterval",
		func(t *testing.T) {
			database := databases.CreateTestDatabase(workspace.ID, storage, notifier)
			defer databases.RemoveTestDatabase(database)

			// Setup mock notifier sender
			mockSender := &MockHealthcheckAttemptSender{}
			mockSender.On("SendNotification", mock.Anything, mock.Anything, mock.Anything).Return()

			// Setup mock database service - connection succeeds
			mockDatabaseService := &MockDatabaseService{}
			mockDatabaseService.On("TestDatabaseConnectionDirect", database).Return(nil)
			availableStatus := databases.HealthStatusAvailable
			mockDatabaseService.On("SetHealthStatus", database.ID, &availableStatus).
				Return(nil)
			mockDatabaseService.On("GetDatabaseByID", database.ID).
				Return(database, nil)

			// Setup healthcheck config with 5 minute interval
			healthcheckConfig := &healthcheck_config.HealthcheckConfig{
				DatabaseID:                        database.ID,
				IsHealthcheckEnabled:              true,
				IsSentNotificationWhenUnavailable: true,
				IntervalMinutes:                   5, // 5 minute interval
				AttemptsBeforeConcideredAsDown:    1,
				StoreAttemptsDays:                 7,
			}

			// Create use case with mock sender
			useCase := &CheckPgHealthUseCase{
				healthcheckAttemptRepository: &HealthcheckAttemptRepository{},
				healthcheckAttemptSender:     mockSender,
				databaseService:              mockDatabaseService,
			}

			// Execute first healthcheck
			now := time.Now().UTC()
			err := useCase.Execute(now, healthcheckConfig)
			assert.NoError(t, err)

			// Verify first attempt was created
			attempts, err := useCase.healthcheckAttemptRepository.FindByDatabaseIDWithLimit(
				database.ID,
				10,
			)
			assert.NoError(t, err)
			assert.Len(t, attempts, 1)
			assert.Equal(t, databases.HealthStatusAvailable, attempts[0].Status)

			// Try to execute second healthcheck immediately (should be skipped)
			err = useCase.Execute(now.Add(1*time.Second), healthcheckConfig)
			assert.NoError(t, err)

			// Verify no new attempt was created (still only 1 attempt)
			attempts, err = useCase.healthcheckAttemptRepository.FindByDatabaseIDWithLimit(
				database.ID,
				10,
			)
			assert.NoError(t, err)
			assert.Len(
				t,
				attempts,
				1,
				"Second healthcheck should not have been executed due to interval constraint",
			)

			// Try to execute third healthcheck after 4 minutes (still too early)
			err = useCase.Execute(now.Add(4*time.Minute), healthcheckConfig)
			assert.NoError(t, err)

			// Verify still only 1 attempt
			attempts, err = useCase.healthcheckAttemptRepository.FindByDatabaseIDWithLimit(
				database.ID,
				10,
			)
			assert.NoError(t, err)
			assert.Len(
				t,
				attempts,
				1,
				"Third healthcheck should not have been executed due to interval constraint",
			)

			// Execute fourth healthcheck after 5 minutes (should be executed)
			err = useCase.Execute(now.Add(5*time.Minute), healthcheckConfig)
			assert.NoError(t, err)

			// Verify new attempt was created (now should be 2 attempts)
			attempts, err = useCase.healthcheckAttemptRepository.FindByDatabaseIDWithLimit(
				database.ID,
				10,
			)
			assert.NoError(t, err)
			assert.Len(
				t,
				attempts,
				2,
				"Fourth healthcheck should have been executed after interval passed",
			)
			assert.Equal(t, databases.HealthStatusAvailable, attempts[0].Status)
			assert.Equal(t, databases.HealthStatusAvailable, attempts[1].Status)
		},
	)
}
