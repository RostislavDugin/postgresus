package users_services

import (
	"postgresus-backend/internal/features/encryption/secrets"
	users_repositories "postgresus-backend/internal/features/users/repositories"
)

var userService = &UserService{
	users_repositories.GetUserRepository(),
	secrets.GetSecretKeyService(),
	settingsService,
	nil,
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
