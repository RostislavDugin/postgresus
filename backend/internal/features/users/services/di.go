package users_services

import (
	"sync/atomic"

	"databasus-backend/internal/features/email"
	"databasus-backend/internal/features/encryption/secrets"
	users_oidc "databasus-backend/internal/features/users/oidc"
	users_repositories "databasus-backend/internal/features/users/repositories"
	"databasus-backend/internal/util/cache"
	"databasus-backend/internal/util/logger"
	"databasus-backend/internal/util/ratelimiter"
)

var userService = &UserService{
	users_repositories.GetUserRepository(),
	secrets.GetSecretKeyService(),
	settingsService,
	nil,
	email.GetEmailSMTPSender(),
	users_repositories.GetPasswordResetRepository(),
	users_repositories.GetTwoFactorRepository(),
	users_oidc.NewAttemptStore(cache.GetStore()),
	ratelimiter.GetCounter(),
	logger.GetLogger(),
}

var settingsService = &SettingsService{
	users_repositories.GetUsersSettingsRepository(),
	users_repositories.GetUserRepository(),
	nil,
	email.GetEmailSMTPSender(),
}

var signInCodeBackgroundService = &SignInCodeBackgroundService{
	userService,
	logger.GetLogger(),
	atomic.Bool{},
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

func GetSignInCodeBackgroundService() *SignInCodeBackgroundService {
	return signInCodeBackgroundService
}
