package users_services

import (
	"databasus-backend/internal/features/email"
	"databasus-backend/internal/features/encryption/secrets"
	users_repositories "databasus-backend/internal/features/users/repositories"
	"databasus-backend/internal/util/logger"
)

var userService = &UserService{
	users_repositories.GetUserRepository(),
	secrets.GetSecretKeyService(),
	settingsService,
	nil,
	email.GetEmailSMTPSender(),
	users_repositories.GetPasswordResetRepository(),
	logger.GetLogger(),
}

var settingsService = &SettingsService{
	users_repositories.GetUsersSettingsRepository(),
	nil,
}

var managementService = &UserManagementService{
	users_repositories.GetUserRepository(),
	nil,
}

func GetUserService() *UserService {
	return userService
}

func GetSettingsService() *SettingsService {
	return settingsService
}

func GetManagementService() *UserManagementService {
	return managementService
}
