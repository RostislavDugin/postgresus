package users_controllers

import (
	users_services "databasus-backend/internal/features/users/services"
	"databasus-backend/internal/util/logger"
	"databasus-backend/internal/util/ratelimiter"
)

var userController = &UserController{
	users_services.GetUserService(),
	ratelimiter.GetCounter(),
	logger.GetLogger(),
}

var settingsController = &SettingsController{
	users_services.GetSettingsService(),
}

var managementController = &ManagementController{
	users_services.GetManagementService(),
}

func GetUserController() *UserController {
	return userController
}

func GetSettingsController() *SettingsController {
	return settingsController
}

func GetManagementController() *ManagementController {
	return managementController
}
