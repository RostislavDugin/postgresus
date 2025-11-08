package notifiers

import (
	"fmt"
	"net/http"
	"testing"

	"postgresus-backend/internal/config"
	audit_logs "postgresus-backend/internal/features/audit_logs"
	discord_notifier "postgresus-backend/internal/features/notifiers/models/discord"
	email_notifier "postgresus-backend/internal/features/notifiers/models/email_notifier"
	slack_notifier "postgresus-backend/internal/features/notifiers/models/slack"
	teams_notifier "postgresus-backend/internal/features/notifiers/models/teams"
	telegram_notifier "postgresus-backend/internal/features/notifiers/models/telegram"
	webhook_notifier "postgresus-backend/internal/features/notifiers/models/webhook"
	users_enums "postgresus-backend/internal/features/users/enums"
	users_middleware "postgresus-backend/internal/features/users/middleware"
	users_services "postgresus-backend/internal/features/users/services"
	users_testing "postgresus-backend/internal/features/users/testing"
	workspaces_controllers "postgresus-backend/internal/features/workspaces/controllers"
	workspaces_testing "postgresus-backend/internal/features/workspaces/testing"
	test_utils "postgresus-backend/internal/util/testing"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

func Test_SaveNewNotifier_NotifierReturnedViaGet(t *testing.T) {
	owner := users_testing.CreateTestUser(users_enums.UserRoleMember)
	router := createRouter()
	workspace := workspaces_testing.CreateTestWorkspace("Test Workspace", owner, router)

	notifier := createNewNotifier(workspace.ID)

	var savedNotifier Notifier
	test_utils.MakePostRequestAndUnmarshal(
		t,
		router,
		"/api/v1/notifiers",
		"Bearer "+owner.Token,
		*notifier,
		http.StatusOK,
		&savedNotifier,
	)

	verifyNotifierData(t, notifier, &savedNotifier)
	assert.NotEmpty(t, savedNotifier.ID)

	// Verify notifier is returned via GET
	var retrievedNotifier Notifier
	test_utils.MakeGetRequestAndUnmarshal(
		t,
		router,
		fmt.Sprintf("/api/v1/notifiers/%s", savedNotifier.ID.String()),
		"Bearer "+owner.Token,
		http.StatusOK,
		&retrievedNotifier,
	)

	verifyNotifierData(t, &savedNotifier, &retrievedNotifier)

	// Verify notifier is returned via GET all notifiers
	var notifiers []Notifier
	test_utils.MakeGetRequestAndUnmarshal(
		t,
		router,
		fmt.Sprintf("/api/v1/notifiers?workspace_id=%s", workspace.ID.String()),
		"Bearer "+owner.Token,
		http.StatusOK,
		&notifiers,
	)

	assert.Len(t, notifiers, 1)

	deleteNotifier(t, router, savedNotifier.ID, workspace.ID, owner.Token)
	workspaces_testing.RemoveTestWorkspace(workspace, router)
}

func Test_UpdateExistingNotifier_UpdatedNotifierReturnedViaGet(t *testing.T) {
	owner := users_testing.CreateTestUser(users_enums.UserRoleMember)
	router := createRouter()
	workspace := workspaces_testing.CreateTestWorkspace("Test Workspace", owner, router)

	notifier := createNewNotifier(workspace.ID)

	var savedNotifier Notifier
	test_utils.MakePostRequestAndUnmarshal(
		t,
		router,
		"/api/v1/notifiers",
		"Bearer "+owner.Token,
		*notifier,
		http.StatusOK,
		&savedNotifier,
	)

	updatedName := "Updated Notifier " + uuid.New().String()
	savedNotifier.Name = updatedName

	var updatedNotifier Notifier
	test_utils.MakePostRequestAndUnmarshal(
		t,
		router,
		"/api/v1/notifiers",
		"Bearer "+owner.Token,
		savedNotifier,
		http.StatusOK,
		&updatedNotifier,
	)

	assert.Equal(t, updatedName, updatedNotifier.Name)
	assert.Equal(t, savedNotifier.ID, updatedNotifier.ID)

	deleteNotifier(t, router, updatedNotifier.ID, workspace.ID, owner.Token)
	workspaces_testing.RemoveTestWorkspace(workspace, router)
}

func Test_DeleteNotifier_NotifierNotReturnedViaGet(t *testing.T) {
	owner := users_testing.CreateTestUser(users_enums.UserRoleMember)
	router := createRouter()
	workspace := workspaces_testing.CreateTestWorkspace("Test Workspace", owner, router)

	notifier := createNewNotifier(workspace.ID)

	var savedNotifier Notifier
	test_utils.MakePostRequestAndUnmarshal(
		t,
		router,
		"/api/v1/notifiers",
		"Bearer "+owner.Token,
		*notifier,
		http.StatusOK,
		&savedNotifier,
	)

	test_utils.MakeDeleteRequest(
		t,
		router,
		fmt.Sprintf("/api/v1/notifiers/%s", savedNotifier.ID.String()),
		"Bearer "+owner.Token,
		http.StatusOK,
	)

	response := test_utils.MakeGetRequest(
		t,
		router,
		fmt.Sprintf("/api/v1/notifiers/%s", savedNotifier.ID.String()),
		"Bearer "+owner.Token,
		http.StatusBadRequest,
	)

	assert.Contains(t, string(response.Body), "error")
	workspaces_testing.RemoveTestWorkspace(workspace, router)
}

func Test_SendTestNotificationDirect_NotificationSent(t *testing.T) {
	owner := users_testing.CreateTestUser(users_enums.UserRoleMember)
	router := createRouter()
	workspace := workspaces_testing.CreateTestWorkspace("Test Workspace", owner, router)

	notifier := createTelegramNotifier(workspace.ID)

	response := test_utils.MakePostRequest(
		t, router, "/api/v1/notifiers/direct-test", "Bearer "+owner.Token, *notifier, http.StatusOK,
	)

	assert.Contains(t, string(response.Body), "successful")
	workspaces_testing.RemoveTestWorkspace(workspace, router)
}

func Test_SendTestNotificationExisting_NotificationSent(t *testing.T) {
	owner := users_testing.CreateTestUser(users_enums.UserRoleMember)
	router := createRouter()
	workspace := workspaces_testing.CreateTestWorkspace("Test Workspace", owner, router)

	notifier := createTelegramNotifier(workspace.ID)

	var savedNotifier Notifier
	test_utils.MakePostRequestAndUnmarshal(
		t,
		router,
		"/api/v1/notifiers",
		"Bearer "+owner.Token,
		*notifier,
		http.StatusOK,
		&savedNotifier,
	)

	response := test_utils.MakePostRequest(
		t,
		router,
		fmt.Sprintf("/api/v1/notifiers/%s/test", savedNotifier.ID.String()),
		"Bearer "+owner.Token,
		nil,
		http.StatusOK,
	)

	assert.Contains(t, string(response.Body), "successful")

	deleteNotifier(t, router, savedNotifier.ID, workspace.ID, owner.Token)
	workspaces_testing.RemoveTestWorkspace(workspace, router)
}

func Test_ViewerCanViewNotifiers_ButCannotModify(t *testing.T) {
	owner := users_testing.CreateTestUser(users_enums.UserRoleMember)
	viewer := users_testing.CreateTestUser(users_enums.UserRoleMember)
	router := createRouter()
	workspace := workspaces_testing.CreateTestWorkspace("Test Workspace", owner, router)
	workspaces_testing.AddMemberToWorkspace(
		workspace,
		viewer,
		users_enums.WorkspaceRoleViewer,
		owner.Token,
		router,
	)

	notifier := createNewNotifier(workspace.ID)

	var savedNotifier Notifier
	test_utils.MakePostRequestAndUnmarshal(
		t,
		router,
		"/api/v1/notifiers",
		"Bearer "+owner.Token,
		*notifier,
		http.StatusOK,
		&savedNotifier,
	)

	// Viewer can GET notifiers
	var notifiers []Notifier
	test_utils.MakeGetRequestAndUnmarshal(
		t,
		router,
		fmt.Sprintf("/api/v1/notifiers?workspace_id=%s", workspace.ID.String()),
		"Bearer "+viewer.Token,
		http.StatusOK,
		&notifiers,
	)
	assert.Len(t, notifiers, 1)

	// Viewer cannot CREATE notifier
	newNotifier := createNewNotifier(workspace.ID)
	test_utils.MakePostRequest(
		t, router, "/api/v1/notifiers", "Bearer "+viewer.Token, *newNotifier, http.StatusForbidden,
	)

	// Viewer cannot UPDATE notifier
	savedNotifier.Name = "Updated by viewer"
	test_utils.MakePostRequest(
		t, router, "/api/v1/notifiers", "Bearer "+viewer.Token, savedNotifier, http.StatusForbidden,
	)

	// Viewer cannot DELETE notifier
	test_utils.MakeDeleteRequest(
		t,
		router,
		fmt.Sprintf("/api/v1/notifiers/%s", savedNotifier.ID.String()),
		"Bearer "+viewer.Token,
		http.StatusForbidden,
	)

	deleteNotifier(t, router, savedNotifier.ID, workspace.ID, owner.Token)
	workspaces_testing.RemoveTestWorkspace(workspace, router)
}

func Test_MemberCanManageNotifiers(t *testing.T) {
	owner := users_testing.CreateTestUser(users_enums.UserRoleMember)
	member := users_testing.CreateTestUser(users_enums.UserRoleMember)
	router := createRouter()
	workspace := workspaces_testing.CreateTestWorkspace("Test Workspace", owner, router)
	workspaces_testing.AddMemberToWorkspace(
		workspace,
		member,
		users_enums.WorkspaceRoleMember,
		owner.Token,
		router,
	)

	notifier := createNewNotifier(workspace.ID)

	// Member can CREATE notifier
	var savedNotifier Notifier
	test_utils.MakePostRequestAndUnmarshal(
		t,
		router,
		"/api/v1/notifiers",
		"Bearer "+member.Token,
		*notifier,
		http.StatusOK,
		&savedNotifier,
	)
	assert.NotEmpty(t, savedNotifier.ID)

	// Member can UPDATE notifier
	savedNotifier.Name = "Updated by member"
	var updatedNotifier Notifier
	test_utils.MakePostRequestAndUnmarshal(
		t,
		router,
		"/api/v1/notifiers",
		"Bearer "+member.Token,
		savedNotifier,
		http.StatusOK,
		&updatedNotifier,
	)
	assert.Equal(t, "Updated by member", updatedNotifier.Name)

	// Member can DELETE notifier
	test_utils.MakeDeleteRequest(
		t,
		router,
		fmt.Sprintf("/api/v1/notifiers/%s", savedNotifier.ID.String()),
		"Bearer "+member.Token,
		http.StatusOK,
	)

	workspaces_testing.RemoveTestWorkspace(workspace, router)
}

func Test_AdminCanManageNotifiers(t *testing.T) {
	owner := users_testing.CreateTestUser(users_enums.UserRoleMember)
	admin := users_testing.CreateTestUser(users_enums.UserRoleMember)
	router := createRouter()
	workspace := workspaces_testing.CreateTestWorkspace("Test Workspace", owner, router)
	workspaces_testing.AddMemberToWorkspace(
		workspace,
		admin,
		users_enums.WorkspaceRoleAdmin,
		owner.Token,
		router,
	)

	notifier := createNewNotifier(workspace.ID)

	// Admin can CREATE, UPDATE, DELETE
	var savedNotifier Notifier
	test_utils.MakePostRequestAndUnmarshal(
		t,
		router,
		"/api/v1/notifiers",
		"Bearer "+admin.Token,
		*notifier,
		http.StatusOK,
		&savedNotifier,
	)

	savedNotifier.Name = "Updated by admin"
	test_utils.MakePostRequest(
		t, router, "/api/v1/notifiers", "Bearer "+admin.Token, savedNotifier, http.StatusOK,
	)

	test_utils.MakeDeleteRequest(
		t,
		router,
		fmt.Sprintf("/api/v1/notifiers/%s", savedNotifier.ID.String()),
		"Bearer "+admin.Token,
		http.StatusOK,
	)

	workspaces_testing.RemoveTestWorkspace(workspace, router)
}

func Test_UserNotInWorkspace_CannotAccessNotifiers(t *testing.T) {
	owner := users_testing.CreateTestUser(users_enums.UserRoleMember)
	outsider := users_testing.CreateTestUser(users_enums.UserRoleMember)
	router := createRouter()
	workspace := workspaces_testing.CreateTestWorkspace("Test Workspace", owner, router)

	notifier := createNewNotifier(workspace.ID)

	var savedNotifier Notifier
	test_utils.MakePostRequestAndUnmarshal(
		t,
		router,
		"/api/v1/notifiers",
		"Bearer "+owner.Token,
		*notifier,
		http.StatusOK,
		&savedNotifier,
	)

	// Outsider cannot GET notifiers
	test_utils.MakeGetRequest(
		t,
		router,
		fmt.Sprintf("/api/v1/notifiers?workspace_id=%s", workspace.ID.String()),
		"Bearer "+outsider.Token,
		http.StatusForbidden,
	)

	// Outsider cannot CREATE notifier
	test_utils.MakePostRequest(
		t, router, "/api/v1/notifiers", "Bearer "+outsider.Token, *notifier, http.StatusForbidden,
	)

	// Outsider cannot UPDATE notifier
	test_utils.MakePostRequest(
		t,
		router,
		"/api/v1/notifiers",
		"Bearer "+outsider.Token,
		savedNotifier,
		http.StatusForbidden,
	)

	// Outsider cannot DELETE notifier
	test_utils.MakeDeleteRequest(
		t,
		router,
		fmt.Sprintf("/api/v1/notifiers/%s", savedNotifier.ID.String()),
		"Bearer "+outsider.Token,
		http.StatusForbidden,
	)

	deleteNotifier(t, router, savedNotifier.ID, workspace.ID, owner.Token)
	workspaces_testing.RemoveTestWorkspace(workspace, router)
}

func Test_CrossWorkspaceSecurity_CannotAccessNotifierFromAnotherWorkspace(t *testing.T) {
	owner1 := users_testing.CreateTestUser(users_enums.UserRoleMember)
	owner2 := users_testing.CreateTestUser(users_enums.UserRoleMember)
	router := createRouter()
	workspace1 := workspaces_testing.CreateTestWorkspace("Workspace 1", owner1, router)
	workspace2 := workspaces_testing.CreateTestWorkspace("Workspace 2", owner2, router)

	notifier1 := createNewNotifier(workspace1.ID)

	var savedNotifier Notifier
	test_utils.MakePostRequestAndUnmarshal(
		t,
		router,
		"/api/v1/notifiers",
		"Bearer "+owner1.Token,
		*notifier1,
		http.StatusOK,
		&savedNotifier,
	)

	// Try to access workspace1's notifier with owner2 from workspace2
	response := test_utils.MakeGetRequest(
		t,
		router,
		fmt.Sprintf("/api/v1/notifiers/%s", savedNotifier.ID.String()),
		"Bearer "+owner2.Token,
		http.StatusForbidden,
	)
	assert.Contains(t, string(response.Body), "insufficient permissions")

	deleteNotifier(t, router, savedNotifier.ID, workspace1.ID, owner1.Token)
	workspaces_testing.RemoveTestWorkspace(workspace1, router)
	workspaces_testing.RemoveTestWorkspace(workspace2, router)
}

func createRouter() *gin.Engine {
	gin.SetMode(gin.TestMode)
	router := gin.New()

	v1 := router.Group("/api/v1")
	protected := v1.Group("").Use(users_middleware.AuthMiddleware(users_services.GetUserService()))

	if routerGroup, ok := protected.(*gin.RouterGroup); ok {
		GetNotifierController().RegisterRoutes(routerGroup)
		workspaces_controllers.GetWorkspaceController().RegisterRoutes(routerGroup)
		workspaces_controllers.GetMembershipController().RegisterRoutes(routerGroup)
	}

	audit_logs.SetupDependencies()

	return router
}

func createNewNotifier(workspaceID uuid.UUID) *Notifier {
	return &Notifier{
		WorkspaceID:  workspaceID,
		Name:         "Test Notifier " + uuid.New().String(),
		NotifierType: NotifierTypeWebhook,
		WebhookNotifier: &webhook_notifier.WebhookNotifier{
			WebhookURL:    "https://webhook.site/test-" + uuid.New().String(),
			WebhookMethod: webhook_notifier.WebhookMethodPOST,
		},
	}
}

func createTelegramNotifier(workspaceID uuid.UUID) *Notifier {
	env := config.GetEnv()
	return &Notifier{
		WorkspaceID:  workspaceID,
		Name:         "Test Telegram Notifier " + uuid.New().String(),
		NotifierType: NotifierTypeTelegram,
		TelegramNotifier: &telegram_notifier.TelegramNotifier{
			BotToken:     env.TestTelegramBotToken,
			TargetChatID: env.TestTelegramChatID,
		},
	}
}

func verifyNotifierData(t *testing.T, expected *Notifier, actual *Notifier) {
	assert.Equal(t, expected.Name, actual.Name)
	assert.Equal(t, expected.NotifierType, actual.NotifierType)
	assert.Equal(t, expected.WorkspaceID, actual.WorkspaceID)
}

func deleteNotifier(
	t *testing.T,
	router *gin.Engine,
	notifierID, workspaceID uuid.UUID,
	token string,
) {
	test_utils.MakeDeleteRequest(
		t,
		router,
		fmt.Sprintf("/api/v1/notifiers/%s", notifierID.String()),
		"Bearer "+token,
		http.StatusOK,
	)
}

func Test_NotifierSensitiveDataLifecycle_AllTypes(t *testing.T) {
	testCases := []struct {
		name                string
		notifierType        NotifierType
		createNotifier      func(workspaceID uuid.UUID) *Notifier
		updateNotifier      func(workspaceID uuid.UUID, notifierID uuid.UUID) *Notifier
		verifySensitiveData func(t *testing.T, notifier *Notifier)
		verifyHiddenData    func(t *testing.T, notifier *Notifier)
	}{
		{
			name:         "Telegram Notifier",
			notifierType: NotifierTypeTelegram,
			createNotifier: func(workspaceID uuid.UUID) *Notifier {
				return &Notifier{
					WorkspaceID:  workspaceID,
					Name:         "Test Telegram Notifier",
					NotifierType: NotifierTypeTelegram,
					TelegramNotifier: &telegram_notifier.TelegramNotifier{
						BotToken:     "original-bot-token-12345",
						TargetChatID: "123456789",
					},
				}
			},
			updateNotifier: func(workspaceID uuid.UUID, notifierID uuid.UUID) *Notifier {
				return &Notifier{
					ID:           notifierID,
					WorkspaceID:  workspaceID,
					Name:         "Updated Telegram Notifier",
					NotifierType: NotifierTypeTelegram,
					TelegramNotifier: &telegram_notifier.TelegramNotifier{
						BotToken:     "",
						TargetChatID: "987654321",
					},
				}
			},
			verifySensitiveData: func(t *testing.T, notifier *Notifier) {
				assert.Equal(t, "original-bot-token-12345", notifier.TelegramNotifier.BotToken)
			},
			verifyHiddenData: func(t *testing.T, notifier *Notifier) {
				assert.Equal(t, "", notifier.TelegramNotifier.BotToken)
			},
		},
		{
			name:         "Email Notifier",
			notifierType: NotifierTypeEmail,
			createNotifier: func(workspaceID uuid.UUID) *Notifier {
				return &Notifier{
					WorkspaceID:  workspaceID,
					Name:         "Test Email Notifier",
					NotifierType: NotifierTypeEmail,
					EmailNotifier: &email_notifier.EmailNotifier{
						TargetEmail:  "test@example.com",
						SMTPHost:     "smtp.example.com",
						SMTPPort:     587,
						SMTPUser:     "user@example.com",
						SMTPPassword: "original-password-secret",
					},
				}
			},
			updateNotifier: func(workspaceID uuid.UUID, notifierID uuid.UUID) *Notifier {
				return &Notifier{
					ID:           notifierID,
					WorkspaceID:  workspaceID,
					Name:         "Updated Email Notifier",
					NotifierType: NotifierTypeEmail,
					EmailNotifier: &email_notifier.EmailNotifier{
						TargetEmail:  "updated@example.com",
						SMTPHost:     "smtp.newhost.com",
						SMTPPort:     465,
						SMTPUser:     "newuser@example.com",
						SMTPPassword: "",
					},
				}
			},
			verifySensitiveData: func(t *testing.T, notifier *Notifier) {
				assert.Equal(t, "original-password-secret", notifier.EmailNotifier.SMTPPassword)
			},
			verifyHiddenData: func(t *testing.T, notifier *Notifier) {
				assert.Equal(t, "", notifier.EmailNotifier.SMTPPassword)
			},
		},
		{
			name:         "Slack Notifier",
			notifierType: NotifierTypeSlack,
			createNotifier: func(workspaceID uuid.UUID) *Notifier {
				return &Notifier{
					WorkspaceID:  workspaceID,
					Name:         "Test Slack Notifier",
					NotifierType: NotifierTypeSlack,
					SlackNotifier: &slack_notifier.SlackNotifier{
						BotToken:     "xoxb-original-slack-token",
						TargetChatID: "C123456",
					},
				}
			},
			updateNotifier: func(workspaceID uuid.UUID, notifierID uuid.UUID) *Notifier {
				return &Notifier{
					ID:           notifierID,
					WorkspaceID:  workspaceID,
					Name:         "Updated Slack Notifier",
					NotifierType: NotifierTypeSlack,
					SlackNotifier: &slack_notifier.SlackNotifier{
						BotToken:     "",
						TargetChatID: "C789012",
					},
				}
			},
			verifySensitiveData: func(t *testing.T, notifier *Notifier) {
				assert.Equal(t, "xoxb-original-slack-token", notifier.SlackNotifier.BotToken)
			},
			verifyHiddenData: func(t *testing.T, notifier *Notifier) {
				assert.Equal(t, "", notifier.SlackNotifier.BotToken)
			},
		},
		{
			name:         "Discord Notifier",
			notifierType: NotifierTypeDiscord,
			createNotifier: func(workspaceID uuid.UUID) *Notifier {
				return &Notifier{
					WorkspaceID:  workspaceID,
					Name:         "Test Discord Notifier",
					NotifierType: NotifierTypeDiscord,
					DiscordNotifier: &discord_notifier.DiscordNotifier{
						ChannelWebhookURL: "https://discord.com/api/webhooks/123/original-token",
					},
				}
			},
			updateNotifier: func(workspaceID uuid.UUID, notifierID uuid.UUID) *Notifier {
				return &Notifier{
					ID:           notifierID,
					WorkspaceID:  workspaceID,
					Name:         "Updated Discord Notifier",
					NotifierType: NotifierTypeDiscord,
					DiscordNotifier: &discord_notifier.DiscordNotifier{
						ChannelWebhookURL: "",
					},
				}
			},
			verifySensitiveData: func(t *testing.T, notifier *Notifier) {
				assert.Equal(
					t,
					"https://discord.com/api/webhooks/123/original-token",
					notifier.DiscordNotifier.ChannelWebhookURL,
				)
			},
			verifyHiddenData: func(t *testing.T, notifier *Notifier) {
				assert.Equal(t, "", notifier.DiscordNotifier.ChannelWebhookURL)
			},
		},
		{
			name:         "Teams Notifier",
			notifierType: NotifierTypeTeams,
			createNotifier: func(workspaceID uuid.UUID) *Notifier {
				return &Notifier{
					WorkspaceID:  workspaceID,
					Name:         "Test Teams Notifier",
					NotifierType: NotifierTypeTeams,
					TeamsNotifier: &teams_notifier.TeamsNotifier{
						WebhookURL: "https://outlook.office.com/webhook/original-token",
					},
				}
			},
			updateNotifier: func(workspaceID uuid.UUID, notifierID uuid.UUID) *Notifier {
				return &Notifier{
					ID:           notifierID,
					WorkspaceID:  workspaceID,
					Name:         "Updated Teams Notifier",
					NotifierType: NotifierTypeTeams,
					TeamsNotifier: &teams_notifier.TeamsNotifier{
						WebhookURL: "",
					},
				}
			},
			verifySensitiveData: func(t *testing.T, notifier *Notifier) {
				assert.Equal(
					t,
					"https://outlook.office.com/webhook/original-token",
					notifier.TeamsNotifier.WebhookURL,
				)
			},
			verifyHiddenData: func(t *testing.T, notifier *Notifier) {
				assert.Equal(t, "", notifier.TeamsNotifier.WebhookURL)
			},
		},
		{
			name:         "Webhook Notifier",
			notifierType: NotifierTypeWebhook,
			createNotifier: func(workspaceID uuid.UUID) *Notifier {
				return &Notifier{
					WorkspaceID:  workspaceID,
					Name:         "Test Webhook Notifier",
					NotifierType: NotifierTypeWebhook,
					WebhookNotifier: &webhook_notifier.WebhookNotifier{
						WebhookURL:    "https://webhook.example.com/test",
						WebhookMethod: webhook_notifier.WebhookMethodPOST,
					},
				}
			},
			updateNotifier: func(workspaceID uuid.UUID, notifierID uuid.UUID) *Notifier {
				return &Notifier{
					ID:           notifierID,
					WorkspaceID:  workspaceID,
					Name:         "Updated Webhook Notifier",
					NotifierType: NotifierTypeWebhook,
					WebhookNotifier: &webhook_notifier.WebhookNotifier{
						WebhookURL:    "https://webhook.example.com/updated",
						WebhookMethod: webhook_notifier.WebhookMethodGET,
					},
				}
			},
			verifySensitiveData: func(t *testing.T, notifier *Notifier) {
				// No sensitive data to verify for webhook
			},
			verifyHiddenData: func(t *testing.T, notifier *Notifier) {
				// No sensitive data to hide for webhook
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			owner := users_testing.CreateTestUser(users_enums.UserRoleMember)
			router := createRouter()
			workspace := workspaces_testing.CreateTestWorkspace("Test Workspace", owner, router)

			// Phase 1: Create notifier with sensitive data
			initialNotifier := tc.createNotifier(workspace.ID)
			var createdNotifier Notifier
			test_utils.MakePostRequestAndUnmarshal(
				t,
				router,
				"/api/v1/notifiers",
				"Bearer "+owner.Token,
				*initialNotifier,
				http.StatusOK,
				&createdNotifier,
			)
			assert.NotEmpty(t, createdNotifier.ID)
			assert.Equal(t, initialNotifier.Name, createdNotifier.Name)

			// Phase 2: Read via service - sensitive data should be hidden
			var retrievedNotifier Notifier
			test_utils.MakeGetRequestAndUnmarshal(
				t,
				router,
				fmt.Sprintf("/api/v1/notifiers/%s", createdNotifier.ID.String()),
				"Bearer "+owner.Token,
				http.StatusOK,
				&retrievedNotifier,
			)
			tc.verifyHiddenData(t, &retrievedNotifier)
			assert.Equal(t, initialNotifier.Name, retrievedNotifier.Name)

			// Phase 3: Update with non-sensitive changes only (sensitive fields empty)
			updatedNotifier := tc.updateNotifier(workspace.ID, createdNotifier.ID)
			var updateResponse Notifier
			test_utils.MakePostRequestAndUnmarshal(
				t,
				router,
				"/api/v1/notifiers",
				"Bearer "+owner.Token,
				*updatedNotifier,
				http.StatusOK,
				&updateResponse,
			)
			// Verify non-sensitive fields were updated
			assert.Equal(t, updatedNotifier.Name, updateResponse.Name)

			// Phase 4: Retrieve directly from repository to verify sensitive data preservation
			repository := &NotifierRepository{}
			notifierFromDB, err := repository.FindByID(createdNotifier.ID)
			assert.NoError(t, err)

			// Verify original sensitive data is still present in DB
			tc.verifySensitiveData(t, notifierFromDB)

			// Verify non-sensitive fields were updated in DB
			assert.Equal(t, updatedNotifier.Name, notifierFromDB.Name)

			// Phase 5: Additional verification - Check via GET that data is still hidden
			var finalRetrieved Notifier
			test_utils.MakeGetRequestAndUnmarshal(
				t,
				router,
				fmt.Sprintf("/api/v1/notifiers/%s", createdNotifier.ID.String()),
				"Bearer "+owner.Token,
				http.StatusOK,
				&finalRetrieved,
			)
			tc.verifyHiddenData(t, &finalRetrieved)

			deleteNotifier(t, router, createdNotifier.ID, workspace.ID, owner.Token)
			workspaces_testing.RemoveTestWorkspace(workspace, router)
		})
	}
}
