package verification_runs

import (
	"sync"
	"sync/atomic"

	"databasus-backend/internal/features/audit_logs"
	backuping_logical "databasus-backend/internal/features/backups/backups/backuping/logical"
	backups_services "databasus-backend/internal/features/backups/backups/services"
	"databasus-backend/internal/features/databases"
	"databasus-backend/internal/features/notifiers"
	verification_agents "databasus-backend/internal/features/verification/agents"
	verification_config "databasus-backend/internal/features/verification/config"
	workspaces_services "databasus-backend/internal/features/workspaces/services"
	"databasus-backend/internal/util/logger"
)

var verificationRepository = &VerificationRepository{}

var verificationService = &VerificationService{
	verificationRepository,
	databases.GetDatabaseService(),
	backups_services.GetBackupService(),
	verification_config.GetVerificationConfigService(),
	notifiers.GetNotifierService(),
	workspaces_services.GetWorkspaceService(),
	audit_logs.GetAuditLogService(),
	logger.GetLogger(),
}

var verificationScheduler = &VerificationScheduler{
	verificationRepository,
	verificationService,
	verification_config.GetVerificationConfigService(),
	verification_agents.GetAgentService(),
	backups_services.GetBackupService(),
	databases.GetDatabaseService(),
	logger.GetLogger(),
	atomic.Bool{},
}

var verificationController = &VerificationController{
	verificationService,
}

var verificationAgentController = &VerificationAgentController{
	verificationService,
	verification_agents.GetAgentService(),
	logger.GetLogger(),
}

func GetVerificationService() *VerificationService {
	return verificationService
}

func GetVerificationScheduler() *VerificationScheduler {
	return verificationScheduler
}

func GetVerificationController() *VerificationController {
	return verificationController
}

func GetVerificationAgentController() *VerificationAgentController {
	return verificationAgentController
}

var SetupDependencies = sync.OnceFunc(func() {
	verification_agents.GetAgentService().AddAgentHeartbeatedListener(verificationService)
	backuping_logical.GetBackupsScheduler().AddBackupCompletionListener(verificationService)
})
