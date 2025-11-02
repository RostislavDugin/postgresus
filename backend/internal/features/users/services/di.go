package users_services

import (
	user_repositories "postgresus-backend/internal/features/users/repositories"
)

var secretKeyRepository = &user_repositories.SecretKeyRepository{}
var userRepository = &user_repositories.UserRepository{}
var usersSettingsRepository = &user_repositories.UsersSettingsRepository{}

var userService = &UserService{
	userRepository,
	secretKeyRepository,
	settingsService,
	nil,
}
var settingsService = &SettingsService{
	usersSettingsRepository,
	nil,
}
var managementService = &UserManagementService{
	userRepository,
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
