package users_controllers

import (
	users_services "postgresus-backend/internal/features/users/services"

	"golang.org/x/time/rate"
)

var userController = &UserController{
	users_services.GetUserService(),
	rate.NewLimiter(rate.Limit(3), 3), // 3 rps with 3 burst
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
