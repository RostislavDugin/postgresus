package backups_controllers_physical_test

import (
	"archive/tar"
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/klauspost/compress/zstd"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	audit_logs "databasus-backend/internal/features/audit_logs"
	backuping_physical "databasus-backend/internal/features/backups/backups/backuping/physical"
	backups_controllers_physical "databasus-backend/internal/features/backups/backups/controllers/physical"
	backups_core_enums "databasus-backend/internal/features/backups/backups/core/enums"
	physical_enums "databasus-backend/internal/features/backups/backups/core/physical/enums"
	physical_models "databasus-backend/internal/features/backups/backups/core/physical/models"
	physical_repositories "databasus-backend/internal/features/backups/backups/core/physical/repositories"
	physical_testing "databasus-backend/internal/features/backups/backups/core/physical/testing"
	backups_download "databasus-backend/internal/features/backups/backups/download"
	backups_dto_physical "databasus-backend/internal/features/backups/backups/dto/physical"
	backups_config_physical "databasus-backend/internal/features/backups/config/physical"
	"databasus-backend/internal/features/databases"
	"databasus-backend/internal/features/notifiers"
	"databasus-backend/internal/features/storages"
	users_dto "databasus-backend/internal/features/users/dto"
	users_enums "databasus-backend/internal/features/users/enums"
	users_middleware "databasus-backend/internal/features/users/middleware"
	users_services "databasus-backend/internal/features/users/services"
	users_testing "databasus-backend/internal/features/users/testing"
	workspaces_controllers "databasus-backend/internal/features/workspaces/controllers"
	workspaces_models "databasus-backend/internal/features/workspaces/models"
	workspaces_testing "databasus-backend/internal/features/workspaces/testing"
	"databasus-backend/internal/storage"
	"databasus-backend/internal/util/encryption"
	"databasus-backend/internal/util/logger"
	test_utils "databasus-backend/internal/util/testing"
	"databasus-backend/internal/util/walmath"
)

const segmentBytes = 16 * 1024 * 1024

type physicalControllerPrereqs struct {
	user      *users_dto.SignInResponseDTO
	workspace *workspaces_models.Workspace
	storage   *storages.Storage
	notifier  *notifiers.Notifier
	database  *databases.Database
	router    *gin.Engine
}

func createPhysicalControllerPrereqs(t *testing.T) *physicalControllerPrereqs {
	t.Helper()

	router := newPhysicalControllerRouter()
	user := users_testing.CreateTestUser(t.Context(), users_enums.UserRoleMember)
	workspace := workspaces_testing.CreateTestWorkspace(t.Context(), "Physical Ctrl "+uuid.NewString(), user, router)
	storage := storages.CreateTestStorage(workspace.ID)
	notifier := notifiers.CreateTestNotifier(workspace.ID)
	database := databases.CreateTestPhysicalPostgresDatabase(workspace.ID, notifier, "17")

	t.Cleanup(func() {
		physical_testing.DeleteAllPhysicalCatalogForDatabase(t, database.ID)
		databases.RemoveTestDatabase(t.Context(), database)
		notifiers.RemoveTestNotifier(notifier)
		storages.RemoveTestStorage(t.Context(), storage.ID)
	})

	return &physicalControllerPrereqs{
		user:      user,
		workspace: workspace,
		storage:   storage,
		notifier:  notifier,
		database:  database,
		router:    router,
	}
}

// newPhysicalControllerRouter wires the physical controller's protected and
// public routes (CreateTestRouter only registers protected ones) plus the
// supporting controllers and feature dependencies.
func newPhysicalControllerRouter() *gin.Engine {
	gin.SetMode(gin.TestMode)
	router := gin.New()

	v1 := router.Group("/api/v1")

	backups_controllers_physical.GetPhysicalBackupController().RegisterPublicRoutes(v1)

	protected := v1.Group("").Use(users_middleware.AuthMiddleware(users_services.GetUserService()))
	if routerGroup, ok := protected.(*gin.RouterGroup); ok {
		workspaces_controllers.GetWorkspaceController().RegisterRoutes(routerGroup)
		workspaces_controllers.GetMembershipController().RegisterRoutes(routerGroup)
		databases.GetDatabaseController().RegisterRoutes(routerGroup)
		backups_controllers_physical.GetPhysicalBackupController().RegisterRoutes(routerGroup)
	}

	storages.SetupDependencies()
	databases.SetupDependencies()
	notifiers.SetupDependencies()
	backups_config_physical.SetupDependencies()
	backuping_physical.SetupDependencies()

	return router
}

func Test_GetBackups_WhenOwner_ReturnsFlatListSortedByCreatedAt(t *testing.T) {
	prereqs := createPhysicalControllerPrereqs(t)

	base := time.Now().UTC().Add(-1 * time.Hour)

	fullModel := physical_testing.NewTestCompletedFullBackup(
		prereqs.database.ID, prereqs.storage.ID, 1, walmath.LSN(0), walmath.LSN(segmentBytes))
	fullModel.CreatedAt = base
	full := physical_testing.CreateTestFullBackup(t, fullModel)

	incrModel := physical_testing.NewTestCompletedIncrementalBackup(
		prereqs.database.ID, prereqs.storage.ID, full.ID, nil, 1,
		walmath.LSN(segmentBytes), walmath.LSN(2*segmentBytes))
	incrModel.CreatedAt = base.Add(10 * time.Minute)
	incr := physical_testing.CreateTestIncrementalBackup(t, incrModel)

	walModel := physical_testing.NewTestWalSegment(
		prereqs.database.ID, prereqs.storage.ID, 1, "000000010000000000000002",
		walmath.LSN(2*segmentBytes), walmath.LSN(3*segmentBytes))
	walModel.ReceivedAt = base.Add(20 * time.Minute)
	walSegment := physical_testing.CreateTestWalSegment(t, walModel)

	var response backups_dto_physical.GetPhysicalBackupsResponse
	test_utils.MakeGetRequestAndUnmarshal(t, prereqs.router,
		"/api/v1/backups/physical/database/"+prereqs.database.ID.String()+"/backups",
		"Bearer "+prereqs.user.Token, http.StatusOK, &response)

	require.Len(t, response.Backups, 3)

	// Newest-first: WAL (received last), then incremental, then full.
	assert.Equal(t, walSegment.ID, response.Backups[0].ID)
	assert.Equal(t, physical_enums.PhysicalBackupTypeWal, response.Backups[0].Type)
	assert.Equal(t, incr.ID, response.Backups[1].ID)
	assert.Equal(t, physical_enums.PhysicalBackupTypeIncremental, response.Backups[1].Type)
	assert.Equal(t, full.ID, response.Backups[2].ID)
	assert.Equal(t, physical_enums.PhysicalBackupTypeFull, response.Backups[2].Type)

	assert.Equal(t, full.ID, *response.Backups[1].RootFullBackupID)
	require.NotNil(t, response.Backups[0].FromWalFilename)
	assert.Equal(t, "000000010000000000000002", *response.Backups[0].FromWalFilename)
	assert.Positive(t, response.TotalUsageMb)
}

func Test_GetBackups_WhenBackupFailed_ReturnsFailMessage(t *testing.T) {
	prereqs := createPhysicalControllerPrereqs(t)

	base := time.Now().UTC().Add(-1 * time.Hour)

	completedModel := physical_testing.NewTestCompletedFullBackup(
		prereqs.database.ID, prereqs.storage.ID, 1, walmath.LSN(0), walmath.LSN(segmentBytes))
	completedModel.CreatedAt = base
	completed := physical_testing.CreateTestFullBackup(t, completedModel)

	failReason := physical_enums.PhysicalBackupErrorPgBasebackupFailed
	failedModel := physical_testing.NewTestCompletedFullBackup(
		prereqs.database.ID, prereqs.storage.ID, 1, walmath.LSN(0), walmath.LSN(segmentBytes))
	failedModel.CreatedAt = base.Add(10 * time.Minute)
	failedModel.Status = physical_enums.PhysicalBackupStatusError
	failedModel.ErrorReason = &failReason
	failedModel.FailMessage = new("pg_basebackup failed; stderr: connection refused")
	failed := physical_testing.CreateTestFullBackup(t, failedModel)

	var response backups_dto_physical.GetPhysicalBackupsResponse
	test_utils.MakeGetRequestAndUnmarshal(t, prereqs.router,
		"/api/v1/backups/physical/database/"+prereqs.database.ID.String()+"/backups",
		"Bearer "+prereqs.user.Token, http.StatusOK, &response)

	require.Len(t, response.Backups, 2)

	// Newest-first: the failed backup, then the completed one.
	assert.Equal(t, failed.ID, response.Backups[0].ID)
	require.NotNil(t, response.Backups[0].FailMessage)
	assert.Equal(t, "pg_basebackup failed; stderr: connection refused", *response.Backups[0].FailMessage)

	assert.Equal(t, completed.ID, response.Backups[1].ID)
	assert.Nil(t, response.Backups[1].FailMessage,
		"a completed backup carries no failure message")
}

func Test_GetBackups_FollowingNextCursor_ReturnsEveryRowOnce(t *testing.T) {
	prereqs := createPhysicalControllerPrereqs(t)

	fulls := createFullBackupsMinutesApart(t, prereqs, 5, time.Now().UTC().Add(-1*time.Hour))

	firstPage := getBackups(t, prereqs, "?limit=2")
	require.Len(t, firstPage.Backups, 2)
	require.NotEmpty(t, firstPage.NextCursor)

	secondPage := getBackups(t, prereqs, "?limit=2&cursor="+firstPage.NextCursor)
	require.Len(t, secondPage.Backups, 2)
	require.NotEmpty(t, secondPage.NextCursor)

	lastPage := getBackups(t, prereqs, "?limit=2&cursor="+secondPage.NextCursor)
	require.Len(t, lastPage.Backups, 1)
	assert.Empty(t, lastPage.NextCursor, "the page holding the oldest row carries no cursor")

	listedIDs := collectBackupIDs(firstPage, secondPage, lastPage)
	assert.Equal(t, []uuid.UUID{fulls[4].ID, fulls[3].ID, fulls[2].ID, fulls[1].ID, fulls[0].ID}, listedIDs)
}

func Test_GetBackups_WhenBackupCreatedBetweenPages_NoRowRepeatedOrSkipped(t *testing.T) {
	prereqs := createPhysicalControllerPrereqs(t)

	fulls := createFullBackupsMinutesApart(t, prereqs, 3, time.Now().UTC().Add(-1*time.Hour))

	firstPage := getBackups(t, prereqs, "?limit=2")
	require.NotEmpty(t, firstPage.NextCursor)

	physical_testing.CreateTestFullBackup(t, physical_testing.NewTestCompletedFullBackup(
		prereqs.database.ID, prereqs.storage.ID, 1, walmath.LSN(0), walmath.LSN(segmentBytes)))

	secondPage := getBackups(t, prereqs, "?limit=2&cursor="+firstPage.NextCursor)

	listedIDs := collectBackupIDs(firstPage, secondPage)
	assert.Equal(t, []uuid.UUID{fulls[2].ID, fulls[1].ID, fulls[0].ID}, listedIDs)
}

func Test_GetBackups_WhenCursorMalformed_Returns400(t *testing.T) {
	prereqs := createPhysicalControllerPrereqs(t)

	listURL := "/api/v1/backups/physical/database/" + prereqs.database.ID.String() + "/backups"

	for _, malformedCursor := range []string{"not-a-cursor", "e30", "eyJ0IjoieCJ9"} {
		test_utils.MakeGetRequest(t, prereqs.router,
			listURL+"?cursor="+malformedCursor, "Bearer "+prereqs.user.Token, http.StatusBadRequest)
	}
}

func Test_GetBackups_WithCursorFromAnotherDatabase_ReturnsOnlyRequestedDatabaseRows(t *testing.T) {
	prereqs := createPhysicalControllerPrereqs(t)

	base := time.Now().UTC().Add(-1 * time.Hour)
	ownFulls := createFullBackupsMinutesApart(t, prereqs, 2, base)

	other := createSecondPhysicalDatabase(t, prereqs)
	for index := range 2 {
		otherModel := physical_testing.NewTestCompletedFullBackup(
			other.ID, prereqs.storage.ID, 1, walmath.LSN(0), walmath.LSN(segmentBytes))
		otherModel.CreatedAt = base.Add(time.Duration(30+index) * time.Minute)
		physical_testing.CreateTestFullBackup(t, otherModel)
	}

	var otherPage backups_dto_physical.GetPhysicalBackupsResponse
	test_utils.MakeGetRequestAndUnmarshal(t, prereqs.router,
		"/api/v1/backups/physical/database/"+other.ID.String()+"/backups?limit=1",
		"Bearer "+prereqs.user.Token, http.StatusOK, &otherPage)
	require.NotEmpty(t, otherPage.NextCursor)

	page := getBackups(t, prereqs, "?cursor="+otherPage.NextCursor)

	assert.Equal(t, []uuid.UUID{ownFulls[1].ID, ownFulls[0].ID}, collectBackupIDs(page))
}

func Test_GetBackups_WithCursor_WhenNonMember_ReturnsError(t *testing.T) {
	prereqs := createPhysicalControllerPrereqs(t)

	createFullBackupsMinutesApart(t, prereqs, 2, time.Now().UTC().Add(-1*time.Hour))

	firstPage := getBackups(t, prereqs, "?limit=1")
	require.NotEmpty(t, firstPage.NextCursor)

	outsider := users_testing.CreateTestUser(t.Context(), users_enums.UserRoleMember)

	test_utils.MakeGetRequest(t, prereqs.router,
		"/api/v1/backups/physical/database/"+prereqs.database.ID.String()+"/backups?cursor="+firstPage.NextCursor,
		"Bearer "+outsider.Token, http.StatusBadRequest)
}

func Test_GetBackups_WhenNonMember_ReturnsError(t *testing.T) {
	prereqs := createPhysicalControllerPrereqs(t)

	outsider := users_testing.CreateTestUser(t.Context(), users_enums.UserRoleMember)

	test_utils.MakeGetRequest(t, prereqs.router,
		"/api/v1/backups/physical/database/"+prereqs.database.ID.String()+"/backups",
		"Bearer "+outsider.Token, http.StatusBadRequest)
}

func Test_TriggerBackup_Full_SetsForceFullFlag(t *testing.T) {
	prereqs := createPhysicalControllerPrereqs(t)

	// Touch the config so a row exists for the request flag to land on.
	_, err := backups_config_physical.GetBackupConfigService().GetBackupConfigByDbId(prereqs.database.ID)
	require.NoError(t, err)

	test_utils.MakePostRequest(t, prereqs.router,
		"/api/v1/backups/physical/database/"+prereqs.database.ID.String()+"/trigger",
		"Bearer "+prereqs.user.Token,
		backups_dto_physical.TriggerBackupRequest{Type: backups_dto_physical.TriggerBackupTypeFull},
		http.StatusAccepted)

	config, err := backups_config_physical.GetBackupConfigService().GetBackupConfigByDbId(prereqs.database.ID)
	require.NoError(t, err)
	assert.NotNil(t, config.ForceFullRequestedAt, "trigger full must set the force-full request flag")
}

func Test_TriggerBackup_IncrementalWithoutExtendableChain_Returns409(t *testing.T) {
	prereqs := createPhysicalControllerPrereqs(t)

	test_utils.MakePostRequest(t, prereqs.router,
		"/api/v1/backups/physical/database/"+prereqs.database.ID.String()+"/trigger",
		"Bearer "+prereqs.user.Token,
		backups_dto_physical.TriggerBackupRequest{Type: backups_dto_physical.TriggerBackupTypeIncremental},
		http.StatusConflict)
}

func Test_CancelBackup_WhenInProgress_CancelsAndReleasesClaim(t *testing.T) {
	prereqs := createPhysicalControllerPrereqs(t)

	full := physical_testing.CreateTestFullBackup(t,
		physical_testing.NewTestInProgressFullBackup(prereqs.database.ID, prereqs.storage.ID, 1))
	physical_testing.CreateTestInFlightClaim(t, prereqs.database.ID, full.ID, physical_enums.PhysicalBackupTypeFull)

	test_utils.MakePostRequest(t, prereqs.router,
		"/api/v1/backups/physical/backups/"+full.ID.String()+"/cancel",
		"Bearer "+prereqs.user.Token, nil, http.StatusNoContent)

	claim, err := physical_repositories.GetInFlightBackupRepository().FindByDatabaseID(prereqs.database.ID)
	require.NoError(t, err)
	assert.Nil(t, claim, "cancel must release the in-flight claim")
}

func Test_CancelBackup_WhenNotInProgress_Returns400(t *testing.T) {
	prereqs := createPhysicalControllerPrereqs(t)

	full := physical_testing.CreateTestFullBackup(t, physical_testing.NewTestCompletedFullBackup(
		prereqs.database.ID, prereqs.storage.ID, 1, walmath.LSN(0), walmath.LSN(segmentBytes)))

	test_utils.MakePostRequest(t, prereqs.router,
		"/api/v1/backups/physical/backups/"+full.ID.String()+"/cancel",
		"Bearer "+prereqs.user.Token, nil, http.StatusBadRequest)
}

func Test_DeleteBackup_Full_CascadesChain(t *testing.T) {
	prereqs := createPhysicalControllerPrereqs(t)

	full := physical_testing.CreateTestFullBackup(t, physical_testing.NewTestCompletedFullBackup(
		prereqs.database.ID, prereqs.storage.ID, 1, walmath.LSN(0), walmath.LSN(segmentBytes)))
	incr := physical_testing.CreateTestIncrementalBackup(t, physical_testing.NewTestCompletedIncrementalBackup(
		prereqs.database.ID, prereqs.storage.ID, full.ID, nil, 1,
		walmath.LSN(segmentBytes), walmath.LSN(2*segmentBytes)))
	walSegment := physical_testing.CreateTestWalSegment(t, physical_testing.NewTestWalSegment(
		prereqs.database.ID, prereqs.storage.ID, 1, "000000010000000000000001",
		walmath.LSN(segmentBytes), walmath.LSN(2*segmentBytes)))

	test_utils.MakeDeleteRequest(t, prereqs.router,
		"/api/v1/backups/physical/backups/"+full.ID.String(),
		"Bearer "+prereqs.user.Token, http.StatusNoContent)

	assertFullGone(t, full.ID)
	assertIncrementalGone(t, incr.ID)
	assertWalGone(t, walSegment.ID)
}

func Test_DeleteBackup_Incremental_RemovesDescendantsKeepsWalAndFull(t *testing.T) {
	prereqs := createPhysicalControllerPrereqs(t)

	full := physical_testing.CreateTestFullBackup(t, physical_testing.NewTestCompletedFullBackup(
		prereqs.database.ID, prereqs.storage.ID, 1, walmath.LSN(0), walmath.LSN(segmentBytes)))
	firstIncr := physical_testing.CreateTestIncrementalBackup(t, physical_testing.NewTestCompletedIncrementalBackup(
		prereqs.database.ID, prereqs.storage.ID, full.ID, nil, 1,
		walmath.LSN(segmentBytes), walmath.LSN(2*segmentBytes)))
	secondIncr := physical_testing.CreateTestIncrementalBackup(t, physical_testing.NewTestCompletedIncrementalBackup(
		prereqs.database.ID, prereqs.storage.ID, full.ID, &firstIncr.ID, 1,
		walmath.LSN(2*segmentBytes), walmath.LSN(3*segmentBytes)))
	walSegment := physical_testing.CreateTestWalSegment(t, physical_testing.NewTestWalSegment(
		prereqs.database.ID, prereqs.storage.ID, 1, "000000010000000000000001",
		walmath.LSN(segmentBytes), walmath.LSN(2*segmentBytes)))

	test_utils.MakeDeleteRequest(t, prereqs.router,
		"/api/v1/backups/physical/backups/"+firstIncr.ID.String(),
		"Bearer "+prereqs.user.Token, http.StatusNoContent)

	assertIncrementalGone(t, firstIncr.ID)
	assertIncrementalGone(t, secondIncr.ID)
	assertFullPresent(t, full.ID)
	assertWalPresent(t, walSegment.ID)
}

func Test_DeleteBackup_Wal_RemovesFollowingWalUpToNextAnchor(t *testing.T) {
	prereqs := createPhysicalControllerPrereqs(t)

	full := physical_testing.CreateTestFullBackup(t, physical_testing.NewTestCompletedFullBackup(
		prereqs.database.ID, prereqs.storage.ID, 1, walmath.LSN(0), walmath.LSN(segmentBytes)))

	firstWal := physical_testing.CreateTestWalSegment(t, physical_testing.NewTestWalSegment(
		prereqs.database.ID, prereqs.storage.ID, 1, "000000010000000000000001",
		walmath.LSN(segmentBytes), walmath.LSN(2*segmentBytes)))
	secondWal := physical_testing.CreateTestWalSegment(t, physical_testing.NewTestWalSegment(
		prereqs.database.ID, prereqs.storage.ID, 1, "000000010000000000000002",
		walmath.LSN(2*segmentBytes), walmath.LSN(3*segmentBytes)))
	thirdWal := physical_testing.CreateTestWalSegment(t, physical_testing.NewTestWalSegment(
		prereqs.database.ID, prereqs.storage.ID, 1, "000000010000000000000003",
		walmath.LSN(3*segmentBytes), walmath.LSN(4*segmentBytes)))

	// An incremental anchored at LSN 3 bounds the WAL delete: segments at or
	// after it survive.
	anchorIncr := physical_testing.CreateTestIncrementalBackup(t, physical_testing.NewTestCompletedIncrementalBackup(
		prereqs.database.ID, prereqs.storage.ID, full.ID, nil, 1,
		walmath.LSN(3*segmentBytes), walmath.LSN(4*segmentBytes)))

	test_utils.MakeDeleteRequest(t, prereqs.router,
		"/api/v1/backups/physical/backups/"+firstWal.ID.String(),
		"Bearer "+prereqs.user.Token, http.StatusNoContent)

	assertWalGone(t, firstWal.ID)
	assertWalGone(t, secondWal.ID)
	assertWalPresent(t, thirdWal.ID)
	assertFullPresent(t, full.ID)
	assertIncrementalPresent(t, anchorIncr.ID)
}

func Test_GenerateRestoreToken_ReturnsTokenAndURL(t *testing.T) {
	prereqs := createPhysicalControllerPrereqs(t)

	// A token is only issued for a resolvable restore set, so the chain must
	// exist before requesting one.
	physical_testing.CreateTestFullBackup(t, physical_testing.NewTestCompletedFullBackup(
		prereqs.database.ID, prereqs.storage.ID, 1, walmath.LSN(0), walmath.LSN(segmentBytes)))

	var response backups_dto_physical.GenerateRestoreTokenResponse
	test_utils.MakePostRequestAndUnmarshal(t, prereqs.router,
		"/api/v1/backups/physical/database/"+prereqs.database.ID.String()+"/restore-token",
		"Bearer "+prereqs.user.Token,
		backups_dto_physical.GenerateRestoreTokenRequest{},
		http.StatusOK, &response)

	assert.NotEmpty(t, response.Token)
	assert.Contains(t, response.URL, response.Token)
}

func Test_GenerateRestoreToken_WhenNoChain_Returns404(t *testing.T) {
	prereqs := createPhysicalControllerPrereqs(t)

	test_utils.MakePostRequest(t, prereqs.router,
		"/api/v1/backups/physical/database/"+prereqs.database.ID.String()+"/restore-token",
		"Bearer "+prereqs.user.Token,
		backups_dto_physical.GenerateRestoreTokenRequest{},
		http.StatusNotFound)
}

// Test_GenerateRestoreToken_WhenTargetReachable_ReturnsToken seeds a contiguous
// WAL run past the chain and asks for a point that the run actually reaches, so
// the restore set resolves and a token is minted.
func Test_GenerateRestoreToken_WhenTargetReachable_ReturnsToken(t *testing.T) {
	prereqs := createPhysicalControllerPrereqs(t)

	base := time.Now().UTC().Add(-2 * time.Hour)

	full := physical_testing.NewTestCompletedFullBackup(
		prereqs.database.ID, prereqs.storage.ID, 1, walmath.LSN(0), walmath.LSN(segmentBytes))
	full.CompletedAt = ptrTime(base)
	physical_testing.CreateTestFullBackup(t, full)

	incr := physical_testing.NewTestCompletedIncrementalBackup(
		prereqs.database.ID, prereqs.storage.ID, full.ID, nil, 1,
		walmath.LSN(segmentBytes), walmath.LSN(2*segmentBytes))
	incr.CompletedAt = ptrTime(base.Add(10 * time.Minute))
	physical_testing.CreateTestIncrementalBackup(t, incr)

	// A WAL segment contiguous with the incremental, received after it.
	walSegment := physical_testing.NewTestWalSegment(
		prereqs.database.ID, prereqs.storage.ID, 1, "000000010000000000000002",
		walmath.LSN(2*segmentBytes), walmath.LSN(3*segmentBytes))
	walSegment.ReceivedAt = base.Add(12 * time.Minute)
	physical_testing.CreateTestWalSegment(t, walSegment)

	targetTime := base.Add(11 * time.Minute)

	var response backups_dto_physical.GenerateRestoreTokenResponse
	test_utils.MakePostRequestAndUnmarshal(t, prereqs.router,
		"/api/v1/backups/physical/database/"+prereqs.database.ID.String()+"/restore-token",
		"Bearer "+prereqs.user.Token,
		backups_dto_physical.GenerateRestoreTokenRequest{TargetTime: &targetTime},
		http.StatusOK, &response)

	assert.NotEmpty(t, response.Token)
}

func Test_GenerateRestoreToken_WhenTargetDuringNewerFull_ReturnsToken(t *testing.T) {
	prereqs := createPhysicalControllerPrereqs(t)
	base := time.Now().UTC().Add(-2 * time.Hour)

	olderFull := physical_testing.NewTestCompletedFullBackup(
		prereqs.database.ID,
		prereqs.storage.ID,
		1,
		walmath.LSN(0),
		walmath.LSN(segmentBytes),
	)
	olderFull.CreatedAt = base.Add(-time.Minute)
	olderFull.CompletedAt = ptrTime(base)
	physical_testing.CreateTestFullBackup(t, olderFull)

	newerFull := physical_testing.NewTestCompletedFullBackup(
		prereqs.database.ID,
		prereqs.storage.ID,
		1,
		walmath.LSN(2*segmentBytes),
		walmath.LSN(2*segmentBytes+physical_testing.FullLSNSpan),
	)
	newerFull.CreatedAt = base.Add(5 * time.Minute)
	newerFull.CompletedAt = ptrTime(base.Add(10 * time.Minute))
	physical_testing.CreateTestFullBackup(t, newerFull)

	for walIndex := 1; walIndex <= 3; walIndex++ {
		walSegment := physical_testing.NewTestWalSegment(
			prereqs.database.ID,
			prereqs.storage.ID,
			1,
			"00000001000000000000000"+string(rune('0'+walIndex)),
			walmath.LSN(walIndex*segmentBytes),
			walmath.LSN((walIndex+1)*segmentBytes),
		)
		walSegment.ReceivedAt = base.Add(time.Duration(3+walIndex*2) * time.Minute)
		physical_testing.CreateTestWalSegment(t, walSegment)
	}

	targetTime := base.Add(7 * time.Minute)
	var response backups_dto_physical.GenerateRestoreTokenResponse
	test_utils.MakePostRequestAndUnmarshal(t, prereqs.router,
		"/api/v1/backups/physical/database/"+prereqs.database.ID.String()+"/restore-token",
		"Bearer "+prereqs.user.Token,
		backups_dto_physical.GenerateRestoreTokenRequest{TargetTime: &targetTime},
		http.StatusOK,
		&response,
	)

	assert.NotEmpty(t, response.Token)
}

// Test_GenerateRestoreToken_WhenTargetPastWalGap_Returns422 verifies the gap is
// caught at token-issue time (the restore set is resolved before a token is
// minted), so the user never burns a token on an unreachable target.
func Test_GenerateRestoreToken_WhenTargetPastWalGap_Returns422(t *testing.T) {
	prereqs := createPhysicalControllerPrereqs(t)

	base := time.Now().UTC().Add(-2 * time.Hour)

	full := physical_testing.NewTestCompletedFullBackup(
		prereqs.database.ID, prereqs.storage.ID, 1, walmath.LSN(0), walmath.LSN(segmentBytes))
	full.CompletedAt = ptrTime(base)
	physical_testing.CreateTestFullBackup(t, full)

	incr := physical_testing.NewTestCompletedIncrementalBackup(
		prereqs.database.ID, prereqs.storage.ID, full.ID, nil, 1,
		walmath.LSN(segmentBytes), walmath.LSN(2*segmentBytes))
	incr.CompletedAt = ptrTime(base.Add(10 * time.Minute))
	physical_testing.CreateTestIncrementalBackup(t, incr)

	// One WAL segment then a gap (no segment covering [3,4)).
	w2 := physical_testing.NewTestWalSegment(
		prereqs.database.ID, prereqs.storage.ID, 1, "000000010000000000000002",
		walmath.LSN(2*segmentBytes), walmath.LSN(3*segmentBytes))
	w2.ReceivedAt = base.Add(12 * time.Minute)
	physical_testing.CreateTestWalSegment(t, w2)

	targetTime := base.Add(60 * time.Minute)

	response := test_utils.MakePostRequest(t, prereqs.router,
		"/api/v1/backups/physical/database/"+prereqs.database.ID.String()+"/restore-token",
		"Bearer "+prereqs.user.Token,
		backups_dto_physical.GenerateRestoreTokenRequest{TargetTime: &targetTime},
		http.StatusUnprocessableEntity)

	var body map[string]string
	require.NoError(t, json.Unmarshal(response.Body, &body))
	assert.Contains(t, body["error"], "wal gap")
}

func Test_GenerateBackupRestoreToken_ForFull_ReturnsTokenAndURL(t *testing.T) {
	prereqs := createPhysicalControllerPrereqs(t)

	full := physical_testing.CreateTestFullBackup(t, physical_testing.NewTestCompletedFullBackup(
		prereqs.database.ID, prereqs.storage.ID, 1, walmath.LSN(0), walmath.LSN(segmentBytes)))

	var response backups_dto_physical.GenerateRestoreTokenResponse
	test_utils.MakePostRequestAndUnmarshal(t, prereqs.router,
		"/api/v1/backups/physical/backups/"+full.ID.String()+"/restore-token",
		"Bearer "+prereqs.user.Token, nil, http.StatusOK, &response)

	assert.NotEmpty(t, response.Token)
	assert.Contains(t, response.URL, response.Token)
}

func Test_GenerateBackupRestoreToken_WhenBackupMissing_Returns404(t *testing.T) {
	prereqs := createPhysicalControllerPrereqs(t)

	test_utils.MakePostRequest(t, prereqs.router,
		"/api/v1/backups/physical/backups/"+uuid.NewString()+"/restore-token",
		"Bearer "+prereqs.user.Token, nil, http.StatusNotFound)
}

func Test_GetRestoreStream_WhenTokenInvalid_Returns401(t *testing.T) {
	prereqs := createPhysicalControllerPrereqs(t)

	test_utils.MakeGetRequest(t, prereqs.router,
		"/api/v1/backups/physical/restore-stream?token=not-a-real-token",
		"", http.StatusUnauthorized)
}

func assertFullGone(t *testing.T, id uuid.UUID) {
	t.Helper()

	full, err := physical_repositories.GetFullBackupRepository().FindByID(id)
	require.NoError(t, err)
	assert.Nil(t, full, "full row must be gone")
}

func assertFullPresent(t *testing.T, id uuid.UUID) {
	t.Helper()

	full, err := physical_repositories.GetFullBackupRepository().FindByID(id)
	require.NoError(t, err)
	assert.NotNil(t, full, "full row must survive")
}

func assertIncrementalGone(t *testing.T, id uuid.UUID) {
	t.Helper()

	incremental, err := physical_repositories.GetIncrementalBackupRepository().FindByID(id)
	require.NoError(t, err)
	assert.Nil(t, incremental, "incremental row must be gone")
}

func assertIncrementalPresent(t *testing.T, id uuid.UUID) {
	t.Helper()

	incremental, err := physical_repositories.GetIncrementalBackupRepository().FindByID(id)
	require.NoError(t, err)
	assert.NotNil(t, incremental, "incremental row must survive")
}

func assertWalGone(t *testing.T, id uuid.UUID) {
	t.Helper()

	walSegment, err := physical_repositories.GetWalSegmentRepository().FindByID(id)
	require.NoError(t, err)
	assert.Nil(t, walSegment, "wal row must be gone")
}

func assertWalPresent(t *testing.T, id uuid.UUID) {
	t.Helper()

	walSegment, err := physical_repositories.GetWalSegmentRepository().FindByID(id)
	require.NoError(t, err)
	assert.NotNil(t, walSegment, "wal row must survive")
}

func ptrTime(t time.Time) *time.Time {
	return &t
}

// --- Permission matrix -------------------------------------------------------

func Test_GetBackups_PermissionsEnforced(t *testing.T) {
	prereqs := createPhysicalControllerPrereqs(t)

	listURL := "/api/v1/backups/physical/database/" + prereqs.database.ID.String() + "/backups"

	// Listing is a read: a viewer is allowed, only a non-member is rejected.
	runRolePermissionMatrix(t, prereqs, false, http.StatusOK, func(authHeader string) int {
		return workspaces_testing.MakeAPIRequest(prereqs.router, "GET", listURL, authHeader, nil).Code
	})
}

func Test_TriggerBackup_PermissionsEnforced(t *testing.T) {
	prereqs := createPhysicalControllerPrereqs(t)

	_, err := backups_config_physical.GetBackupConfigService().GetBackupConfigByDbId(prereqs.database.ID)
	require.NoError(t, err)

	triggerURL := "/api/v1/backups/physical/database/" + prereqs.database.ID.String() + "/trigger"
	request := backups_dto_physical.TriggerBackupRequest{Type: backups_dto_physical.TriggerBackupTypeFull}

	// Triggering changes state: a viewer is rejected.
	runRolePermissionMatrix(t, prereqs, true, http.StatusAccepted, func(authHeader string) int {
		return workspaces_testing.MakeAPIRequest(prereqs.router, "POST", triggerURL, authHeader, request).Code
	})
}

// A viewer could delete physical backups before authorization was tightened to
// manage-DBs; this pins the fix.
func Test_DeleteBackup_WhenViewer_Returns400AndKeepsBackup(t *testing.T) {
	prereqs := createPhysicalControllerPrereqs(t)

	full := physical_testing.CreateTestFullBackup(t, physical_testing.NewTestCompletedFullBackup(
		prereqs.database.ID, prereqs.storage.ID, 1, walmath.LSN(0), walmath.LSN(segmentBytes)))

	viewer := addWorkspaceViewer(t, prereqs)

	test_utils.MakeDeleteRequest(t, prereqs.router,
		"/api/v1/backups/physical/backups/"+full.ID.String(),
		"Bearer "+viewer.Token, http.StatusBadRequest)

	assertFullPresent(t, full.ID)
}

func Test_CancelBackup_WhenViewer_Returns400AndKeepsClaim(t *testing.T) {
	prereqs := createPhysicalControllerPrereqs(t)

	full := physical_testing.CreateTestFullBackup(t,
		physical_testing.NewTestInProgressFullBackup(prereqs.database.ID, prereqs.storage.ID, 1))
	physical_testing.CreateTestInFlightClaim(t, prereqs.database.ID, full.ID, physical_enums.PhysicalBackupTypeFull)

	viewer := addWorkspaceViewer(t, prereqs)

	test_utils.MakePostRequest(t, prereqs.router,
		"/api/v1/backups/physical/backups/"+full.ID.String()+"/cancel",
		"Bearer "+viewer.Token, nil, http.StatusBadRequest)

	claim, err := physical_repositories.GetInFlightBackupRepository().FindByDatabaseID(prereqs.database.ID)
	require.NoError(t, err)
	assert.NotNil(t, claim, "a viewer's rejected cancel must not release the claim")
}

// --- Audit logging -----------------------------------------------------------

func Test_DeleteBackup_WritesAuditLog(t *testing.T) {
	prereqs := createPhysicalControllerPrereqs(t)

	full := physical_testing.CreateTestFullBackup(t, physical_testing.NewTestCompletedFullBackup(
		prereqs.database.ID, prereqs.storage.ID, 1, walmath.LSN(0), walmath.LSN(segmentBytes)))

	test_utils.MakeDeleteRequest(t, prereqs.router,
		"/api/v1/backups/physical/backups/"+full.ID.String(),
		"Bearer "+prereqs.user.Token, http.StatusNoContent)

	assertAuditLogContains(t, prereqs.workspace.ID, "Physical full backup deleted", prereqs.database.Name)
}

func Test_CancelBackup_WritesAuditLog(t *testing.T) {
	prereqs := createPhysicalControllerPrereqs(t)

	full := physical_testing.CreateTestFullBackup(t,
		physical_testing.NewTestInProgressFullBackup(prereqs.database.ID, prereqs.storage.ID, 1))
	physical_testing.CreateTestInFlightClaim(t, prereqs.database.ID, full.ID, physical_enums.PhysicalBackupTypeFull)

	test_utils.MakePostRequest(t, prereqs.router,
		"/api/v1/backups/physical/backups/"+full.ID.String()+"/cancel",
		"Bearer "+prereqs.user.Token, nil, http.StatusNoContent)

	assertAuditLogContains(t, prereqs.workspace.ID, "Physical backup cancelled", prereqs.database.Name)
}

// --- Filters -----------------------------------------------------------------

func Test_GetBackups_FilterByType_ReturnsOnlyThatType(t *testing.T) {
	prereqs := createPhysicalControllerPrereqs(t)

	full, incr, walSegment := seedOneOfEachType(t, prereqs)

	fullPage := getBackups(t, prereqs, "?type=FULL")
	require.Len(t, fullPage.Backups, 1)
	assert.Equal(t, full.ID, fullPage.Backups[0].ID)

	incrPage := getBackups(t, prereqs, "?type=INCREMENTAL")
	require.Len(t, incrPage.Backups, 1)
	assert.Equal(t, incr.ID, incrPage.Backups[0].ID)

	walPage := getBackups(t, prereqs, "?type=WAL")
	require.Len(t, walPage.Backups, 1)
	assert.Equal(t, walSegment.ID, walPage.Backups[0].ID)
}

func Test_GetBackups_FilterByStatus_ReturnsOnlyThatStatus(t *testing.T) {
	prereqs := createPhysicalControllerPrereqs(t)

	completed := physical_testing.CreateTestFullBackup(t, physical_testing.NewTestCompletedFullBackup(
		prereqs.database.ID, prereqs.storage.ID, 1, walmath.LSN(0), walmath.LSN(segmentBytes)))
	inProgress := physical_testing.CreateTestFullBackup(t,
		physical_testing.NewTestInProgressFullBackup(prereqs.database.ID, prereqs.storage.ID, 1))

	completedPage := getBackups(t, prereqs, "?status=COMPLETED")
	require.Len(t, completedPage.Backups, 1)
	assert.Equal(t, completed.ID, completedPage.Backups[0].ID)

	inProgressPage := getBackups(t, prereqs, "?status=IN_PROGRESS")
	require.Len(t, inProgressPage.Backups, 1)
	assert.Equal(t, inProgress.ID, inProgressPage.Backups[0].ID)
}

func Test_GetBackups_FilterByBeforeDate_ExcludesNewer(t *testing.T) {
	prereqs := createPhysicalControllerPrereqs(t)

	now := time.Now().UTC()

	olderModel := physical_testing.NewTestCompletedFullBackup(
		prereqs.database.ID, prereqs.storage.ID, 1, walmath.LSN(0), walmath.LSN(segmentBytes))
	olderModel.CreatedAt = now.Add(-3 * time.Hour)
	older := physical_testing.CreateTestFullBackup(t, olderModel)

	newerModel := physical_testing.NewTestCompletedFullBackup(
		prereqs.database.ID, prereqs.storage.ID, 1, walmath.LSN(2*segmentBytes), walmath.LSN(3*segmentBytes))
	newerModel.CreatedAt = now
	physical_testing.CreateTestFullBackup(t, newerModel)

	cutoff := now.Add(-1 * time.Hour).Format(time.RFC3339)
	page := getBackups(t, prereqs, "?beforeDate="+cutoff)

	require.Len(t, page.Backups, 1)
	assert.Equal(t, older.ID, page.Backups[0].ID)
}

func Test_GetBackups_FilterByMultipleTypes_ReturnsThoseTypes(t *testing.T) {
	prereqs := createPhysicalControllerPrereqs(t)

	full, _, walSegment := seedOneOfEachType(t, prereqs)

	page := getBackups(t, prereqs, "?type=FULL&type=WAL")

	require.Len(t, page.Backups, 2)

	returnedIDs := []uuid.UUID{page.Backups[0].ID, page.Backups[1].ID}
	assert.Contains(t, returnedIDs, full.ID)
	assert.Contains(t, returnedIDs, walSegment.ID)
}

func Test_GetBackups_FilterByMultipleStatuses_ReturnsThoseStatuses(t *testing.T) {
	prereqs := createPhysicalControllerPrereqs(t)

	completed := physical_testing.CreateTestFullBackup(t, physical_testing.NewTestCompletedFullBackup(
		prereqs.database.ID, prereqs.storage.ID, 1, walmath.LSN(0), walmath.LSN(segmentBytes)))
	inProgress := physical_testing.CreateTestFullBackup(t,
		physical_testing.NewTestInProgressFullBackup(prereqs.database.ID, prereqs.storage.ID, 1))

	erroredModel := physical_testing.NewTestCompletedFullBackup(
		prereqs.database.ID, prereqs.storage.ID, 1, walmath.LSN(2*segmentBytes), walmath.LSN(3*segmentBytes))
	erroredModel.Status = physical_enums.PhysicalBackupStatusError
	physical_testing.CreateTestFullBackup(t, erroredModel)

	page := getBackups(t, prereqs, "?status=COMPLETED&status=IN_PROGRESS")

	require.Len(t, page.Backups, 2)

	returnedIDs := []uuid.UUID{page.Backups[0].ID, page.Backups[1].ID}
	assert.Contains(t, returnedIDs, completed.ID)
	assert.Contains(t, returnedIDs, inProgress.ID)
}

func Test_GetBackups_FilterByTypesAndStatuses_AppliesAndAcrossDimensions(t *testing.T) {
	prereqs := createPhysicalControllerPrereqs(t)

	completedFull := physical_testing.CreateTestFullBackup(t, physical_testing.NewTestCompletedFullBackup(
		prereqs.database.ID, prereqs.storage.ID, 1, walmath.LSN(0), walmath.LSN(segmentBytes)))

	// Excluded by the status dimension.
	physical_testing.CreateTestFullBackup(t,
		physical_testing.NewTestInProgressFullBackup(prereqs.database.ID, prereqs.storage.ID, 1))

	// Excluded by the type dimension (COMPLETED but INCREMENTAL).
	physical_testing.CreateTestIncrementalBackup(t, physical_testing.NewTestCompletedIncrementalBackup(
		prereqs.database.ID, prereqs.storage.ID, completedFull.ID, nil, 1,
		walmath.LSN(segmentBytes), walmath.LSN(2*segmentBytes)))

	page := getBackups(t, prereqs, "?type=FULL&status=COMPLETED")

	require.Len(t, page.Backups, 1)
	assert.Equal(t, completedFull.ID, page.Backups[0].ID)
}

// --- List edges --------------------------------------------------------------

func Test_GetBackups_WhenEmpty_ReturnsEmptyPage(t *testing.T) {
	prereqs := createPhysicalControllerPrereqs(t)

	page := getBackups(t, prereqs, "")

	assert.Empty(t, page.Backups)
	assert.Empty(t, page.NextCursor)
	assert.Zero(t, page.TotalUsageMb)
}

func Test_GetBackups_IsolatesByDatabase(t *testing.T) {
	prereqs := createPhysicalControllerPrereqs(t)

	ownFull := physical_testing.CreateTestFullBackup(t, physical_testing.NewTestCompletedFullBackup(
		prereqs.database.ID, prereqs.storage.ID, 1, walmath.LSN(0), walmath.LSN(segmentBytes)))

	other := createSecondPhysicalDatabase(t, prereqs)
	physical_testing.CreateTestFullBackup(t, physical_testing.NewTestCompletedFullBackup(
		other.ID, prereqs.storage.ID, 1, walmath.LSN(0), walmath.LSN(segmentBytes)))

	page := getBackups(t, prereqs, "")

	require.Len(t, page.Backups, 1)
	assert.Equal(t, ownFull.ID, page.Backups[0].ID)
}

func Test_GetBackups_DefaultsAndCapsPageLimit(t *testing.T) {
	prereqs := createPhysicalControllerPrereqs(t)

	createCompletedFullBackupsInBulk(t, prereqs, 1001)

	unset := getBackups(t, prereqs, "")
	assert.Len(t, unset.Backups, 50)
	assert.NotEmpty(t, unset.NextCursor)

	capped := getBackups(t, prereqs, "?limit=5000")
	assert.Len(t, capped.Backups, 1000)
	assert.NotEmpty(t, capped.NextCursor)
}

// --- WAL ranges --------------------------------------------------------------

func Test_GetBackups_WithContiguousWalSegments_ReturnsOneRangeRow(t *testing.T) {
	prereqs := createPhysicalControllerPrereqs(t)

	base := time.Now().UTC().Add(-1 * time.Hour)

	fullModel := physical_testing.NewTestCompletedFullBackup(
		prereqs.database.ID, prereqs.storage.ID, 1, walmath.LSN(0), walmath.LSN(segmentBytes))
	fullModel.CreatedAt = base
	full := physical_testing.CreateTestFullBackup(t, fullModel)

	segments := createContiguousWalSegments(t, prereqs, 1, 1, 5, base.Add(time.Minute))

	page := getBackups(t, prereqs, "")
	require.Len(t, page.Backups, 2)

	walRange := page.Backups[0]
	assert.Equal(t, physical_enums.PhysicalBackupTypeWal, walRange.Type)
	assert.Equal(t, segments[0].ID, walRange.ID)
	assert.Equal(t, segments[0].WalFilename, *walRange.FromWalFilename)
	assert.Equal(t, segments[4].WalFilename, *walRange.ToWalFilename)
	assert.Equal(t, 5, *walRange.WalSegmentsCount)
	assert.InDelta(t, 80, walRange.SizeMb, 0.001)
	assert.Equal(t, segments[0].StartLSN.String(), walRange.StartLSN)
	assert.Equal(t, segments[4].EndLSN.String(), walRange.StopLSN)
	assert.WithinDuration(t, segments[0].ReceivedAt, walRange.CreatedAt, time.Millisecond)
	require.NotNil(t, walRange.CompletedAt)
	assert.WithinDuration(t, segments[4].ReceivedAt, *walRange.CompletedAt, time.Millisecond)
	assert.False(t, walRange.HasGapBefore)

	assert.Equal(t, full.ID, page.Backups[1].ID)
}

func Test_GetBackups_WhenWalSegmentMissing_SplitsRangeAndMarksGap(t *testing.T) {
	prereqs := createPhysicalControllerPrereqs(t)

	base := time.Now().UTC().Add(-1 * time.Hour)
	olderSegments := createContiguousWalSegments(t, prereqs, 1, 1, 2, base)
	newerSegments := createContiguousWalSegments(t, prereqs, 1, 4, 2, base.Add(10*time.Minute))

	page := getBackups(t, prereqs, "")
	require.Len(t, page.Backups, 2)

	assert.Equal(t, newerSegments[0].ID, page.Backups[0].ID)
	assert.True(t, page.Backups[0].HasGapBefore, "the range after the missing segment is marked")

	assert.Equal(t, olderSegments[0].ID, page.Backups[1].ID)
	assert.False(t, page.Backups[1].HasGapBefore, "the oldest retained range has nothing before it")
}

func Test_GetBackups_WhenTimelineSwitches_StartsNewRangeWithoutGapMark(t *testing.T) {
	prereqs := createPhysicalControllerPrereqs(t)

	base := time.Now().UTC().Add(-1 * time.Hour)
	oldTimelineSegments := createContiguousWalSegments(t, prereqs, 1, 1, 2, base)
	newTimelineSegments := createContiguousWalSegments(t, prereqs, 2, 3, 2, base.Add(10*time.Minute))

	page := getBackups(t, prereqs, "")
	require.Len(t, page.Backups, 2)

	assert.Equal(t, newTimelineSegments[0].ID, page.Backups[0].ID)
	assert.Equal(t, 2, page.Backups[0].TimelineID)
	assert.False(t, page.Backups[0].HasGapBefore, "the first range on a timeline is not a gap")

	assert.Equal(t, oldTimelineSegments[0].ID, page.Backups[1].ID)
	assert.Equal(t, 1, page.Backups[1].TimelineID)
}

func Test_GetBackups_WhenFullBetweenSegments_SplitsRangeAroundFull(t *testing.T) {
	prereqs := createPhysicalControllerPrereqs(t)

	olderSegments, full, newerSegments := seedWalAroundFull(t, prereqs)

	page := getBackups(t, prereqs, "")
	require.Len(t, page.Backups, 3)

	assert.Equal(t, newerSegments[0].ID, page.Backups[0].ID)
	assert.Equal(t, 2, *page.Backups[0].WalSegmentsCount)
	assert.False(t, page.Backups[0].HasGapBefore)

	assert.Equal(t, full.ID, page.Backups[1].ID)

	assert.Equal(t, olderSegments[0].ID, page.Backups[2].ID)
	assert.Equal(t, 2, *page.Backups[2].WalSegmentsCount)
}

func Test_GetBackups_WhenFilteredToWal_MergesRangeAcrossFull(t *testing.T) {
	prereqs := createPhysicalControllerPrereqs(t)

	olderSegments, _, newerSegments := seedWalAroundFull(t, prereqs)

	page := getBackups(t, prereqs, "?type=WAL")
	require.Len(t, page.Backups, 1)

	assert.Equal(t, olderSegments[0].ID, page.Backups[0].ID)
	assert.Equal(t, 4, *page.Backups[0].WalSegmentsCount)
	assert.Equal(t, newerSegments[1].WalFilename, *page.Backups[0].ToWalFilename)
}

func Test_GetBackups_WithBeforeDateInsideRange_RangeCoversOlderSegmentsOnly(t *testing.T) {
	prereqs := createPhysicalControllerPrereqs(t)

	base := time.Now().UTC().Add(-1 * time.Hour)
	segments := createContiguousWalSegments(t, prereqs, 1, 1, 4, base)

	cutoff := segments[1].ReceivedAt.Add(30 * time.Second).Format(time.RFC3339Nano)
	page := getBackups(t, prereqs, "?beforeDate="+cutoff)
	require.Len(t, page.Backups, 1)

	assert.Equal(t, 2, *page.Backups[0].WalSegmentsCount)
	assert.Equal(t, segments[1].WalFilename, *page.Backups[0].ToWalFilename)
	assert.InDelta(t, 32, page.Backups[0].SizeMb, 0.001)
}

func Test_GetBackups_WithStatusThatWalNeverHas_ReturnsNoWalRows(t *testing.T) {
	prereqs := createPhysicalControllerPrereqs(t)

	base := time.Now().UTC().Add(-1 * time.Hour)
	createContiguousWalSegments(t, prereqs, 1, 1, 2, base)

	failedModel := physical_testing.NewTestCompletedFullBackup(
		prereqs.database.ID, prereqs.storage.ID, 1, walmath.LSN(0), walmath.LSN(segmentBytes))
	failedModel.Status = physical_enums.PhysicalBackupStatusError
	failed := physical_testing.CreateTestFullBackup(t, failedModel)

	page := getBackups(t, prereqs, "?status=ERROR")

	assert.Equal(t, []uuid.UUID{failed.ID}, collectBackupIDs(page))
}

func Test_GetBackups_WhenSegmentAppendedToNewestRange_RangeGrowsInPlace(t *testing.T) {
	prereqs := createPhysicalControllerPrereqs(t)

	base := time.Now().UTC().Add(-1 * time.Hour)
	segments := createContiguousWalSegments(t, prereqs, 1, 1, 2, base)

	pageBefore := getBackups(t, prereqs, "")
	require.Len(t, pageBefore.Backups, 1)

	appendedSegment := createContiguousWalSegments(t, prereqs, 1, 3, 1, base.Add(5*time.Minute))[0]

	pageAfter := getBackups(t, prereqs, "")
	require.Len(t, pageAfter.Backups, 1)

	grownRange := pageAfter.Backups[0]
	assert.Equal(t, segments[0].ID, grownRange.ID)
	assert.Equal(t, 3, *grownRange.WalSegmentsCount)
	assert.Equal(t, appendedSegment.WalFilename, *grownRange.ToWalFilename)
	assert.Greater(t, grownRange.SizeMb, pageBefore.Backups[0].SizeMb)
	assert.WithinDuration(t, appendedSegment.ReceivedAt, *grownRange.CompletedAt, time.Millisecond)
}

func Test_GetBackups_WithManyWalSegments_RangeCountsAsOneRow(t *testing.T) {
	prereqs := createPhysicalControllerPrereqs(t)

	base := time.Now().UTC().Add(-2 * time.Hour)

	fullModel := physical_testing.NewTestCompletedFullBackup(
		prereqs.database.ID, prereqs.storage.ID, 1, walmath.LSN(0), walmath.LSN(segmentBytes))
	fullModel.CreatedAt = base
	full := physical_testing.CreateTestFullBackup(t, fullModel)

	segments := createContiguousWalSegments(t, prereqs, 1, 1, 60, base.Add(time.Minute))

	page := getBackups(t, prereqs, "?limit=2")

	assert.Equal(t, []uuid.UUID{segments[0].ID, full.ID}, collectBackupIDs(page))
	assert.Equal(t, 60, *page.Backups[0].WalSegmentsCount)
	assert.Empty(t, page.NextCursor)
}

// --- Cancel / delete edge cases ---------------------------------------------

func Test_DeleteBackup_WhenUnknownId_Returns404(t *testing.T) {
	prereqs := createPhysicalControllerPrereqs(t)

	test_utils.MakeDeleteRequest(t, prereqs.router,
		"/api/v1/backups/physical/backups/"+uuid.NewString(),
		"Bearer "+prereqs.user.Token, http.StatusNotFound)
}

func Test_CancelBackup_WhenUnknownId_Returns404(t *testing.T) {
	prereqs := createPhysicalControllerPrereqs(t)

	test_utils.MakePostRequest(t, prereqs.router,
		"/api/v1/backups/physical/backups/"+uuid.NewString()+"/cancel",
		"Bearer "+prereqs.user.Token, nil, http.StatusNotFound)
}

// A WAL segment has no trigger/cancel identity, so its id is not cancellable.
func Test_CancelBackup_WhenWalSegment_Returns404(t *testing.T) {
	prereqs := createPhysicalControllerPrereqs(t)

	walSegment := physical_testing.CreateTestWalSegment(t, physical_testing.NewTestWalSegment(
		prereqs.database.ID, prereqs.storage.ID, 1, "000000010000000000000001",
		walmath.LSN(segmentBytes), walmath.LSN(2*segmentBytes)))

	test_utils.MakePostRequest(t, prereqs.router,
		"/api/v1/backups/physical/backups/"+walSegment.ID.String()+"/cancel",
		"Bearer "+prereqs.user.Token, nil, http.StatusNotFound)
}

func Test_DeleteBackup_WhenInProgress_CancelsThenDeletes(t *testing.T) {
	prereqs := createPhysicalControllerPrereqs(t)

	full := physical_testing.CreateTestFullBackup(t,
		physical_testing.NewTestInProgressFullBackup(prereqs.database.ID, prereqs.storage.ID, 1))
	physical_testing.CreateTestInFlightClaim(t, prereqs.database.ID, full.ID, physical_enums.PhysicalBackupTypeFull)

	test_utils.MakeDeleteRequest(t, prereqs.router,
		"/api/v1/backups/physical/backups/"+full.ID.String(),
		"Bearer "+prereqs.user.Token, http.StatusNoContent)

	assertFullGone(t, full.ID)

	claim, err := physical_repositories.GetInFlightBackupRepository().FindByDatabaseID(prereqs.database.ID)
	require.NoError(t, err)
	assert.Nil(t, claim, "deleting an in-progress backup must release its claim")
}

func Test_TriggerBackup_Auto_Returns202(t *testing.T) {
	prereqs := createPhysicalControllerPrereqs(t)

	_, err := backups_config_physical.GetBackupConfigService().GetBackupConfigByDbId(prereqs.database.ID)
	require.NoError(t, err)

	test_utils.MakePostRequest(t, prereqs.router,
		"/api/v1/backups/physical/database/"+prereqs.database.ID.String()+"/trigger",
		"Bearer "+prereqs.user.Token,
		backups_dto_physical.TriggerBackupRequest{Type: backups_dto_physical.TriggerBackupTypeAuto},
		http.StatusAccepted)
}

// --- Restore stream happy path ----------------------------------------------

// Deep tar layout is covered by restore_stream's writer_test; this asserts the
// controller path end to end (token → resolve → stream → headers) and the
// per-backup invariant that no WAL rides along.
func Test_GetRestoreStream_WithBackupToken_StreamsTarWithoutWal(t *testing.T) {
	prereqs := createPhysicalControllerPrereqs(t)

	full := createStreamableFullBackup(t, prereqs)

	var tokenResponse backups_dto_physical.GenerateRestoreTokenResponse
	test_utils.MakePostRequestAndUnmarshal(t, prereqs.router,
		"/api/v1/backups/physical/backups/"+full.ID.String()+"/restore-token",
		"Bearer "+prereqs.user.Token, nil, http.StatusOK, &tokenResponse)

	response := test_utils.MakeGetRequest(t, prereqs.router,
		"/api/v1/backups/physical/restore-stream?token="+tokenResponse.Token,
		"", http.StatusOK)

	assert.Equal(t, "application/x-tar", response.Headers.Get("Content-Type"))
	assert.Contains(t, response.Headers.Get("Content-Disposition"), "restore-")

	names := readTarEntryNames(t, response.Body)
	assert.Contains(t, names, "full/base.tar.zst")
	assert.Contains(t, names, "full/backup_manifest")
	for _, name := range names {
		assert.False(t, strings.HasPrefix(name, "wal/"),
			"a per-backup restore must carry no WAL, got %q", name)
	}
}

// The recovery script is a static, secret-free helper served without auth; the UI
// tells the user to curl it. Lock its content-type and that it is a runnable POSIX
// script that drives pg_combinebackup.
func Test_GetRecoveryScript_ReturnsRunnableShellScript(t *testing.T) {
	router := newPhysicalControllerRouter()

	response := test_utils.MakeGetRequest(t, router,
		"/api/v1/backups/physical/recovery-script", "", http.StatusOK)

	assert.Contains(t, response.Headers.Get("Content-Type"), "text/x-shellscript")
	assert.True(t, strings.HasPrefix(string(response.Body), "#!/bin/sh"),
		"the recovery script must be a runnable POSIX sh script")
	assert.Contains(t, string(response.Body), "pg_combinebackup",
		"the recovery script must reconstruct the cluster")
	assert.Contains(t, string(response.Body), "--pg-bin",
		"the recovery script must accept a PostgreSQL bin path")
	assert.Contains(t, string(response.Body), "--target-time",
		"the recovery script must accept the PITR target time as an argument")
	assert.NotContains(t, string(response.Body), "ZSTD_BIN",
		"WAL is decompressed on the host up front; the script must not wire zstd into restore_command")
}

// A cluster whose source keeps its configuration outside the data directory (the Debian/Ubuntu
// layout) restores without postgresql.conf and refuses to start. The served script must diagnose
// that and instruct - never invent an access-control policy for the restored data (issue #685).
func Test_GetRecoveryScript_IncludesConfigFileCheck(t *testing.T) {
	router := newPhysicalControllerRouter()

	response := test_utils.MakeGetRequest(t, router,
		"/api/v1/backups/physical/recovery-script", "", http.StatusOK)

	servedScript := string(response.Body)

	assert.Contains(t, servedScript, "ACTION REQUIRED: the restored cluster has no configuration files",
		"the script must name the failure instead of leaving it to the server's startup error")
	assert.Contains(t, servedScript, "/etc/postgresql/",
		"the instructions must name where Debian/Ubuntu keeps the originals")
	assert.Contains(t, servedScript, "SHOW config_file",
		"the instructions must show how to ask the source cluster for the real paths")
	assert.Contains(t, servedScript, "pg_lsclusters",
		"the instructions must show how to tell two clusters on one host apart")
	// Blanket, not rule-shaped: any future mention deserves a second look, because the whole point
	// is that the restore never hands the user an unauthenticated rule to paste.
	assert.NotContains(t, servedScript, "trust",
		"the script must never emit a trust rule for the restored cluster")
}

func Test_GetRestoreStream_TokenIsSingleUse(t *testing.T) {
	prereqs := createPhysicalControllerPrereqs(t)

	full := createStreamableFullBackup(t, prereqs)

	var tokenResponse backups_dto_physical.GenerateRestoreTokenResponse
	test_utils.MakePostRequestAndUnmarshal(t, prereqs.router,
		"/api/v1/backups/physical/backups/"+full.ID.String()+"/restore-token",
		"Bearer "+prereqs.user.Token, nil, http.StatusOK, &tokenResponse)

	streamURL := "/api/v1/backups/physical/restore-stream?token=" + tokenResponse.Token

	test_utils.MakeGetRequest(t, prereqs.router, streamURL, "", http.StatusOK)
	test_utils.MakeGetRequest(t, prereqs.router, streamURL, "", http.StatusUnauthorized)
}

// A second heavy stream while the user already has one in progress is rejected
// with 409: downloads and restores share one per-user slot in the stream guard.
// The slot is occupied directly so the race is deterministic, not timing-based.
func Test_GetRestoreStream_WhenRestoreInProgress_Returns409(t *testing.T) {
	prereqs := createPhysicalControllerPrereqs(t)

	full := physical_testing.CreateTestFullBackup(t, physical_testing.NewTestCompletedFullBackup(
		prereqs.database.ID, prereqs.storage.ID, 1, walmath.LSN(0), walmath.LSN(segmentBytes)))

	// Issue the token while idle (issuance is gated on the same slot), then occupy
	// the slot before consuming it — the stream's slot acquisition fails with 409.
	var tokenResponse backups_dto_physical.GenerateRestoreTokenResponse
	test_utils.MakePostRequestAndUnmarshal(t, prereqs.router,
		"/api/v1/backups/physical/backups/"+full.ID.String()+"/restore-token",
		"Bearer "+prereqs.user.Token, nil, http.StatusOK, &tokenResponse)

	occupyUserStreamSlot(t, prereqs.user.UserID)

	test_utils.MakeGetRequest(t, prereqs.router,
		"/api/v1/backups/physical/restore-stream?token="+tokenResponse.Token,
		"", http.StatusConflict)
}

// Issuing a restore token while the user already has a stream in progress is
// rejected up front with 409, so they never mint a token they can't use.
func Test_GenerateBackupRestoreToken_WhenRestoreInProgress_Returns409(t *testing.T) {
	prereqs := createPhysicalControllerPrereqs(t)

	full := physical_testing.CreateTestFullBackup(t, physical_testing.NewTestCompletedFullBackup(
		prereqs.database.ID, prereqs.storage.ID, 1, walmath.LSN(0), walmath.LSN(segmentBytes)))

	occupyUserStreamSlot(t, prereqs.user.UserID)

	test_utils.MakePostRequest(t, prereqs.router,
		"/api/v1/backups/physical/backups/"+full.ID.String()+"/restore-token",
		"Bearer "+prereqs.user.Token, nil, http.StatusConflict)
}

// --- Helpers -----------------------------------------------------------------

// runRolePermissionMatrix exercises one endpoint across workspace roles via
// doRequest, which returns the HTTP status for the given Authorization header.
// successCode is what authorized roles receive; managersOnly=true means a viewer
// is rejected (state-changing ops), false means a viewer is allowed (reads). A
// non-member is always rejected; a global admin is always allowed.
func runRolePermissionMatrix(
	t *testing.T,
	prereqs *physicalControllerPrereqs,
	managersOnly bool,
	successCode int,
	doRequest func(authHeader string) int,
) {
	t.Helper()

	member := users_testing.CreateTestUser(t.Context(), users_enums.UserRoleMember)
	workspaces_testing.AddMemberToWorkspace(
		prereqs.workspace, member, users_enums.WorkspaceRoleMember, prereqs.user.Token, prereqs.router)

	viewer := addWorkspaceViewer(t, prereqs)
	nonMember := users_testing.CreateTestUser(t.Context(), users_enums.UserRoleMember)
	admin := users_testing.CreateTestUser(t.Context(), users_enums.UserRoleAdmin)

	viewerCode := successCode
	if managersOnly {
		viewerCode = http.StatusBadRequest
	}

	assert.Equal(t, successCode, doRequest("Bearer "+prereqs.user.Token), "owner")
	assert.Equal(t, successCode, doRequest("Bearer "+member.Token), "member")
	assert.Equal(t, viewerCode, doRequest("Bearer "+viewer.Token), "viewer")
	assert.Equal(t, http.StatusBadRequest, doRequest("Bearer "+nonMember.Token), "non-member")
	assert.Equal(t, successCode, doRequest("Bearer "+admin.Token), "global admin")
}

// occupyUserStreamSlot takes the user's shared download/restore slot to stand in
// for an in-flight heavy stream, releasing it at test end. Both token services
// embed the same guard, so this blocks token issuance and a second stream alike —
// letting the 409 path be asserted deterministically instead of via a timing race.
func occupyUserStreamSlot(t *testing.T, userID uuid.UUID) {
	t.Helper()

	restoreTokenService := backups_download.GetRestoreTokenService()
	if err := restoreTokenService.AcquireSlot(t.Context(), userID); err != nil {
		t.Fatalf("occupy stream slot: %v", err)
	}

	t.Cleanup(func() {
		restoreTokenService.ReleaseDownloadLock(t.Context(), userID)
	})
}

func addWorkspaceViewer(t *testing.T, prereqs *physicalControllerPrereqs) *users_dto.SignInResponseDTO {
	t.Helper()

	viewer := users_testing.CreateTestUser(t.Context(), users_enums.UserRoleMember)
	workspaces_testing.AddMemberToWorkspace(
		prereqs.workspace, viewer, users_enums.WorkspaceRoleViewer, prereqs.user.Token, prereqs.router)

	return viewer
}

func getBackups(
	t *testing.T,
	prereqs *physicalControllerPrereqs,
	query string,
) backups_dto_physical.GetPhysicalBackupsResponse {
	t.Helper()

	var response backups_dto_physical.GetPhysicalBackupsResponse
	test_utils.MakeGetRequestAndUnmarshal(t, prereqs.router,
		"/api/v1/backups/physical/database/"+prereqs.database.ID.String()+"/backups"+query,
		"Bearer "+prereqs.user.Token, http.StatusOK, &response)

	return response
}

// seedOneOfEachType creates one COMPLETED full, one incremental on it, and one
// committed WAL segment for the prereqs database — the minimal mix for filter and
// listing assertions.
func seedOneOfEachType(t *testing.T, prereqs *physicalControllerPrereqs) (
	*physical_models.PhysicalFullBackup,
	*physical_models.PhysicalIncrementalBackup,
	*physical_models.PhysicalWalSegment,
) {
	t.Helper()

	full := physical_testing.CreateTestFullBackup(t, physical_testing.NewTestCompletedFullBackup(
		prereqs.database.ID, prereqs.storage.ID, 1, walmath.LSN(0), walmath.LSN(segmentBytes)))
	incr := physical_testing.CreateTestIncrementalBackup(t, physical_testing.NewTestCompletedIncrementalBackup(
		prereqs.database.ID, prereqs.storage.ID, full.ID, nil, 1,
		walmath.LSN(segmentBytes), walmath.LSN(2*segmentBytes)))
	walSegment := physical_testing.CreateTestWalSegment(t, physical_testing.NewTestWalSegment(
		prereqs.database.ID, prereqs.storage.ID, 1, "000000010000000000000002",
		walmath.LSN(2*segmentBytes), walmath.LSN(3*segmentBytes)))

	return full, incr, walSegment
}

func collectBackupIDs(pages ...backups_dto_physical.GetPhysicalBackupsResponse) []uuid.UUID {
	var backupIDs []uuid.UUID

	for _, page := range pages {
		for _, backup := range page.Backups {
			backupIDs = append(backupIDs, backup.ID)
		}
	}

	return backupIDs
}

// createFullBackupsMinutesApart returns the created FULLs oldest first.
func createFullBackupsMinutesApart(
	t *testing.T,
	prereqs *physicalControllerPrereqs,
	count int,
	firstCreatedAt time.Time,
) []*physical_models.PhysicalFullBackup {
	t.Helper()

	fulls := make([]*physical_models.PhysicalFullBackup, 0, count)

	for index := range count {
		fullModel := physical_testing.NewTestCompletedFullBackup(
			prereqs.database.ID, prereqs.storage.ID, 1, walmath.LSN(0), walmath.LSN(segmentBytes))
		fullModel.CreatedAt = firstCreatedAt.Add(time.Duration(index) * time.Minute)
		fulls = append(fulls, physical_testing.CreateTestFullBackup(t, fullModel))
	}

	return fulls
}

// createCompletedFullBackupsInBulk inserts count FULL rows a second apart in
// one statement: saving them one by one through the repository takes tens of
// seconds at the page-limit cap.
func createCompletedFullBackupsInBulk(t *testing.T, prereqs *physicalControllerPrereqs, count int) {
	t.Helper()

	err := storage.GetDb().Exec(`
		INSERT INTO physical_full_backups
		    (database_id, storage_id, timeline_id, status, start_lsn, stop_lsn, created_at, completed_at)
		SELECT ?, ?, 1, ?, '0/0'::pg_lsn, '0/100'::pg_lsn,
		       NOW() - make_interval(secs => series_index), NOW() - make_interval(secs => series_index)
		FROM generate_series(1, ?) AS series_index`,
		prereqs.database.ID, prereqs.storage.ID, physical_enums.PhysicalBackupStatusCompleted, count,
	).Error
	require.NoError(t, err)
}

// createContiguousWalSegments commits count segments that follow each other
// without a gap, received a minute apart, and returns them in LSN order.
func createContiguousWalSegments(
	t *testing.T,
	prereqs *physicalControllerPrereqs,
	timelineID int,
	firstSegmentNo uint64,
	count int,
	firstReceivedAt time.Time,
) []*physical_models.PhysicalWalSegment {
	t.Helper()

	segments := make([]*physical_models.PhysicalWalSegment, 0, count)

	for index := range count {
		segmentNo := walmath.WalSegmentNo(firstSegmentNo + uint64(index))

		segmentModel := physical_testing.NewTestWalSegment(
			prereqs.database.ID, prereqs.storage.ID, timelineID,
			segmentNo.GetFilename(uint32(timelineID)), segmentNo.FirstLSN(), segmentNo.Next().FirstLSN())
		segmentModel.ReceivedAt = firstReceivedAt.Add(time.Duration(index) * time.Minute)

		segments = append(segments, physical_testing.CreateTestWalSegment(t, segmentModel))
	}

	return segments
}

// seedWalAroundFull commits four contiguous segments with a FULL created after
// the second one, so only the FULL separates the two halves in the list.
func seedWalAroundFull(t *testing.T, prereqs *physicalControllerPrereqs) (
	[]*physical_models.PhysicalWalSegment,
	*physical_models.PhysicalFullBackup,
	[]*physical_models.PhysicalWalSegment,
) {
	t.Helper()

	base := time.Now().UTC().Add(-1 * time.Hour)
	olderSegments := createContiguousWalSegments(t, prereqs, 1, 1, 2, base)

	fullModel := physical_testing.NewTestCompletedFullBackup(
		prereqs.database.ID, prereqs.storage.ID, 1,
		walmath.WalSegmentNo(3).FirstLSN()+physical_testing.FirstRecordOffset,
		walmath.WalSegmentNo(3).FirstLSN()+physical_testing.FirstRecordOffset+physical_testing.FullLSNSpan)
	fullModel.CreatedAt = base.Add(5 * time.Minute)
	full := physical_testing.CreateTestFullBackup(t, fullModel)

	newerSegments := createContiguousWalSegments(t, prereqs, 1, 3, 2, base.Add(10*time.Minute))

	return olderSegments, full, newerSegments
}

func createSecondPhysicalDatabase(t *testing.T, prereqs *physicalControllerPrereqs) *databases.Database {
	t.Helper()

	database := databases.CreateTestPhysicalPostgresDatabase(prereqs.workspace.ID, prereqs.notifier, "17")
	t.Cleanup(func() {
		physical_testing.DeleteAllPhysicalCatalogForDatabase(t, database.ID)
		databases.RemoveTestDatabase(t.Context(), database)
	})

	return database
}

func assertAuditLogContains(t *testing.T, workspaceID uuid.UUID, messageFragment, databaseName string) {
	t.Helper()

	logs, err := audit_logs.GetAuditLogService().GetWorkspaceAuditLogs(t.Context(),
		workspaceID, &audit_logs.GetAuditLogsRequest{Limit: 100, Offset: 0})
	require.NoError(t, err)

	for _, entry := range logs.AuditLogs {
		if strings.Contains(entry.Message, messageFragment) && strings.Contains(entry.Message, databaseName) {
			return
		}
	}

	t.Fatalf("audit log %q for database %q not found", messageFragment, databaseName)
}

// createStreamableFullBackup seeds a COMPLETED full whose stored artifact and
// manifest sidecar actually exist in storage, so the restore stream can assemble a
// real tar. NewTestCompletedFullBackup only writes a catalog row with no bytes
// behind it, which cannot be streamed. The artifact is a minimal zstd-compressed
// PGDATA tar; the manifest is stored raw, matching the executor's convention.
func createStreamableFullBackup(t *testing.T, prereqs *physicalControllerPrereqs) *physical_models.PhysicalFullBackup {
	t.Helper()

	full := physical_testing.NewTestCompletedFullBackup(
		prereqs.database.ID, prereqs.storage.ID, 1, walmath.LSN(0), walmath.LSN(segmentBytes))
	full.Encryption = backups_core_enums.BackupEncryptionNone
	full.Compression = physical_enums.PhysicalBackupCompressionZstd
	full.ManifestFileName = new(*full.FileName + ".manifest")

	blob := zstdTar(t, map[string]string{
		"PG_VERSION":  "17\n",
		"base/1/1259": "heap-bytes",
	})
	full.BackupSizeMb = new(float64(len(blob)) / (1024 * 1024))

	saveStorageObject(t, prereqs.storage, *full.FileName, blob)
	saveStorageObject(t, prereqs.storage, *full.ManifestFileName,
		[]byte(`{ "PostgreSQL-Backup-Manifest-Version": 2 }`))

	return physical_testing.CreateTestFullBackup(t, full)
}

func saveStorageObject(t *testing.T, storage *storages.Storage, name string, body []byte) {
	t.Helper()

	encryptor := encryption.GetFieldEncryptor()
	if err := storage.SaveFile(
		context.Background(), encryptor, logger.GetLogger(), name, bytes.NewReader(body)); err != nil {
		t.Fatalf("save storage object %s: %v", name, err)
	}

	t.Cleanup(func() { _ = storage.DeleteFile(t.Context(), encryptor, logger.GetLogger(), name) })
}

func zstdTar(t *testing.T, files map[string]string) []byte {
	t.Helper()

	var rawTar bytes.Buffer
	tarWriter := tar.NewWriter(&rawTar)
	for name, content := range files {
		require.NoError(t, tarWriter.WriteHeader(&tar.Header{
			Name: name, Mode: 0o600, Size: int64(len(content)), Typeflag: tar.TypeReg,
		}))
		_, err := tarWriter.Write([]byte(content))
		require.NoError(t, err)
	}
	require.NoError(t, tarWriter.Close())

	var compressed bytes.Buffer
	encoder, err := zstd.NewWriter(&compressed)
	require.NoError(t, err)
	_, err = encoder.Write(rawTar.Bytes())
	require.NoError(t, err)
	require.NoError(t, encoder.Close())

	return compressed.Bytes()
}

func readTarEntryNames(t *testing.T, body []byte) []string {
	t.Helper()

	var names []string
	tarReader := tar.NewReader(bytes.NewReader(body))
	for {
		header, err := tarReader.Next()
		if err == io.EOF {
			break
		}
		require.NoError(t, err)
		names = append(names, header.Name)
	}

	return names
}
