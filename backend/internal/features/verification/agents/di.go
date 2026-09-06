package verification_agents

import (
	audit_logs "databasus-backend/internal/features/audit_logs"
	"databasus-backend/internal/util/logger"
	"databasus-backend/internal/util/ratelimiter"
)

var agentRepository = &AgentRepository{}

var agentService = &AgentService{
	agentRepository,
	audit_logs.GetAuditLogService(),
	ratelimiter.GetCounter(),
	logger.GetLogger(),
	nil,
}

var agentController = &AgentController{
	agentService,
}

var agentFacingController = &AgentFacingController{
	agentService,
}

func GetAgentService() *AgentService {
	return agentService
}

func GetAgentController() *AgentController {
	return agentController
}

func GetAgentFacingController() *AgentFacingController {
	return agentFacingController
}
