package users_repositories

var userRepository = &UserRepository{}
var usersSettingsRepository = &UsersSettingsRepository{}

func GetUserRepository() *UserRepository {
	return userRepository
}

func GetUsersSettingsRepository() *UsersSettingsRepository {
	return usersSettingsRepository
}
