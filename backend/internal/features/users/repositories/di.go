package users_repositories

var secretKeyRepository = &SecretKeyRepository{}
var userRepository = &UserRepository{}
var usersSettingsRepository = &UsersSettingsRepository{}

func GetSecretKeyRepository() *SecretKeyRepository {
	return secretKeyRepository
}

func GetUserRepository() *UserRepository {
	return userRepository
}

func GetUsersSettingsRepository() *UsersSettingsRepository {
	return usersSettingsRepository
}
